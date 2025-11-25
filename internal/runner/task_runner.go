package taskrunner

import (
	"context"
	"fmt"
)

// StatusCallback reports status changes and completion/errors back to controller.
type StatusCallback interface {
	OnStatusChange(taskID string, status string) error
	OnComplete(taskID string, result *RunResult) error
	OnError(taskID string, err error) error
}

// RunRequest bundles inputs for running a task.
type RunRequest struct {
	TaskID       string
	Model        string
	SystemPrompt string
	Messages     []ChatMessage
	Agent        AgentInfo
	Callback     StatusCallback
}

// TaskRunner defines the contract for task execution.
type TaskRunner interface {
	Run(ctx context.Context, req *RunRequest) (*RunResult, error)
	CheckStatus(ctx context.Context) RunnerStatus
	SendMessage(ctx context.Context, msg Message) error
	Subscribe(o TaskRunnerObserver)
	Unsubscribe(o TaskRunnerObserver)
}

// ensure Runner implements TaskRunner.
var _ TaskRunner = (*Runner)(nil)

// Run implements TaskRunner interface.
// 스켈레톤 유지: 시스템 프롬프트와 메시지를 결합해 마지막 사용자 메시지를 프롬프트로 사용.
func (r *Runner) Run(ctx context.Context, req *RunRequest) (*RunResult, error) {
	if req == nil {
		return nil, fmt.Errorf("RunRequest is nil")
	}

	// 시스템 프롬프트와 메시지를 결합
	messages := make([]ChatMessage, 0, len(req.Messages)+1)
	if req.SystemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}
	messages = append(messages, req.Messages...)

	// 마지막 사용자 메시지를 prompt로 사용
	var prompt string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			prompt = messages[i].Content
			break
		}
	}

	result, err := r.RunWithResult(ctx, req.Model, req.TaskID, prompt)
	if req.Callback != nil {
		if err != nil {
			_ = req.Callback.OnError(req.TaskID, err)
		} else {
			_ = req.Callback.OnComplete(req.TaskID, result)
		}
	}
	return result, err
}
