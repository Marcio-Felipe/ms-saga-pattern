package saga

import "testing"

func TestSuccessPath(t *testing.T) {
	orchestrator, bus := BuildOrchestrator(nil, nil, nil)
	result := orchestrator.Start("ORDER-OK", 1, 99.9, "10 A Street", "SAGA-OK")

	if result.Status != StatusCompleted {
		t.Fatalf("expected status %s, got %s", StatusCompleted, result.Status)
	}
	if !contains(result.Steps, "SAGA_COMPLETED") {
		t.Fatalf("expected SAGA_COMPLETED step")
	}
	if len(result.Compensations) != 0 {
		t.Fatalf("expected no compensations, got %v", result.Compensations)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %v", result.Errors)
	}

	found := false
	for _, event := range bus.History {
		if event.Name == "shipping.created" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected shipping.created event")
	}
}

func TestPaymentFailureWithInventoryCompensation(t *testing.T) {
	orchestrator, _ := BuildOrchestrator(nil, map[string]bool{"ORDER-PAY-FAIL": true}, nil)
	result := orchestrator.Start("ORDER-PAY-FAIL", 1, 150.0, "11 B Street", "SAGA-PAY-FAIL")

	if result.Status != StatusFailedCompensated {
		t.Fatalf("expected status %s, got %s", StatusFailedCompensated, result.Status)
	}
	if !contains(result.Steps, "PAYMENT_FAILED") {
		t.Fatalf("expected PAYMENT_FAILED step")
	}
	if !contains(result.Compensations, "INVENTORY_RELEASED") {
		t.Fatalf("expected INVENTORY_RELEASED compensation")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected one error, got %v", result.Errors)
	}
}

func TestShippingFailureWithDoubleCompensation(t *testing.T) {
	orchestrator, _ := BuildOrchestrator(nil, nil, map[string]bool{"ORDER-SHIP-FAIL": true})
	result := orchestrator.Start("ORDER-SHIP-FAIL", 1, 200.0, "12 C Street", "SAGA-SHIP-FAIL")

	if result.Status != StatusFailedCompensated {
		t.Fatalf("expected status %s, got %s", StatusFailedCompensated, result.Status)
	}
	if !contains(result.Steps, "SHIPPING_FAILED") {
		t.Fatalf("expected SHIPPING_FAILED step")
	}
	if !contains(result.Compensations, "PAYMENT_REFUNDED") {
		t.Fatalf("expected PAYMENT_REFUNDED compensation")
	}
	if !contains(result.Compensations, "INVENTORY_RELEASED") {
		t.Fatalf("expected INVENTORY_RELEASED compensation")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected one error, got %v", result.Errors)
	}
}

func TestInventoryFailureWithoutCompensation(t *testing.T) {
	orchestrator, _ := BuildOrchestrator(map[string]bool{"ORDER-INV-FAIL": true}, nil, nil)
	result := orchestrator.Start("ORDER-INV-FAIL", 1, 10.0, "13 D Street", "SAGA-INV-FAIL")

	if result.Status != StatusFailed {
		t.Fatalf("expected status %s, got %s", StatusFailed, result.Status)
	}
	if !contains(result.Steps, "INVENTORY_FAILED") {
		t.Fatalf("expected INVENTORY_FAILED step")
	}
	if len(result.Compensations) != 0 {
		t.Fatalf("expected no compensations, got %v", result.Compensations)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected one error, got %v", result.Errors)
	}
}
