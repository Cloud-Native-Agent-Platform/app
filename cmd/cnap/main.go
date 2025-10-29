package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Logger 초기화
	logger, err := initLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting application", 
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
	)

	// Context 생성
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown을 위한 signal 처리
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 메인 애플리케이션 로직 실행
	go func() {
		if err := run(ctx, logger); err != nil {
			logger.Error("Application error", zap.Error(err))
			cancel()
		}
	}()

	// Shutdown signal 대기
	<-sigChan
	logger.Info("Shutdown signal received")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := shutdown(shutdownCtx, logger); err != nil {
		logger.Error("Shutdown error", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Application stopped gracefully")
}

// initLogger는 zap logger를 초기화합니다.
func initLogger() (*zap.Logger, error) {
	env := os.Getenv("ENV")
	logLevel := os.Getenv("LOG_LEVEL")

	var config zap.Config
	if env == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	// LOG_LEVEL 환경변수가 설정되어 있으면 적용
	if logLevel != "" {
		level, err := zap.ParseAtomicLevel(logLevel)
		if err == nil {
			config.Level = level
		}
	}

	return config.Build()
}

// run은 애플리케이션의 메인 로직을 실행합니다.
func run(ctx context.Context, logger *zap.Logger) error {
	logger.Info("Application running")

	// 메인 로직 (여기서는 간단히 context 취소를 대기)
	<-ctx.Done()
	return ctx.Err()
}

// shutdown은 애플리케이션을 정상적으로 종료합니다.
func shutdown(ctx context.Context, logger *zap.Logger) error {
	logger.Info("Starting graceful shutdown")

	// 여기에 cleanup 로직 추가
	// 예: 데이터베이스 연결 종료, 진행 중인 작업 완료 등

	select {
	case <-ctx.Done():
		logger.Warn("Shutdown timeout exceeded")
		return ctx.Err()
	case <-time.After(100 * time.Millisecond):
		logger.Info("Shutdown completed")
		return nil
	}
}
