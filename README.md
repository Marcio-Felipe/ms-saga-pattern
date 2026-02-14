# Robust Saga Pattern Example (Event-Driven) in Go

This project demonstrates an **event-driven Saga Pattern** orchestration flow for an e-commerce checkout.

## Scenario

Main saga flow (`OrderSaga`):

1. Reserve inventory
2. Charge payment
3. Create shipment
4. Confirm order

When a step fails, compensating actions are triggered:

- Payment failure -> release inventory
- Shipping failure -> refund payment and release inventory
- Inventory failure -> stop without compensation (nothing committed yet)

## Project Structure

- `saga/event_bus.go`: event bus with in-memory event history.
- `saga/services.go`: Inventory, Payment, and Shipping services.
- `saga/orchestrator.go`: saga orchestration and compensation logic.
- `cmd/demo/main.go`: manual run with scenario outputs.
- `saga/orchestrator_test.go`: success and failure test scenarios.

## Run the demo

```bash
go run ./cmd/demo
```

## Run tests

```bash
go test ./...
```

## Observability

The system emits logs for:

- event publication and dispatch,
- saga state transitions,
- step failures,
- compensations.

This makes it easy to understand behavior in both success and failure paths.
