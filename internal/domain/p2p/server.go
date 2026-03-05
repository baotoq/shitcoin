package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/mempool"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
)

// handshakeTimeout is the deadline for completing the version handshake.
const handshakeTimeout = 10 * time.Second

// Server manages TCP connections, peer lifecycle, and the P2P protocol.
type Server struct {
	mu        sync.RWMutex
	peers     map[string]*Peer
	listener  net.Listener
	chain     *chain.Chain
	mempool   *mempool.Mempool
	utxoSet   *utxo.Set
	chainRepo chain.Repository
	listenPort int
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewServer creates a new P2P server.
func NewServer(ch *chain.Chain, pool *mempool.Mempool, us *utxo.Set, repo chain.Repository, port int) *Server {
	return &Server{
		peers:      make(map[string]*Peer),
		chain:      ch,
		mempool:    pool,
		utxoSet:    us,
		chainRepo:  repo,
		listenPort: port,
	}
}

// Start begins listening for incoming TCP connections on localhost:{port}.
func (s *Server) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	addr := fmt.Sprintf("localhost:%d", s.listenPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	s.listener = listener

	slog.Info("P2P server listening", "addr", addr)

	go s.acceptLoop()
	return nil
}

// acceptLoop accepts incoming TCP connections until the context is cancelled.
func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				slog.Error("accept error", "err", err)
				continue
			}
		}
		go s.handleInbound(conn)
	}
}

// Connect dials a remote peer and performs an outbound handshake.
func (s *Server) Connect(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, handshakeTimeout)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}

	peer := NewPeer(conn, addr)

	if err := s.outboundHandshake(peer); err != nil {
		peer.Stop()
		return fmt.Errorf("outbound handshake with %s: %w", addr, err)
	}

	s.registerPeer(peer)
	slog.Info("connected to peer", "addr", addr, "height", peer.Height())
	return nil
}

// handleInbound handles an incoming connection by performing the inbound handshake.
func (s *Server) handleInbound(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	peer := NewPeer(conn, remoteAddr)

	if err := s.inboundHandshake(peer); err != nil {
		slog.Warn("inbound handshake failed", "addr", remoteAddr, "err", err)
		peer.Stop()
		return
	}

	s.registerPeer(peer)
	slog.Info("accepted peer", "addr", remoteAddr, "height", peer.Height())
}

// outboundHandshake performs the version handshake as the initiating side.
// Flow: send Version -> receive Version -> check genesis -> send Verack -> receive Verack
func (s *Server) outboundHandshake(peer *Peer) error {
	conn := peer.conn

	// Set handshake deadline
	conn.SetDeadline(time.Now().Add(handshakeTimeout))
	defer conn.SetDeadline(time.Time{}) // clear deadline after handshake

	// Send our version
	versionMsg, err := s.buildVersionMessage()
	if err != nil {
		return fmt.Errorf("build version message: %w", err)
	}
	if err := WriteMessage(conn, versionMsg); err != nil {
		return fmt.Errorf("send version: %w", err)
	}

	// Receive their version
	msg, err := ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("receive version: %w", err)
	}
	if msg.Command != CmdVersion {
		return fmt.Errorf("%w: expected version, got command %d", ErrProtocolViolation, msg.Command)
	}

	var theirVersion VersionPayload
	if err := json.Unmarshal(msg.Payload, &theirVersion); err != nil {
		return fmt.Errorf("unmarshal version: %w", err)
	}

	// Check genesis hash compatibility
	if err := s.checkGenesis(theirVersion.GenesisHash); err != nil {
		return err
	}

	peer.SetHeight(theirVersion.Height)
	peer.SetVersion(theirVersion.Version)

	// Send verack
	verack := Message{Command: CmdVerack, Payload: []byte("{}")}
	if err := WriteMessage(conn, verack); err != nil {
		return fmt.Errorf("send verack: %w", err)
	}

	// Receive verack
	msg, err = ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("receive verack: %w", err)
	}
	if msg.Command != CmdVerack {
		return fmt.Errorf("%w: expected verack, got command %d", ErrProtocolViolation, msg.Command)
	}

	return nil
}

