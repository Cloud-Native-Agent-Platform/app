package mocks

import (
	"context"
	"fmt"

	taskrunner "github.com/cnap-oss/app/internal/runner"
)

// MockRunner는 테스트용 TaskRunner 구현입니다.
type MockRunner struct {
	Responses map[string]string
	Errors    map[string]error
	Calls     []*taskrunner.RunRequest

	DefaultResponse string
	Status          taskrunner.RunnerStatus

	observers []taskrunner.TaskRunnerObserver
}

// NewMockRunner는 새로운 MockRunner를 생성합니다.
func NewMockRunner() *MockRunner {
	return &MockRunner{
		Responses:       make(map[string]string),
		Errors:          make(map[string]error),
		Calls:           make([]*taskrunner.RunRequest, 0),
		DefaultResponse: "Mock response",
		Status:          taskrunner.StatusPending,
		observers:       make([]taskrunner.TaskRunnerObserver, 0),
	}
}

// ensure MockRunner implements TaskRunner.
var _ taskrunner.TaskRunner = (*MockRunner)(nil)

// Run implements TaskRunner interface.
func (m *MockRunner) Run(ctx context.Context, req *taskrunner.RunRequest) (*taskrunner.RunResult, error) {
	// 호출 기록
	m.Calls = append(m.Calls, req)

	// 에러 체크
	if err, ok := m.Errors[req.TaskID]; ok {
		return nil, err
	}

	// 응답 조회
	response := m.DefaultResponse
	if resp, ok := m.Responses[req.TaskID]; ok {
		response = resp
	}

	return &taskrunner.RunResult{
		Agent:   req.Model,
		Name:    req.TaskID,
		Success: true,
		Output:  response,
		Error:   nil,
	}, nil
}

// CheckStatus는 설정된 Status를 반환합니다.
func (m *MockRunner) CheckStatus(ctx context.Context) taskrunner.RunnerStatus {
	return m.Status
}

// SendMessage는 메시지를 전송했다고 가정하고 옵저버에게 알립니다.
func (m *MockRunner) SendMessage(ctx context.Context, msg taskrunner.Message) error {
	for _, ob := range m.observers {
		ob.OnMessage("mock", msg)
	}
	return nil
}

// Subscribe는 옵저버를 등록합니다.
func (m *MockRunner) Subscribe(o taskrunner.TaskRunnerObserver) {
	if o == nil {
		return
	}
	m.observers = append(m.observers, o)
}

// Unsubscribe는 옵저버 등록을 해제합니다.
func (m *MockRunner) Unsubscribe(o taskrunner.TaskRunnerObserver) {
	if o == nil {
		return
	}
	for i, ob := range m.observers {
		if ob == o {
			m.observers = append(m.observers[:i], m.observers[i+1:]...)
			break
		}
	}
}

// SetResponse는 특정 taskID에 대한 응답을 설정합니다.
func (m *MockRunner) SetResponse(taskID, response string) {
	m.Responses[taskID] = response
}

// SetError는 특정 taskID에 대한 에러를 설정합니다.
func (m *MockRunner) SetError(taskID string, err error) {
	m.Errors[taskID] = err
}

// SetErrorMessage는 특정 taskID에 대한 에러 메시지를 설정합니다.
func (m *MockRunner) SetErrorMessage(taskID, message string) {
	m.Errors[taskID] = fmt.Errorf("%s", message)
}

// GetCallCount는 Run 호출 횟수를 반환합니다.
func (m *MockRunner) GetCallCount() int {
	return len(m.Calls)
}

// GetLastCall은 마지막 Run 호출을 반환합니다.
func (m *MockRunner) GetLastCall() *taskrunner.RunRequest {
	if len(m.Calls) == 0 {
		return nil
	}
	return m.Calls[len(m.Calls)-1]
}

// Reset은 모든 호출 기록을 초기화합니다.
func (m *MockRunner) Reset() {
	m.Calls = make([]*taskrunner.RunRequest, 0)
}
