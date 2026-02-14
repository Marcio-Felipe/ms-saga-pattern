import logging
from typing import Dict, Set

from saga.event_bus import EventBus
from saga.models import Event


class InventoryService:
    def __init__(self, bus: EventBus, logger: logging.Logger, fail_for_orders: Set[str] | None = None) -> None:
        self.bus = bus
        self.logger = logger
        self.fail_for_orders = fail_for_orders or set()
        self.reservations: Dict[str, int] = {}

        self.bus.subscribe("inventory.reserve.requested", self.reserve)
        self.bus.subscribe("inventory.release.requested", self.release)

    def reserve(self, event: Event) -> None:
        order_id = event.payload["order_id"]
        qty = event.payload["quantity"]

        self.logger.info("inventory_reserve_started", extra={"saga_id": event.saga_id, "order_id": order_id, "qty": qty})

        if order_id in self.fail_for_orders:
            error = f"Estoque insuficiente para pedido {order_id}"
            self.logger.error("inventory_reserve_failed", extra={"saga_id": event.saga_id, "error": error})
            self.bus.publish(Event("inventory.reserve.failed", event.saga_id, {"order_id": order_id, "error": error}))
            return

        self.reservations[order_id] = self.reservations.get(order_id, 0) + qty
        self.logger.info("inventory_reserved", extra={"saga_id": event.saga_id, "order_id": order_id, "qty": qty})
        self.bus.publish(Event("inventory.reserved", event.saga_id, {"order_id": order_id, "quantity": qty}))

    def release(self, event: Event) -> None:
        order_id = event.payload["order_id"]
        qty = self.reservations.pop(order_id, 0)
        self.logger.warning("inventory_released", extra={"saga_id": event.saga_id, "order_id": order_id, "qty": qty})
        self.bus.publish(Event("inventory.released", event.saga_id, {"order_id": order_id, "quantity": qty}))


class PaymentService:
    def __init__(self, bus: EventBus, logger: logging.Logger, fail_for_orders: Set[str] | None = None) -> None:
        self.bus = bus
        self.logger = logger
        self.fail_for_orders = fail_for_orders or set()
        self.charges: Dict[str, float] = {}

        self.bus.subscribe("payment.charge.requested", self.charge)
        self.bus.subscribe("payment.refund.requested", self.refund)

    def charge(self, event: Event) -> None:
        order_id = event.payload["order_id"]
        amount = event.payload["amount"]

        self.logger.info("payment_charge_started", extra={"saga_id": event.saga_id, "order_id": order_id, "amount": amount})

        if order_id in self.fail_for_orders:
            error = f"Cartão recusado para pedido {order_id}"
            self.logger.error("payment_charge_failed", extra={"saga_id": event.saga_id, "error": error})
            self.bus.publish(Event("payment.charge.failed", event.saga_id, {"order_id": order_id, "error": error}))
            return

        self.charges[order_id] = amount
        self.logger.info("payment_charged", extra={"saga_id": event.saga_id, "order_id": order_id, "amount": amount})
        self.bus.publish(Event("payment.charged", event.saga_id, {"order_id": order_id, "amount": amount}))

    def refund(self, event: Event) -> None:
        order_id = event.payload["order_id"]
        amount = self.charges.pop(order_id, 0.0)
        self.logger.warning("payment_refunded", extra={"saga_id": event.saga_id, "order_id": order_id, "amount": amount})
        self.bus.publish(Event("payment.refunded", event.saga_id, {"order_id": order_id, "amount": amount}))


class ShippingService:
    def __init__(self, bus: EventBus, logger: logging.Logger, fail_for_orders: Set[str] | None = None) -> None:
        self.bus = bus
        self.logger = logger
        self.fail_for_orders = fail_for_orders or set()
        self.shipments: Dict[str, str] = {}

        self.bus.subscribe("shipping.create.requested", self.create)

    def create(self, event: Event) -> None:
        order_id = event.payload["order_id"]
        address = event.payload["address"]

        self.logger.info("shipping_create_started", extra={"saga_id": event.saga_id, "order_id": order_id, "address": address})

        if order_id in self.fail_for_orders:
            error = f"Transportadora indisponível para pedido {order_id}"
            self.logger.error("shipping_create_failed", extra={"saga_id": event.saga_id, "error": error})
            self.bus.publish(Event("shipping.create.failed", event.saga_id, {"order_id": order_id, "error": error}))
            return

        tracking_id = f"TRK-{order_id}"
        self.shipments[order_id] = tracking_id
        self.logger.info("shipping_created", extra={"saga_id": event.saga_id, "order_id": order_id, "tracking_id": tracking_id})
        self.bus.publish(Event("shipping.created", event.saga_id, {"order_id": order_id, "tracking_id": tracking_id}))
