# CNAP

## 프로젝트 구조

```text
cnap-app/
├── cmd/                  # 메인 애플리케이션
├── internal/             # 내부 패키지
│   ├── connector/             # Discord 봇
│   ├── controller/       # 에이전트 관리 및 서버 제어
│   ├── runner/          # OpenCode 러너
│   └── storage/         # GORM 기반 영속 계층
├── go.mod
├── Makefile
└── README.md
```

## 시스템 요구사항

- **OS**: Linux (커널 3.10+)
- **Go**: 1.23+
- **runc**: 최신 버전
- **권한**: root

### 4. 빌드

```bash
make build
# 또는
go build -o cnap ./cmd/cnap
```

### 5. 테스트

저장소 동작과 컨트롤러 흐름은 GORM의 인메모리(SQLite) 드라이버를 활용한 단위 테스트로 검증됩니다.

```bash
go test ./...
```

## 데이터베이스 설정

CNAP은 PostgreSQL과 GORM을 사용하여 다음 엔티티를 관리합니다.

- `agents`: 로직 멀티테넌시를 위한 에이전트 메타데이터
- `tasks`: 에이전트별 작업 실행 단위
- `msg_index`: 메시지 본문이 저장된 로컬 JSON 파일 경로 인덱스
- `run_steps`: 작업 단계 기록
- `checkpoints`: Git 스냅샷(해시) 기록

### 환경 변수

| 변수 | 필수 | 설명 | 기본값 |
| --- | --- | --- | --- |
| `DATABASE_URL` | ✅ | PostgreSQL 연결 DSN (예: `postgres://user:pass@localhost:5432/cnap?sslmode=disable`) | 없음 |
| `DB_LOG_LEVEL` |  | GORM 로그 레벨 (`silent`, `error`, `warn`, `info`) | `warn` |
| `DB_MAX_IDLE` |  | 연결 풀 idle 개수 | `5` |
| `DB_MAX_OPEN` |  | 연결 풀 최대 개수 | `20` |
| `DB_CONN_LIFETIME` |  | 연결 최대 수명 (예: `1h`) | `30m` |
| `DB_SKIP_DEFAULT_TXN` |  | 기본 트랜잭션 생략 여부 (`true`/`false`) | `true` |
| `DB_PREPARE_STMT` |  | Prepare statement 캐시 활성화 여부 | `false` |
| `DB_DISABLE_AUTO_PING` |  | GORM 자동 `Ping` 비활성화 | `false` |

애플리케이션이 시작될 때 자동으로 스키마 마이그레이션을 수행하며, 메시지 본문은 데이터베이스가 아닌 로컬 JSON 파일로 유지됩니다.

### Docker Compose로 애플리케이션 실행

`docker/docker-compose.yml`은 CNAP 애플리케이션과 PostgreSQL을 함께 실행합니다. 데이터 및 메시지 파일은 `.gitignore`에 포함된 `docker-data/` 경로에 저장됩니다.

```bash
docker compose -f docker/docker-compose.yml up -d
```

환경 변수는 다음과 같이 기본값을 재정의할 수 있습니다.

- 데이터베이스: `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_PORT`
- 애플리케이션: `APP_ENV`, `APP_LOG_LEVEL`

## 라이선스

MIT License
