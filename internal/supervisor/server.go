package supervisor

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Server는 supervisor 서버를 나타냅니다.
type Server struct {
	logger *zap.Logger
}

// NewServer는 새로운 supervisor 서버를 생성합니다.
func NewServer(logger *zap.Logger) *Server {
	return &Server{
		logger: logger,
	}
}

// Start는 supervisor 서버를 시작합니다.
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting supervisor server")

	// 더미 프로세스 - 실제 구현 시 여기에 supervisor 로직 추가
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Supervisor server shutting down")
			return ctx.Err()
		case <-ticker.C:
			s.logger.Debug("Supervisor heartbeat")
		}
	}
}

// Stop은 supervisor 서버를 정상적으로 종료합니다.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping supervisor server")
	
	// 정리 작업 수행
	select {
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout exceeded")
	case <-time.After(100 * time.Millisecond):
		s.logger.Info("Supervisor server stopped")
		return nil
	}
}