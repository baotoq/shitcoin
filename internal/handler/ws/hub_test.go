package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/baotoq/shitcoin/internal/domain/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHub_RegisterClient(t *testing.T) {
	bus := events.NewBus()
	hub := NewHub(bus)

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 256),
	}

	hub.register <- client

	// Give hub goroutine time to process
	time.Sleep(10 * time.Millisecond)

	hub.mu.RLock()
	_, exists := hub.clients[client]
	hub.mu.RUnlock()

	assert.True(t, exists, "client should be registered in hub")
}

func TestHub_UnregisterClient(t *testing.T) {
	bus := events.NewBus()
	hub := NewHub(bus)

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 256),
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	hub.unregister <- client
	time.Sleep(10 * time.Millisecond)

	hub.mu.RLock()
	_, exists := hub.clients[client]
	hub.mu.RUnlock()

	assert.False(t, exists, "client should be removed from hub after unregister")
}

func TestHub_BroadcastToAllClients(t *testing.T) {
	bus := events.NewBus()
	hub := NewHub(bus)

	client1 := &Client{hub: hub, send: make(chan []byte, 256)}
	client2 := &Client{hub: hub, send: make(chan []byte, 256)}

	hub.register <- client1
	hub.register <- client2
	time.Sleep(10 * time.Millisecond)

	msg := []byte(`{"type":"test","payload":"hello"}`)
	hub.broadcast <- msg
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, len(client1.send), "client1 should receive broadcast")
	assert.Equal(t, 1, len(client2.send), "client2 should receive broadcast")

	got1 := <-client1.send
	got2 := <-client2.send
	assert.Equal(t, msg, got1)
	assert.Equal(t, msg, got2)
}

func TestHub_DropsMessageWhenClientFull(t *testing.T) {
	bus := events.NewBus()
	hub := NewHub(bus)

	// Create client with tiny buffer
	client := &Client{hub: hub, send: make(chan []byte, 1)}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Fill the buffer
	hub.broadcast <- []byte(`{"type":"msg1"}`)
	time.Sleep(10 * time.Millisecond)

	// This should be dropped (non-blocking), and client should be evicted
	hub.broadcast <- []byte(`{"type":"msg2"}`)
	time.Sleep(10 * time.Millisecond)

	hub.mu.RLock()
	_, exists := hub.clients[client]
	hub.mu.RUnlock()

	assert.False(t, exists, "slow client should be evicted when send channel is full")
}

func TestHub_ForwardsEventBusEventsAsJSON(t *testing.T) {
	bus := events.NewBus()
	hub := NewHub(bus)

	client := &Client{hub: hub, send: make(chan []byte, 256)}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Publish an event on the event bus
	bus.Publish(events.Event{
		Type:    events.EventNewBlock,
		Payload: map[string]string{"hash": "abc123"},
	})

	// Wait for event to propagate through hub
	select {
	case msg := <-client.send:
		var wsMsg WSMessage
		err := json.Unmarshal(msg, &wsMsg)
		require.NoError(t, err)
		assert.Equal(t, string(events.EventNewBlock), wsMsg.Type)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for event bus message on client send channel")
	}
}
