package p2p_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/p2p"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleTx_ValidTransaction(t *testing.T) {
	minerAddr := "miner-A"
	w := testutil.MustCreateWallet(t)

	srv, ch, pool, _ := makeRelayTestNode(t, minerAddr)

	// Mine a block to create UTXOs for miner
	ctx := context.Background()
	_, err := ch.MineBlock(ctx, w.Address(), nil, 0)
	require.NoError(t, err)

	// Get the UTXO set for the wallet
	// Build a signed tx spending the miner reward
	// We need the utxo set from the node -- get it via chain
	utxoRepo := testutil.NewMockUTXORepo()
	// Instead of using the node's utxo set directly, build a tx payload manually
	// using the coinbase output from height 1
	latestBlock := ch.LatestBlock()
	var coinbaseTx *tx.Transaction
	for _, rawTx := range latestBlock.RawTransactions() {
		if t2, ok := rawTx.(*tx.Transaction); ok {
			coinbaseTx = t2
			break
		}
	}
	require.NotNil(t, coinbaseTx)
	_ = utxoRepo // unused, node has its own

	// Build a signed transaction spending coinbase output
	input := tx.NewTxInput(coinbaseTx.ID(), 0)
	output := tx.NewTxOutput(coinbaseTx.Outputs()[0].Value()-1000, "1RecipientAddr")
	spendTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{output})
	err = tx.SignTransaction(spendTx, w.PrivateKey())
	require.NoError(t, err)

	// Convert to payload and send via raw connection
	txPayload := p2p.TxPayloadFromDomain(spendTx)
	msg, err := p2p.NewMessage(p2p.CmdTx, txPayload)
	require.NoError(t, err)

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	require.NoError(t, p2p.WriteMessage(conn, msg))

	// Verify mempool accepted the transaction
	require.Eventually(t, func() bool {
		return pool.Count() == 1
	}, 3*time.Second, 50*time.Millisecond, "expected mempool count to be 1")
}

func TestHandleTx_InvalidPayload(t *testing.T) {
	srv, ch, pool, _ := makeRelayTestNode(t, "miner-A")

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	// Send malformed JSON as CmdTx
	badMsg := p2p.Message{Command: p2p.CmdTx, Payload: []byte(`{invalid json`)}
	require.NoError(t, p2p.WriteMessage(conn, badMsg))

	// Mempool should remain empty -- message rejected
	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, 0, pool.Count(), "mempool should be empty after invalid payload")
}

func TestHandleTx_RejectedByMempool(t *testing.T) {
	// Send a transaction with invalid signature -- mempool rejects it
	srv, ch, pool, _ := makeRelayTestNode(t, "miner-A")

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	// Create a fake unsigned tx (will fail signature verification)
	fakeHash := block.Hash{}
	input := tx.NewTxInput(fakeHash, 0)
	output := tx.NewTxOutput(100, "1SomeAddr")
	fakeTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{output})

	txPayload := p2p.TxPayloadFromDomain(fakeTx)
	msg, err := p2p.NewMessage(p2p.CmdTx, txPayload)
	require.NoError(t, err)

	require.NoError(t, p2p.WriteMessage(conn, msg))

	// Mempool should remain empty
	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, 0, pool.Count(), "mempool should reject unsigned transaction")
}

func TestHandleGetData_Block(t *testing.T) {
	srv, ch, _, _ := makeRelayTestNode(t, "miner-A")

	// Genesis block exists at height 0
	genesis := ch.LatestBlock()
	require.NotNil(t, genesis)

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	// Request genesis block via getdata
	inv := p2p.InvPayload{
		Type:   "block",
		Hashes: []string{genesis.Hash().String()},
	}
	getDataMsg, err := p2p.NewMessage(p2p.CmdGetData, inv)
	require.NoError(t, err)
	require.NoError(t, p2p.WriteMessage(conn, getDataMsg))

	// Read response -- should be a CmdBlock message
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	resp, err := p2p.ReadMessage(conn)
	require.NoError(t, err)
	require.Equal(t, p2p.CmdBlock, resp.Command)

	// Verify the block data
	var bp p2p.BlockPayload
	require.NoError(t, json.Unmarshal(resp.Payload, &bp))
	assert.Equal(t, genesis.Hash().String(), bp.Hash)
	assert.Equal(t, uint64(0), bp.Height)
}

