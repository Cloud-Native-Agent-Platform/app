# CLI 작동 테스트 가이드

CNAP Controller CLI의 작동을 확인하고 테스트하는 방법을 안내합니다.

## 목차
- [환경 설정](#환경-설정)
- [기본 테스트](#기본-테스트)
- [메시지 관리 테스트](#메시지-관리-테스트)
- [전체 워크플로우 테스트](#전체-워크플로우-테스트)
- [Docker 환경 테스트](#docker-환경-테스트)
- [데이터베이스 검증](#데이터베이스-검증)
- [에러 케이스 테스트](#에러-케이스-테스트)
- [자동화 테스트 스크립트](#자동화-테스트-스크립트)

---

## 환경 설정

### 방법 1: Docker Compose 사용 (권장)

Docker Compose를 사용하면 PostgreSQL과 애플리케이션이 함께 실행됩니다.

```bash
# 프로젝트 루트에서
cd /Users/leo/Documents/development/app

# Docker Compose 실행
docker compose -f docker/docker-compose.yml up -d

# 컨테이너 상태 확인
docker compose -f docker/docker-compose.yml ps

# 로그 확인
docker logs cnap-app -f
```

**환경 변수 확인:**
```bash
docker exec cnap-app env | grep DATABASE_URL
```

### 방법 2: 로컬 PostgreSQL 사용

로컬에 PostgreSQL이 설치되어 있는 경우:

```bash
# 데이터베이스 생성 (처음 한 번만)
createdb -U postgres cnap

# 사용자 생성 및 권한 부여
psql -U postgres -d cnap << EOF
CREATE USER cnap WITH PASSWORD 'cnap';
GRANT ALL PRIVILEGES ON DATABASE cnap TO cnap;
ALTER DATABASE cnap OWNER TO cnap;
EOF

# 환경 변수 설정
export DATABASE_URL="postgres://cnap:cnap@localhost:5432/cnap?sslmode=disable"

# 확인
echo $DATABASE_URL
```

### CLI 빌드

```bash
# Makefile 사용
make build

# 또는 직접 빌드
go build -o bin/cnap ./cmd/cnap

# 빌드 확인
./bin/cnap --version
./bin/cnap --help
```

---

## 기본 테스트

### 1. Agent 생성 테스트

**대화형 모드로 생성:**

```bash
./bin/cnap agent create
```

**입력 예시:**
```
Agent 이름: test-bot
설명: 테스트용 챗봇
모델 (예: gpt-4): gpt-4
프롬프트 (역할 정의): 당신은 친절한 테스트 봇입니다.
```

**예상 출력:**
```
✓ Agent 'test-bot' 생성 완료
```

**스크립트로 자동 입력:**
```bash
echo -e "test-bot\n테스트용 챗봇\ngpt-4\n당신은 친절한 테스트 봇입니다." | ./bin/cnap agent create
```

### 2. Agent 목록 조회

```bash
./bin/cnap agent list
```

**예상 출력:**
```
NAME       STATUS  MODEL  DESCRIPTION      CREATED
----       ------  -----  -----------      -------
test-bot   active  gpt-4  테스트용 챗봇     2025-11-18 13:30
```

### 3. Agent 상세 정보

```bash
./bin/cnap agent view test-bot
```

**예상 출력:**
```
=== Agent 정보: test-bot ===

이름:        test-bot
상태:        active
모델:        gpt-4
설명:        테스트용 챗봇
프롬프트:
당신은 친절한 테스트 봇입니다.

생성일:      2025-11-18 13:30:00
수정일:      2025-11-18 13:30:00
```

### 4. Agent 수정

```bash
./bin/cnap agent edit test-bot
```

**입력 예시 (빈 값은 기존 값 유지):**
```
설명 (현재: 테스트용 챗봇): 업데이트된 테스트 봇
모델 (현재: gpt-4):
프롬프트 (현재: 당신은 친절한 테스트 봇입니다.):
```

**스크립트로 자동 입력:**
```bash
echo -e "업데이트된 테스트 봇\n\n" | ./bin/cnap agent edit test-bot
```

### 5. Task 생성

```bash
./bin/cnap task create test-bot task-001
```

**예상 출력:**
```
✓ Task 'task-001' 생성 완료 (Agent: test-bot)
```

### 6. Task 목록 조회

```bash
./bin/cnap task list test-bot
```

**예상 출력:**
```
TASK ID    STATUS   CREATED           UPDATED
-------    ------   -------           -------
task-001   pending  2025-11-18 13:35  2025-11-18 13:35
```

### 7. Task 상태 변경

```bash
# running으로 변경
./bin/cnap task update-status task-001 running

# 확인
./bin/cnap task view task-001
```

**예상 출력:**
```
=== Task 정보: task-001 ===

Task ID:     task-001
Agent ID:    test-bot
상태:        running
생성일:      2025-11-18 13:35:00
수정일:      2025-11-18 13:36:00
```

### 8. Task 취소

```bash
./bin/cnap task cancel task-001
```

**예상 출력:**
```
✓ Task 'task-001' 상태 변경: canceled
```

### 9. Agent 삭제

```bash
./bin/cnap agent delete test-bot
```

**입력:**
```
Agent 'test-bot'을(를) 삭제하시겠습니까? (y/N): y
```

**예상 출력:**
```
✓ Agent 'test-bot' 삭제 완료
```

**스크립트로 자동 확인:**
```bash
echo "y" | ./bin/cnap agent delete test-bot
```

---

## 메시지 관리 테스트

MessageIndex는 Task의 대화 메시지를 추적하는 기능입니다. 메시지의 역할(user/assistant/system), 순서, 파일 경로를 DB에 저장하여 빠른 검색과 대화 흐름 파악을 가능하게 합니다.

### 주요 개념

- **ConversationIndex**: Task별 메시지 순서 번호 (0부터 자동 증가)
- **Role**: 메시지 발신자 구분 (`user`, `assistant`, `system`)
- **FilePath**: 실제 메시지 내용이 저장된 JSON 파일 경로

### 1. 메시지 추가 (프로그래밍 방식)

현재 CLI에서는 MessageIndex 직접 관리 기능이 없으므로, Go 코드나 데이터베이스를 통해 테스트합니다.

**Go 코드 예시:**
```go
// Agent와 Task가 이미 생성되어 있다고 가정
msg, err := repo.AppendMessageIndex(ctx, "task-001", storage.MessageRoleUser, "/data/msg-000.json")
// msg.ConversationIndex는 자동으로 0, 1, 2... 증가
```

### 2. 데이터베이스에서 직접 메시지 추가

**PostgreSQL 접속:**
```bash
# Docker 환경
docker exec -it cnap-app psql -U cnap -d cnap

# 로컬 환경
psql -U cnap -d cnap
```

**메시지 추가 예시:**
```sql
-- Task 'task-001'에 user 메시지 추가
INSERT INTO msg_index (task_id, conversation_index, role, file_path, created_at, updated_at)
VALUES ('task-001', 0, 'user', '/data/task-001/msg-000.json', NOW(), NOW());

-- assistant 응답 추가
INSERT INTO msg_index (task_id, conversation_index, role, file_path, created_at, updated_at)
VALUES ('task-001', 1, 'assistant', '/data/task-001/msg-001.json', NOW(), NOW());

-- 추가 user 메시지
INSERT INTO msg_index (task_id, conversation_index, role, file_path, created_at, updated_at)
VALUES ('task-001', 2, 'user', '/data/task-001/msg-002.json', NOW(), NOW());
```

### 3. 메시지 조회 및 검증

**특정 Task의 모든 메시지 조회 (순서대로):**
```sql
SELECT conversation_index, role, file_path, created_at
FROM msg_index
WHERE task_id = 'task-001'
ORDER BY conversation_index ASC;
```

**예상 출력:**
```
 conversation_index |   role    |          file_path          |      created_at
--------------------+-----------+-----------------------------+---------------------
                  0 | user      | /data/task-001/msg-000.json | 2025-11-19 10:00:00
                  1 | assistant | /data/task-001/msg-001.json | 2025-11-19 10:00:05
                  2 | user      | /data/task-001/msg-002.json | 2025-11-19 10:00:10
```

**Role별 메시지 개수 확인:**
```sql
SELECT role, COUNT(*) as message_count
FROM msg_index
WHERE task_id = 'task-001'
GROUP BY role;
```

**예상 출력:**
```
   role    | message_count
-----------+--------------
 user      |             2
 assistant |             1
```

### 4. ConversationIndex 자동 증가 검증

**다음 인덱스 확인 쿼리:**
```sql
SELECT COALESCE(MAX(conversation_index), -1) + 1 as next_index
FROM msg_index
WHERE task_id = 'task-001';
```

이 쿼리는 `GetNextConversationIndex()` 메서드가 내부적으로 사용하는 로직과 동일합니다.

### 5. 메시지 순서 연속성 검증

**누락된 인덱스가 있는지 확인:**
```sql
WITH RECURSIVE expected_indices AS (
  SELECT 0 AS idx
  UNION ALL
  SELECT idx + 1
  FROM expected_indices
  WHERE idx < (SELECT MAX(conversation_index) FROM msg_index WHERE task_id = 'task-001')
)
SELECT ei.idx as missing_index
FROM expected_indices ei
LEFT JOIN msg_index mi ON ei.idx = mi.conversation_index AND mi.task_id = 'task-001'
WHERE mi.id IS NULL;
```

결과가 비어 있으면 순서가 올바릅니다.

### 6. 테스트 시나리오

**완전한 대화 흐름 테스트:**
```sql
-- 1. Agent와 Task 생성 (CLI 사용)
-- ./bin/cnap agent create
-- ./bin/cnap task create test-bot task-chat-001

-- 2. 대화 시뮬레이션
BEGIN;

INSERT INTO msg_index (task_id, conversation_index, role, file_path, created_at, updated_at)
VALUES
  ('task-chat-001', 0, 'user', '/data/task-chat-001/msg-000.json', NOW(), NOW()),
  ('task-chat-001', 1, 'assistant', '/data/task-chat-001/msg-001.json', NOW(), NOW()),
  ('task-chat-001', 2, 'user', '/data/task-chat-001/msg-002.json', NOW(), NOW()),
  ('task-chat-001', 3, 'assistant', '/data/task-chat-001/msg-003.json', NOW(), NOW());

-- 3. 검증
SELECT
  conversation_index,
  role,
  file_path,
  created_at
FROM msg_index
WHERE task_id = 'task-chat-001'
ORDER BY conversation_index;

COMMIT;
```

### 7. Repository 메서드 테스트 (Go)

**단위 테스트 실행:**
```bash
# MessageIndex 관련 테스트만 실행
go test ./internal/storage/... -v -run TestRepositoryMessageIndexAutoIncrement

# 전체 storage 테스트
go test ./internal/storage/... -v
```

**예상 출력:**
```
=== RUN   TestRepositoryMessageIndexAutoIncrement
--- PASS: TestRepositoryMessageIndexAutoIncrement (0.00s)
PASS
```

---

## 전체 워크플로우 테스트

### 시나리오: 챗봇 생성부터 Task 관리까지

```bash
# 1. Agent 생성
echo -e "chatbot-01\n고객 지원 챗봇\ngpt-4\n친절한 고객 지원 담당자입니다." | ./bin/cnap agent create

# 2. 생성 확인
./bin/cnap agent view chatbot-01

# 3. 여러 Task 생성
./bin/cnap task create chatbot-01 task-001
./bin/cnap task create chatbot-01 task-002
./bin/cnap task create chatbot-01 task-003

# 4. Task 목록 확인
./bin/cnap task list chatbot-01

# 5. 각 Task 상태 변경
./bin/cnap task update-status task-001 running
./bin/cnap task update-status task-002 completed
./bin/cnap task cancel task-003

# 6. 최종 상태 확인
./bin/cnap task list chatbot-01

# 7. Agent 정보 수정
echo -e "고급 AI 고객 지원 챗봇\ngpt-4-turbo\n" | ./bin/cnap agent edit chatbot-01

# 8. 수정 확인
./bin/cnap agent view chatbot-01

# 9. 전체 Agent 목록
./bin/cnap agent list
```

### 시나리오: 다중 Agent 관리

```bash
# 여러 Agent 생성
echo -e "support-bot\n고객 지원\ngpt-4\n고객 지원 담당" | ./bin/cnap agent create
echo -e "sales-bot\n영업 지원\ngpt-3.5-turbo\n영업 담당" | ./bin/cnap agent create
echo -e "dev-bot\n개발 지원\ngpt-4\n개발 담당" | ./bin/cnap agent create

# Agent 목록 확인
./bin/cnap agent list

# 각 Agent에 Task 생성
./bin/cnap task create support-bot support-task-001
./bin/cnap task create sales-bot sales-task-001
./bin/cnap task create dev-bot dev-task-001

# 각 Agent의 Task 확인
for agent in support-bot sales-bot dev-bot; do
    echo "=== $agent Tasks ==="
    ./bin/cnap task list $agent
    echo ""
done
```

---

## Docker 환경 테스트

### 컨테이너 내부에서 CLI 실행

```bash
# 컨테이너 내부로 진입
docker exec -it cnap-app /bin/bash

# 내부에서 CLI 사용
cd /app
./bin/cnap agent list
./bin/cnap task list <agent-name>

# 종료
exit
```

### 컨테이너 외부에서 직접 실행

```bash
# Agent 목록 조회
docker exec cnap-app /app/bin/cnap agent list

# Agent 생성 (대화형은 불가능)
# 스크립트로 입력 전달
docker exec -i cnap-app /app/bin/cnap agent create << EOF
docker-bot
Docker 테스트 봇
gpt-4
Docker 환경에서 실행되는 봇
EOF

# Task 생성
docker exec cnap-app /app/bin/cnap task create docker-bot docker-task-001

# Task 목록
docker exec cnap-app /app/bin/cnap task list docker-bot
```

### Docker 로그 모니터링

```bash
# 실시간 로그 확인
docker logs cnap-app -f

# 최근 100줄만 확인
docker logs cnap-app --tail 100

# 타임스탬프 포함
docker logs cnap-app -f -t
```

---

## 데이터베이스 검증

CLI가 올바르게 데이터를 저장하는지 데이터베이스에서 직접 확인합니다.

### PostgreSQL 접속

**Docker 환경:**
```bash
docker exec -it cnap-app psql -U cnap -d cnap
```

**로컬 환경:**
```bash
psql -U cnap -d cnap
```

### 데이터 확인 쿼리

```sql
-- 모든 Agent 조회
SELECT agent_id, description, model, status, created_at, updated_at
FROM agents
ORDER BY created_at DESC;

-- 모든 Task 조회
SELECT task_id, agent_id, status, created_at, updated_at
FROM tasks
ORDER BY created_at DESC;

-- Agent와 Task 조인 조회
SELECT
    a.agent_id,
    a.model,
    a.status as agent_status,
    COUNT(t.id) as task_count,
    COUNT(CASE WHEN t.status = 'pending' THEN 1 END) as pending_tasks,
    COUNT(CASE WHEN t.status = 'running' THEN 1 END) as running_tasks,
    COUNT(CASE WHEN t.status = 'completed' THEN 1 END) as completed_tasks
FROM agents a
LEFT JOIN tasks t ON a.agent_id = t.agent_id
GROUP BY a.agent_id, a.model, a.status;

-- 특정 Agent의 Task 상세
SELECT task_id, status, created_at, updated_at
FROM tasks
WHERE agent_id = 'test-bot'
ORDER BY created_at DESC;

-- 모든 MessageIndex 조회
SELECT task_id, conversation_index, role, file_path, created_at
FROM msg_index
ORDER BY task_id, conversation_index;

-- 특정 Task의 메시지 조회 (대화 순서대로)
SELECT conversation_index, role, file_path, created_at
FROM msg_index
WHERE task_id = 'task-001'
ORDER BY conversation_index ASC;

-- Task별 메시지 통계
SELECT
    task_id,
    COUNT(*) as total_messages,
    COUNT(CASE WHEN role = 'user' THEN 1 END) as user_messages,
    COUNT(CASE WHEN role = 'assistant' THEN 1 END) as assistant_messages,
    COUNT(CASE WHEN role = 'system' THEN 1 END) as system_messages,
    MAX(conversation_index) as last_index
FROM msg_index
GROUP BY task_id;

-- Task와 MessageIndex 조인 조회
SELECT
    t.task_id,
    t.agent_id,
    t.status as task_status,
    COUNT(m.id) as message_count,
    MAX(m.conversation_index) as last_message_index,
    MAX(m.created_at) as last_message_time
FROM tasks t
LEFT JOIN msg_index m ON t.task_id = m.task_id
GROUP BY t.task_id, t.agent_id, t.status;

-- 테이블 구조 확인
\d agents
\d tasks
\d msg_index

-- 데이터베이스 종료
\q
```

### 데이터 정합성 검증

```sql
-- Agent 없이 존재하는 Task 확인 (있으면 안 됨)
SELECT t.task_id, t.agent_id
FROM tasks t
LEFT JOIN agents a ON t.agent_id = a.agent_id
WHERE a.id IS NULL;

-- 중복된 Agent ID 확인 (있으면 안 됨)
SELECT agent_id, COUNT(*)
FROM agents
GROUP BY agent_id
HAVING COUNT(*) > 1;

-- 중복된 Task ID 확인 (있으면 안 됨)
SELECT task_id, COUNT(*)
FROM tasks
GROUP BY task_id
HAVING COUNT(*) > 1;

-- Task 없이 존재하는 MessageIndex 확인 (있으면 안 됨)
SELECT m.task_id, COUNT(*) as orphaned_messages
FROM msg_index m
LEFT JOIN tasks t ON m.task_id = t.task_id
WHERE t.id IS NULL
GROUP BY m.task_id;

-- 중복된 MessageIndex 확인 (task_id + conversation_index는 unique해야 함)
SELECT task_id, conversation_index, COUNT(*)
FROM msg_index
GROUP BY task_id, conversation_index
HAVING COUNT(*) > 1;

-- 잘못된 Role 값 확인 (user, assistant, system만 허용)
SELECT task_id, conversation_index, role
FROM msg_index
WHERE role NOT IN ('user', 'assistant', 'system');

-- ConversationIndex 연속성 검증 (누락된 인덱스 확인)
WITH task_max_indices AS (
  SELECT task_id, MAX(conversation_index) as max_idx
  FROM msg_index
  GROUP BY task_id
),
expected_counts AS (
  SELECT task_id, max_idx + 1 as expected_count
  FROM task_max_indices
),
actual_counts AS (
  SELECT task_id, COUNT(*) as actual_count
  FROM msg_index
  GROUP BY task_id
)
SELECT
  e.task_id,
  e.expected_count,
  a.actual_count,
  e.expected_count - a.actual_count as missing_indices
FROM expected_counts e
JOIN actual_counts a ON e.task_id = a.task_id
WHERE e.expected_count != a.actual_count;
```

---

## 에러 케이스 테스트

의도적으로 에러를 발생시켜 에러 처리가 올바른지 확인합니다.

### 1. 존재하지 않는 리소스 접근

```bash
# 존재하지 않는 Agent 조회
./bin/cnap agent view nonexistent-bot
```
**예상 출력:** `Agent 조회 실패: agent not found: nonexistent-bot`

```bash
# 존재하지 않는 Task 조회
./bin/cnap task view nonexistent-task
```
**예상 출력:** `Task 조회 실패: task not found: nonexistent-task`

```bash
# 존재하지 않는 Agent에 Task 생성
./bin/cnap task create nonexistent-bot task-test
```
**예상 출력:** `Task 생성 실패: agent not found: nonexistent-bot`

### 2. 잘못된 입력값

```bash
# 잘못된 Task 상태
./bin/cnap task update-status task-001 invalid-status
```
**예상 출력:** `유효하지 않은 상태: invalid-status (사용 가능: [pending running completed failed canceled])`

```bash
# 너무 긴 Agent 이름 (65자 이상)
echo -e "$(python3 -c 'print("a"*65)')\n테스트\ngpt-4\n프롬프트" | ./bin/cnap agent create
```
**예상 출력:** `유효하지 않은 Agent 이름: agent name too long (max 64 characters)`

```bash
# 빈 Agent 이름
echo -e "\n테스트\ngpt-4\n프롬프트" | ./bin/cnap agent create
```
**예상 출력:** `유효하지 않은 Agent 이름: agent name cannot be empty`

### 3. 데이터베이스 연결 오류

```bash
# DATABASE_URL 없이 실행
unset DATABASE_URL
./bin/cnap agent list
```
**예상 출력:** `컨트롤러 초기화 실패: ...`

```bash
# 잘못된 DATABASE_URL
export DATABASE_URL="postgres://invalid:invalid@localhost:9999/invalid"
./bin/cnap agent list
```
**예상 출력:** 연결 에러 메시지

### 4. 중복 생성

```bash
# 같은 이름의 Agent 생성 시도
echo -e "test-bot\n테스트\ngpt-4\n프롬프트" | ./bin/cnap agent create
echo -e "test-bot\n테스트2\ngpt-4\n프롬프트2" | ./bin/cnap agent create
```
**예상 출력:** 두 번째 생성 시 중복 에러

### 5. MessageIndex 에러 케이스

**잘못된 Role 값:**
```sql
-- 데이터베이스에 직접 접속
docker exec -it cnap-app psql -U cnap -d cnap

-- 잘못된 role 값으로 삽입 시도
INSERT INTO msg_index (task_id, conversation_index, role, file_path, created_at, updated_at)
VALUES ('task-001', 0, 'invalid-role', '/data/msg.json', NOW(), NOW());
```
**예상 결과:** DB 제약조건은 없으나, 애플리케이션 레벨에서 검증 필요

**중복된 ConversationIndex:**
```sql
-- 같은 task_id와 conversation_index로 두 번 삽입 시도
INSERT INTO msg_index (task_id, conversation_index, role, file_path, created_at, updated_at)
VALUES ('task-001', 0, 'user', '/data/msg-000.json', NOW(), NOW());

INSERT INTO msg_index (task_id, conversation_index, role, file_path, created_at, updated_at)
VALUES ('task-001', 0, 'assistant', '/data/msg-001.json', NOW(), NOW());
```
**예상 결과:** `ERROR: duplicate key value violates unique constraint "idx_msg_idx_task_conv"`

**존재하지 않는 Task에 메시지 추가:**
```sql
INSERT INTO msg_index (task_id, conversation_index, role, file_path, created_at, updated_at)
VALUES ('nonexistent-task', 0, 'user', '/data/msg.json', NOW(), NOW());
```
**예상 결과:** 성공 (외래 키 제약조건 없음, 애플리케이션 레벨에서 검증 필요)

**빈 Role 값:**
```sql
INSERT INTO msg_index (task_id, conversation_index, role, file_path, created_at, updated_at)
VALUES ('task-001', 0, '', '/data/msg.json', NOW(), NOW());
```
**예상 결과:** `ERROR: null value in column "role" violates not-null constraint`

**Repository 메서드 에러 테스트 (Go):**
```go
// AppendMessageIndex에 빈 값 전달
_, err := repo.AppendMessageIndex(ctx, "", storage.MessageRoleUser, "/data/msg.json")
// 예상: "storage: empty taskID" 에러

_, err = repo.AppendMessageIndex(ctx, "task-001", "", "/data/msg.json")
// 예상: "storage: empty role" 에러

_, err = repo.AppendMessageIndex(ctx, "task-001", storage.MessageRoleUser, "")
// 예상: "storage: empty filePath" 에러
```

---

## 자동화 테스트 스크립트

### 기본 테스트 스크립트

`test-cli-basic.sh` 파일 생성:

```bash
#!/bin/bash

set -e  # 에러 발생 시 종료

echo "========================================="
echo "CNAP CLI 기본 테스트 시작"
echo "========================================="

# 환경 변수 확인
if [ -z "$DATABASE_URL" ]; then
    echo "DATABASE_URL 환경 변수가 설정되지 않았습니다."
    echo "Docker Compose 환경을 사용합니다."
    export DATABASE_URL="postgres://cnap:cnap@localhost:5432/cnap?sslmode=disable"
fi

echo ""
echo "1. CLI 버전 확인"
./bin/cnap --version

echo ""
echo "2. Agent 생성"
echo -e "test-bot-$$\n테스트 봇\ngpt-4\n테스트 프롬프트" | ./bin/cnap agent create

AGENT_NAME="test-bot-$$"

echo ""
echo "3. Agent 목록 확인"
./bin/cnap agent list

echo ""
echo "4. Agent 상세 정보"
./bin/cnap agent view $AGENT_NAME

echo ""
echo "5. Task 생성"
./bin/cnap task create $AGENT_NAME task-001-$$
./bin/cnap task create $AGENT_NAME task-002-$$

echo ""
echo "6. Task 목록 확인"
./bin/cnap task list $AGENT_NAME

echo ""
echo "7. Task 상태 변경"
./bin/cnap task update-status task-001-$$ running
./bin/cnap task update-status task-002-$$ completed

echo ""
echo "8. Task 최종 상태"
./bin/cnap task list $AGENT_NAME

echo ""
echo "9. 정리"
echo "y" | ./bin/cnap agent delete $AGENT_NAME

echo ""
echo "========================================="
echo "테스트 완료!"
echo "========================================="
```

실행:
```bash
chmod +x test-cli-basic.sh
./test-cli-basic.sh
```

### 종합 테스트 스크립트

`test-cli-comprehensive.sh` 파일 생성:

```bash
#!/bin/bash

set -e

PASSED=0
FAILED=0

function test_case() {
    echo ""
    echo "TEST: $1"
    if eval "$2" > /dev/null 2>&1; then
        echo "✓ PASS"
        ((PASSED++))
    else
        echo "✗ FAIL"
        ((FAILED++))
    fi
}

echo "========================================="
echo "CNAP CLI 종합 테스트"
echo "========================================="

# Agent 생성
AGENT="test-agent-$$"
echo -e "$AGENT\n테스트\ngpt-4\n프롬프트" | ./bin/cnap agent create > /dev/null 2>&1

test_case "Agent 목록 조회" "./bin/cnap agent list | grep -q $AGENT"
test_case "Agent 상세 조회" "./bin/cnap agent view $AGENT | grep -q 'gpt-4'"
test_case "존재하지 않는 Agent 조회 실패" "! ./bin/cnap agent view nonexistent"

# Task 테스트
TASK="task-$$"
./bin/cnap task create $AGENT $TASK > /dev/null 2>&1

test_case "Task 생성 확인" "./bin/cnap task list $AGENT | grep -q $TASK"
test_case "Task 상태 변경" "./bin/cnap task update-status $TASK running"
test_case "Task 취소" "./bin/cnap task cancel $TASK"
test_case "잘못된 상태로 변경 실패" "! ./bin/cnap task update-status $TASK invalid"

# 정리
echo "y" | ./bin/cnap agent delete $AGENT > /dev/null 2>&1

echo ""
echo "========================================="
echo "테스트 결과"
echo "========================================="
echo "통과: $PASSED"
echo "실패: $FAILED"
echo "========================================="

if [ $FAILED -eq 0 ]; then
    echo "모든 테스트 통과!"
    exit 0
else
    echo "일부 테스트 실패"
    exit 1
fi
```

실행:
```bash
chmod +x test-cli-comprehensive.sh
./test-cli-comprehensive.sh
```

---

## 도움말 확인

```bash
# 전체 명령어 확인
./bin/cnap --help

# Agent 명령어 도움말
./bin/cnap agent --help

# Task 명령어 도움말
./bin/cnap task --help

# 특정 하위 명령어 도움말
./bin/cnap agent create --help
./bin/cnap task update-status --help
```

---

## 로그 및 디버깅

### 상세 로그 활성화

```bash
# DEBUG 레벨 로그
export LOG_LEVEL=debug

# CLI 실행
./bin/cnap agent list

# 로그 확인
```

### Docker 로그 확인

```bash
# 실시간 로그
docker logs cnap-app -f

# 최근 로그만
docker logs cnap-app --tail 50

# 특정 시간 이후 로그
docker logs cnap-app --since 10m
```

---

## 문제 해결

### DATABASE_URL 에러

**증상:**
```
컨트롤러 초기화 실패: DATABASE_URL environment variable not set
```

**해결:**
```bash
export DATABASE_URL="postgres://cnap:cnap@localhost:5432/cnap?sslmode=disable"
```

### 연결 거부 에러

**증상:**
```
dial tcp [::1]:5432: connect: connection refused
```

**해결:**
```bash
# PostgreSQL 상태 확인
docker compose -f docker/docker-compose.yml ps

# 재시작
docker compose -f docker/docker-compose.yml restart
```

### 빌드 에러

**증상:**
```
package not found
```

**해결:**
```bash
# 의존성 설치
go mod download
go mod tidy

# 재빌드
make build
```

---

## 참고 자료

- [Controller CLI 사용 가이드](./controller-cli-guide.md)
- [Docker 가이드](./docker-guide.md)
- [로컬 개발 가이드](./local-development.md)
