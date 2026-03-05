package p2p_test

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/mempool"
	"github.com/baotoq/shitcoin/internal/domain/p2p"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fullMockChainRepo stores all blocks for relay test scenarios.
type fullMockChainRepo struct {
	mu       sync.RWMutex
	blocks   map[block.Hash]*block.Block
	byHeight map[uint64]*block.Block
	latest   *block.Block
}

func newFullMockChainRepo() *fullMockChainRepo {
	return &fullMockChainRepo{
		blocks:   make(map[block.Hash]*block.Block),
		byHeight: make(map[uint64]*block.Block),
	}
}

func (m *fullMockChainRepo) SaveBlock(_ context.Context, b *block.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blocks[b.Hash()] = b
	m.byHeight[b.Height()] = b
	m.latest = b
	return nil
}

func (m *fullMockChainRepo) SaveBlockWithUTXOs(_ context.Context, b *block.Block, _ *utxo.UndoEntry) error {
	return m.SaveBlock(context.Background(), b)
}

func (m *fullMockChainRepo) GetBlock(_ context.Context, hash block.Hash) (*block.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if b, ok := m.blocks[hash]; ok {
		return b, nil
	}
	return nil, chain.ErrChainEmpty
}

func (m *fullMockChainRepo) GetBlockByHeight(_ context.Context, height uint64) (*block.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if b, ok := m.byHeight[height]; ok {
		return b, nil
	}
	return nil, chain.ErrChainEmpty
}

func (m *fullMockChainRepo) GetLatestBlock(_ context.Context) (*block.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.latest != nil {
		return m.latest, nil
	}
	return nil, chain.ErrChainEmpty
}

func (m *fullMockChainRepo) GetChainHeight(_ context.Context) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.latest != nil {
		return m.latest.Height(), nil
	}
	return 0, chain.ErrChainEmpty
}

func (m *fullMockChainRepo) GetBlocksInRange(_ context.Context, start, end uint64) ([]*block.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*block.Block
	for h := start; h <= end; h++ {
		if b, ok := m.byHeight[h]; ok {
			result = append(result, b)
		}
	}
	return result, nil
}

func (m *fullMockChainRepo) GetUndoEntry(_ context.Context, blockHeight uint64) (*utxo.UndoEntry, error) {
	return nil, utxo.ErrUndoEntryNotFound
}

func (m *fullMockChainRepo) DeleteBlocksAbove(_ context.Context, height uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for h, b := range m.byHeight {
		if h > height {
			delete(m.blocks, b.Hash())
			delete(m.byHeight, h)
		}
	}
	if b, ok := m.byHeight[height]; ok {
		m.latest = b
	}
	return nil
}

// mockUTXORepo implements utxo.Repository in-memory.
type mockUTXORepo struct {
	mu    sync.Mutex
	utxos map[string]utxo.UTXO // key = "txid_hex:vout"
}

func newMockUTXORepo() *mockUTXORepo {
	return &mockUTXORepo{utxos: make(map[string]utxo.UTXO)}
}

func (r *mockUTXORepo) utxoKey(txID block.Hash, vout uint32) string {
	return txID.String() + ":" + string(rune(vout+'0'))
}

func (r *mockUTXORepo) Put(u utxo.UTXO) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.utxos[r.utxoKey(u.TxID(), u.Vout())] = u
	return nil
}

func (r *mockUTXORepo) Get(txID block.Hash, vout uint32) (utxo.UTXO, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.utxos[r.utxoKey(txID, vout)]; ok {
		return u, nil
	}
	return utxo.UTXO{}, utxo.ErrUTXONotFound
}

func (r *mockUTXORepo) Delete(txID block.Hash, vout uint32) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.utxos, r.utxoKey(txID, vout))
	return nil
}

func (r *mockUTXORepo) GetByAddress(address string) ([]utxo.UTXO, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []utxo.UTXO
	for _, u := range r.utxos {
		if u.Address() == address {
			result = append(result, u)
		}
	}
	return result, nil
}

func (r *mockUTXORepo) SaveUndoEntry(_ *utxo.UndoEntry) error { return nil }
func (r *mockUTXORepo) GetUndoEntry(_ uint64) (*utxo.UndoEntry, error) {
	return nil, utxo.ErrUndoEntryNotFound
}
func (r *mockUTXORepo) DeleteUndoEntry(_ uint64) error { return nil }

// makeRelayTestNode creates a fully wired test node with UTXO support.
func makeRelayTestNode(t *testing.T, minerAddr string) (*p2p.Server, *chain.Chain, *mempool.Mempool, *fullMockChainRepo) {
	t.Helper()

	repo := newFullMockChainRepo()
	utxoRepo := newMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1, // very low for fast test mining
		GenesisMessage:    "relay-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	ctx := context.Background()
	require.NoError(t, ch.Initialize(ctx, minerAddr))

	pool := mempool.New(utxoSet)

	srv := p2p.NewServer(ch, pool, utxoSet, repo, 0)
	require.NoError(t, srv.Start(ctx))

	t.Cleanup(func() {
		srv.Stop()
	})

	return srv, ch, pool, repo
}

