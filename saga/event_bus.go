package saga

import (
	"fmt"
	"log/slog"
	"sync"
)

type Handler func(event Event)

type Transport interface {
	Publish(event Event) error
	Close() error
}

type EventBus struct {
	logger    *slog.Logger
	handlers  map[string][]Handler
	history   []Event
	transport Transport
	mu        sync.RWMutex
}

func NewEventBus(logger *slog.Logger, transport Transport) *EventBus {
	if transport == nil {
		transport = NewInMemoryTransport()
	}
	return &EventBus{
		logger:    logger,
		handlers:  map[string][]Handler{},
		history:   []Event{},
		transport: transport,
	}
}

func (b *EventBus) Subscribe(eventName string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventName] = append(b.handlers[eventName], handler)
	b.logger.Debug("handler_subscribed", "event", eventName)
}

func (b *EventBus) Publish(event Event) error {
	if event.Payload == nil {
		event.Payload = map[string]any{}
	}

	b.mu.Lock()
	b.history = append(b.history, event)
	handlers := append([]Handler(nil), b.handlers[event.Name]...)
	b.mu.Unlock()

	b.logger.Info("event_published",
		"event", event.Name,
		"saga_id", event.SagaID,
		"payload", event.Payload,
		"handler_count", len(handlers),
	)

	for _, handler := range handlers {
		handler(event)
	}

	if err := b.transport.Publish(event); err != nil {
		return fmt.Errorf("publish to transport: %w", err)
	}
	return nil
}

func (b *EventBus) MustPublish(event Event) {
	if err := b.Publish(event); err != nil {
		panic(err)
	}
}

func (b *EventBus) History() []Event {
	b.mu.RLock()
	defer b.mu.RUnlock()
	copyHistory := make([]Event, len(b.history))
	copy(copyHistory, b.history)
	return copyHistory
}

func (b *EventBus) Close() error {
	if b.transport == nil {
		return nil
	}
	return b.transport.Close()
}
