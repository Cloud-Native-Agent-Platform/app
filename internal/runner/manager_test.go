package TaskRunner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockAgent is a mock implementation of the Agent interface.
type MockAgent struct{}

type fakeRuntime struct {
	started bool
	status  RunnerStatus
	sent    []Message
}

func (f *fakeRuntime) Start(ctx context.Context, cfg RunnerConfig) error {
	f.started = true
	f.status = StatusRunning
	return nil
}

func (f *fakeRuntime) Send(ctx context.Context, msg Message) error {
	f.sent = append(f.sent, msg)
	return nil
}

func (f *fakeRuntime) Status(ctx context.Context) (RunnerStatus, error) {
	return f.status, nil
}

func (f *fakeRuntime) Stop(ctx context.Context) error {
	f.status = StatusCanceled
	return nil
}

func TestRunnerManager_Singleton(t *testing.T) {
	rm1 := GetRunnerManager()
	rm2 := GetRunnerManager()

	assert.Equal(t, rm1, rm2, "GetRunnerManager should return the same instance")
}

func TestRunnerManager_CRUD(t *testing.T) {
	rm := GetRunnerManager()

	// Ensure clean state for test (though singleton persists, so we might need to clear it if tests run in same process)
	// Since we can't easily reset the singleton once, we just work with what we have or clear the map manually.
	rm.mu.Lock()
	rm.runners = make(map[string]*TaskRunner)
	rm.mu.Unlock()

	taskId := "task-1"
	runtime := &fakeRuntime{}
	cfg := RunnerConfig{Image: "test-image"}

	// Create
	runner, err := rm.CreateRunner(context.Background(), taskId, runtime, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, runner)
	assert.Equal(t, taskId, runner.ID)
	assert.Equal(t, StatusRunning, runner.Status)

	// List
	runners := rm.ListRunner()
	assert.NotNil(t, runners)
	assert.Len(t, *runners, 1)
	assert.Equal(t, taskId, (*runners)[0].ID)

	// Delete
	deletedRunner := rm.DeleteRunner(taskId)
	assert.NotNil(t, deletedRunner)
	assert.Equal(t, taskId, deletedRunner.ID)

	// List after delete
	runners = rm.ListRunner()
	assert.Len(t, *runners, 0)

	// Delete non-existent
	nilRunner := rm.DeleteRunner("non-existent")
	assert.Nil(t, nilRunner)
}
