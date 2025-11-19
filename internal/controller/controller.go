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

// CreateTask는 새로운 작업을 생성합니다.
func (c *Controller) CreateTask(ctx context.Context, agentID, taskID string) error {
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
		Status:  storage.TaskStatusPending,
	}

	if err := c.repo.CreateTask(ctx, task); err != nil {
		c.logger.Error("Failed to create task", zap.Error(err))
		return err
	}
	
	// TODO: Create TaskRunner with RunnerManager

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
