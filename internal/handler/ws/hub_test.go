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

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		_, ok := hub.clients[client]
		return ok
	}, time.Second, 10*time.Millisecond, "client should be registered in hub")
}

func TestHub_UnregisterClient(t *testing.T) {
	bus := events.NewBus()
	hub := NewHub(bus)

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 256),
	}

	hub.register <- client

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		_, ok := hub.clients[client]
		return ok
	}, time.Second, 10*time.Millisecond, "client should be registered")

	hub.unregister <- client

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		_, ok := hub.clients[client]
		return !ok
	}, time.Second, 10*time.Millisecond, "client should be removed from hub after unregister")
}

func TestHub_BroadcastToAllClients(t *testing.T) {
	bus := events.NewBus()
	hub := NewHub(bus)

	client1 := &Client{hub: hub, send: make(chan []byte, 256)}
	client2 := &Client{hub: hub, send: make(chan []byte, 256)}

	hub.register <- client1
	hub.register <- client2

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.clients) == 2
	}, time.Second, 10*time.Millisecond, "both clients should be registered")

	msg := []byte(`{"type":"test","payload":"hello"}`)
	hub.broadcast <- msg

	require.Eventually(t, func() bool {
		return len(client1.send) == 1 && len(client2.send) == 1
	}, time.Second, 10*time.Millisecond, "both clients should receive broadcast")

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

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		_, ok := hub.clients[client]
		return ok
	}, time.Second, 10*time.Millisecond, "client should be registered")

	// Fill the buffer
	hub.broadcast <- []byte(`{"type":"msg1"}`)

	require.Eventually(t, func() bool {
		return len(client.send) == 1
	}, time.Second, 10*time.Millisecond, "first message should arrive")

	// This should be dropped (non-blocking), and client should be evicted
	hub.broadcast <- []byte(`{"type":"msg2"}`)

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		_, ok := hub.clients[client]
		return !ok
	}, time.Second, 10*time.Millisecond, "slow client should be evicted when send channel is full")
}

func TestHub_ForwardsEventBusEventsAsJSON(t *testing.T) {
	bus := events.NewBus()
	hub := NewHub(bus)

	client := &Client{hub: hub, send: make(chan []byte, 256)}

	hub.register <- client

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		_, ok := hub.clients[client]
		return ok
	}, time.Second, 10*time.Millisecond, "client should be registered")

	// Publish events in a loop until one arrives (subscriber goroutine may not be ready yet)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				bus.Publish(events.Event{
					Type:    events.EventNewBlock,
					Payload: map[string]string{"hash": "abc123"},
				})
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	// Wait for event to propagate through hub to client
	require.Eventually(t, func() bool {
		return len(client.send) > 0
	}, time.Second, 10*time.Millisecond, "event bus message should arrive on client send channel")
	close(done)

	msg := <-client.send
	var wsMsg WSMessage
	err := json.Unmarshal(msg, &wsMsg)
	require.NoError(t, err)
	assert.Equal(t, string(events.EventNewBlock), wsMsg.Type)
}
