# CNAP

## 프로젝트 구조

```text
cnap-app/
├── cmd/                  # 메인 애플리케이션
├── internal/             # 내부 패키지
│   ├── connector/             # Discord 봇
│   ├── controller/       # 에이전트 관리 및 서버 제어
│   └── runner/          # OpenCode 러너
├── go.mod
├── Makefile
└── README.md
```

## 시스템 요구사항

- **OS**: Linux (커널 3.10+)
- **Go**: 1.25+
- **runc**: 최신 버전
- **권한**: root

### 4. 빌드

```bash
make build
# 또는
go build -o cnap ./cmd/cnap
```

## 라이선스

MIT License
