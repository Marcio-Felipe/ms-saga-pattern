import unittest

from saga.models import SagaStatus
from saga.orchestrator import build_orchestrator


class SagaPatternTests(unittest.TestCase):
    def test_success_path(self) -> None:
        orchestrator, bus = build_orchestrator()
        result = orchestrator.start(
            order_id="ORDER-OK",
            quantity=1,
            amount=99.9,
            address="Rua A, 10",
            saga_id="SAGA-OK",
        )

        self.assertEqual(result.status, SagaStatus.COMPLETED)
        self.assertIn("SAGA_COMPLETED", result.steps)
        self.assertEqual(result.compensations, [])
        self.assertEqual(result.errors, [])
        self.assertIn("shipping.created", [event.name for event in bus.history])

    def test_payment_failure_with_inventory_compensation(self) -> None:
        orchestrator, _ = build_orchestrator(fail_payment_for={"ORDER-PAY-FAIL"})
        result = orchestrator.start(
            order_id="ORDER-PAY-FAIL",
            quantity=1,
            amount=150.0,
            address="Rua B, 11",
            saga_id="SAGA-PAY-FAIL",
        )

        self.assertEqual(result.status, SagaStatus.FAILED_COMPENSATED)
        self.assertIn("PAYMENT_FAILED", result.steps)
        self.assertIn("INVENTORY_RELEASED", result.compensations)
        self.assertEqual(len(result.errors), 1)

    def test_shipping_failure_with_double_compensation(self) -> None:
        orchestrator, _ = build_orchestrator(fail_shipping_for={"ORDER-SHIP-FAIL"})
        result = orchestrator.start(
            order_id="ORDER-SHIP-FAIL",
            quantity=1,
            amount=200.0,
            address="Rua C, 12",
            saga_id="SAGA-SHIP-FAIL",
        )

        self.assertEqual(result.status, SagaStatus.FAILED_COMPENSATED)
        self.assertIn("SHIPPING_FAILED", result.steps)
        self.assertIn("PAYMENT_REFUNDED", result.compensations)
        self.assertIn("INVENTORY_RELEASED", result.compensations)
        self.assertEqual(len(result.errors), 1)

    def test_inventory_failure_without_compensation(self) -> None:
        orchestrator, _ = build_orchestrator(fail_inventory_for={"ORDER-INV-FAIL"})
        result = orchestrator.start(
            order_id="ORDER-INV-FAIL",
            quantity=1,
            amount=10.0,
            address="Rua D, 13",
            saga_id="SAGA-INV-FAIL",
        )

        self.assertEqual(result.status, SagaStatus.FAILED)
        self.assertIn("INVENTORY_FAILED", result.steps)
        self.assertEqual(result.compensations, [])
        self.assertEqual(len(result.errors), 1)


if __name__ == "__main__":
    unittest.main()