// inboundHandshake performs the version handshake as the receiving side.
// Flow: receive Version -> check genesis -> send Version + Verack -> receive Verack
func (s *Server) inboundHandshake(peer *Peer) error {
	conn := peer.conn

	// Set handshake deadline
	conn.SetDeadline(time.Now().Add(handshakeTimeout))
	defer conn.SetDeadline(time.Time{}) // clear deadline after handshake

	// Receive their version
	msg, err := ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("receive version: %w", err)
	}
	if msg.Command != CmdVersion {
		return fmt.Errorf("%w: expected version, got command %d", ErrProtocolViolation, msg.Command)
	}

	var theirVersion VersionPayload
	if err := json.Unmarshal(msg.Payload, &theirVersion); err != nil {
		return fmt.Errorf("unmarshal version: %w", err)
	}

	// Check genesis hash compatibility
	if err := s.checkGenesis(theirVersion.GenesisHash); err != nil {
		return err
	}

	peer.SetHeight(theirVersion.Height)
	peer.SetVersion(theirVersion.Version)

	// Send our version
	versionMsg, err := s.buildVersionMessage()
	if err != nil {
		return fmt.Errorf("build version message: %w", err)
	}
	if err := WriteMessage(conn, versionMsg); err != nil {
		return fmt.Errorf("send version: %w", err)
	}

	// Send verack
	verack := Message{Command: CmdVerack, Payload: []byte("{}")}
	if err := WriteMessage(conn, verack); err != nil {
		return fmt.Errorf("send verack: %w", err)
	}

	// Receive verack
	msg, err = ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("receive verack: %w", err)
	}
	if msg.Command != CmdVerack {
		return fmt.Errorf("%w: expected verack, got command %d", ErrProtocolViolation, msg.Command)
	}

	return nil
}

// buildVersionMessage creates a version message with the current chain state.
func (s *Server) buildVersionMessage() (Message, error) {
	genesisHash := ""
	if latest := s.chain.LatestBlock(); latest != nil {
		// Walk to genesis is expensive; for now, we use the chain's genesis
		// which is the block at height 0. We store genesis hash at startup.
		genesisHash = s.getGenesisHash()
	}

	payload := VersionPayload{
		Version:     ProtocolVersion,
		Height:      s.chain.Height(),
		GenesisHash: genesisHash,
		ListenPort:  s.listenPort,
	}

	return NewMessage(CmdVersion, payload)
}

// getGenesisHash returns the genesis block hash as a hex string.
func (s *Server) getGenesisHash() string {
	ctx := context.Background()
	genesis, err := s.chainRepo.GetBlockByHeight(ctx, 0)
	if err != nil {
		return ""
	}
	return genesis.Hash().String()
}

// checkGenesis validates that the remote peer has a compatible genesis hash.
func (s *Server) checkGenesis(remoteGenesisHash string) error {
	ourGenesis := s.getGenesisHash()
	// If either side has no genesis (empty chain), skip check
	if ourGenesis == "" || remoteGenesisHash == "" {
		return nil
	}
	if ourGenesis != remoteGenesisHash {
		return fmt.Errorf("%w: ours=%s theirs=%s", ErrIncompatibleGenesis, ourGenesis[:16], remoteGenesisHash[:min(16, len(remoteGenesisHash))])
	}
	return nil
}

// registerPeer adds a peer to the registry and starts its read/write loops.
func (s *Server) registerPeer(peer *Peer) {
	s.mu.Lock()
	s.peers[peer.Addr()] = peer
	s.mu.Unlock()

	peer.Start(s.handleMessage)
}

// removePeer removes a peer from the registry.
func (s *Server) removePeer(addr string) {
	s.mu.Lock()
	if p, ok := s.peers[addr]; ok {
		p.Stop()
		delete(s.peers, addr)
	}
	s.mu.Unlock()
}

// Broadcast sends a message to all connected peers except the excluded address.
func (s *Server) Broadcast(msg Message, excludeAddr string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for addr, peer := range s.peers {
		if addr != excludeAddr {
			peer.Send(msg)
		}
	}
}

// Stop gracefully shuts down the server, closing the listener and all peers.
func (s *Server) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.listener != nil {
		s.listener.Close()
	}

	s.mu.Lock()
	for addr, peer := range s.peers {
		peer.Stop()
		delete(s.peers, addr)
	}
	s.mu.Unlock()

	slog.Info("P2P server stopped")
}

// PeerCount returns the number of connected peers.
func (s *Server) PeerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.peers)
}

// ListenAddr returns the listener's address (useful for tests with port 0).
func (s *Server) ListenAddr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}
