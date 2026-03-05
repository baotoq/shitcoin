package p2p

import (
	"log/slog"
	"net"
	"sync"
)

// Peer represents a connected remote node with send/receive goroutines.
type Peer struct {
	conn    net.Conn
	addr    string
	sendCh  chan Message
	height  uint64
	version uint32
	done    chan struct{}
	once    sync.Once
}

// NewPeer creates a new Peer wrapping the given connection.
func NewPeer(conn net.Conn, addr string) *Peer {
	return &Peer{
		conn:   conn,
		addr:   addr,
		sendCh: make(chan Message, 64),
		done:   make(chan struct{}),
	}
}

// Addr returns the peer's address.
func (p *Peer) Addr() string {
	return p.addr
}

// Height returns the peer's last known chain height.
func (p *Peer) Height() uint64 {
	return p.height
}

// SetHeight updates the peer's known chain height.
func (p *Peer) SetHeight(h uint64) {
	p.height = h
}

// SetVersion updates the peer's protocol version.
func (p *Peer) SetVersion(v uint32) {
	p.version = v
}

// Start launches read and write goroutines for the peer.
// The handler function is called for each message received from the remote peer.
func (p *Peer) Start(handler func(*Peer, Message)) {
	go p.writeLoop()
	go p.readLoop(handler)
}

// Send enqueues a message for the peer. Non-blocking: drops if buffer is full.
func (p *Peer) Send(msg Message) {
	select {
	case p.sendCh <- msg:
	default:
		slog.Warn("peer send buffer full, dropping message", "addr", p.addr, "cmd", msg.Command)
	}
}

// Stop closes the peer connection and signals goroutines to exit.
func (p *Peer) Stop() {
	p.once.Do(func() {
		close(p.done)
		p.conn.Close()
	})
}

// writeLoop reads messages from sendCh and writes them to the connection.
func (p *Peer) writeLoop() {
	defer p.Stop()
	for msg := range p.sendCh {
		if err := WriteMessage(p.conn, msg); err != nil {
			slog.Debug("peer write error", "addr", p.addr, "err", err)
			return
		}
	}
}

// readLoop reads messages from the connection and calls the handler.
func (p *Peer) readLoop(handler func(*Peer, Message)) {
	defer p.Stop()
	for {
		msg, err := ReadMessage(p.conn)
		if err != nil {
			select {
			case <-p.done:
				// Peer was stopped intentionally
			default:
				slog.Debug("peer read error", "addr", p.addr, "err", err)
			}
			return
		}
		handler(p, msg)
	}
}
