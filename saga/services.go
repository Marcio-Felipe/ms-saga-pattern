package saga

import (
	"fmt"
	"log/slog"
)

type InventoryService struct {
	bus           *EventBus
	logger        *slog.Logger
	failForOrders map[string]bool
	reservations  map[string]int
}

func NewInventoryService(bus *EventBus, logger *slog.Logger, failForOrders map[string]bool) *InventoryService {
	if failForOrders == nil {
		failForOrders = map[string]bool{}
	}
	s := &InventoryService{bus: bus, logger: logger, failForOrders: failForOrders, reservations: map[string]int{}}
	bus.Subscribe("inventory.reserve.requested", s.Reserve)
	bus.Subscribe("inventory.release.requested", s.Release)
	return s
}

func (s *InventoryService) Reserve(event Event) {
	orderID := event.Payload["order_id"].(string)
	qty := event.Payload["quantity"].(int)

	if s.failForOrders[orderID] {
		err := fmt.Sprintf("insufficient stock for order %s", orderID)
		s.logger.Error("inventory_reserve_failed", "saga_id", event.SagaID, "error", err)
		s.bus.MustPublish(Event{Name: "inventory.reserve.failed", SagaID: event.SagaID, Payload: map[string]any{"order_id": orderID, "error": err}})
		return
	}

	s.reservations[orderID] += qty
	s.bus.MustPublish(Event{Name: "inventory.reserved", SagaID: event.SagaID, Payload: map[string]any{"order_id": orderID, "quantity": qty}})
}

func (s *InventoryService) Release(event Event) {
	orderID := event.Payload["order_id"].(string)
	qty := s.reservations[orderID]
	delete(s.reservations, orderID)
	s.bus.MustPublish(Event{Name: "inventory.released", SagaID: event.SagaID, Payload: map[string]any{"order_id": orderID, "quantity": qty}})
}

type PaymentService struct {
	bus           *EventBus
	logger        *slog.Logger
	failForOrders map[string]bool
	charges       map[string]float64
}

func NewPaymentService(bus *EventBus, logger *slog.Logger, failForOrders map[string]bool) *PaymentService {
	if failForOrders == nil {
		failForOrders = map[string]bool{}
	}
	s := &PaymentService{bus: bus, logger: logger, failForOrders: failForOrders, charges: map[string]float64{}}
	bus.Subscribe("payment.charge.requested", s.Charge)
	bus.Subscribe("payment.refund.requested", s.Refund)
	return s
}

func (s *PaymentService) Charge(event Event) {
	orderID := event.Payload["order_id"].(string)
	amount := event.Payload["amount"].(float64)

	if s.failForOrders[orderID] {
		err := fmt.Sprintf("card declined for order %s", orderID)
		s.logger.Error("payment_charge_failed", "saga_id", event.SagaID, "error", err)
		s.bus.MustPublish(Event{Name: "payment.charge.failed", SagaID: event.SagaID, Payload: map[string]any{"order_id": orderID, "error": err}})
		return
	}

	s.charges[orderID] = amount
	s.bus.MustPublish(Event{Name: "payment.charged", SagaID: event.SagaID, Payload: map[string]any{"order_id": orderID, "amount": amount}})
}

func (s *PaymentService) Refund(event Event) {
	orderID := event.Payload["order_id"].(string)
	amount := s.charges[orderID]
	delete(s.charges, orderID)
	s.bus.MustPublish(Event{Name: "payment.refunded", SagaID: event.SagaID, Payload: map[string]any{"order_id": orderID, "amount": amount}})
}

type ShippingService struct {
	bus           *EventBus
	logger        *slog.Logger
	failForOrders map[string]bool
	shipments     map[string]string
}

func NewShippingService(bus *EventBus, logger *slog.Logger, failForOrders map[string]bool) *ShippingService {
	if failForOrders == nil {
		failForOrders = map[string]bool{}
	}
	s := &ShippingService{bus: bus, logger: logger, failForOrders: failForOrders, shipments: map[string]string{}}
	bus.Subscribe("shipping.create.requested", s.Create)
	return s
}

func (s *ShippingService) Create(event Event) {
	orderID := event.Payload["order_id"].(string)
	address := event.Payload["address"].(string)

	if s.failForOrders[orderID] {
		err := fmt.Sprintf("shipping unavailable for order %s", orderID)
		s.logger.Error("shipping_create_failed", "saga_id", event.SagaID, "error", err)
		s.bus.MustPublish(Event{Name: "shipping.create.failed", SagaID: event.SagaID, Payload: map[string]any{"order_id": orderID, "error": err}})
		return
	}

	trackingID := fmt.Sprintf("TRK-%s", orderID)
	s.shipments[orderID] = address
	s.bus.MustPublish(Event{Name: "shipping.created", SagaID: event.SagaID, Payload: map[string]any{"order_id": orderID, "tracking_id": trackingID}})
}
