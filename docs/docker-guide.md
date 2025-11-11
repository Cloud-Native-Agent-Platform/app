# Docker 가이드

이 문서는 Docker를 사용하여 CNAP을 실행하는 방법을 설명합니다.

## 목차

- [아키텍처](#아키텍처)
- [빠른 시작](#빠른-시작)
- [설정](#설정)
- [명령어](#명령어)
- [볼륨 관리](#볼륨-관리)
- [문제 해결](#문제-해결)

## 아키텍처

### 멀티 컨테이너 구조

CNAP은 두 개의 독립적인 컨테이너로 구성됩니다:

```
┌─────────────────────────────────────┐
│         Docker Network              │
│                                     │
│  ┌──────────────┐  ┌─────────────┐ │
│  │   postgres   │  │     app     │ │
│  │              │  │             │ │
│  │ PostgreSQL   │◄─┤  CNAP App   │ │
│  │   16-alpine  │  │             │ │
│  └──────────────┘  └─────────────┘ │
│         ▲                           │
└─────────┼───────────────────────────┘
          │
    ┌─────┴──────┐
    │   Volumes  │
    │            │
    │ postgres_  │
    │   data     │
    └────────────┘
```

### 왜 멀티 컨테이너인가?

1. **관심사 분리**: 데이터베이스와 애플리케이션 독립 관리
2. **보안**: 격리된 환경으로 보안 강화
3. **확장성**: 각 컴포넌트 독립적 스케일링
4. **유지보수**: PostgreSQL 버전 업그레이드 용이
5. **표준 준수**: Docker best practices 준수

## 빠른 시작

### 전제 조건

- Docker 20.10+
- Docker Compose 2.0+

### 실행

```bash
# 프로젝트 디렉토리에서
cd docker
docker compose up -d
```

서비스 확인:
```bash
docker compose ps
```

로그 확인:
```bash
# 전체 로그
docker compose logs -f

# 특정 서비스만
docker compose logs -f app
docker compose logs -f postgres
```

종료:
```bash
docker compose down
```

## 설정

### 환경 변수

`.env` 파일을 생성하여 설정을 커스터마이즈할 수 있습니다:

```bash
# docker/.env
POSTGRES_USER=myuser
POSTGRES_PASSWORD=securepassword
POSTGRES_DB=mydb
POSTGRES_PORT=5432
APP_ENV=production
APP_LOG_LEVEL=info
```

### 기본값

환경변수를 설정하지 않으면 다음 기본값이 사용됩니다:

| 변수 | 기본값 | 설명 |
|------|--------|------|
| POSTGRES_USER | cnap | PostgreSQL 사용자명 |
| POSTGRES_PASSWORD | cnap | PostgreSQL 비밀번호 |
| POSTGRES_DB | cnap | 데이터베이스 이름 |
| POSTGRES_PORT | 5432 | 외부 노출 포트 |
| APP_ENV | development | 애플리케이션 환경 |
| APP_LOG_LEVEL | info | 로그 레벨 |

## 명령어

### 기본 명령어

```bash
# 서비스 시작 (백그라운드)
docker compose up -d

# 서비스 시작 (포어그라운드, 로그 표시)
docker compose up

# 서비스 중지
docker compose stop

# 서비스 중지 및 컨테이너 삭제
docker compose down

# 볼륨까지 삭제 (데이터 손실 주의!)
docker compose down -v

# 재시작
docker compose restart

# 특정 서비스만 재시작
docker compose restart app
```

### 빌드 관련

```bash
# 이미지 빌드 (캐시 사용)
docker compose build

# 이미지 빌드 (캐시 없이)
docker compose build --no-cache

# 빌드 후 즉시 실행
docker compose up --build
```

### 로그 및 디버깅

```bash
# 전체 로그
docker compose logs

# 실시간 로그 스트리밍
docker compose logs -f

# 최근 100줄만
docker compose logs --tail=100

# 특정 서비스 로그
docker compose logs -f app
docker compose logs -f postgres

# 컨테이너 상태 확인
docker compose ps

# 리소스 사용량 확인
docker stats cnap-app cnap-postgres
```

### 데이터베이스 접속

```bash
# PostgreSQL 접속
docker compose exec postgres psql -U cnap -d cnap

# SQL 쿼리 실행
docker compose exec postgres psql -U cnap -d cnap -c "SELECT * FROM agents;"

# 데이터베이스 백업
docker compose exec postgres pg_dump -U cnap cnap > backup.sql

# 데이터베이스 복원
docker compose exec -T postgres psql -U cnap -d cnap < backup.sql
```

### 애플리케이션 접속

```bash
# 애플리케이션 컨테이너 쉘 접속
docker compose exec app sh

# 헬스체크 확인
docker compose exec app /app/cnap health
```

## 볼륨 관리

### 볼륨 구조

```
volumes:
  postgres_data:    # PostgreSQL 데이터 저장
  app_data:         # SQLite 폴백용 (사용 안 함)
```

### 볼륨 명령어

```bash
# 볼륨 목록
docker volume ls

# 볼륨 상세 정보
docker volume inspect docker_postgres_data

# 볼륨 삭제 (주의: 데이터 손실!)
docker compose down -v

# 특정 볼륨만 삭제
docker volume rm docker_postgres_data
```

### 데이터 백업

```bash
# PostgreSQL 데이터 백업
docker compose exec postgres pg_dump -U cnap -Fc cnap > backup_$(date +%Y%m%d).dump

# 볼륨 전체 백업 (tar)
docker run --rm -v docker_postgres_data:/data -v $(pwd):/backup alpine tar czf /backup/postgres_data_backup.tar.gz -C /data .
```

### 데이터 복원

```bash
# PostgreSQL 덤프 복원
docker compose exec -T postgres pg_restore -U cnap -d cnap < backup.dump

# 볼륨 전체 복원
docker run --rm -v docker_postgres_data:/data -v $(pwd):/backup alpine sh -c "cd /data && tar xzf /backup/postgres_data_backup.tar.gz"
```

## 문제 해결

### 서비스가 시작되지 않음

```bash
# 로그 확인
docker compose logs

# 컨테이너 상태 확인
docker compose ps -a

# 네트워크 확인
docker network ls
docker network inspect docker_cnap-network
```

### PostgreSQL 연결 실패

```bash
# PostgreSQL 컨테이너가 healthy 상태인지 확인
docker compose ps

# PostgreSQL 로그 확인
docker compose logs postgres

# 네트워크 연결 테스트
docker compose exec app ping postgres

# 직접 연결 테스트
docker compose exec app nc -zv postgres 5432
```

### 데이터베이스 초기화

```bash
# 1. 서비스 중지
docker compose down

# 2. 볼륨 삭제 (데이터 손실!)
docker volume rm docker_postgres_data

# 3. 재시작
docker compose up -d
```

### 이미지 재빌드

```bash
# 캐시 없이 완전히 재빌드
docker compose build --no-cache app

# 빌드 후 즉시 실행
docker compose up --build
```

### 포트 충돌

```bash
# 다른 포트 사용
POSTGRES_PORT=5433 docker compose up -d

# 또는 .env 파일에서
# POSTGRES_PORT=5433
```

### 권한 문제

애플리케이션 컨테이너는 non-root 유저(`cnap:1000`)로 실행됩니다. 볼륨 권한 문제 시:

```bash
# 볼륨 권한 확인
docker compose exec app ls -la /app/data

# 필요시 권한 수정 (개발 환경에만 사용)
docker compose exec --user root app chown -R cnap:cnap /app/data
```

## 프로덕션 체크리스트

프로덕션 배포 전 확인사항:

- [ ] 강력한 PostgreSQL 비밀번호 설정
- [ ] 환경변수를 `.env` 파일이 아닌 시크릿 관리 도구로 관리
- [ ] 볼륨 백업 전략 수립
- [ ] 리소스 제한 설정 (CPU, 메모리)
- [ ] 로그 수집 및 모니터링 설정
- [ ] SSL/TLS 설정 (PostgreSQL sslmode=require)
- [ ] 방화벽 규칙 확인
- [ ] 헬스체크 엔드포인트 모니터링

### 프로덕션 추가 설정 예시

```yaml
services:
  app:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## 추가 리소스

- [Docker Compose 문서](https://docs.docker.com/compose/)
- [PostgreSQL Docker 이미지](https://hub.docker.com/_/postgres)
- [로컬 개발 가이드](./local-development.md) (SQLite 사용)
