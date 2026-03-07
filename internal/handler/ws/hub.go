package ws

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/baotoq/shitcoin/internal/domain/events"
)

// Hub maintains the set of active WebSocket clients and broadcasts messages to them.
// It subscribes to the domain event bus and forwards all events as JSON to connected clients.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	eventBus   *events.Bus
}

// NewHub creates a new Hub that subscribes to the given event bus.
// Call Run() to start the hub goroutine.
func NewHub(bus *events.Bus) *Hub {
	h := &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		eventBus:   bus,
	}
	go h.Run()
	return h
}

// Run processes register, unregister, and broadcast events in a select loop.
// It also starts a goroutine to forward event bus events to the broadcast channel.
func (h *Hub) Run() {
	// Start event bus subscriber goroutine
	go h.subscribeEventBus()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Slow client -- evict
					delete(h.clients, client)
					close(client.send)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// subscribeEventBus subscribes to the event bus and converts events to JSON
// WSMessage bytes, then sends them to the broadcast channel.
func (h *Hub) subscribeEventBus() {
	ch := h.eventBus.Subscribe()
	for event := range ch {
		msg := WSMessage{
			Type:    string(event.Type),
			Payload: event.Payload,
		}
		data, err := json.Marshal(msg)
		if err != nil {
			slog.Error("failed to marshal event for WebSocket", "type", event.Type, "error", err)
			continue
		}
		h.broadcast <- data
	}
}
