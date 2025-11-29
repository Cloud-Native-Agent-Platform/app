package taskrunner

import (
	"os"
	"sync"

	"go.uber.org/zap"
)

// RunnerManager manages Runner instances.
type RunnerManager struct {
	runners map[string]*Runner
	logger  *zap.Logger
	apiKey  string
	mu      sync.RWMutex
}

var (
	instance *RunnerManager
	once     sync.Once
)

// GetRunnerManager returns the singleton instance of RunnerManager.
func GetRunnerManager(logger *zap.Logger) *RunnerManager {
	once.Do(func() {
		apiKey := os.Getenv("OPEN_CODE_API_KEY")
		if apiKey == "" {
			logger.Fatal("환경 변수 OPEN_CODE_API_KEY가 설정되어 있지 않습니다")
		}

		instance = &RunnerManager{
			runners: make(map[string]*Runner),
			logger:  logger,
			apiKey:  apiKey,
		}
	})
	return instance
}

// CreateRunner creates a new Runner and adds it to the manager.
func (rm *RunnerManager) CreateRunner(taskId string, agent AgentInfo, callback StatusCallback) *Runner {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	runner := &Runner{
		ID:       taskId,
		Status:   "Pending",
		logger:   rm.logger,
		apiKey:   rm.apiKey,
		callback: callback,
	}
	rm.runners[taskId] = runner
	return runner
}

// GetRunner returns a Runner by its ID.
func (rm *RunnerManager) GetRunner(taskId string) *Runner {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.runners[taskId]
}

// ListRunner returns a list of all Runners.
func (rm *RunnerManager) ListRunner() *[]Runner {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	runnersList := make([]Runner, 0, len(rm.runners))
	for _, runner := range rm.runners {
		if runner != nil {
			runnersList = append(runnersList, *runner)
		}
	}
	return &runnersList
}

// DeleteRunner removes a Runner by its ID.
func (rm *RunnerManager) DeleteRunner(taskId string) *Runner {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	runner, exists := rm.runners[taskId]
	if !exists {
		return nil
	}

	delete(rm.runners, taskId)
	return runner
}
