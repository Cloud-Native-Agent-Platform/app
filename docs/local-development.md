# 로컬 개발 가이드

이 문서는 Docker 없이 SQLite를 사용하여 로컬에서 개발하는 방법을 설명합니다.

## 목차

- [개요](#개요)
- [빠른 시작](#빠른-시작)
- [Makefile 명령어](#makefile-명령어)
- [환경 설정](#환경-설정)
- [데이터베이스 선택](#데이터베이스-선택)
- [자주 묻는 질문](#자주-묻는-질문)

## 개요

CNAP은 PostgreSQL과 SQLite 두 가지 데이터베이스를 지원합니다:

- **PostgreSQL**: 프로덕션 환경 권장 (Docker Compose 사용)
- **SQLite**: 로컬 개발 및 테스트용 (Docker 불필요)

SQLite를 사용하면 Docker 설치 없이 즉시 개발을 시작할 수 있어 진입장벽이 낮습니다.

## 빠른 시작

```bash
# 1. 의존성 설치
make deps

# 2. SQLite로 바로 실행 (DATABASE_URL 설정 불필요)
make run-local
```

이게 전부입니다! 애플리케이션이 자동으로 `./data/cnap.db`에 SQLite 데이터베이스를 생성하고 실행됩니다.

## Makefile 명령어

### 로컬 개발 명령어 (SQLite)

```bash
# SQLite로 애플리케이션 실행 (소스코드에서 직접 실행)
make run-local

# 빌드 후 SQLite로 실행
make dev

# 인메모리 SQLite로 테스트 실행
make test-local

# 로컬 SQLite 데이터베이스 파일 삭제
make clean-db
```

### 일반 명령어

```bash
# 도움말 보기
make help

# 빌드만 실행
make build

# 테스트 실행 (기본 설정)
make test

# 코드 포맷팅
make fmt

# 린트 검사
make lint

# 포맷팅, 린트, 테스트 모두 실행
make check
```

## 환경 설정

### SQLite 사용 (로컬 개발)

환경변수 설정이 **불필요**합니다. `DATABASE_URL`이 설정되지 않으면 자동으로 SQLite를 사용합니다.

기본 데이터베이스 파일 위치:
```
./data/cnap.db
```

커스텀 SQLite 파일 경로 지정 (선택사항):
```bash
export SQLITE_DATABASE="./my-custom-db.db"
make run-local
```

인메모리 SQLite 사용 (테스트용):
```bash
export SQLITE_DATABASE=":memory:"
go run ./cmd/cnap start
```

### PostgreSQL 사용 (프로덕션/Docker)

PostgreSQL을 사용하려면 `DATABASE_URL` 환경변수를 설정하세요:

```bash
export DATABASE_URL="postgres://user:password@localhost:5432/cnap?sslmode=disable"
go run ./cmd/cnap start
```

또는 Docker Compose 사용:
```bash
docker compose up
```

## 데이터베이스 선택

애플리케이션은 DSN(Data Source Name)을 분석하여 자동으로 적절한 드라이버를 선택합니다:

| DSN 패턴 | 사용되는 드라이버 |
|----------|------------------|
| `DATABASE_URL` 미설정 | SQLite (기본값) |
| `file:...` | SQLite |
| `sqlite:...` | SQLite |
| `*.db` | SQLite |
| `:memory:` | SQLite (인메모리) |
| 기타 (postgres://...) | PostgreSQL |

### 예시

```bash
# SQLite 파일 데이터베이스
export DATABASE_URL="./data/cnap.db"
export DATABASE_URL="file:./data/cnap.db"
export DATABASE_URL="sqlite:./data/cnap.db"

# SQLite 인메모리 (테스트용)
export DATABASE_URL=":memory:"

# PostgreSQL
export DATABASE_URL="postgres://user:pass@localhost:5432/cnap"
```

## 자주 묻는 질문

### Q: SQLite와 PostgreSQL 중 어떤 것을 사용해야 하나요?

- **로컬 개발/테스트**: SQLite 권장 (빠르고 간편함)
- **프로덕션/배포**: PostgreSQL 권장 (성능, 동시성, 안정성)

### Q: 데이터베이스를 초기화하려면 어떻게 하나요?

SQLite 데이터베이스를 삭제하면 다음 실행 시 자동으로 재생성됩니다:

```bash
make clean-db
make run-local
```

### Q: 테스트는 어떤 데이터베이스를 사용하나요?

- `make test`: 환경변수에 따름 (기본값은 SQLite)
- `make test-local`: 항상 인메모리 SQLite 사용 (격리되고 빠름)

### Q: 데이터베이스 마이그레이션은 어떻게 동작하나요?

GORM의 AutoMigrate가 애플리케이션 시작 시 자동으로 실행됩니다. 스키마 변경사항이 있으면 자동으로 적용됩니다.

### Q: SQLite 사용 시 제약사항이 있나요?

SQLite는 훌륭한 개발용 데이터베이스이지만 다음 제약사항이 있습니다:

- 동시 쓰기 성능 제한
- 일부 고급 PostgreSQL 기능 미지원
- 대용량 데이터 처리 시 성능 저하 가능

따라서 **프로덕션에서는 PostgreSQL 사용을 강력히 권장**합니다.

### Q: 개발 중에 데이터베이스를 전환할 수 있나요?

네, 언제든지 전환 가능합니다:

```bash
# SQLite로 개발
make run-local

# PostgreSQL로 전환 (Docker Compose 시작)
docker compose up
```

데이터는 각 데이터베이스에 독립적으로 저장되므로 주의하세요.

## 추가 설정

### 데이터베이스 로그 레벨 조정

```bash
# 로그 레벨: silent, error, warn, info
export DB_LOG_LEVEL=info
make run-local
```

### 커넥션 풀 설정

```bash
export DB_MAX_IDLE=5
export DB_MAX_OPEN=20
export DB_CONN_LIFETIME=30m
make run-local
```

## 문제 해결

### "database is locked" 오류

SQLite는 단일 쓰기자 제한이 있습니다. 여러 인스턴스를 동시에 실행하고 있다면 하나만 남기고 종료하세요.

### 데이터베이스 파일이 생성되지 않음

`./data` 디렉토리 생성 권한을 확인하세요:

```bash
mkdir -p ./data
ls -la ./data
```

### PostgreSQL 연결 실패

Docker Compose가 실행 중인지 확인:

```bash
docker compose ps
docker compose logs postgres
```

## 추가 리소스

- [GORM 문서](https://gorm.io/docs/)
- [SQLite 문서](https://www.sqlite.org/docs.html)
- [PostgreSQL 문서](https://www.postgresql.org/docs/)
- [Docker Compose 가이드](../README.md#docker-compose)
