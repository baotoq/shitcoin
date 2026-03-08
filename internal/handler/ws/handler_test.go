package ws

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/baotoq/shitcoin/internal/domain/events"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dialWs(t *testing.T, server *httptest.Server) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	return conn
}

func waitForClients(t *testing.T, hub *Hub, count int) {
	t.Helper()
	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.clients) == count
	}, time.Second, 10*time.Millisecond)
}

func TestServeWs_ClientReceivesBroadcast(t *testing.T) {
	bus := events.NewBus()
	hub := NewHub(bus)

	server := httptest.NewServer(ServeWs(hub))
	defer server.Close()

	conn := dialWs(t, server)
	defer conn.Close()

	waitForClients(t, hub, 1)

	// Publish event on bus
	bus.Publish(events.Event{
		Type:    events.EventNewBlock,
		Payload: map[string]string{"hash": "abc123"},
	})

	// Read message from WebSocket
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)

	var wsMsg WSMessage
	err = json.Unmarshal(msg, &wsMsg)
	require.NoError(t, err)
	assert.Equal(t, "new_block", wsMsg.Type)
}

func TestServeWs_ClientDisconnectUnregisters(t *testing.T) {
	bus := events.NewBus()
	hub := NewHub(bus)

	server := httptest.NewServer(ServeWs(hub))
	defer server.Close()

	conn := dialWs(t, server)

	waitForClients(t, hub, 1)

	// Close the connection
	conn.Close()

	// Verify client was unregistered
	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.clients) == 0
	}, time.Second, 10*time.Millisecond)
}

func TestServeWs_MultipleClients(t *testing.T) {
	bus := events.NewBus()
	hub := NewHub(bus)

	server := httptest.NewServer(ServeWs(hub))
	defer server.Close()

	conn1 := dialWs(t, server)
	defer conn1.Close()

	conn2 := dialWs(t, server)
	defer conn2.Close()

	waitForClients(t, hub, 2)

	// Broadcast a message via hub
	hub.broadcast <- []byte(`{"type":"test","payload":"hello"}`)

	// Both clients should receive the message
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg1, err := conn1.ReadMessage()
	require.NoError(t, err)

	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg2, err := conn2.ReadMessage()
	require.NoError(t, err)

	assert.Equal(t, `{"type":"test","payload":"hello"}`, string(msg1))
	assert.Equal(t, `{"type":"test","payload":"hello"}`, string(msg2))
}

func TestServeWs_EventBusIntegration(t *testing.T) {
	bus := events.NewBus()
	hub := NewHub(bus)

	server := httptest.NewServer(ServeWs(hub))
	defer server.Close()

	conn := dialWs(t, server)
	defer conn.Close()

	waitForClients(t, hub, 1)

	// Publish multiple events
	bus.Publish(events.Event{
		Type:    events.EventNewBlock,
		Payload: map[string]string{"hash": "block1"},
	})
	bus.Publish(events.Event{
		Type:    events.EventMempoolChanged,
		Payload: map[string]int{"count": 5},
	})

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// writePump may batch queued messages into a single frame separated by newlines.
	// Collect all JSON objects from all frames until we have both event types.
	var received []WSMessage
	for len(received) < 2 {
		_, raw, err := conn.ReadMessage()
		require.NoError(t, err)

		// Split by newline in case of batched messages
		parts := strings.Split(string(raw), "\n")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			var wsMsg WSMessage
			err := json.Unmarshal([]byte(part), &wsMsg)
			require.NoError(t, err)
			received = append(received, wsMsg)
		}
	}

	// Verify both event types arrived
	types := make(map[string]bool)
	for _, msg := range received {
		types[msg.Type] = true
	}
	assert.True(t, types["new_block"], "should receive new_block event")
	assert.True(t, types["mempool_changed"], "should receive mempool_changed event")
}
