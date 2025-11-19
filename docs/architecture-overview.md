# Architecture Overview

본 문서는 CNAP 시스템의 도메인 구조, Controller API, 그리고 데이터베이스 스키마를 설명합니다.

## 목차

1. [도메인 모델](#도메인-모델)
2. [Controller API](#controller-api)
3. [데이터베이스 스키마](#데이터베이스-스키마)
4. [데이터 흐름](#데이터-흐름)

---

## 도메인 모델

CNAP는 AI 에이전트와 작업(Task)을 관리하는 시스템으로, 다음과 같은 핵심 도메인 엔티티로 구성됩니다.

### 엔티티 관계도

```
Agent (1) ──→ (N) Task ──→ (N) MessageIndex
                    │
                    ├──→ (N) RunStep
                    └──→ (N) Checkpoint
```

### 1. Agent (에이전트)

**설명**: AI 에이전트를 나타내는 멀티테넌트 논리 단위

**속성**:
- `AgentID` (string): 에이전트 고유 식별자 (최대 64자)
- `Description` (string): 에이전트 설명
- `Model` (string): 사용하는 AI 모델명 (예: gpt-4, claude-3)
- `Prompt` (string): 시스템 프롬프트
- `Status` (string): 에이전트 상태 (active, idle, busy, deleted)

**상태 전이**:
```
         create
           │
           ▼
   ┌──── active ◄────┐
   │       │         │
   │       ▼         │
   │     idle ──► busy
   │                 │
   │                 ▼
   └──────────► deleted
```

**비즈니스 규칙**:
- `AgentID`는 1~64자여야 함
- 삭제된 에이전트는 `status = deleted`로 soft delete
- 하나의 에이전트는 여러 작업을 가질 수 있음

---

### 2. Task (작업)

**설명**: 에이전트가 수행하는 실행 단위

**속성**:
- `TaskID` (string): 작업 고유 식별자 (최대 64자)
- `AgentID` (string): 작업을 수행하는 에이전트 ID
- `Status` (string): 작업 상태 (pending, running, completed, failed, canceled)

**상태 전이**:
```
         create
           │
           ▼
        pending
           │
           ▼
        running ──┬──► completed
           │      │
           │      ├──► failed
           │      │
           │      └──► canceled
           ▼
        canceled
```

**비즈니스 규칙**:
- 작업 생성 시 해당 에이전트가 반드시 존재해야 함
- `TaskID`는 1~64자여야 함
- 작업은 한 번에 하나의 에이전트에만 속함

---

### 3. MessageIndex (메시지 인덱스)

**설명**: 작업별 대화 메시지의 파일 경로를 추적

**속성**:
- `TaskID` (string): 작업 식별자
- `ConversationIndex` (int): 대화 순서 번호
- `FilePath` (string): JSON 메시지 파일 경로

**비즈니스 규칙**:
- 실제 메시지 본문은 DB가 아닌 로컬 JSON 파일에 저장
- DB에는 파일 경로만 저장하여 참조
- `(TaskID, ConversationIndex)` 조합은 고유해야 함

---

### 4. RunStep (실행 단계)

**설명**: 작업의 세부 실행 단계를 추적

**속성**:
- `TaskID` (string): 작업 식별자
- `StepNo` (int): 단계 번호
- `Type` (string): 단계 유형 (system, tool, model, checkpoint)
- `Status` (string): 단계 상태 (pending, running, completed, failed)

**단계 유형**:
- `system`: 시스템 이벤트
- `tool`: 도구 호출
- `model`: AI 모델 호출
- `checkpoint`: Git 체크포인트 생성

**비즈니스 규칙**:
- `(TaskID, StepNo)` 조합은 고유해야 함
- 단계 번호는 순차적으로 증가

---

### 5. Checkpoint (체크포인트)

**설명**: 작업의 Git 스냅샷 참조

**속성**:
- `TaskID` (string): 작업 식별자
- `GitHash` (string): Git 커밋 해시

**비즈니스 규칙**:
- `(TaskID, GitHash)` 조합은 고유해야 함
- 중복 체크포인트는 무시 (upsert with DoNothing)

---

## Controller API

Controller는 에이전트와 작업의 생명주기를 관리하는 핵심 계층입니다.

### Agent 관리 API

#### CreateAgent
새로운 에이전트를 생성합니다.

```go
func (c *Controller) CreateAgent(
    ctx context.Context,
    agentID string,
    description string,
    model string,
    prompt string,
) error
```

**파라미터**:
- `agentID`: 에이전트 고유 식별자
- `description`: 에이전트 설명
- `model`: AI 모델명
- `prompt`: 시스템 프롬프트

**반환**: 에러 (nil이면 성공)

**에러 케이스**:
- Repository가 설정되지 않음
- 데이터베이스 저장 실패
- 중복된 `agentID`

**참조**: `internal/controller/controller.go:62`

---

#### DeleteAgent
에이전트를 삭제합니다 (soft delete).

```go
func (c *Controller) DeleteAgent(
    ctx context.Context,
    agent string,
) error
```

**파라미터**:
- `agent`: 삭제할 에이전트 ID

**동작**:
- 에이전트 상태를 `deleted`로 변경
- 실제 레코드는 삭제하지 않음

**참조**: `internal/controller/controller.go:93`

---

#### UpdateAgent
에이전트 정보를 수정합니다.

```go
func (c *Controller) UpdateAgent(
    ctx context.Context,
    agentID string,
    description string,
    model string,
    prompt string,
) error
```

**파라미터**:
- `agentID`: 수정할 에이전트 ID
- `description`: 새 설명
- `model`: 새 모델명
- `prompt`: 새 프롬프트

**에러 케이스**:
- 에이전트가 존재하지 않음

**참조**: `internal/controller/controller.go:371`

---

#### ListAgents
모든 에이전트 ID 목록을 반환합니다.

```go
func (c *Controller) ListAgents(ctx context.Context) ([]string, error)
```

**반환**: 에이전트 ID 문자열 배열

**참조**: `internal/controller/controller.go:113`

---

#### ListAgentsWithInfo
상세 정보를 포함한 에이전트 목록을 반환합니다.

```go
func (c *Controller) ListAgentsWithInfo(ctx context.Context) ([]*AgentInfo, error)
```

**반환**: `AgentInfo` 구조체 배열

**AgentInfo 구조**:
```go
type AgentInfo struct {
    Name        string
    Description string
    Model       string
    Prompt      string
    Status      string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

**참조**: `internal/controller/controller.go:405`

---

#### GetAgentInfo
특정 에이전트의 상세 정보를 조회합니다.

```go
func (c *Controller) GetAgentInfo(
    ctx context.Context,
    agent string,
) (*AgentInfo, error)
```

**에러 케이스**:
- 에이전트가 존재하지 않음 → `"agent not found: {agentID}"`

**참조**: `internal/controller/controller.go:148`

---

#### ValidateAgent
에이전트 이름의 유효성을 검증합니다.

```go
func (c *Controller) ValidateAgent(agent string) error
```

**검증 규칙**:
- 빈 문자열 불가
- 최대 64자

**참조**: `internal/controller/controller.go:183`

---

### Task 관리 API

#### CreateTask
새로운 작업을 생성합니다.

```go
func (c *Controller) CreateTask(
    ctx context.Context,
    agentID string,
    taskID string,
) error
```

**파라미터**:
- `agentID`: 작업을 수행할 에이전트 ID
- `taskID`: 작업 고유 식별자

**전제 조건**:
- 해당 에이전트가 반드시 존재해야 함

**에러 케이스**:
- 에이전트가 존재하지 않음 → `"agent not found: {agentID}"`
- 중복된 `taskID`

**참조**: `internal/controller/controller.go:197`

---

#### GetTask
작업 정보를 조회합니다.

```go
func (c *Controller) GetTask(
    ctx context.Context,
    taskID string,
) (*storage.Task, error)
```

**반환**: `storage.Task` 구조체 포인터

**에러 케이스**:
- 작업이 존재하지 않음 → `"task not found: {taskID}"`

**참조**: `internal/controller/controller.go:235`

---

#### GetTaskInfo
작업의 상세 정보를 반환합니다.

```go
func (c *Controller) GetTaskInfo(
    ctx context.Context,
    taskID string,
) (*TaskInfo, error)
```

**TaskInfo 구조**:
```go
type TaskInfo struct {
    TaskID    string
    AgentID   string
    Status    string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

**참조**: `internal/controller/controller.go:325`

---

#### UpdateTaskStatus
작업 상태를 업데이트합니다.

```go
func (c *Controller) UpdateTaskStatus(
    ctx context.Context,
    taskID string,
    status string,
) error
```

**파라미터**:
- `taskID`: 작업 ID
- `status`: 새 상태 (pending, running, completed, failed, canceled)

**에러 케이스**:
- 작업이 존재하지 않음

**참조**: `internal/controller/controller.go:260`

---

#### ListTasksByAgent
에이전트별 작업 목록을 반환합니다.

```go
func (c *Controller) ListTasksByAgent(
    ctx context.Context,
    agentID string,
) ([]storage.Task, error)
```

**반환**: `storage.Task` 배열 (생성 시간 순 정렬)

**참조**: `internal/controller/controller.go:294`

---

#### ValidateTask
작업 ID의 유효성을 검증합니다.

```go
func (c *Controller) ValidateTask(taskID string) error
```

**검증 규칙**:
- 빈 문자열 불가
- 최대 64자

**참조**: `internal/controller/controller.go:358`

---

### Server 관리 API

#### Start
Controller 서버를 시작합니다.

```go
func (c *Controller) Start(ctx context.Context) error
```

**동작**:
- 5초마다 heartbeat 로그 출력 (현재는 더미 구현)
- Context 취소 시 graceful shutdown

**참조**: `internal/controller/controller.go:29`

---

#### Stop
Controller 서버를 정상 종료합니다.

```go
func (c *Controller) Stop(ctx context.Context) error
```

**동작**:
- 정리 작업 수행 후 종료
- 100ms 대기 후 종료 완료

**참조**: `internal/controller/controller.go:48`

---

## 데이터베이스 스키마

### ERD (Entity Relationship Diagram)

```
┌─────────────────────────────┐
│         agents              │
├─────────────────────────────┤
│ PK  id          BIGSERIAL   │
│ UK  agent_id    VARCHAR(64) │
│     description TEXT         │
│     model       VARCHAR(64) │
│     prompt      TEXT         │
│     status      VARCHAR(32) │
│     created_at  TIMESTAMP   │
│     updated_at  TIMESTAMP   │
└──────────┬──────────────────┘
           │
           │ 1:N
           │
┌──────────▼──────────────────┐
│         tasks               │
├─────────────────────────────┤
│ PK  id         BIGSERIAL    │
│ UK  task_id    VARCHAR(64)  │
│ FK  agent_id   VARCHAR(64)  │────┐
│     status     VARCHAR(32)  │    │
│     created_at TIMESTAMP    │    │
│     updated_at TIMESTAMP    │    │
└──────────┬──────────────────┘    │
           │                       │
      ┌────┴────┬─────────┐        │
      │         │         │        │
      │ 1:N     │ 1:N     │ 1:N    │
      ▼         ▼         ▼        │
┌───────────┐ ┌─────────┐ ┌───────▼─────┐
│msg_index  │ │run_steps│ │checkpoints  │
├───────────┤ ├─────────┤ ├─────────────┤
│PK id      │ │PK id    │ │PK id        │
│FK task_id │ │FK task_id│ │FK task_id  │
│   conv_idx│ │   step_no│ │   git_hash │
│   filepath│ │   type   │ │   created_at│
│updated_at │ │   status │ └─────────────┘
└───────────┘ │created_at│
              └──────────┘
```

---

### 테이블 상세

#### 1. agents

| 컬럼명      | 타입         | 제약 조건                          | 설명                  |
|-------------|--------------|-----------------------------------|----------------------|
| id          | BIGSERIAL    | PRIMARY KEY                        | 자동 증가 ID          |
| agent_id    | VARCHAR(64)  | NOT NULL, UNIQUE INDEX             | 에이전트 고유 식별자   |
| description | TEXT         |                                    | 에이전트 설명         |
| model       | VARCHAR(64)  |                                    | AI 모델명             |
| prompt      | TEXT         |                                    | 시스템 프롬프트       |
| status      | VARCHAR(32)  | NOT NULL, DEFAULT 'active'         | 상태 (active/idle/busy/deleted) |
| created_at  | TIMESTAMP    | NOT NULL, AUTO CREATE TIME         | 생성 시간             |
| updated_at  | TIMESTAMP    | NOT NULL, AUTO UPDATE TIME         | 수정 시간             |

**인덱스**:
- `idx_agents_agent_id`: UNIQUE INDEX on `agent_id`

---

#### 2. tasks

| 컬럼명      | 타입         | 제약 조건                          | 설명                  |
|-------------|--------------|-----------------------------------|----------------------|
| id          | BIGSERIAL    | PRIMARY KEY                        | 자동 증가 ID          |
| task_id     | VARCHAR(64)  | NOT NULL, UNIQUE INDEX             | 작업 고유 식별자      |
| agent_id    | VARCHAR(64)  | NOT NULL, INDEX                    | 에이전트 ID (FK)      |
| status      | VARCHAR(32)  | NOT NULL                           | 상태 (pending/running/completed/failed/canceled) |
| created_at  | TIMESTAMP    | NOT NULL, AUTO CREATE TIME         | 생성 시간             |
| updated_at  | TIMESTAMP    | NOT NULL, AUTO UPDATE TIME         | 수정 시간             |

**인덱스**:
- `idx_tasks_task_id`: UNIQUE INDEX on `task_id`
- `idx_tasks_agent_id`: INDEX on `agent_id`

**외래 키**:
- `agent_id` → `agents.agent_id` (논리적 FK, DB 제약은 미설정)

---

#### 3. msg_index

| 컬럼명              | 타입         | 제약 조건                          | 설명                  |
|---------------------|--------------|-----------------------------------|----------------------|
| id                  | BIGSERIAL    | PRIMARY KEY                        | 자동 증가 ID          |
| task_id             | VARCHAR(64)  | NOT NULL, INDEX, UNIQUE COMPOSITE  | 작업 ID (FK)          |
| conversation_index  | INT          | NOT NULL, UNIQUE COMPOSITE         | 대화 순서 번호        |
| file_path           | TEXT         | NOT NULL                           | JSON 파일 경로        |
| updated_at          | TIMESTAMP    | NOT NULL, AUTO UPDATE TIME         | 수정 시간             |

**인덱스**:
- `idx_msg_index_task`: INDEX on `task_id`
- `idx_msg_idx_task_conv`: UNIQUE INDEX on `(task_id, conversation_index)`

**외래 키**:
- `task_id` → `tasks.task_id` (논리적 FK, DB 제약은 미설정)

---

#### 4. run_steps

| 컬럼명      | 타입         | 제약 조건                          | 설명                  |
|-------------|--------------|-----------------------------------|----------------------|
| id          | BIGSERIAL    | PRIMARY KEY                        | 자동 증가 ID          |
| task_id     | VARCHAR(64)  | NOT NULL, INDEX, UNIQUE COMPOSITE  | 작업 ID (FK)          |
| step_no     | INT          | NOT NULL, UNIQUE COMPOSITE         | 단계 번호             |
| type        | VARCHAR(32)  | NOT NULL                           | 단계 유형 (system/tool/model/checkpoint) |
| status      | VARCHAR(32)  | NOT NULL                           | 상태 (pending/running/completed/failed) |
| created_at  | TIMESTAMP    | NOT NULL, AUTO CREATE TIME         | 생성 시간             |

**인덱스**:
- `idx_run_steps_task`: INDEX on `task_id`
- `idx_run_steps_task_step`: UNIQUE INDEX on `(task_id, step_no)`

**외래 키**:
- `task_id` → `tasks.task_id` (논리적 FK, DB 제약은 미설정)

---

#### 5. checkpoints

| 컬럼명      | 타입         | 제약 조건                          | 설명                  |
|-------------|--------------|-----------------------------------|----------------------|
| id          | BIGSERIAL    | PRIMARY KEY                        | 자동 증가 ID          |
| task_id     | VARCHAR(64)  | NOT NULL, INDEX, UNIQUE COMPOSITE  | 작업 ID (FK)          |
| git_hash    | VARCHAR(64)  | NOT NULL, UNIQUE COMPOSITE         | Git 커밋 해시         |
| created_at  | TIMESTAMP    | NOT NULL, AUTO CREATE TIME         | 생성 시간             |

**인덱스**:
- `idx_checkpoints_task`: INDEX on `task_id`
- `idx_checkpoints_task_git`: UNIQUE INDEX on `(task_id, git_hash)`

**외래 키**:
- `task_id` → `tasks.task_id` (논리적 FK, DB 제약은 미설정)

---

### Repository 패턴 메서드

Storage 계층은 Repository 패턴을 사용하여 데이터 액세스를 추상화합니다.

#### Agent Repository 메서드

| 메서드명              | 설명                                           | 참조                            |
|-----------------------|------------------------------------------------|--------------------------------|
| `CreateAgent`         | 새 에이전트 레코드 생성                         | `internal/storage/repository.go:31` |
| `UpsertAgentStatus`   | 에이전트 상태 upsert                           | `internal/storage/repository.go:39` |
| `GetAgent`            | 에이전트 ID로 조회                             | `internal/storage/repository.go:55` |
| `ListAgents`          | 에이전트 목록 조회 (상태 필터 가능)             | `internal/storage/repository.go:69` |
| `UpdateAgent`         | 에이전트 정보 업데이트                         | `internal/storage/repository.go:82` |

#### Task Repository 메서드

| 메서드명              | 설명                                           | 참조                            |
|-----------------------|------------------------------------------------|--------------------------------|
| `CreateTask`          | 새 작업 레코드 생성                            | `internal/storage/repository.go:101` |
| `UpsertTaskStatus`    | 작업 상태 upsert                               | `internal/storage/repository.go:109` |
| `GetTask`             | 작업 ID로 조회                                 | `internal/storage/repository.go:126` |
| `ListTasksByAgent`    | 에이전트별 작업 목록 조회                       | `internal/storage/repository.go:137` |

#### MessageIndex Repository 메서드

| 메서드명                | 설명                                           | 참조                            |
|-------------------------|------------------------------------------------|--------------------------------|
| `UpsertMessageIndex`    | 메시지 인덱스 upsert                           | `internal/storage/repository.go:149` |
| `ListMessageIndexByTask`| 작업별 메시지 인덱스 목록 조회                  | `internal/storage/repository.go:165` |

#### RunStep Repository 메서드

| 메서드명              | 설명                                           | 참조                            |
|-----------------------|------------------------------------------------|--------------------------------|
| `UpsertRunStep`       | 실행 단계 upsert                               | `internal/storage/repository.go:177` |
| `ListRunSteps`        | 작업별 실행 단계 목록 조회                      | `internal/storage/repository.go:193` |

#### Checkpoint Repository 메서드

| 메서드명              | 설명                                           | 참조                            |
|-----------------------|------------------------------------------------|--------------------------------|
| `CreateCheckpoint`    | 체크포인트 생성 (중복 시 무시)                  | `internal/storage/repository.go:205` |
| `ListCheckpoints`     | 작업별 체크포인트 목록 조회                     | `internal/storage/repository.go:221` |

---

## 데이터 흐름

### 1. 에이전트 생성 플로우

```
사용자 요청
    │
    ▼
CLI (cmd/cnap/agent.go)
    │
    │ createAgentCmd
    ▼
Controller.CreateAgent()
    │
    │ validation
    ▼
Repository.CreateAgent()
    │
    │ GORM INSERT
    ▼
PostgreSQL agents 테이블
```

**SQL**:
```sql
INSERT INTO agents (agent_id, description, model, prompt, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, 'active', NOW(), NOW());
```

---

### 2. 작업 생성 플로우

```
사용자 요청
    │
    ▼
CLI (cmd/cnap/agent.go)
    │
    │ runCmd
    ▼
Controller.CreateTask()
    │
    │ 1. Agent 존재 확인
    │    Repository.GetAgent()
    │
    │ 2. Task 생성
    ▼
Repository.CreateTask()
    │
    │ GORM INSERT
    ▼
PostgreSQL tasks 테이블
```

**SQL**:
```sql
-- 1. Agent 존재 확인
SELECT * FROM agents WHERE agent_id = $1 LIMIT 1;

-- 2. Task 생성
INSERT INTO tasks (task_id, agent_id, status, created_at, updated_at)
VALUES ($1, $2, 'pending', NOW(), NOW());
```

---

### 3. 작업 상태 업데이트 플로우

```
Controller.UpdateTaskStatus()
    │
    │ 1. Task 존재 확인
    │    Repository.GetTask()
    │
    │ 2. 상태 업데이트
    ▼
Repository.UpsertTaskStatus()
    │
    │ UPSERT (ON CONFLICT UPDATE)
    ▼
PostgreSQL tasks 테이블
```

**SQL**:
```sql
-- 1. Task 존재 확인
SELECT * FROM tasks WHERE task_id = $1 LIMIT 1;

-- 2. 상태 업데이트 (Upsert)
INSERT INTO tasks (task_id, agent_id, status, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (task_id)
DO UPDATE SET status = EXCLUDED.status, updated_at = EXCLUDED.updated_at;
```

---

### 4. 메시지 인덱스 저장 플로우 (향후 구현 예정)

```
Task 실행 중 메시지 발생
    │
    ▼
메시지를 JSON 파일로 저장
    │  (예: data/messages/task-123/conv-001.json)
    │
    ▼
Repository.UpsertMessageIndex()
    │
    │ INSERT ON CONFLICT UPDATE
    ▼
PostgreSQL msg_index 테이블
```

**SQL**:
```sql
INSERT INTO msg_index (task_id, conversation_index, file_path, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (task_id, conversation_index)
DO UPDATE SET file_path = EXCLUDED.file_path, updated_at = EXCLUDED.updated_at;
```

---

## 상태 상수

### Agent 상태

| 상수                 | 값        | 설명                          |
|----------------------|-----------|------------------------------|
| `AgentStatusActive`  | "active"  | 활성 상태                     |
| `AgentStatusIdle`    | "idle"    | 유휴 상태                     |
| `AgentStatusBusy`    | "busy"    | 작업 수행 중                  |
| `AgentStatusDeleted` | "deleted" | 삭제됨 (soft delete)          |

### Task 상태

| 상수                   | 값          | 설명                          |
|------------------------|-------------|------------------------------|
| `TaskStatusPending`    | "pending"   | 대기 중                       |
| `TaskStatusRunning`    | "running"   | 실행 중                       |
| `TaskStatusCompleted`  | "completed" | 완료됨                        |
| `TaskStatusFailed`     | "failed"    | 실패함                        |
| `TaskStatusCanceled`   | "canceled"  | 취소됨                        |

### RunStep 상태

| 상수                      | 값          | 설명                          |
|---------------------------|-------------|------------------------------|
| `RunStepStatusPending`    | "pending"   | 대기 중                       |
| `RunStepStatusRunning`    | "running"   | 실행 중                       |
| `RunStepStatusCompleted`  | "completed" | 완료됨                        |
| `RunStepStatusFailed`     | "failed"    | 실패함                        |

### RunStep 타입

| 상수                      | 값           | 설명                          |
|---------------------------|--------------|------------------------------|
| `RunStepTypeSystem`       | "system"     | 시스템 이벤트                 |
| `RunStepTypeTool`         | "tool"       | 도구 호출                     |
| `RunStepTypeModel`        | "model"      | AI 모델 호출                  |
| `RunStepTypeCheckpoint`   | "checkpoint" | Git 체크포인트 생성           |

**참조**: `internal/storage/constants.go`

---

## 구현 상태

### ✅ 완료된 기능
- Agent CRUD 작업
- Task CRUD 작업
- Repository 패턴 구현
- 데이터베이스 마이그레이션
- 상태 관리 상수 정의

### 🚧 향후 구현 예정
- Discord bot과의 통합 (connector 계층)
- 실제 Task 실행 로직
- MessageIndex를 이용한 메시지 저장
- RunStep 추적 기능
- Checkpoint 생성 및 Git 스냅샷 관리
- Connector ↔ Controller 통신 메커니즘

---

## 참고 문서

- [로컬 개발 가이드](./local-development.md)
- [Docker 가이드](./docker-guide.md)
- [CLI 테스팅 가이드](./cli-testing-guide.md)
- [Controller CLI 가이드](./controller-cli-guide.md)
