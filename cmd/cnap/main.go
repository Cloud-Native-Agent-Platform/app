package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/cnap-oss/app/internal/connector"
	"github.com/cnap-oss/app/internal/controller"
	"github.com/cnap-oss/app/internal/runner"
	"github.com/cnap-oss/app/internal/supervisor"
	"github.com/spf13/cobra"
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

	rootCmd := &cobra.Command{
		Use:   "cnap",
		Short: "CNAP - AI Agent Supervisor CLI",
		Long:  `CNAP is a command-line interface for managing AI agent supervisor and connector servers.`,
		Version: fmt.Sprintf("%s (built at %s)", Version, BuildTime),
	}

	// start 명령어
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start supervisor and connector server processes",
		Long:  `Start the server processes for internal/supervisor and internal/connector.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(logger)
		},
	}

	// agent 명령어 그룹
	agentCmd := &cobra.Command{
		Use:   "agent",
		Short: "Agent operations",
		Long:  `Commands for managing and running agents.`,
	}

	// agent run 명령어
	agentRunCmd := &cobra.Command{
		Use:   "run <agent> <name> <prompt>",
		Short: "Run an agent",
		Long:  `Run a specified agent with given name and prompt`,
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			agent := args[0]
			name := args[1]
			prompt := args[2]
			return runAgent(logger, agent, name, prompt)
		},
	}

	// agent create 명령어
	agentCreateCmd := &cobra.Command{
		Use:   "create <agent>",
		Short: "Agent",
		Long:  `Create a new agent`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agent := args[0]
			return createAgent(logger, agent)
		},
	}

	// 명령어 구성
	agentCmd.AddCommand(agentRunCmd)
	agentCmd.AddCommand(agentCreateCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(agentCmd)

	if err := rootCmd.Execute(); err != nil {
		logger.Error("Command execution failed", zap.Error(err))
		os.Exit(1)
	}
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

// runStart는 supervisor와 connector 서버를 시작합니다.
func runStart(logger *zap.Logger) error {
	logger.Info("Starting CNAP servers",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
	)

	// Context 생성
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown을 위한 signal 처리
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 서버 인스턴스 생성
	supervisorServer := supervisor.NewServer(logger.Named("supervisor"))
	connectorServer := connector.NewServer(logger.Named("connector"))

	// 에러 채널
	errChan := make(chan error, 2)
	var wg sync.WaitGroup

	// Supervisor 서버 시작
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := supervisorServer.Start(ctx); err != nil && err != context.Canceled {
			errChan <- fmt.Errorf("supervisor error: %w", err)
		}
	}()

	// Connector 서버 시작
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := connectorServer.Start(ctx); err != nil && err != context.Canceled {
			errChan <- fmt.Errorf("connector error: %w", err)
		}
	}()

	// 종료 대기
	select {
	case <-sigChan:
		logger.Info("Shutdown signal received")
		cancel()
	case err := <-errChan:
		logger.Error("Server error", zap.Error(err))
		cancel()
		return err
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	shutdownErrChan := make(chan error, 2)
	
	go func() {
		shutdownErrChan <- supervisorServer.Stop(shutdownCtx)
	}()
	
	go func() {
		shutdownErrChan <- connectorServer.Stop(shutdownCtx)
	}()

	// 모든 고루틴이 종료될 때까지 대기
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Shutdown 에러 확인
	for i := 0; i < 2; i++ {
		if err := <-shutdownErrChan; err != nil {
			logger.Error("Shutdown error", zap.Error(err))
		}
	}

	logger.Info("Servers stopped gracefully")
	return nil
}

// runAgent는 에이전트를 실행합니다.
func runAgent(logger *zap.Logger, agent, name, prompt string) error {
	logger.Info("Running agent",
		zap.String("agent", agent),
		zap.String("name", name),
		zap.String("prompt", prompt),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	r := runner.NewRunner(logger.Named("runner"))
	
	result, err := r.RunWithResult(ctx, agent, name, prompt)
	if err != nil {
		logger.Error("Failed to run agent", zap.Error(err))
		return err
	}

	if result.Success {
		fmt.Printf("✓ Agent executed successfully\n")
		fmt.Printf("Output: %s\n", result.Output)
	} else {
		fmt.Printf("✗ Agent execution failed\n")
		if result.Error != nil {
			fmt.Printf("Error: %s\n", result.Error.Error())
		}
		return fmt.Errorf("agent execution failed")
	}

	return nil
}

// createAgent는 새로운 에이전트를 생성합니다.
func createAgent(logger *zap.Logger, agent string) error {
	logger.Info("Creating agent",
		zap.String("agent", agent),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	ctrl := controller.NewController(logger.Named("controller"))
	
	// 에이전트 이름 검증
	if err := ctrl.ValidateAgent(agent); err != nil {
		logger.Error("Invalid agent name", zap.Error(err))
		return err
	}

	// 에이전트 생성
	if err := ctrl.CreateAgent(ctx, agent); err != nil {
		logger.Error("Failed to create agent", zap.Error(err))
		return err
	}

	fmt.Printf("✓ Agent '%s' created successfully\n", agent)
	return nil
}