func TestHandleGetData_Tx(t *testing.T) {
	minerAddr := "miner-A"
	w := testutil.MustCreateWallet(t)

	srv, ch, pool, _ := makeRelayTestNode(t, minerAddr)

	// Mine a block to create UTXOs
	ctx := context.Background()
	_, err := ch.MineBlock(ctx, w.Address(), nil, 0)
	require.NoError(t, err)

	// Build and add a transaction to mempool
	latestBlock := ch.LatestBlock()
	var coinbaseTx *tx.Transaction
	for _, rawTx := range latestBlock.RawTransactions() {
		if t2, ok := rawTx.(*tx.Transaction); ok {
			coinbaseTx = t2
			break
		}
	}
	require.NotNil(t, coinbaseTx)

	input := tx.NewTxInput(coinbaseTx.ID(), 0)
	output := tx.NewTxOutput(coinbaseTx.Outputs()[0].Value()-1000, "1RecipientAddr")
	spendTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{output})
	err = tx.SignTransaction(spendTx, w.PrivateKey())
	require.NoError(t, err)

	require.NoError(t, pool.Add(spendTx))
	require.Equal(t, 1, pool.Count())

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	// Request the tx via getdata
	inv := p2p.InvPayload{
		Type:   "tx",
		Hashes: []string{spendTx.ID().String()},
	}
	getDataMsg, err := p2p.NewMessage(p2p.CmdGetData, inv)
	require.NoError(t, err)
	require.NoError(t, p2p.WriteMessage(conn, getDataMsg))

	// Read response -- should be CmdTx
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	resp, err := p2p.ReadMessage(conn)
	require.NoError(t, err)
	require.Equal(t, p2p.CmdTx, resp.Command)

	// Verify tx data
	var tp p2p.TxPayload
	require.NoError(t, json.Unmarshal(resp.Payload, &tp))
	assert.Equal(t, spendTx.ID().String(), tp.ID)
}

