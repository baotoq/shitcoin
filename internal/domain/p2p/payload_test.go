package p2p_test

import (
	"net"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/p2p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createPipe creates a net.Pipe and registers cleanup.
func createPipe(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	c, s := net.Pipe()
	t.Cleanup(func() {
		c.Close()
		s.Close()
	})
	return c, s
}

func TestToTransaction_InvalidHex(t *testing.T) {
	tests := []struct {
		name    string
		payload p2p.TxPayload
	}{
		{
			name: "invalid tx ID hex",
			payload: p2p.TxPayload{
				ID:      "not-valid-hex",
				Inputs:  nil,
				Outputs: nil,
			},
		},
		{
			name: "invalid input txid hex",
			payload: p2p.TxPayload{
				ID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Inputs: []p2p.TxInputPayload{
					{TxID: "zzz-invalid-hex", Vout: 0},
				},
				Outputs: nil,
			},
		},
		{
			name: "invalid signature hex",
			payload: p2p.TxPayload{
				ID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Inputs: []p2p.TxInputPayload{
					{
						TxID:      "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
						Vout:      0,
						Signature: "not-valid-hex-sig",
					},
				},
				Outputs: nil,
			},
		},
		{
			name: "invalid pubkey hex",
			payload: p2p.TxPayload{
				ID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Inputs: []p2p.TxInputPayload{
					{
						TxID:      "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
						Vout:      0,
						Signature: "deadbeef",
						PubKey:    "not-valid-hex-pk",
					},
				},
				Outputs: nil,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.payload.ToTransaction()
			require.Error(t, err)
		})
	}
}

func TestToBlock_InvalidHex(t *testing.T) {
	tests := []struct {
		name    string
		payload p2p.BlockPayload
	}{
		{
			name: "invalid block hash hex",
			payload: p2p.BlockPayload{
				Hash:   "not-valid-hex",
				Height: 1,
				Header: p2p.HeaderPayload{
					PrevBlockHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					MerkleRoot:    "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				},
			},
		},
		{
			name: "invalid prev block hash hex",
			payload: p2p.BlockPayload{
				Hash:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Height: 1,
				Header: p2p.HeaderPayload{
					PrevBlockHash: "not-valid-hex",
					MerkleRoot:    "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				},
			},
		},
		{
			name: "invalid merkle root hex",
			payload: p2p.BlockPayload{
				Hash:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Height: 1,
				Header: p2p.HeaderPayload{
					PrevBlockHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
					MerkleRoot:    "not-valid-hex",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.payload.ToBlock()
			require.Error(t, err)
		})
	}
}

func TestNewMessage_MarshalError(t *testing.T) {
	// Channels cannot be marshaled to JSON
	unmarshalable := make(chan int)
	_, err := p2p.NewMessage(p2p.CmdTx, unmarshalable)
	assert.Error(t, err)
}

func TestWriteMessage_ClosedConn(t *testing.T) {
	// Write to a closed pipe end should return an error
	client, server := createPipe(t)
	server.Close() // close server end so writes to client fail

	msg := p2p.Message{Command: p2p.CmdVerack, Payload: []byte(`{}`)}
	err := p2p.WriteMessage(client, msg)
	assert.Error(t, err)
}

func TestPeerSend_AfterStop(t *testing.T) {
	// Sending after Stop should not panic
	_, server := createPipe(t)

	peer := p2p.NewPeer(server, "test-stopped-peer")
	peer.Start(func(p *p2p.Peer, msg p2p.Message) {})
	peer.Stop()

	// Should not panic -- message is either dropped or channel is still drainable
	assert.NotPanics(t, func() {
		msg := p2p.Message{Command: p2p.CmdVerack, Payload: []byte(`{}`)}
		peer.Send(msg)
	})
}
