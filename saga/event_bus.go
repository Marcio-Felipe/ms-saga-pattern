package saga

import "log/slog"

type Handler func(event Event)

type EventBus struct {
	logger   *slog.Logger
	handlers map[string][]Handler
	History  []Event
}

func NewEventBus(logger *slog.Logger) *EventBus {
	return &EventBus{
		logger:   logger,
		handlers: map[string][]Handler{},
		History:  []Event{},
	}
}

func (b *EventBus) Subscribe(eventName string, handler Handler) {
	b.handlers[eventName] = append(b.handlers[eventName], handler)
	b.logger.Debug("handler_subscribed", "event", eventName)
}

func (b *EventBus) Publish(event Event) {
	if event.Payload == nil {
		event.Payload = map[string]any{}
	}
	b.History = append(b.History, event)
	handlers := b.handlers[event.Name]

	b.logger.Info("event_published",
		"event", event.Name,
		"saga_id", event.SagaID,
		"payload", event.Payload,
		"handler_count", len(handlers),
	)

	for _, handler := range handlers {
		handler(event)
	}
}
