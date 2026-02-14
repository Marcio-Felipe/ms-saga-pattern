package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"ms-saga-pattern/saga"
)

func buildTransport() saga.Transport {
	endpoint := os.Getenv("RABBITMQ_HTTP_URL")
	if endpoint == "" {
		fmt.Println("RABBITMQ_HTTP_URL not set, using in-memory transport only")
		return nil
	}

	username := os.Getenv("RABBITMQ_USERNAME")
	password := os.Getenv("RABBITMQ_PASSWORD")
	vhost := os.Getenv("RABBITMQ_VHOST")

	transport, err := saga.NewRabbitMQTransport(endpoint, username, password, vhost, saga.DefaultExchange)
	if err != nil {
		log.Printf("failed to configure RabbitMQ transport (%v), using in-memory transport only", err)
		return nil
	}
	fmt.Println("RabbitMQ transport enabled via Management HTTP API")
	return transport
}

func startMetricsServer() {
	addr := os.Getenv("METRICS_ADDR")
	if addr == "" {
		addr = ":2112"
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", saga.DefaultMetrics.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	go func() {
		log.Printf("metrics endpoint listening on %s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("metrics server stopped: %v", err)
		}
	}()
}

func runCase(title, orderID, sagaID string, failInventoryFor, failPaymentFor, failShippingFor map[string]bool, transport saga.Transport) {
	fmt.Printf("\n%s\n%s\n", title, "================================================================================")
	orchestrator, bus := saga.BuildOrchestrator(failInventoryFor, failPaymentFor, failShippingFor, transport)

	result := orchestrator.Start(orderID, 2, 199.90, "123 Flower Street, Sao Paulo", sagaID)

	fmt.Println("Saga result:")
	fmt.Printf("- saga_id: %s\n", result.SagaID)
	fmt.Printf("- status: %s\n", result.Status)
	fmt.Printf("- steps: %v\n", result.Steps)
	fmt.Printf("- compensations: %v\n", result.Compensations)
	fmt.Printf("- errors: %v\n", result.Errors)
	fmt.Printf("- total_events: %d\n", len(bus.History()))
}

func runScenarios(transport saga.Transport) {
	runCase("SCENARIO 1: SUCCESS", "ORDER-OK", fmt.Sprintf("SAGA-OK-%d", time.Now().UnixNano()), nil, nil, nil, transport)
	runCase("SCENARIO 2: PAYMENT FAILURE (inventory compensation)", "ORDER-PAY-FAIL", fmt.Sprintf("SAGA-PAY-FAIL-%d", time.Now().UnixNano()), nil, map[string]bool{"ORDER-PAY-FAIL": true}, nil, transport)
	runCase("SCENARIO 3: SHIPPING FAILURE (refund + release)", "ORDER-SHIP-FAIL", fmt.Sprintf("SAGA-SHIP-FAIL-%d", time.Now().UnixNano()), nil, nil, map[string]bool{"ORDER-SHIP-FAIL": true}, transport)
	runCase("SCENARIO 4: INVENTORY FAILURE (no compensation)", "ORDER-INV-FAIL", fmt.Sprintf("SAGA-INV-FAIL-%d", time.Now().UnixNano()), map[string]bool{"ORDER-INV-FAIL": true}, nil, nil, transport)
}

func main() {
	startMetricsServer()
	transport := buildTransport()
	runScenarios(transport)

	if os.Getenv("RUN_CONTINUOUS") == "true" {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			runScenarios(transport)
		}
	}
}
