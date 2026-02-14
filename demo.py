from saga.orchestrator import build_orchestrator


def run_case(title: str, **kwargs: object) -> None:
    print(f"\n{'=' * 80}\n{title}\n{'=' * 80}")
    orchestrator, bus = build_orchestrator(
        fail_inventory_for=kwargs.get("fail_inventory_for"),
        fail_payment_for=kwargs.get("fail_payment_for"),
        fail_shipping_for=kwargs.get("fail_shipping_for"),
    )
    result = orchestrator.start(
        order_id=kwargs["order_id"],
        quantity=kwargs.get("quantity", 2),
        amount=kwargs.get("amount", 199.90),
        address=kwargs.get("address", "Rua das Flores, 123 - São Paulo"),
        saga_id=kwargs.get("saga_id"),
    )

    print("Resultado da saga:")
    print(f"- saga_id: {result.saga_id}")
    print(f"- status: {result.status}")
    print(f"- steps: {result.steps}")
    print(f"- compensations: {result.compensations}")
    print(f"- errors: {result.errors}")
    print(f"- total_eventos: {len(bus.history)}")


if __name__ == "__main__":
    run_case("CENÁRIO 1: SUCESSO", order_id="ORDER-OK", saga_id="SAGA-OK")
    run_case(
        "CENÁRIO 2: FALHA NO PAGAMENTO (com compensação de estoque)",
        order_id="ORDER-PAY-FAIL",
        saga_id="SAGA-PAY-FAIL",
        fail_payment_for={"ORDER-PAY-FAIL"},
    )
    run_case(
        "CENÁRIO 3: FALHA NO ENVIO (com estorno + liberação)",
        order_id="ORDER-SHIP-FAIL",
        saga_id="SAGA-SHIP-FAIL",
        fail_shipping_for={"ORDER-SHIP-FAIL"},
    )
    run_case(
        "CENÁRIO 4: FALHA NO ESTOQUE (sem compensação)",
        order_id="ORDER-INV-FAIL",
        saga_id="SAGA-INV-FAIL",
        fail_inventory_for={"ORDER-INV-FAIL"},
    )
