package events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBusPublishSendsToAllSubscribers(t *testing.T) {
	bus := NewBus()

	ch1 := bus.Subscribe()
	ch2 := bus.Subscribe()

	event := Event{Type: EventNewBlock, Payload: "block-1"}
	bus.Publish(event)

	select {
	case got := <-ch1:
		assert.Equal(t, event, got)
	case <-time.After(time.Second):
		t.Fatal("ch1 did not receive event")
	}

	select {
	case got := <-ch2:
		assert.Equal(t, event, got)
	case <-time.After(time.Second):
		t.Fatal("ch2 did not receive event")
	}
}

func TestBusPublishDropsWhenChannelFull(t *testing.T) {
	bus := NewBus()

	ch := bus.Subscribe()

	// Fill the channel buffer (capacity 64)
	for i := range 64 {
		bus.Publish(Event{Type: EventNewTx, Payload: i})
	}

	// This should not block -- event is dropped
	bus.Publish(Event{Type: EventNewTx, Payload: "dropped"})

	// Drain and verify we got 64 events, not 65
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	assert.Equal(t, 64, count)
}

func TestBusUnsubscribeRemovesAndCloses(t *testing.T) {
	bus := NewBus()

	ch := bus.Subscribe()
	bus.Unsubscribe(ch)

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed after unsubscribe")

	// Publishing after unsubscribe should not panic
	assert.NotPanics(t, func() {
		bus.Publish(Event{Type: EventNewBlock, Payload: nil})
	})
}

func TestBusMultipleUnsubscribe(t *testing.T) {
	bus := NewBus()

	ch1 := bus.Subscribe()
	ch2 := bus.Subscribe()

	bus.Unsubscribe(ch1)

	// ch2 should still receive events
	bus.Publish(Event{Type: EventStatus, Payload: "test"})

	select {
	case got := <-ch2:
		assert.Equal(t, EventStatus, got.Type)
	case <-time.After(time.Second):
		t.Fatal("ch2 should still receive events after ch1 unsubscribed")
	}

	bus.Unsubscribe(ch2)

	// Both closed
	_, ok := <-ch1
	assert.False(t, ok)
	_, ok = <-ch2
	assert.False(t, ok)
}

func TestBusSubscribeReturnsBufferedChannel(t *testing.T) {
	bus := NewBus()
	ch := bus.Subscribe()
	require.Equal(t, 64, cap(ch))
	bus.Unsubscribe(ch)
}
