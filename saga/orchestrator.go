package saga

import (
	"fmt"
	"log/slog"
)

type OrderSagaOrchestrator struct {
	bus     *EventBus
	logger  *slog.Logger
	results map[string]*SagaResult
}

func NewOrderSagaOrchestrator(bus *EventBus, logger *slog.Logger) *OrderSagaOrchestrator {
	o := &OrderSagaOrchestrator{bus: bus, logger: logger, results: map[string]*SagaResult{}}
	bus.Subscribe("inventory.reserved", o.onInventoryReserved)
	bus.Subscribe("inventory.reserve.failed", o.onInventoryFailed)
	bus.Subscribe("payment.charged", o.onPaymentCharged)
	bus.Subscribe("payment.charge.failed", o.onPaymentFailed)
	bus.Subscribe("shipping.created", o.onShippingCreated)
	bus.Subscribe("shipping.create.failed", o.onShippingFailed)
	bus.Subscribe("inventory.released", o.onInventoryReleased)
	bus.Subscribe("payment.refunded", o.onPaymentRefunded)
	return o
}

func (o *OrderSagaOrchestrator) Start(orderID string, quantity int, amount float64, address string, sagaID string) *SagaResult {
	result := &SagaResult{SagaID: sagaID, Status: StatusStarted}
	o.results[sagaID] = result
	o.appendStep(sagaID, "SAGA_STARTED")
	o.bus.Publish(Event{Name: "inventory.reserve.requested", SagaID: sagaID, Payload: map[string]any{"order_id": orderID, "quantity": quantity, "amount": amount, "address": address}})
	return o.results[sagaID]
}

func (o *OrderSagaOrchestrator) onInventoryReserved(event Event) {
	result := o.results[event.SagaID]
	result.Status = StatusInProgress
	o.appendStep(event.SagaID, "INVENTORY_RESERVED")
	amount := o.sourcePayload(event.SagaID, "inventory.reserve.requested")["amount"].(float64)
	o.bus.Publish(Event{Name: "payment.charge.requested", SagaID: event.SagaID, Payload: map[string]any{"order_id": event.Payload["order_id"], "amount": amount}})
}

func (o *OrderSagaOrchestrator) onInventoryFailed(event Event) {
	result := o.results[event.SagaID]
	result.Status = StatusFailed
	result.Errors = append(result.Errors, event.Payload["error"].(string))
	o.appendStep(event.SagaID, "INVENTORY_FAILED")
}

func (o *OrderSagaOrchestrator) onPaymentCharged(event Event) {
	o.appendStep(event.SagaID, "PAYMENT_CHARGED")
	address := o.sourcePayload(event.SagaID, "inventory.reserve.requested")["address"].(string)
	o.bus.Publish(Event{Name: "shipping.create.requested", SagaID: event.SagaID, Payload: map[string]any{"order_id": event.Payload["order_id"], "address": address}})
}

func (o *OrderSagaOrchestrator) onPaymentFailed(event Event) {
	result := o.results[event.SagaID]
	result.Status = StatusFailed
	result.Errors = append(result.Errors, event.Payload["error"].(string))
	o.appendStep(event.SagaID, "PAYMENT_FAILED")
	o.appendCompensation(event.SagaID, "INVENTORY_RELEASE_REQUESTED")
	o.bus.Publish(Event{Name: "inventory.release.requested", SagaID: event.SagaID, Payload: map[string]any{"order_id": event.Payload["order_id"]}})
}

func (o *OrderSagaOrchestrator) onShippingCreated(event Event) {
	result := o.results[event.SagaID]
	result.Status = StatusCompleted
	o.appendStep(event.SagaID, "SHIPPING_CREATED")
	o.appendStep(event.SagaID, "SAGA_COMPLETED")
}

func (o *OrderSagaOrchestrator) onShippingFailed(event Event) {
	result := o.results[event.SagaID]
	result.Status = StatusFailed
	result.Errors = append(result.Errors, event.Payload["error"].(string))
	o.appendStep(event.SagaID, "SHIPPING_FAILED")
	o.appendCompensation(event.SagaID, "PAYMENT_REFUND_REQUESTED")
	o.bus.Publish(Event{Name: "payment.refund.requested", SagaID: event.SagaID, Payload: map[string]any{"order_id": event.Payload["order_id"]}})
	o.appendCompensation(event.SagaID, "INVENTORY_RELEASE_REQUESTED")
	o.bus.Publish(Event{Name: "inventory.release.requested", SagaID: event.SagaID, Payload: map[string]any{"order_id": event.Payload["order_id"]}})
}

func (o *OrderSagaOrchestrator) onInventoryReleased(event Event) {
	o.appendCompensation(event.SagaID, "INVENTORY_RELEASED")
	o.refreshFailedCompensated(event.SagaID)
}

func (o *OrderSagaOrchestrator) onPaymentRefunded(event Event) {
	o.appendCompensation(event.SagaID, "PAYMENT_REFUNDED")
	o.refreshFailedCompensated(event.SagaID)
}

func (o *OrderSagaOrchestrator) appendStep(sagaID, step string) {
	o.results[sagaID].Steps = append(o.results[sagaID].Steps, step)
}

func (o *OrderSagaOrchestrator) appendCompensation(sagaID, action string) {
	o.results[sagaID].Compensations = append(o.results[sagaID].Compensations, action)
}

func (o *OrderSagaOrchestrator) sourcePayload(sagaID, eventName string) map[string]any {
	for _, item := range o.bus.History {
		if item.SagaID == sagaID && item.Name == eventName {
			return item.Payload
		}
	}
	panic(fmt.Sprintf("source event not found: %s for saga %s", eventName, sagaID))
}

func (o *OrderSagaOrchestrator) refreshFailedCompensated(sagaID string) {
	result := o.results[sagaID]
	if result.Status != StatusFailed {
		return
	}

	hasRelease := contains(result.Compensations, "INVENTORY_RELEASED")
	paymentWasCharged := contains(result.Steps, "PAYMENT_CHARGED")
	hasRefund := contains(result.Compensations, "PAYMENT_REFUNDED")

	if paymentWasCharged {
		if hasRelease && hasRefund {
			result.Status = StatusFailedCompensated
			o.appendStep(sagaID, "SAGA_FAILED_COMPENSATED")
		}
		return
	}

	if hasRelease {
		result.Status = StatusFailedCompensated
		o.appendStep(sagaID, "SAGA_FAILED_COMPENSATED")
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func BuildOrchestrator(failInventoryFor, failPaymentFor, failShippingFor map[string]bool) (*OrderSagaOrchestrator, *EventBus) {
	logger := slog.Default()
	bus := NewEventBus(logger)
	_ = NewInventoryService(bus, logger, failInventoryFor)
	_ = NewPaymentService(bus, logger, failPaymentFor)
	_ = NewShippingService(bus, logger, failShippingFor)
	orchestrator := NewOrderSagaOrchestrator(bus, logger)
	return orchestrator, bus
}
