package taskrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AgentInfo represents minimal agent metadata.
type AgentInfo struct {
	AgentID string
	Model   string
	Prompt  string
}

// RunnerStatus represents the lifecycle status of a task runner.
type RunnerStatus string

const (
	StatusPending   RunnerStatus = "Pending"
	StatusRunning   RunnerStatus = "Running"
	StatusCompleted RunnerStatus = "Completed"
	StatusFailed    RunnerStatus = "Failed"
	StatusCanceled  RunnerStatus = "Canceled"
	StatusUnknown   RunnerStatus = "Unknown"
)

// Message is the unit exchanged with the agent/container.
type Message struct {
	Role    string
	Content string
}

// TaskRunnerObserver allows callers to observe status/message events.
type TaskRunnerObserver interface {
	OnStatusChange(taskID string, status RunnerStatus)
	OnMessage(taskID string, msg Message)
}

// RunnerConfig carries runtime-specific configuration.
type RunnerConfig struct {
	Image      string
	Command    []string
	Env        map[string]string
	WorkingDir string
}

// ContainerRuntime abstracts container/libcontainer operations.
type ContainerRuntime interface {
	Start(ctx context.Context, cfg RunnerConfig) error
	Send(ctx context.Context, msg Message) error
	Status(ctx context.Context) (RunnerStatus, error)
	Stop(ctx context.Context) error
}

// Runner는 short-living 에이전트 실행을 담당합니다.
type Runner struct {
	ID        string
	Status    RunnerStatus
	logger    *zap.Logger
	apiKey    string
	runtime   ContainerRuntime
	cfg       RunnerConfig
	observers []TaskRunnerObserver
	mu        sync.Mutex
}

// OpenCodeRequest는 OpenCode Zen API 요청 바디입니다.
type OpenCodeRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

// ChatMessage는 OpenCode Zen API 요청 바디의 messages 필드입니다.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenCodeResponse는 OpenCode Zen API 응답 바디입니다.
type OpenCodeResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewRunner는 새로운 Runner를 생성하고 컨테이너 런타임을 시작합니다.
func NewRunner(ctx context.Context, id string, runtime ContainerRuntime, cfg RunnerConfig, logger *zap.Logger) (*Runner, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	apiKey := os.Getenv("OPEN_CODE_API_KEY")
	if apiKey == "" {
		logger.Warn("환경 변수 OPEN_CODE_API_KEY가 설정되어 있지 않습니다 (RunWithResult 사용 시 필요)")
	}

	r := &Runner{
		ID:        id,
		Status:    StatusPending,
		logger:    logger,
		apiKey:    apiKey,
		runtime:   runtime,
		cfg:       cfg,
		observers: make([]TaskRunnerObserver, 0),
	}

	// 컨테이너 프로세스 생성만 수행, 별도 지시는 없음.
	if runtime != nil {
		if err := runtime.Start(ctx, cfg); err != nil {
			r.setStatus(StatusFailed)
			return nil, fmt.Errorf("runtime start 실패: %w", err)
		}
		r.setStatus(StatusRunning)
	}

	return r, nil
}

// RunWithResult는 프롬프트를 OpenCode Zen API의 chat/completions 엔드포인트로 보내고 결과를 반환합니다.
func (r *Runner) RunWithResult(ctx context.Context, model, name, prompt string) (*RunResult, error) {
	if r.logger == nil {
		r.logger = zap.NewNop()
	}
	if r.apiKey == "" {
		return nil, fmt.Errorf("OPEN_CODE_API_KEY가 설정되어 있지 않아 RunWithResult를 실행할 수 없습니다")
	}

	promptPreview := prompt
	if len(promptPreview) > 200 {
		promptPreview = promptPreview[:200] + "..."
	}

	// 요청 정보 로그 출력
	r.logger.Info("Sending request to OpenCode Zen API (Chat Completions endpoint)",
		zap.String("model", model),
		zap.String("name", name),
		zap.String("prompt_preview", promptPreview),
	)

	// 요청 본문 구성
	reqBody := OpenCodeRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("요청 바디 직렬화 실패: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://opencode.ai/zen/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.apiKey)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API 요청 실패: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	r.logger.Debug("Response received",
		zap.String("content_type", contentType),
		zap.String("body_preview", summarizeBody(bodyBytes)),
	)

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("API 응답 오류: %s - %s", resp.Status, summarizeBody(bodyBytes))
	}

	var apiResp OpenCodeResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w\n\n[응답 원문]\n%s", err, string(bodyBytes))
	}

	// 에러 필드 처리
	if apiResp.Error != nil {
		return nil, fmt.Errorf("API 에러: %s - %s", apiResp.Error.Type, apiResp.Error.Message)
	}

	output := "(empty result)"
	if len(apiResp.Choices) > 0 {
		output = apiResp.Choices[0].Message.Content
	}

	r.logger.Info("OpenCode 응답 수신 완료",
		zap.String("output_preview", summarizeBody([]byte(output))),
	)

	return &RunResult{
		Agent:   model,
		Name:    name,
		Success: true,
		Output:  output,
		Error:   nil,
	}, nil
}

// CheckStatus는 런타임으로부터 상태를 조회하고 반환합니다.
func (r *Runner) CheckStatus(ctx context.Context) RunnerStatus {
	if r.logger == nil {
		r.logger = zap.NewNop()
	}
	r.mu.Lock()
	current := r.Status
	r.mu.Unlock()

	if r.runtime == nil {
		return current
	}

	status, err := r.runtime.Status(ctx)
	if err != nil {
		r.logger.Warn("runtime status 조회 실패", zap.Error(err))
		return current
	}
	r.setStatus(status)
	return status
}

// SendMessage는 컨테이너/에이전트로 메시지를 전달합니다.
func (r *Runner) SendMessage(ctx context.Context, msg Message) error {
	if r.runtime == nil {
		return fmt.Errorf("runtime이 초기화되지 않았습니다")
	}
	if err := r.runtime.Send(ctx, msg); err != nil {
		r.setStatus(StatusFailed)
		return err
	}
	r.notifyMessage(msg)
	return nil
}

// Subscribe는 옵저버를 등록합니다.
func (r *Runner) Subscribe(o TaskRunnerObserver) {
	if o == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observers = append(r.observers, o)
}

// Unsubscribe는 옵저버 등록을 해제합니다.
func (r *Runner) Unsubscribe(o TaskRunnerObserver) {
	if o == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, ob := range r.observers {
		if ob == o {
			r.observers = append(r.observers[:i], r.observers[i+1:]...)
			break
		}
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

func (r *Runner) setStatus(status RunnerStatus) {
	r.mu.Lock()
	changed := r.Status != status
	r.Status = status
	observers := append([]TaskRunnerObserver(nil), r.observers...)
	r.mu.Unlock()

	if changed {
		for _, ob := range observers {
			ob.OnStatusChange(r.ID, status)
		}
	}
}

func (r *Runner) notifyMessage(msg Message) {
	r.mu.Lock()
	observers := append([]TaskRunnerObserver(nil), r.observers...)
	r.mu.Unlock()

	for _, ob := range observers {
		ob.OnMessage(r.ID, msg)
	}
}

func summarizeBody(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "<empty>"
	}
	if len(trimmed) > 200 {
		return trimmed[:200] + "..."
	}
	return trimmed
}
