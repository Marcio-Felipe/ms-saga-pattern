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
	Name    string
	SagaID  string
	Payload map[string]any
}

type SagaResult struct {
	SagaID        string
	Status        SagaStatus
	Steps         []string
	Compensations []string
	Errors        []string
}
