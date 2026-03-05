package p2p

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"testing"
	"time"
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
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	if got.Command != original.Command {
		t.Errorf("Command = %d; want %d", got.Command, original.Command)
	}
	if string(got.Payload) != string(original.Payload) {
		t.Errorf("Payload = %q; want %q", got.Payload, original.Payload)
	}
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
	if err == nil {
		t.Fatal("expected ErrMessageTooLarge, got nil")
	}
	if err != ErrMessageTooLarge {
		t.Errorf("expected ErrMessageTooLarge, got: %v", err)
	}
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
	if err == nil {
		t.Fatal("expected error on truncated frame, got nil")
	}
}

func TestVersionPayloadSerialization(t *testing.T) {
	original := VersionPayload{
		Version:     1,
		Height:      42,
		GenesisHash: "deadbeef",
		ListenPort:  3000,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded VersionPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, original)
	}
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
	if err != nil {
		t.Fatalf("ReadMessage from peer's connection failed: %v", err)
	}

	if got.Command != CmdVerack {
		t.Errorf("Command = %d; want %d", got.Command, CmdVerack)
	}

	// Now write a message to the peer's connection (simulating remote)
	testMsg := Message{Command: CmdVersion, Payload: []byte(`{"version":1}`)}
	if err := WriteMessage(client, testMsg); err != nil {
		t.Fatalf("WriteMessage to peer's connection failed: %v", err)
	}

	// The peer's read loop should invoke the handler
	select {
	case r := <-received:
		if r.Command != CmdVersion {
			t.Errorf("received Command = %d; want %d", r.Command, CmdVersion)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for handler callback")
	}

	peer.Stop()
}
