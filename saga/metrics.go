package saga

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type Metrics struct {
	mu                   sync.RWMutex
	eventsPublished      map[string]float64
	sagaStartedTotal     float64
	sagaCompletedTotal   float64
	sagaFailedTotal      float64
	sagaCompensatedTotal float64
	sagaDurationBuckets  map[float64]float64
	sagaDurationCount    float64
	sagaDurationSum      float64
}

func NewMetrics() *Metrics {
	return &Metrics{
		eventsPublished: map[string]float64{},
		sagaDurationBuckets: map[float64]float64{
			0.1: 0,
			0.5: 0,
			1:   0,
			2:   0,
			5:   0,
			10:  0,
		},
	}
}

var DefaultMetrics = NewMetrics()

func (m *Metrics) ObserveEventPublished(eventName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventsPublished[eventName]++
}

func (m *Metrics) ObserveSagaStarted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sagaStartedTotal++
}

func (m *Metrics) ObserveSagaResult(status SagaStatus, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch status {
	case StatusCompleted:
		m.sagaCompletedTotal++
	case StatusFailed:
		m.sagaFailedTotal++
	case StatusFailedCompensated:
		m.sagaCompensatedTotal++
	}

	seconds := duration.Seconds()
	m.sagaDurationCount++
	m.sagaDurationSum += seconds
	for bucket := range m.sagaDurationBuckets {
		if seconds <= bucket {
			m.sagaDurationBuckets[bucket]++
		}
	}
}

func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = w.Write([]byte(m.RenderPrometheus()))
	})
}

func (m *Metrics) RenderPrometheus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sb strings.Builder
	writeLine := func(line string) {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	writeLine("# HELP saga_events_published_total Total events published by event name")
	writeLine("# TYPE saga_events_published_total counter")
	eventNames := make([]string, 0, len(m.eventsPublished))
	for name := range m.eventsPublished {
		eventNames = append(eventNames, name)
	}
	sort.Strings(eventNames)
	for _, name := range eventNames {
		writeLine(fmt.Sprintf("saga_events_published_total{event=%q} %.0f", name, m.eventsPublished[name]))
	}

	writeLine("# HELP saga_started_total Total started sagas")
	writeLine("# TYPE saga_started_total counter")
	writeLine(fmt.Sprintf("saga_started_total %.0f", m.sagaStartedTotal))

	writeLine("# HELP saga_completed_total Total completed sagas")
	writeLine("# TYPE saga_completed_total counter")
	writeLine(fmt.Sprintf("saga_completed_total %.0f", m.sagaCompletedTotal))

	writeLine("# HELP saga_failed_total Total failed sagas without full compensation")
	writeLine("# TYPE saga_failed_total counter")
	writeLine(fmt.Sprintf("saga_failed_total %.0f", m.sagaFailedTotal))

	writeLine("# HELP saga_failed_compensated_total Total failed but compensated sagas")
	writeLine("# TYPE saga_failed_compensated_total counter")
	writeLine(fmt.Sprintf("saga_failed_compensated_total %.0f", m.sagaCompensatedTotal))

	writeLine("# HELP saga_duration_seconds Saga execution duration histogram")
	writeLine("# TYPE saga_duration_seconds histogram")
	buckets := make([]float64, 0, len(m.sagaDurationBuckets))
	for b := range m.sagaDurationBuckets {
		buckets = append(buckets, b)
	}
	sort.Float64s(buckets)
	for _, b := range buckets {
		writeLine(fmt.Sprintf("saga_duration_seconds_bucket{le=%q} %.0f", fmt.Sprintf("%.1f", b), m.sagaDurationBuckets[b]))
	}
	writeLine(fmt.Sprintf("saga_duration_seconds_bucket{le=\"+Inf\"} %.0f", m.sagaDurationCount))
	writeLine(fmt.Sprintf("saga_duration_seconds_sum %.6f", m.sagaDurationSum))
	writeLine(fmt.Sprintf("saga_duration_seconds_count %.0f", m.sagaDurationCount))

	return sb.String()
}
