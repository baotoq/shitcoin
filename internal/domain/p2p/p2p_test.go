package p2p

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteReadMessageRoundTrip(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	original := Message{
		Command: CmdVersion,
		Payload: []byte(`{"version":1,"height":10,"genesis_hash":"abc","listen_port":3000}`),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- WriteMessage(client, original)
	}()

	got, err := ReadMessage(server)
	require.NoError(t, err)

	require.NoError(t, <-errCh)

	assert.Equal(t, original.Command, got.Command)
	assert.Equal(t, string(original.Payload), string(got.Payload))
}

func TestReadMessageTooLarge(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	// Write a length header that exceeds MaxMessageSize
	go func() {
		var header [4]byte
		binary.BigEndian.PutUint32(header[:], MaxMessageSize+1)
		client.Write(header[:])
	}()

	_, err := ReadMessage(server)
	assert.ErrorIs(t, err, ErrMessageTooLarge)
}

func TestReadMessageTruncated(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()

	// Write a valid length header but then close the connection (truncated frame)
	go func() {
		var header [4]byte
		binary.BigEndian.PutUint32(header[:], 10) // claim 10 bytes
		client.Write(header[:])
		client.Close() // truncate
	}()

	_, err := ReadMessage(server)
	require.Error(t, err)
}

func TestVersionPayloadSerialization(t *testing.T) {
	original := VersionPayload{
		Version:     1,
		Height:      42,
		GenesisHash: "deadbeef",
		ListenPort:  3000,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded VersionPayload
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, original, decoded)
}

func TestPeerStartSendStop(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()

	peer := NewPeer(server, "test-addr")

	received := make(chan Message, 1)
	peer.Start(func(p *Peer, msg Message) {
		received <- msg
	})

	// Send a message TO the peer (it should be written to the connection)
	msg := Message{Command: CmdVerack, Payload: []byte(`{}`)}
	peer.Send(msg)

	// Read what the peer wrote to its connection
	got, err := ReadMessage(client)
	require.NoError(t, err)

	assert.Equal(t, CmdVerack, got.Command)

	// Now write a message to the peer's connection (simulating remote)
	testMsg := Message{Command: CmdVersion, Payload: []byte(`{"version":1}`)}
	require.NoError(t, WriteMessage(client, testMsg))

	// The peer's read loop should invoke the handler
	select {
	case r := <-received:
		assert.Equal(t, CmdVersion, r.Command)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for handler callback")
	}

	peer.Stop()
}
