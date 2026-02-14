# Saga Pattern in Go with RabbitMQ Integration

This repository is now **100% Go** and demonstrates an event-driven Saga Pattern for e-commerce checkout.

## Flow

`OrderSaga` happy-path:

1. Reserve inventory
2. Charge payment
3. Create shipment
4. Complete saga

Compensations on failures:

- Payment failure -> release inventory
- Shipping failure -> refund payment + release inventory
- Inventory failure -> fail directly (nothing to compensate)

## Project Structure

- `saga/models.go`: saga entities and statuses.
- `saga/event_bus.go`: in-process event bus + transport abstraction.
- `saga/transport_memory.go`: in-memory transport (default for tests/dev).
- `saga/transport_rabbitmq.go`: RabbitMQ publish integration via Management HTTP API.
- `saga/services.go`: inventory, payment and shipping services.
- `saga/orchestrator.go`: saga orchestration and compensation logic.
- `saga/orchestrator_test.go`: unit tests for success/failure scenarios.
- `cmd/demo/main.go`: executable demo.

## Run tests

```bash
go test ./...
```

## Run demo (in-memory)

```bash
go run ./cmd/demo
```

## Run demo with RabbitMQ

The demo publishes every event to RabbitMQ using the Management HTTP API while still processing the saga locally.

```bash
export RABBITMQ_HTTP_URL="http://localhost:15672"
export RABBITMQ_USERNAME="guest"
export RABBITMQ_PASSWORD="guest"
export RABBITMQ_VHOST="/"
go run ./cmd/demo
```

> Requirements: RabbitMQ Management Plugin enabled and an exchange named `saga.events` available (type `topic`).
