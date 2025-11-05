package storage

const (
	AgentStatusActive  = "active"
	AgentStatusIdle    = "idle"
	AgentStatusBusy    = "busy"
	AgentStatusDeleted = "deleted"

	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusFailed    = "failed"
	TaskStatusCanceled  = "canceled"

	RunStepStatusPending   = "pending"
	RunStepStatusRunning   = "running"
	RunStepStatusCompleted = "completed"
	RunStepStatusFailed    = "failed"

	RunStepTypeSystem     = "system"
	RunStepTypeTool       = "tool"
	RunStepTypeModel      = "model"
	RunStepTypeCheckpoint = "checkpoint"
)
