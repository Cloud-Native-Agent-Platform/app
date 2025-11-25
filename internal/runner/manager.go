package TaskRunner

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// RunnerManager manages TaskRunner instances.
type RunnerManager struct {
	runners map[string]*TaskRunner
	mu      sync.RWMutex
}

var (
	instance *RunnerManager
	once     sync.Once
)

// GetRunnerManager returns the singleton instance of RunnerManager.
func GetRunnerManager() *RunnerManager {
	once.Do(func() {
		instance = &RunnerManager{
			runners: make(map[string]*TaskRunner),
		}
	})
	return instance
}

// CreateRunner creates a new TaskRunner and adds it to the manager.
func (rm *RunnerManager) CreateRunner(ctx context.Context, taskId string, runtime ContainerRuntime, cfg RunnerConfig) (*TaskRunner, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	runner, err := NewTaskRunner(ctx, taskId, runtime, cfg, zap.NewNop())
	if err != nil {
		return nil, err
	}
	rm.runners[taskId] = runner
	return runner, nil
}

// ListRunner returns a list of all TaskRunners.
func (rm *RunnerManager) ListRunner() *[]*TaskRunner {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	runnersList := make([]*TaskRunner, 0, len(rm.runners))
	for _, runner := range rm.runners {
		if runner != nil {
			runnersList = append(runnersList, runner)
		}
	}
	return &runnersList
}

// DeleteRunner removes a TaskRunner by its ID.
func (rm *RunnerManager) DeleteRunner(taskId string) *TaskRunner {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	runner, exists := rm.runners[taskId]
	if !exists {
		return nil
	}

	delete(rm.runners, taskId)
	return runner
}
