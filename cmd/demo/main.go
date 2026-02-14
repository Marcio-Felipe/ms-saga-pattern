package main

import (
	"fmt"

	"ms-saga-pattern/saga"
)

func runCase(title, orderID, sagaID string, failInventoryFor, failPaymentFor, failShippingFor map[string]bool) {
	fmt.Printf("\n%s\n%s\n", title, "================================================================================")
	orchestrator, bus := saga.BuildOrchestrator(failInventoryFor, failPaymentFor, failShippingFor)
	result := orchestrator.Start(orderID, 2, 199.90, "123 Flower Street, São Paulo", sagaID)

	fmt.Println("Saga result:")
	fmt.Printf("- saga_id: %s\n", result.SagaID)
	fmt.Printf("- status: %s\n", result.Status)
	fmt.Printf("- steps: %v\n", result.Steps)
	fmt.Printf("- compensations: %v\n", result.Compensations)
	fmt.Printf("- errors: %v\n", result.Errors)
	fmt.Printf("- total_events: %d\n", len(bus.History))
}

func main() {
	runCase("SCENARIO 1: SUCCESS", "ORDER-OK", "SAGA-OK", nil, nil, nil)
	runCase("SCENARIO 2: PAYMENT FAILURE (inventory compensation)", "ORDER-PAY-FAIL", "SAGA-PAY-FAIL", nil, map[string]bool{"ORDER-PAY-FAIL": true}, nil)
	runCase("SCENARIO 3: SHIPPING FAILURE (refund + release)", "ORDER-SHIP-FAIL", "SAGA-SHIP-FAIL", nil, nil, map[string]bool{"ORDER-SHIP-FAIL": true})
	runCase("SCENARIO 4: INVENTORY FAILURE (no compensation)", "ORDER-INV-FAIL", "SAGA-INV-FAIL", map[string]bool{"ORDER-INV-FAIL": true}, nil, nil)
}