// connectNodes connects two servers and waits for the handshake to complete.
func connectNodes(t *testing.T, from, to *p2p.Server) {
	t.Helper()
	require.NoError(t, from.Connect(to.ListenAddr()))
	// Brief wait for inbound handling
	time.Sleep(200 * time.Millisecond)
}

// dialAndHandshake connects to a server, completes the inbound version handshake,
// and returns the raw connection. The inbound handshake flow (from the server's POV)
// is: receive Version -> check genesis -> send Version + Verack -> receive Verack.
// So as the client we: send Version -> receive Version -> receive Verack -> send Verack.
func dialAndHandshake(t *testing.T, srv *p2p.Server, ch *chain.Chain) net.Conn {
	t.Helper()

	conn, err := net.DialTimeout("tcp", srv.ListenAddr(), 2*time.Second)
	require.NoError(t, err)

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Send our version first (server's inbound handshake expects to receive version first)
	versionPayload := p2p.VersionPayload{
		Version:    1,
		Height:     ch.Height(),
		ListenPort: 0,
	}
	versionPayload.GenesisHash = ""

	versionMsg, err := p2p.NewMessage(p2p.CmdVersion, versionPayload)
	require.NoError(t, err)
	require.NoError(t, p2p.WriteMessage(conn, versionMsg))

	// Receive server's version
	msg, err := p2p.ReadMessage(conn)
	require.NoError(t, err)
	require.Equal(t, p2p.CmdVersion, msg.Command)

	// Receive server's verack
	msg, err = p2p.ReadMessage(conn)
	require.NoError(t, err)
	require.Equal(t, p2p.CmdVerack, msg.Command)

	// Send our verack
	verack := p2p.Message{Command: p2p.CmdVerack, Payload: []byte("{}")}
	require.NoError(t, p2p.WriteMessage(conn, verack))

	// Clear deadline
	conn.SetDeadline(time.Time{})

	return conn
}

func TestBlockBroadcast(t *testing.T) {
	// Two nodes with the same genesis. Mine a block on A, verify B receives it.
	srvA, chainA, _, _ := makeRelayTestNode(t, "miner-A")
	srvB, chainB, _, _ := makeRelayTestNode(t, "miner-A") // same miner for matching genesis

	connectNodes(t, srvB, srvA)

	// Mine a block on node A
	ctx := context.Background()
	blk, err := chainA.MineBlock(ctx, "miner-A", nil)
	require.NoError(t, err)

	// Broadcast the mined block
	srvA.BroadcastBlock(blk, "")

	// Wait for propagation
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			require.FailNow(t, "timed out waiting for block propagation",
				"B height=%d, want=%d", chainB.Height(), blk.Height())
		default:
			if chainB.Height() >= blk.Height() {
				return // success
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func TestBlockValidation_RejectInvalidPoW(t *testing.T) {
	// Send a block with invalid PoW to a node. Verify it's rejected.
	_, chainA, _, _ := makeRelayTestNode(t, "miner-A")
	srvB, chainB, _, _ := makeRelayTestNode(t, "miner-A")

	// Get the genesis block hash from A's chain
	genesis := chainA.LatestBlock()

	// Create a block with extremely high difficulty (bits=255) so nonce=0 won't pass PoW
	fakeTxs := make([]any, 0)
	badBlock, err := block.NewBlock(genesis.Hash(), 1, 255, fakeTxs, block.Hash{})
	require.NoError(t, err)
	// Don't mine it -- PoW is invalid

	// Send the invalid block directly via raw connection
	payload := p2p.BlockPayloadFromDomain(badBlock)
	msg, err := p2p.NewMessage(p2p.CmdBlock, payload)
	require.NoError(t, err)

	conn := dialAndHandshake(t, srvB, chainA)
	defer conn.Close()

	require.NoError(t, p2p.WriteMessage(conn, msg))

	// Wait briefly and verify B's height did NOT increase
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, uint64(0), chainB.Height(), "block should have been rejected")
}

func TestSeenTracking(t *testing.T) {
	// Send same inv twice. Verify only one getdata request is sent back.
	srvA, chainA, _, _ := makeRelayTestNode(t, "miner-A")

	conn := dialAndHandshake(t, srvA, chainA)
	defer conn.Close()

	fakeBlockHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	inv := p2p.InvPayload{
		Type:   "block",
		Hashes: []string{fakeBlockHash},
	}
	invMsg, err := p2p.NewMessage(p2p.CmdInv, inv)
	require.NoError(t, err)

	// Send first inv
	require.NoError(t, p2p.WriteMessage(conn, invMsg))

	// Read the getdata response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	resp, err := p2p.ReadMessage(conn)
	require.NoError(t, err)
	require.Equal(t, p2p.CmdGetData, resp.Command)

	// Send second identical inv
	require.NoError(t, p2p.WriteMessage(conn, invMsg))

	// Should NOT get another getdata (already seen)
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, err = p2p.ReadMessage(conn)
	assert.Error(t, err, "expected no response for second inv (already seen)")
	// timeout error is expected -- means no message was sent
}
