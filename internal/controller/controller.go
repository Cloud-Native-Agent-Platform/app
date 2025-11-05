package controller

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Controller는 에이전트 생성 및 관리를 담당합니다.
type Controller struct {
	logger *zap.Logger
}

// NewController는 새로운 Controller를 생성합니다.
func NewController(logger *zap.Logger) *Controller {
	return &Controller{
		logger: logger,
	}
}

// CreateAgent는 새로운 에이전트를 생성합니다.
func (c *Controller) CreateAgent(ctx context.Context, agent string) error {
	c.logger.Info("Creating agent",
		zap.String("agent", agent),
	)

	// 더미 생성 - 실제 구현 시 여기에 에이전트 생성 로직 추가
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second):
		c.logger.Info("Agent created successfully",
			zap.String("agent", agent),
		)
		return nil
	}
}

// DeleteAgent는 기존 에이전트를 삭제합니다.
func (c *Controller) DeleteAgent(ctx context.Context, agent string) error {
	c.logger.Info("Deleting agent",
		zap.String("agent", agent),
	)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(500 * time.Millisecond):
		c.logger.Info("Agent deleted successfully",
			zap.String("agent", agent),
		)
		return nil
	}
}

// ListAgents는 모든 에이전트 목록을 반환합니다.
func (c *Controller) ListAgents(ctx context.Context) ([]string, error) {
	c.logger.Info("Listing agents")

	// 더미 목록
	agents := []string{
		"agent-1",
		"agent-2",
		"agent-3",
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(500 * time.Millisecond):
		c.logger.Info("Listed agents",
			zap.Int("count", len(agents)),
		)
		return agents, nil
	}
}

// AgentInfo는 에이전트 정보를 나타냅니다.
type AgentInfo struct {
	Name      string
	Status    string
	CreatedAt time.Time
}

// GetAgentInfo는 특정 에이전트의 정보를 반환합니다.
func (c *Controller) GetAgentInfo(ctx context.Context, agent string) (*AgentInfo, error) {
	c.logger.Info("Getting agent info",
		zap.String("agent", agent),
	)

	info := &AgentInfo{
		Name:      agent,
		Status:    "active",
		CreatedAt: time.Now().Add(-24 * time.Hour),
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(300 * time.Millisecond):
		c.logger.Info("Retrieved agent info",
			zap.String("agent", agent),
			zap.String("status", info.Status),
		)
		return info, nil
	}
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
