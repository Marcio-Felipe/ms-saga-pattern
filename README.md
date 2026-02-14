# Saga Pattern in Go + RabbitMQ + Observability (Prometheus/Grafana)

This project is fully implemented in **Go** and demonstrates a **Saga Pattern** orchestration flow with optional RabbitMQ event publishing and built-in observability endpoints.

## Architecture

- `OrderSaga` flow:
  1. Reserve inventory
  2. Charge payment
  3. Create shipment
  4. Complete order
- Compensations:
  - payment failure -> release inventory
  - shipping failure -> refund payment + release inventory
  - inventory failure -> fail directly (no compensation needed)

## Repository structure

- `saga/models.go`: data models and statuses.
- `saga/event_bus.go`: in-process event bus with transport abstraction.
- `saga/transport_memory.go`: in-memory transport (default).
- `saga/transport_rabbitmq.go`: RabbitMQ transport through Management HTTP API.
- `saga/services.go`: inventory/payment/shipping services.
- `saga/orchestrator.go`: saga orchestration + compensation rules.
- `saga/metrics.go`: Prometheus-format metrics registry and `/metrics` handler.
- `cmd/demo/main.go`: runnable app with metrics endpoint.
- `observability/*`: Prometheus, Grafana provisioning, and dashboard.

## Run tests

```bash
go test ./...
```

## Run app locally

```bash
go run ./cmd/demo
```

By default, this starts:
- saga execution (4 scenarios once),
- metrics endpoint on `:2112`.

### Optional environment variables

- `METRICS_ADDR` (default `:2112`)
- `RUN_CONTINUOUS=true` -> re-runs all scenarios every 10s (useful for dashboards)
- `RABBITMQ_HTTP_URL` (ex: `http://localhost:15672`)
- `RABBITMQ_USERNAME` (ex: `guest`)
- `RABBITMQ_PASSWORD` (ex: `guest`)
- `RABBITMQ_VHOST` (default `/`)

---

## Observability manual (Prometheus + Grafana)

### 1) Start infrastructure

```bash
docker compose up -d
```

Services:
- RabbitMQ: `http://localhost:15672` (`guest/guest`)
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` (`admin/admin`)

### 2) Start the Go app with continuous traffic

```bash
export RUN_CONTINUOUS=true
go run ./cmd/demo
```

If you also want RabbitMQ publishing:

```bash
export RABBITMQ_HTTP_URL="http://localhost:15672"
export RABBITMQ_USERNAME="guest"
export RABBITMQ_PASSWORD="guest"
export RABBITMQ_VHOST="/"
export RUN_CONTINUOUS=true
go run ./cmd/demo
```

### 3) Validate metrics endpoint

```bash
curl -s http://localhost:2112/metrics | head -n 40
```

Main metrics exposed:
- `saga_events_published_total{event="..."}`
- `saga_started_total`
- `saga_completed_total`
- `saga_failed_total`
- `saga_failed_compensated_total`
- `saga_duration_seconds_bucket|sum|count`

### 4) Validate Prometheus scraping

Open Prometheus and run:
- `saga_started_total`
- `sum(rate(saga_events_published_total[1m]))`
- `histogram_quantile(0.95, sum(rate(saga_duration_seconds_bucket[5m])) by (le))`

### 5) Open Grafana dashboard

- URL: `http://localhost:3000`
- Login: `admin/admin`
- Dashboard is auto-provisioned: **Saga / Saga Overview**

Panels include:
- event throughput,
- total started sagas,
- outcome counters,
- p50/p95 saga duration.

### 6) Troubleshooting

- If Prometheus has no data, ensure app is running and `http://localhost:2112/metrics` works.
- `docker compose` targets `host.docker.internal:2112`; this is already configured via `extra_hosts`.
- If RabbitMQ integration fails, app falls back to in-memory transport automatically.
