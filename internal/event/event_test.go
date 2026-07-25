package event_test

import (
	"sync"
	"testing"
	"time"

	"mash/internal/event"
)

func TestEventBusPubSub(t *testing.T) {
	bus := event.NewBus()

	var wg sync.WaitGroup
	wg.Add(1)

	var received event.Event
	bus.Subscribe(event.EventObjectCreated, func(e event.Event) {
		received = e
		wg.Done()
	})

	testEvent := event.Event{
		ID:        "evt-1",
		Type:      event.EventObjectCreated,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"foo": "bar",
		},
	}

	bus.Publish(testEvent)

	// Wait with a timeout
	c := make(chan struct{})
	go func() {
		wg.Wait()
		close(c)
	}()

	select {
	case <-c:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for event handler")
	}

	if received.ID != "evt-1" {
		t.Errorf("expected event ID evt-1, got %s", received.ID)
	}

	val, exists := received.Payload["foo"]
	if !exists || val != "bar" {
		t.Errorf("expected payload to contain foo: bar, got %v", received.Payload)
	}
}
