package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cnap-oss/app/internal/storage"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Controller는 에이전트 생성 및 관리를 담당하며, supervisor 기능도 포함합니다.
type Controller struct {
	logger *zap.Logger
	repo   *storage.Repository
}

// NewController는 새로운 Controller를 생성합니다.
func NewController(logger *zap.Logger, repo *storage.Repository) *Controller {
	return &Controller{
		logger: logger,
		repo:   repo,
	}
}

// Start는 controller 서버를 시작합니다.
func (c *Controller) Start(ctx context.Context) error {
	c.logger.Info("Starting controller server")

	// 더미 프로세스 - 실제 구현 시 여기에 controller 로직 추가
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Controller server shutting down")
			return ctx.Err()
		case <-ticker.C:
			c.logger.Debug("Controller heartbeat")
		}
	}
}

// Stop은 controller 서버를 정상적으로 종료합니다.
func (c *Controller) Stop(ctx context.Context) error {
	c.logger.Info("Stopping controller server")

	// 정리 작업 수행
	select {
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout exceeded")
	case <-time.After(100 * time.Millisecond):
		c.logger.Info("Controller server stopped")
		return nil
	}
}

// CreateAgent는 새로운 에이전트를 생성합니다.
func (c *Controller) CreateAgent(ctx context.Context, agentID, description, model, prompt string) error {
	c.logger.Info("Creating agent",
		zap.String("agent_id", agentID),
		zap.String("model", model),
	)

	if c.repo == nil {
		return fmt.Errorf("controller: repository is not configured")
	}

	payload := &storage.Agent{
		AgentID:     agentID,
		Description: description,
		Model:       model,
		Prompt:      prompt,
		Status:      storage.AgentStatusActive,
	}

	if err := c.repo.CreateAgent(ctx, payload); err != nil {
		c.logger.Error("Failed to persist agent", zap.Error(err))
		return err
	}

	c.logger.Info("Agent created successfully",
		zap.String("agent", agentID),
		zap.Int64("id", payload.ID),
	)
	return nil
}

// DeleteAgent는 기존 에이전트를 삭제합니다.
func (c *Controller) DeleteAgent(ctx context.Context, agent string) error {
	c.logger.Info("Deleting agent",
		zap.String("agent", agent),
	)

	if c.repo == nil {
		return fmt.Errorf("controller: repository is not configured")
	}

	if err := c.repo.UpsertAgentStatus(ctx, agent, storage.AgentStatusDeleted); err != nil {
		return err
	}

	c.logger.Info("Agent deleted successfully",
		zap.String("agent", agent),
	)
	return nil
}

// ListAgents는 모든 에이전트 목록을 반환합니다.
func (c *Controller) ListAgents(ctx context.Context) ([]string, error) {
	c.logger.Info("Listing agents")

	if c.repo == nil {
		return nil, fmt.Errorf("controller: repository is not configured")
	}

	records, err := c.repo.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	agents := make([]string, 0, len(records))
	for _, rec := range records {
		agents = append(agents, rec.AgentID)
	}

	c.logger.Info("Listed agents",
		zap.Int("count", len(agents)),
	)
	return agents, nil
}

// AgentInfo는 에이전트 정보를 나타냅니다.
type AgentInfo struct {
	Name        string
	Description string
	Model       string
	Prompt      string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// GetAgentInfo는 특정 에이전트의 정보를 반환합니다.
func (c *Controller) GetAgentInfo(ctx context.Context, agent string) (*AgentInfo, error) {
	c.logger.Info("Getting agent info",
		zap.String("agent", agent),
	)

	if c.repo == nil {
		return nil, fmt.Errorf("controller: repository is not configured")
	}

	rec, err := c.repo.GetAgent(ctx, agent)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("agent not found: %s", agent)
		}
		return nil, err
	}

	info := &AgentInfo{
		Name:        rec.AgentID,
		Description: rec.Description,
		Model:       rec.Model,
		Prompt:      rec.Prompt,
		Status:      rec.Status,
		CreatedAt:   rec.CreatedAt,
		UpdatedAt:   rec.UpdatedAt,
	}

	c.logger.Info("Retrieved agent info",
		zap.String("agent", agent),
		zap.String("status", info.Status),
	)
	return info, nil
}

// ValidateAgent는 에이전트 이름의 유효성을 검증합니다.
func (c *Controller) ValidateAgent(agent string) error {
	if agent == "" {
		return fmt.Errorf("agent name cannot be empty")
	}

	if len(agent) > 64 {
		return fmt.Errorf("agent name too long (max 64 characters)")
	}

	// 추가 검증 로직
	return nil
}

