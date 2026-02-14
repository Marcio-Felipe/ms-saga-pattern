package main

import (
	"fmt"
	"log"
	"os"

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

func main() {
	transport := buildTransport()
	runCase("SCENARIO 1: SUCCESS", "ORDER-OK", "SAGA-OK", nil, nil, nil, transport)
	runCase("SCENARIO 2: PAYMENT FAILURE (inventory compensation)", "ORDER-PAY-FAIL", "SAGA-PAY-FAIL", nil, map[string]bool{"ORDER-PAY-FAIL": true}, nil, transport)
	runCase("SCENARIO 3: SHIPPING FAILURE (refund + release)", "ORDER-SHIP-FAIL", "SAGA-SHIP-FAIL", nil, nil, map[string]bool{"ORDER-SHIP-FAIL": true}, transport)
	runCase("SCENARIO 4: INVENTORY FAILURE (no compensation)", "ORDER-INV-FAIL", "SAGA-INV-FAIL", map[string]bool{"ORDER-INV-FAIL": true}, nil, nil, transport)
}
