# Controller CLI 사용 가이드

CNAP Controller를 CLI를 통해 직접 관리하는 방법을 안내합니다.

## 목차
- [개요](#개요)
- [Agent 관리](#agent-관리)
- [Task 관리](#task-관리)
- [환경 설정](#환경-설정)
- [문제 해결](#문제-해결)

---

## 개요

Controller CLI는 Discord Connector와 독립적으로 Agent 및 Task를 관리할 수 있는 명령줄 인터페이스를 제공합니다. 향후 Connector 구현이 변경되어도 CLI를 통한 관리 기능은 유지됩니다.

**아키텍처:**
```
CLI 명령어
    ↓
Controller 메서드
    ↓
Repository (GORM)
    ↓
PostgreSQL
```

---

## Agent 관리

### Agent 생성

대화형 입력을 통해 새로운 Agent를 생성합니다.

```bash
$ cnap agent create
Agent 이름: support-bot
설명: 고객 지원 챗봇
모델 (예: gpt-4): gpt-4
프롬프트 (역할 정의): 당신은 친절한 고객 지원 담당자입니다.
✓ Agent 'support-bot' 생성 완료
```

**필수 입력 항목:**
- **Agent 이름**: 고유한 식별자 (최대 64자)
- **설명**: Agent에 대한 간단한 설명
- **모델**: 사용할 AI 모델 (예: gpt-4, gpt-3.5-turbo)
- **프롬프트**: Agent의 역할 및 행동 정의

### Agent 목록 조회

등록된 모든 Agent의 목록을 테이블 형식으로 조회합니다.

```bash
$ cnap agent list
NAME          STATUS  MODEL   DESCRIPTION          CREATED
----          ------  -----   -----------          -------
support-bot   active  gpt-4   고객 지원 챗봇        2025-01-18 10:30
sales-bot     active  gpt-3   영업 담당 봇          2025-01-17 14:20
```

**출력 컬럼:**
- **NAME**: Agent 이름
- **STATUS**: 현재 상태 (active, idle, busy, deleted)
- **MODEL**: 사용 중인 AI 모델
- **DESCRIPTION**: 간단한 설명 (40자 초과 시 생략)
- **CREATED**: 생성 날짜 및 시간

### Agent 상세 정보

특정 Agent의 전체 정보를 조회합니다.

```bash
$ cnap agent view support-bot
=== Agent 정보: support-bot ===

이름:        support-bot
상태:        active
모델:        gpt-4
설명:        고객 지원 챗봇
프롬프트:
당신은 친절한 고객 지원 담당자입니다.

생성일:      2025-01-18 10:30:00
수정일:      2025-01-18 10:30:00
```

**표시 정보:**
- Agent의 모든 메타데이터
- 전체 프롬프트 내용
- 생성 및 수정 타임스탬프

### Agent 수정

Agent의 정보를 대화형 입력으로 수정합니다. 빈 값 입력 시 기존 값이 유지됩니다.

```bash
$ cnap agent edit support-bot
설명 (현재: 고객 지원 챗봇): 고급 고객 지원 AI 챗봇
모델 (현재: gpt-4):
프롬프트 (현재: 당신은 친절한 고객 지원 담당자입니다.):
✓ Agent 'support-bot' 수정 완료
```

**수정 가능 항목:**
- 설명
- 모델
- 프롬프트

**참고:** Agent 이름은 변경할 수 없습니다.

### Agent 삭제

Agent를 삭제합니다. 실제로는 상태를 `deleted`로 변경합니다.

```bash
$ cnap agent delete support-bot
Agent 'support-bot'을(를) 삭제하시겠습니까? (y/N): y
✓ Agent 'support-bot' 삭제 완료
```

**주의사항:**
- 삭제는 되돌릴 수 없습니다
- 확인 메시지가 표시됩니다
- 삭제된 Agent는 목록에 표시되지 않습니다

---

## Task 관리

### Task 생성

특정 Agent에 새로운 Task를 생성합니다.

```bash
$ cnap task create support-bot task-20250118-001
✓ Task 'task-20250118-001' 생성 완료 (Agent: support-bot)
```

**인자:**
1. `<agent-name>`: Task를 할당할 Agent 이름
2. `<task-id>`: 고유한 Task 식별자 (최대 64자)

**초기 상태:** `pending`

### Task 목록 조회

특정 Agent의 모든 Task를 조회합니다.

```bash
$ cnap task list support-bot
TASK ID             STATUS     CREATED           UPDATED
-------             ------     -------           -------
task-20250118-001   pending    2025-01-18 10:35  2025-01-18 10:35
task-20250118-002   running    2025-01-18 11:00  2025-01-18 11:05
task-20250117-003   completed  2025-01-17 15:20  2025-01-17 15:45
```

**출력 컬럼:**
- **TASK ID**: Task 식별자
- **STATUS**: 현재 상태
- **CREATED**: 생성 시각
- **UPDATED**: 마지막 업데이트 시각

### Task 상세 정보

특정 Task의 상세 정보를 조회합니다.

```bash
$ cnap task view task-20250118-001
=== Task 정보: task-20250118-001 ===

Task ID:     task-20250118-001
Agent ID:    support-bot
상태:        pending
생성일:      2025-01-18 10:35:00
수정일:      2025-01-18 10:35:00
```

### Task 상태 변경

Task의 상태를 직접 변경합니다.

```bash
$ cnap task update-status task-20250118-001 running
✓ Task 'task-20250118-001' 상태 변경: running
```

**사용 가능한 상태:**
- `pending` - 대기 중
- `running` - 실행 중
- `completed` - 완료
- `failed` - 실패
- `canceled` - 취소

**인자:**
1. `<task-id>`: Task 식별자
2. `<status>`: 변경할 상태값

### Task 취소 (편의 명령어)

Task를 취소 상태로 변경하는 단축 명령어입니다.

```bash
$ cnap task cancel task-20250118-001
✓ Task 'task-20250118-001' 상태 변경: canceled
```

이 명령어는 `cnap task update-status <task-id> canceled`와 동일합니다.

---

## 환경 설정

### 필수 환경 변수

Controller CLI는 PostgreSQL 데이터베이스 연결이 필요합니다.

```bash
export DATABASE_URL="postgres://user:password@localhost:5432/cnap?sslmode=disable"
```

**형식:** `postgres://[username]:[password]@[host]:[port]/[database]?[options]`

### 선택적 환경 변수

```bash
# 로그 레벨 설정 (debug, info, warn, error)
export LOG_LEVEL=info

# 환경 모드 (development, production)
export ENV=development

# 데이터베이스 연결 풀 설정
export DB_MAX_IDLE=5
export DB_MAX_OPEN=20
export DB_CONN_LIFETIME=30m
```

### Docker Compose 사용 시

Docker 컨테이너 내에서 CLI를 사용할 수 있습니다.

```bash
# 컨테이너 실행
docker compose -f docker/docker-compose.yml up -d

# 컨테이너 내부에서 CLI 사용
docker exec cnap-app /app/bin/cnap agent list

# 로그 확인
docker logs cnap-app -f
```

**참고:** Docker 환경에서는 환경 변수가 `docker-compose.yml`에 정의되어 있습니다.

---

## 문제 해결

### "controller: repository is not configured" 에러

**원인:** `DATABASE_URL` 환경 변수가 설정되지 않았습니다.

**해결 방법:**
```bash
export DATABASE_URL="postgres://cnap:cnap@localhost:5432/cnap?sslmode=disable"
```

또는 `.env` 파일에 추가:
```
DATABASE_URL=postgres://cnap:cnap@localhost:5432/cnap?sslmode=disable
```

### "agent not found" 에러

**원인:** 존재하지 않는 Agent를 조회하려 했습니다.

**해결 방법:**
```bash
# 등록된 Agent 목록 확인
cnap agent list
```

### "task not found" 에러

**원인:** 존재하지 않는 Task를 조회하려 했습니다.

**해결 방법:**
```bash
# Agent의 Task 목록 확인
cnap task list <agent-name>
```

### "agent name too long" 에러

**원인:** Agent 이름이 64자를 초과했습니다.

**해결 방법:** 64자 이하의 이름을 사용하세요.

### "유효하지 않은 상태" 에러

**원인:** Task 상태 변경 시 잘못된 상태값을 사용했습니다.

**해결 방법:** 다음 상태값만 사용 가능합니다:
- `pending`, `running`, `completed`, `failed`, `canceled`

### Database 연결 실패

**증상:**
```
Error: failed to connect to `host=localhost user=cnap database=cnap`:
dial error (dial tcp [::1]:5432: connect: connection refused)
```

**해결 방법:**
1. PostgreSQL 서비스가 실행 중인지 확인:
   ```bash
   # macOS
   brew services list

   # Linux
   systemctl status postgresql
   ```

2. Docker Compose 사용 시 컨테이너 상태 확인:
   ```bash
   docker compose -f docker/docker-compose.yml ps
   ```

3. 데이터베이스가 생성되었는지 확인:
   ```bash
   psql -h localhost -U cnap -d cnap -c "\dt"
   ```

---

## 예제 워크플로우

### 새 Agent 생성 및 Task 실행

```bash
# 1. Agent 생성
$ cnap agent create
Agent 이름: my-assistant
설명: 개인 비서 봇
모델 (예: gpt-4): gpt-4
프롬프트 (역할 정의): 효율적인 개인 비서입니다.
✓ Agent 'my-assistant' 생성 완료

# 2. Agent 확인
$ cnap agent view my-assistant
=== Agent 정보: my-assistant ===
이름:        my-assistant
상태:        active
...

# 3. Task 생성
$ cnap task create my-assistant task-001
✓ Task 'task-001' 생성 완료 (Agent: my-assistant)

# 4. Task 상태 업데이트
$ cnap task update-status task-001 running
✓ Task 'task-001' 상태 변경: running

# 5. Task 목록 확인
$ cnap task list my-assistant
TASK ID   STATUS   CREATED           UPDATED
-------   ------   -------           -------
task-001  running  2025-01-18 14:00  2025-01-18 14:01
```

### Agent 정보 수정

```bash
# 1. 현재 정보 확인
$ cnap agent view my-assistant

# 2. 정보 수정
$ cnap agent edit my-assistant
설명 (현재: 개인 비서 봇): 고급 AI 개인 비서
모델 (현재: gpt-4): gpt-4-turbo
프롬프트 (현재: 효율적인 개인 비서입니다.): 매우 효율적이고 친절한 개인 비서입니다.
✓ Agent 'my-assistant' 수정 완료

# 3. 변경 사항 확인
$ cnap agent view my-assistant
```

---

## 추가 정보

### 관련 문서
- [Docker 가이드](./docker-guide.md)
- [로컬 개발 가이드](./local-development.md)
- [프로젝트 개요](../CLAUDE.md)

### 지원
문제가 발생하면 GitHub Issues에 보고해주세요:
https://github.com/cnap-oss/app/issues