// CreateTask는 프롬프트와 함께 새로운 작업을 생성합니다.
// 생성 후 SendMessage를 호출하기 전까지 실행되지 않습니다.
func (c *Controller) CreateTask(ctx context.Context, agentID, taskID, prompt string) error {
	c.logger.Info("Creating task",
		zap.String("agent_id", agentID),
		zap.String("task_id", taskID),
	)

	if c.repo == nil {
		return fmt.Errorf("controller: repository is not configured")
	}

	// Agent 존재 여부 확인
	if _, err := c.repo.GetAgent(ctx, agentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("agent not found: %s", agentID)
		}
		return err
	}

	task := &storage.Task{
		TaskID:  taskID,
		AgentID: agentID,
		Prompt:  prompt,
		Status:  storage.TaskStatusPending,
	}

	if err := c.repo.CreateTask(ctx, task); err != nil {
		c.logger.Error("Failed to create task", zap.Error(err))
		return err
	}

	c.logger.Info("Task created successfully",
		zap.String("task_id", taskID),
		zap.String("agent_id", agentID),
		zap.Int64("id", task.ID),
	)
	return nil
}

// GetTask는 작업 정보를 조회합니다.
func (c *Controller) GetTask(ctx context.Context, taskID string) (*storage.Task, error) {
	c.logger.Info("Getting task",
		zap.String("task_id", taskID),
	)

	if c.repo == nil {
		return nil, fmt.Errorf("controller: repository is not configured")
	}

	task, err := c.repo.GetTask(ctx, taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, err
	}

	c.logger.Info("Retrieved task",
		zap.String("task_id", taskID),
		zap.String("status", task.Status),
	)
	return task, nil
}

// UpdateTaskStatus는 작업 상태를 업데이트합니다.
func (c *Controller) UpdateTaskStatus(ctx context.Context, taskID, status string) error {
	c.logger.Info("Updating task status",
		zap.String("task_id", taskID),
		zap.String("status", status),
	)

	if c.repo == nil {
		return fmt.Errorf("controller: repository is not configured")
	}

	// 작업 존재 여부 확인
	task, err := c.repo.GetTask(ctx, taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("task not found: %s", taskID)
		}
		return err
	}

	// 상태 업데이트
	if err := c.repo.UpsertTaskStatus(ctx, taskID, task.AgentID, status); err != nil {
		c.logger.Error("Failed to update task status", zap.Error(err))
		return err
	}

	c.logger.Info("Task status updated successfully",
		zap.String("task_id", taskID),
		zap.String("old_status", task.Status),
		zap.String("new_status", status),
	)
	return nil
}

// ListTasksByAgent는 에이전트별 작업 목록을 반환합니다.
func (c *Controller) ListTasksByAgent(ctx context.Context, agentID string) ([]storage.Task, error) {
	c.logger.Info("Listing tasks by agent",
		zap.String("agent_id", agentID),
	)

	if c.repo == nil {
		return nil, fmt.Errorf("controller: repository is not configured")
	}

	tasks, err := c.repo.ListTasksByAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}

	c.logger.Info("Listed tasks by agent",
		zap.String("agent_id", agentID),
		zap.Int("count", len(tasks)),
	)
	return tasks, nil
}

// TaskInfo는 작업 정보를 나타냅니다.
type TaskInfo struct {
	TaskID    string
	AgentID   string
	Prompt    string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GetTaskInfo는 작업의 상세 정보를 반환합니다.
func (c *Controller) GetTaskInfo(ctx context.Context, taskID string) (*TaskInfo, error) {
	c.logger.Info("Getting task info",
		zap.String("task_id", taskID),
	)

	if c.repo == nil {
		return nil, fmt.Errorf("controller: repository is not configured")
	}

	task, err := c.repo.GetTask(ctx, taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, err
	}

	info := &TaskInfo{
		TaskID:    task.TaskID,
		AgentID:   task.AgentID,
		Prompt:    task.Prompt,
		Status:    task.Status,
		CreatedAt: task.CreatedAt,
		UpdatedAt: task.UpdatedAt,
	}

	c.logger.Info("Retrieved task info",
		zap.String("task_id", taskID),
		zap.String("status", info.Status),
	)
	return info, nil
}

// ValidateTask는 작업 ID의 유효성을 검증합니다.
func (c *Controller) ValidateTask(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	if len(taskID) > 64 {
		return fmt.Errorf("task ID too long (max 64 characters)")
	}

	return nil
}

// UpdateAgent는 에이전트 정보를 수정합니다.
func (c *Controller) UpdateAgent(ctx context.Context, agentID, description, model, prompt string) error {
	c.logger.Info("Updating agent",
		zap.String("agent_id", agentID),
	)

	if c.repo == nil {
		return fmt.Errorf("controller: repository is not configured")
	}

	// Agent 존재 여부 확인
	if _, err := c.repo.GetAgent(ctx, agentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("agent not found: %s", agentID)
		}
		return err
	}

	agent := &storage.Agent{
		AgentID:     agentID,
		Description: description,
		Model:       model,
		Prompt:      prompt,
	}

	if err := c.repo.UpdateAgent(ctx, agent); err != nil {
		c.logger.Error("Failed to update agent", zap.Error(err))
		return err
	}

	c.logger.Info("Agent updated successfully", zap.String("agent", agentID))
	return nil
}

