import logging
import uuid
from typing import Dict, Optional

from saga.event_bus import EventBus
from saga.models import Event, SagaResult, SagaStatus
from saga.services import InventoryService, PaymentService, ShippingService


class OrderSagaOrchestrator:
    def __init__(
        self,
        bus: EventBus,
        logger: logging.Logger,
        inventory: InventoryService,
        payment: PaymentService,
        shipping: ShippingService,
    ) -> None:
        self.bus = bus
        self.logger = logger
        self.inventory = inventory
        self.payment = payment
        self.shipping = shipping

        self.results: Dict[str, SagaResult] = {}

        self.bus.subscribe("inventory.reserved", self.on_inventory_reserved)
        self.bus.subscribe("inventory.reserve.failed", self.on_inventory_failed)
        self.bus.subscribe("payment.charged", self.on_payment_charged)
        self.bus.subscribe("payment.charge.failed", self.on_payment_failed)
        self.bus.subscribe("shipping.created", self.on_shipping_created)
        self.bus.subscribe("shipping.create.failed", self.on_shipping_failed)
        self.bus.subscribe("inventory.released", self.on_inventory_released)
        self.bus.subscribe("payment.refunded", self.on_payment_refunded)

    def start(self, order_id: str, quantity: int, amount: float, address: str, saga_id: Optional[str] = None) -> SagaResult:
        saga_id = saga_id or str(uuid.uuid4())
        result = SagaResult(saga_id=saga_id, status=SagaStatus.STARTED)
        self.results[saga_id] = result

        self.logger.info("saga_started", extra={"saga_id": saga_id, "order_id": order_id})
        self._append_step(saga_id, "SAGA_STARTED")

        self.bus.publish(
            Event(
                "inventory.reserve.requested",
                saga_id,
                {"order_id": order_id, "quantity": quantity, "amount": amount, "address": address},
            )
        )
        return self.results[saga_id]

    def on_inventory_reserved(self, event: Event) -> None:
        result = self.results[event.saga_id]
        result.status = SagaStatus.IN_PROGRESS
        self._append_step(event.saga_id, "INVENTORY_RESERVED")

        self.bus.publish(
            Event(
                "payment.charge.requested",
                event.saga_id,
                {
                    "order_id": event.payload["order_id"],
                    "amount": self._source_payload(event.saga_id, "inventory.reserve.requested")["amount"],
                },
            )
        )

    def on_inventory_failed(self, event: Event) -> None:
        result = self.results[event.saga_id]
        result.status = SagaStatus.FAILED
        result.errors.append(event.payload["error"])
        self._append_step(event.saga_id, "INVENTORY_FAILED")

    def on_payment_charged(self, event: Event) -> None:
        self._append_step(event.saga_id, "PAYMENT_CHARGED")

        source = self._source_payload(event.saga_id, "inventory.reserve.requested")
        self.bus.publish(
            Event(
                "shipping.create.requested",
                event.saga_id,
                {"order_id": event.payload["order_id"], "address": source["address"]},
            )
        )

    def on_payment_failed(self, event: Event) -> None:
        result = self.results[event.saga_id]
        result.status = SagaStatus.FAILED
        result.errors.append(event.payload["error"])
        self._append_step(event.saga_id, "PAYMENT_FAILED")

        self._append_compensation(event.saga_id, "INVENTORY_RELEASE_REQUESTED")
        self.bus.publish(Event("inventory.release.requested", event.saga_id, {"order_id": event.payload["order_id"]}))

    def on_shipping_created(self, event: Event) -> None:
        result = self.results[event.saga_id]
        result.status = SagaStatus.COMPLETED
        self._append_step(event.saga_id, "SHIPPING_CREATED")
        self._append_step(event.saga_id, "SAGA_COMPLETED")
        self.logger.info("saga_completed", extra={"saga_id": event.saga_id, "tracking_id": event.payload["tracking_id"]})

    def on_shipping_failed(self, event: Event) -> None:
        result = self.results[event.saga_id]
        result.status = SagaStatus.FAILED
        result.errors.append(event.payload["error"])
        self._append_step(event.saga_id, "SHIPPING_FAILED")

        self._append_compensation(event.saga_id, "PAYMENT_REFUND_REQUESTED")
        self.bus.publish(Event("payment.refund.requested", event.saga_id, {"order_id": event.payload["order_id"]}))

        self._append_compensation(event.saga_id, "INVENTORY_RELEASE_REQUESTED")
        self.bus.publish(Event("inventory.release.requested", event.saga_id, {"order_id": event.payload["order_id"]}))

    def on_inventory_released(self, event: Event) -> None:
        self._append_compensation(event.saga_id, "INVENTORY_RELEASED")
        self._refresh_failed_compensated(event.saga_id)

    def on_payment_refunded(self, event: Event) -> None:
        self._append_compensation(event.saga_id, "PAYMENT_REFUNDED")
        self._refresh_failed_compensated(event.saga_id)

    def _append_step(self, saga_id: str, step: str) -> None:
        self.results[saga_id].steps.append(step)
        self.logger.info("saga_step", extra={"saga_id": saga_id, "step": step})

    def _append_compensation(self, saga_id: str, action: str) -> None:
        self.results[saga_id].compensations.append(action)
        self.logger.warning("saga_compensation", extra={"saga_id": saga_id, "action": action})

    def _source_payload(self, saga_id: str, event_name: str) -> Dict[str, object]:
        for item in self.bus.history:
            if item.saga_id == saga_id and item.name == event_name:
                return item.payload
        raise RuntimeError(f"Evento de origem não encontrado: {event_name} para saga {saga_id}")

    def _refresh_failed_compensated(self, saga_id: str) -> None:
        result = self.results[saga_id]
        if result.status != SagaStatus.FAILED:
            return

        has_release = "INVENTORY_RELEASED" in result.compensations
        payment_was_charged = "PAYMENT_CHARGED" in result.steps
        has_refund = "PAYMENT_REFUNDED" in result.compensations

        if payment_was_charged:
            if has_release and has_refund:
                result.status = SagaStatus.FAILED_COMPENSATED
                self._append_step(saga_id, "SAGA_FAILED_COMPENSATED")
        else:
            if has_release:
                result.status = SagaStatus.FAILED_COMPENSATED
                self._append_step(saga_id, "SAGA_FAILED_COMPENSATED")


def build_orchestrator(
    fail_inventory_for: set[str] | None = None,
    fail_payment_for: set[str] | None = None,
    fail_shipping_for: set[str] | None = None,
) -> tuple[OrderSagaOrchestrator, EventBus]:
    logging.basicConfig(level=logging.DEBUG, format="%(asctime)s %(levelname)s %(message)s")
    logger = logging.getLogger("saga")

    bus = EventBus(logger)
    inventory = InventoryService(bus, logger, fail_inventory_for)
    payment = PaymentService(bus, logger, fail_payment_for)
    shipping = ShippingService(bus, logger, fail_shipping_for)
    orchestrator = OrderSagaOrchestrator(bus, logger, inventory, payment, shipping)
    return orchestrator, bus
