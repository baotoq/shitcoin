package events

import "sync"

// EventType identifies the kind of domain event.
type EventType string

const (
	EventNewBlock          EventType = "new_block"
	EventNewTx             EventType = "new_tx"
	EventMiningProgress    EventType = "mining_progress"
	EventMiningStarted     EventType = "mining_started"
	EventMiningStopped     EventType = "mining_stopped"
	EventPeerConnected     EventType = "peer_connected"
	EventPeerDisconnected  EventType = "peer_disconnected"
	EventMempoolChanged    EventType = "mempool_changed"
	EventReorg             EventType = "reorg"
	EventStatus            EventType = "status"
)

// Event is a domain event carrying a typed payload.
type Event struct {
	Type    EventType
	Payload any
}

// Bus is a simple publish/subscribe event bus.
// Subscribers receive events on buffered channels.
// Publishing is non-blocking: slow subscribers have events dropped.
type Bus struct {
	mu          sync.RWMutex
	subscribers []chan Event
}

// NewBus creates a new event bus.
func NewBus() *Bus {
	return &Bus{}
}

// Subscribe returns a buffered channel (capacity 64) that receives published events.
func (b *Bus) Subscribe() chan Event {
	ch := make(chan Event, 64)
	b.mu.Lock()
	b.subscribers = append(b.subscribers, ch)
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel and closes it.
func (b *Bus) Unsubscribe(ch chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, sub := range b.subscribers {
		if sub == ch {
			b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

// Publish sends an event to all subscribers.
// If a subscriber's channel is full, the event is dropped (non-blocking).
func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers {
		select {
		case ch <- e:
		default:
			// Drop event for slow subscriber
		}
	}
}