// ListAgentsWithInfo는 상세 정보를 포함한 에이전트 목록을 반환합니다.
func (c *Controller) ListAgentsWithInfo(ctx context.Context) ([]*AgentInfo, error) {
	c.logger.Info("Listing agents with info")

	if c.repo == nil {
		return nil, fmt.Errorf("controller: repository is not configured")
	}

	records, err := c.repo.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	agents := make([]*AgentInfo, 0, len(records))
	for _, rec := range records {
		agents = append(agents, &AgentInfo{
			Name:        rec.AgentID,
			Description: rec.Description,
			Model:       rec.Model,
			Prompt:      rec.Prompt,
			Status:      rec.Status,
			CreatedAt:   rec.CreatedAt,
			UpdatedAt:   rec.UpdatedAt,
		})
	}

	c.logger.Info("Listed agents with info",
		zap.Int("count", len(agents)),
	)
	return agents, nil
}

// AddMessage adds a message to an existing task without executing it.
// The message will be stored and can be sent later using SendMessage.
func (c *Controller) AddMessage(ctx context.Context, taskID, role, content string) error {
	c.logger.Info("Adding message to task",
		zap.String("task_id", taskID),
		zap.String("role", role),
	)

	if c.repo == nil {
		return fmt.Errorf("controller: repository is not configured")
	}

	// Task 존재 여부 확인
	if _, err := c.repo.GetTask(ctx, taskID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("task not found: %s", taskID)
		}
		return err
	}

	// 메시지를 파일로 저장하고 인덱스 생성
	filePath := c.saveMessageToFile(taskID, content)
	if _, err := c.repo.AppendMessageIndex(ctx, taskID, role, filePath); err != nil {
		c.logger.Error("Failed to add message", zap.Error(err))
		return err
	}

	c.logger.Info("Message added successfully",
		zap.String("task_id", taskID),
		zap.String("role", role),
	)
	return nil
}

// SendMessage triggers the execution of a task.
// This method should be called after creating a task and optionally adding messages.
// The actual execution will be handled by the RunnerManager (to be implemented).
func (c *Controller) SendMessage(ctx context.Context, taskID string) error {
	c.logger.Info("Sending message for task",
		zap.String("task_id", taskID),
	)

	if c.repo == nil {
		return fmt.Errorf("controller: repository is not configured")
	}

	// Task 조회
	task, err := c.repo.GetTask(ctx, taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("task not found: %s", taskID)
		}
		return err
	}

	// 이미 실행 중인 경우 에러
	if task.Status == storage.TaskStatusRunning {
		return fmt.Errorf("task is already running: %s", taskID)
	}

	// 완료된 작업은 재실행 불가
	if task.Status == storage.TaskStatusCompleted || task.Status == storage.TaskStatusFailed {
		return fmt.Errorf("task is already finished: %s (status: %s)", taskID, task.Status)
	}

	// 메시지 목록 조회
	messages, err := c.repo.ListMessageIndexByTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	// 프롬프트나 메시지가 없으면 에러
	if task.Prompt == "" && len(messages) == 0 {
		return fmt.Errorf("no prompt or messages to send for task: %s", taskID)
	}

	// 상태를 running으로 변경
	if err := c.repo.UpsertTaskStatus(ctx, taskID, task.AgentID, storage.TaskStatusRunning); err != nil {
		c.logger.Error("Failed to update task status", zap.Error(err))
		return err
	}

	c.logger.Info("Task execution triggered",
		zap.String("task_id", taskID),
		zap.String("agent_id", task.AgentID),
		zap.Int("message_count", len(messages)),
	)

	// TODO: RunnerManager를 통해 실제 실행 트리거
	// 현재는 상태만 변경하고, RunnerManager 연동 후 실제 실행 로직 추가

	return nil
}

// ListMessages returns all messages for a task in conversation order.
func (c *Controller) ListMessages(ctx context.Context, taskID string) ([]storage.MessageIndex, error) {
	c.logger.Info("Listing messages for task",
		zap.String("task_id", taskID),
	)

	if c.repo == nil {
		return nil, fmt.Errorf("controller: repository is not configured")
	}

	messages, err := c.repo.ListMessageIndexByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	c.logger.Info("Listed messages",
		zap.String("task_id", taskID),
		zap.Int("count", len(messages)),
	)
	return messages, nil
}

// saveMessageToFile saves message content to a file and returns the file path.
// Messages are stored in data/messages/{taskID}/{conversationIndex}.json
func (c *Controller) saveMessageToFile(taskID, content string) string {
	// TODO: 실제 파일 저장 로직 구현
	// 현재는 임시로 경로만 반환
	return fmt.Sprintf("data/messages/%s/%d.json", taskID, time.Now().UnixNano())
}

// loadMessageFromFile loads message content from a file.
func (c *Controller) loadMessageFromFile(filePath string) (string, error) {
	// TODO: 실제 파일 읽기 로직 구현
	return "", fmt.Errorf("not implemented: loadMessageFromFile")
}
