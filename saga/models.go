package saga

type SagaStatus string

const (
	StatusStarted           SagaStatus = "STARTED"
	StatusInProgress        SagaStatus = "IN_PROGRESS"
	StatusCompleted         SagaStatus = "COMPLETED"
	StatusFailed            SagaStatus = "FAILED"
	StatusFailedCompensated SagaStatus = "FAILED_COMPENSATED"
)

type Event struct {
	Name    string         `json:"name"`
	SagaID  string         `json:"saga_id"`
	Payload map[string]any `json:"payload"`
}

type SagaResult struct {
	SagaID        string
	Status        SagaStatus
	Steps         []string
	Compensations []string
	Errors        []string
}