func TestHandleGetData_NotFound(t *testing.T) {
	srv, ch, _, _ := makeRelayTestNode(t, "miner-A")

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	// Request a non-existent block hash
	inv := p2p.InvPayload{
		Type:   "block",
		Hashes: []string{"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
	}
	getDataMsg, err := p2p.NewMessage(p2p.CmdGetData, inv)
	require.NoError(t, err)
	require.NoError(t, p2p.WriteMessage(conn, getDataMsg))

	// Should NOT receive a response (block not found, no crash)
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, err = p2p.ReadMessage(conn)
	assert.Error(t, err, "expected timeout -- no response for missing block")
}

func TestHandleMessage_UnknownCommand(t *testing.T) {
	srv, ch, _, _ := makeRelayTestNode(t, "miner-A")

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	// Send a message with an unknown command byte
	unknownMsg := p2p.Message{Command: 0xFF, Payload: []byte(`{}`)}
	require.NoError(t, p2p.WriteMessage(conn, unknownMsg))

	// Connection should stay alive -- verify by sending another valid message
	// The unknown command is simply logged and ignored
	time.Sleep(200 * time.Millisecond)

	// Send a valid getdata request to verify the connection is still functional
	genesis := ch.LatestBlock()
	inv := p2p.InvPayload{
		Type:   "block",
		Hashes: []string{genesis.Hash().String()},
	}
	getDataMsg, err := p2p.NewMessage(p2p.CmdGetData, inv)
	require.NoError(t, err)
	require.NoError(t, p2p.WriteMessage(conn, getDataMsg))

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	resp, err := p2p.ReadMessage(conn)
	require.NoError(t, err)
	assert.Equal(t, p2p.CmdBlock, resp.Command, "connection should still work after unknown command")
}

func TestHandleGetData_InvalidPayload(t *testing.T) {
	srv, ch, _, _ := makeRelayTestNode(t, "miner-A")

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	// Send malformed JSON as CmdGetData
	badMsg := p2p.Message{Command: p2p.CmdGetData, Payload: []byte(`{invalid`)}
	require.NoError(t, p2p.WriteMessage(conn, badMsg))

	// Should not crash, no response expected
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, err := p2p.ReadMessage(conn)
	assert.Error(t, err, "expected timeout -- invalid payload silently rejected")
}

func TestOnBlockReceived_Callback(t *testing.T) {
	// Verify the OnBlockReceived callback is invoked when a valid block arrives
	srvA, _, _, _ := makeRelayTestNode(t, "miner-A")

	callbackCh := make(chan *block.Block, 1)
	srvA.OnBlockReceived(func(b *block.Block) {
		callbackCh <- b
	})

	srvB, chainB, _, _ := makeRelayTestNode(t, "miner-A")
	connectNodes(t, srvB, srvA)

	// Mine and broadcast from B
	ctx := context.Background()
	blk, err := chainB.MineBlock(ctx, "miner-A", nil, 0)
	require.NoError(t, err)
	srvB.BroadcastBlock(blk, "")

	// Verify callback fires on A
	select {
	case received := <-callbackCh:
		assert.Equal(t, blk.Hash(), received.Hash())
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for OnBlockReceived callback")
	}
}

func TestHandleInv_InvalidPayload(t *testing.T) {
	srv, ch, _, _ := makeRelayTestNode(t, "miner-A")

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	// Send malformed JSON as CmdInv
	badMsg := p2p.Message{Command: p2p.CmdInv, Payload: []byte(`{invalid`)}
	require.NoError(t, p2p.WriteMessage(conn, badMsg))

	// No crash, no response expected
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, err := p2p.ReadMessage(conn)
	assert.Error(t, err, "expected timeout -- invalid inv payload rejected")
}

func TestHandleGetBlocks_InvalidPayload(t *testing.T) {
	srv, ch, _, _ := makeRelayTestNode(t, "miner-A")

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	// Send malformed JSON as CmdGetBlocks
	badMsg := p2p.Message{Command: p2p.CmdGetBlocks, Payload: []byte(`{invalid`)}
	require.NoError(t, p2p.WriteMessage(conn, badMsg))

	// No crash, no response expected
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, err := p2p.ReadMessage(conn)
	assert.Error(t, err, "expected timeout -- invalid getblocks payload rejected")
}

func TestHandleBlock_InvalidPayload(t *testing.T) {
	srv, ch, _, _ := makeRelayTestNode(t, "miner-A")

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	// Send malformed JSON as CmdBlock
	badMsg := p2p.Message{Command: p2p.CmdBlock, Payload: []byte(`{invalid`)}
	require.NoError(t, p2p.WriteMessage(conn, badMsg))

	// No crash, connection stays alive
	time.Sleep(300 * time.Millisecond)

	// Verify connection is alive by sending a valid message
	genesis := ch.LatestBlock()
	inv := p2p.InvPayload{Type: "block", Hashes: []string{genesis.Hash().String()}}
	getDataMsg, err := p2p.NewMessage(p2p.CmdGetData, inv)
	require.NoError(t, err)
	require.NoError(t, p2p.WriteMessage(conn, getDataMsg))

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	resp, err := p2p.ReadMessage(conn)
	require.NoError(t, err)
	assert.Equal(t, p2p.CmdBlock, resp.Command)
}

func TestHandleVerack_AfterHandshake(t *testing.T) {
	// Sending a verack after handshake is a protocol violation (logged but no disconnect)
	srv, ch, _, _ := makeRelayTestNode(t, "miner-A")

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	// Send verack message (protocol violation after handshake)
	verack := p2p.Message{Command: p2p.CmdVerack, Payload: []byte("{}")}
	require.NoError(t, p2p.WriteMessage(conn, verack))

	// Connection should stay alive -- verack after handshake is just logged
	time.Sleep(200 * time.Millisecond)

	genesis := ch.LatestBlock()
	inv := p2p.InvPayload{Type: "block", Hashes: []string{genesis.Hash().String()}}
	getDataMsg, err := p2p.NewMessage(p2p.CmdGetData, inv)
	require.NoError(t, err)
	require.NoError(t, p2p.WriteMessage(conn, getDataMsg))

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	resp, err := p2p.ReadMessage(conn)
	require.NoError(t, err)
	assert.Equal(t, p2p.CmdBlock, resp.Command)
}

func TestMarkSeen_UnknownType(t *testing.T) {
	srv, _, _, _ := makeRelayTestNode(t, "miner-A")

	// Unknown type should return false (not already seen)
	result := srv.MarkSeen("unknown", "somehash")
	assert.False(t, result, "unknown type should return false")
}

func TestHandleGetData_InvalidBlockHash(t *testing.T) {
	// Send getdata with a valid JSON but invalid hash hex string
	srv, ch, _, _ := makeRelayTestNode(t, "miner-A")

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	inv := p2p.InvPayload{
		Type:   "block",
		Hashes: []string{"not-valid-hex-hash"},
	}
	getDataMsg, err := p2p.NewMessage(p2p.CmdGetData, inv)
	require.NoError(t, err)
	require.NoError(t, p2p.WriteMessage(conn, getDataMsg))

	// No response expected (invalid hash logged and skipped)
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, err = p2p.ReadMessage(conn)
	assert.Error(t, err, "expected timeout")
}

func TestHandleGetData_InvalidTxHash(t *testing.T) {
	// Send getdata for tx type with invalid hash hex string
	srv, ch, _, _ := makeRelayTestNode(t, "miner-A")

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	inv := p2p.InvPayload{
		Type:   "tx",
		Hashes: []string{"not-valid-hex-hash"},
	}
	getDataMsg, err := p2p.NewMessage(p2p.CmdGetData, inv)
	require.NoError(t, err)
	require.NoError(t, p2p.WriteMessage(conn, getDataMsg))

	// No response expected (invalid hash logged and skipped)
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, err = p2p.ReadMessage(conn)
	assert.Error(t, err, "expected timeout")
}

func TestHandleGetData_TxNotInMempool(t *testing.T) {
	// Request a valid tx hash that is not in the mempool
	srv, ch, _, _ := makeRelayTestNode(t, "miner-A")

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	inv := p2p.InvPayload{
		Type:   "tx",
		Hashes: []string{"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
	}
	getDataMsg, err := p2p.NewMessage(p2p.CmdGetData, inv)
	require.NoError(t, err)
	require.NoError(t, p2p.WriteMessage(conn, getDataMsg))

	// No response expected
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, err = p2p.ReadMessage(conn)
	assert.Error(t, err, "expected timeout -- tx not in mempool")
}

func TestHandleVersion_AfterHandshake(t *testing.T) {
	// Sending a version message after handshake is a protocol violation
	// The server should remove the peer
	srv, ch, _, _ := makeRelayTestNode(t, "miner-A")

	conn := dialAndHandshake(t, srv, ch)
	defer conn.Close()

	// Server should have 1 peer after handshake
	require.Eventually(t, func() bool {
		return srv.PeerCount() == 1
	}, 2*time.Second, 50*time.Millisecond)

	// Send version message (protocol violation after handshake)
	versionPayload := p2p.VersionPayload{Version: 1, Height: 0}
	versionMsg, err := p2p.NewMessage(p2p.CmdVersion, versionPayload)
	require.NoError(t, err)
	require.NoError(t, p2p.WriteMessage(conn, versionMsg))

	// Server should remove the peer (removePeer called)
	require.Eventually(t, func() bool {
		return srv.PeerCount() == 0
	}, 3*time.Second, 50*time.Millisecond, "peer should be removed after protocol violation")
}
