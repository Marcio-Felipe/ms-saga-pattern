package saga

import (
	"strings"
	"testing"
	"time"
)

func TestMetricsRenderPrometheus(t *testing.T) {
	m := NewMetrics()
	m.ObserveEventPublished("inventory.reserve.requested")
	m.ObserveSagaStarted()
	m.ObserveSagaResult(StatusCompleted, 120*time.Millisecond)

	output := m.RenderPrometheus()
	checks := []string{
		"saga_events_published_total{event=\"inventory.reserve.requested\"} 1",
		"saga_started_total 1",
		"saga_completed_total 1",
		"saga_duration_seconds_count 1",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Fatalf("expected metrics output to contain %q", check)
		}
	}
}
