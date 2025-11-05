package runner

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Runner는 short-living 에이전트 실행을 담당합니다.
type Runner struct {
	logger *zap.Logger
}

// NewRunner는 새로운 Runner를 생성합니다.
func NewRunner(logger *zap.Logger) *Runner {
	return &Runner{
		logger: logger,
	}
}

// Run은 지정된 에이전트를 실행합니다.
func (r *Runner) Run(ctx context.Context, agent, name, prompt string) error {
	r.logger.Info("Running agent",
		zap.String("agent", agent),
		zap.String("name", name),
		zap.String("prompt", prompt),
	)

	// 더미 실행 - 실제 구현 시 여기에 에이전트 실행 로직 추가
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
		r.logger.Info("Agent execution completed",
			zap.String("agent", agent),
			zap.String("name", name),
		)
		return nil
	}
}

// RunResult는 에이전트 실행 결과를 나타냅니다.
type RunResult struct {
	Agent   string
	Name    string
	Success bool
	Output  string
	Error   error
}

// RunWithResult는 에이전트를 실행하고 결과를 반환합니다.
func (r *Runner) RunWithResult(ctx context.Context, agent, name, prompt string) (*RunResult, error) {
	r.logger.Info("Running agent with result",
		zap.String("agent", agent),
		zap.String("name", name),
		zap.String("prompt", prompt),
	)

	result := &RunResult{
		Agent: agent,
		Name:  name,
	}

	// 더미 실행
	select {
	case <-ctx.Done():
		result.Success = false
		result.Error = ctx.Err()
		return result, ctx.Err()
	case <-time.After(2 * time.Second):
		result.Success = true
		result.Output = fmt.Sprintf("Agent %s (%s) executed successfully with prompt: %s", agent, name, prompt)
		r.logger.Info("Agent execution completed with result",
			zap.String("agent", agent),
			zap.String("name", name),
			zap.String("output", result.Output),
		)
		return result, nil
	}
}
