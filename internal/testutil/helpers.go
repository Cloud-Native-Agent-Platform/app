package testutil

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestContext는 테스트에 필요한 컨텍스트와 정리 함수를 포함합니다.
type TestContext struct {
	Ctx     context.Context
	Cancel  context.CancelFunc
	TempDir string
	T       *testing.T
}

// NewTestContext는 새로운 테스트 컨텍스트를 생성합니다.
func NewTestContext(t *testing.T) *TestContext {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	tempDir := t.TempDir()

	tc := &TestContext{
		Ctx:     ctx,
		Cancel:  cancel,
		TempDir: tempDir,
		T:       t,
	}

	t.Cleanup(func() {
		cancel()
	})

	return tc
}

// CreateTempFile은 임시 파일을 생성하고 내용을 작성합니다.
func (tc *TestContext) CreateTempFile(name, content string) string {
	tc.T.Helper()

	filePath := filepath.Join(tc.TempDir, name)
	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		tc.T.Fatalf("디렉토리 생성 실패: %v", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		tc.T.Fatalf("파일 작성 실패: %v", err)
	}

	return filePath
}

// LoadFixture는 테스트 픽스처 파일을 로드합니다.
func LoadFixture(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("픽스처 로드 실패: %v", err)
	}

	return data
}

// LoadJSONFixture는 JSON 픽스처 파일을 로드하고 구조체로 언마샬합니다.
func LoadJSONFixture(t *testing.T, path string, v interface{}) {
	t.Helper()

	data := LoadFixture(t, path)
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("JSON 언마샬 실패: %v", err)
	}
}

// AssertNoError는 에러가 없는지 확인합니다.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("예상치 못한 에러 발생: %v", err)
	}
}

// AssertError는 에러가 발생했는지 확인합니다.
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("에러가 예상되었지만 발생하지 않음")
	}
}

// AssertEqual은 두 값이 같은지 확인합니다.
func AssertEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Fatalf("값이 일치하지 않음: got=%v, want=%v", got, want)
	}
}

// AssertNotEqual은 두 값이 다른지 확인합니다.
func AssertNotEqual(t *testing.T, got, notWant interface{}) {
	t.Helper()
	if got == notWant {
		t.Fatalf("값이 일치함: got=%v, notWant=%v", got, notWant)
	}
}

// AssertTrue는 조건이 참인지 확인합니다.
func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()
	if !condition {
		t.Fatalf("조건이 거짓: %s", message)
	}
}

// AssertFalse는 조건이 거짓인지 확인합니다.
func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()
	if condition {
		t.Fatalf("조건이 참: %s", message)
	}
}

// SetupTestEnvironment는 테스트 환경 변수를 설정합니다.
func SetupTestEnvironment(t *testing.T, envVars map[string]string) {
	t.Helper()

	for key, value := range envVars {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("환경 변수 설정 실패 (%s): %v", key, err)
		}
	}

	t.Cleanup(func() {
		for key := range envVars {
			if err := os.Unsetenv(key); err != nil {
				t.Errorf("환경 변수 제거 실패 (%s): %v", key, err)
			}
		}
	})
}

// GetTestEnvVar는 테스트용 환경 변수를 반환합니다.
func GetTestEnvVar(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// WaitForCondition은 조건이 만족될 때까지 대기합니다.
func WaitForCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("조건 만족 대기 시간 초과")
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// Mock 생성 헬퍼

// MockConfig는 테스트용 설정 Mock을 생성합니다.
type MockConfig struct {
	Data map[string]interface{}
}

// NewMockConfig는 새로운 Mock 설정을 생성합니다.
func NewMockConfig() *MockConfig {
	return &MockConfig{
		Data: make(map[string]interface{}),
	}
}

// Set은 설정 값을 설정합니다.
func (m *MockConfig) Set(key string, value interface{}) {
	m.Data[key] = value
}

// Get은 설정 값을 가져옵니다.
func (m *MockConfig) Get(key string) interface{} {
	return m.Data[key]
}

// GetString은 문자열 설정 값을 가져옵니다.
func (m *MockConfig) GetString(key string) string {
	if v, ok := m.Data[key].(string); ok {
		return v
	}
	return ""
}

// GetInt는 정수 설정 값을 가져옵니다.
func (m *MockConfig) GetInt(key string) int {
	if v, ok := m.Data[key].(int); ok {
		return v
	}
	return 0
}

// GetBool은 불린 설정 값을 가져옵니다.
func (m *MockConfig) GetBool(key string) bool {
	if v, ok := m.Data[key].(bool); ok {
		return v
	}
	return false
}

// MockLogger는 테스트용 로거 Mock을 생성합니다.
type MockLogger struct {
	Logs []string
}

// NewMockLogger는 새로운 Mock 로거를 생성합니다.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		Logs: make([]string, 0),
	}
}

// Info는 정보 로그를 기록합니다.
func (m *MockLogger) Info(msg string) {
	m.Logs = append(m.Logs, "INFO: "+msg)
}

// Error는 에러 로그를 기록합니다.
func (m *MockLogger) Error(msg string) {
	m.Logs = append(m.Logs, "ERROR: "+msg)
}

// Debug는 디버그 로그를 기록합니다.
func (m *MockLogger) Debug(msg string) {
	m.Logs = append(m.Logs, "DEBUG: "+msg)
}

// GetLogs는 모든 로그를 반환합니다.
func (m *MockLogger) GetLogs() []string {
	return m.Logs
}

// HasLog는 특정 로그가 존재하는지 확인합니다.
func (m *MockLogger) HasLog(expected string) bool {
	for _, log := range m.Logs {
		if log == expected {
			return true
		}
	}
	return false
}

// TestServer는 테스트용 서버 설정을 제공합니다.
type TestServer struct {
	Port    int
	BaseURL string
}

// NewTestServer는 새로운 테스트 서버를 생성합니다.
func NewTestServer(port int) *TestServer {
	return &TestServer{
		Port:    port,
		BaseURL: "http://localhost:" + string(rune(port)),
	}
}

// SkipIfShort는 짧은 테스트 모드에서 테스트를 건너뜁니다.
func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("짧은 테스트 모드에서 건너뜀")
	}
}

// SkipCI는 CI 환경에서 테스트를 건너뜁니다.
func SkipCI(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("CI 환경에서 건너뜀")
	}
}
