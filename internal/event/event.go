package event

import (
	"sync"
	"time"
)

// EventType is a strong type representing system event names.
type EventType string

const (
	// Metadata/Discovery events
	EventObjectCreated   EventType = "ObjectCreated"
	EventObjectModified  EventType = "ObjectModified"
	EventObjectDeleted   EventType = "ObjectDeleted"

	// Policy & Decision events
	EventPolicyChanged   EventType = "PolicyChanged"
	EventTransferPlanCreated EventType = "TransferPlanCreated"

	// Transfer jobs
	EventJobStarted      EventType = "JobStarted"
	EventJobCompleted    EventType = "JobCompleted"
	EventJobFailed       EventType = "JobFailed"

	// Provider/System status
	EventProviderOnline  EventType = "ProviderOnline"
	EventProviderOffline EventType = "ProviderOffline"
	EventCopyCorrupted   EventType = "CopyCorrupted"
)

// Event represents an immutable payload traveling through the system.
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// Handler is a callback function to consume system events.
type Handler func(Event)

// Bus provides a thread-safe implementation of an in-memory Pub/Sub event router.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[EventType][]Handler
}

// NewBus creates and returns a new initialized Bus.
func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[EventType][]Handler),
	}
}

// Subscribe registers a handler for a given event type.
func (b *Bus) Subscribe(t EventType, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[t] = append(b.subscribers[t], h)
}

// Publish broadcasts an event asynchronously to all subscribed handlers.
func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	handlers, exists := b.subscribers[e.Type]
	if !exists {
		b.mu.RUnlock()
		return
	}

	// Copy slice under lock to prevent concurrent modification race
	handlersCopy := make([]Handler, len(handlers))
	copy(handlersCopy, handlers)
	b.mu.RUnlock()

	for _, handler := range handlersCopy {
		h := handler
		go h(e)
	}
}
