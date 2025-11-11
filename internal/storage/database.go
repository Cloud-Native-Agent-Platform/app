package storage

import (
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Open은 전달받은 설정으로 GORM 연결을 생성합니다.
// DSN에 따라 자동으로 PostgreSQL 또는 SQLite 드라이버를 선택합니다.
func Open(cfg Config) (*gorm.DB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("storage: empty DSN")
	}

	gormConfig := &gorm.Config{
		Logger:                 logger.Default.LogMode(cfg.LogLevel),
		NamingStrategy:         schema.NamingStrategy{SingularTable: false},
		SkipDefaultTransaction: cfg.SkipDefaultTxn,
		PrepareStmt:            cfg.PrepareStmt,
		DisableAutomaticPing:   cfg.DisableAutomaticPing,
	}

	// DSN에서 데이터베이스 드라이버 자동 감지
	var dialector gorm.Dialector
	if isSQLiteDSN(cfg.DSN) {
		dialector = sqlite.Open(cfg.DSN)
	} else {
		dialector = postgres.Open(cfg.DSN)
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("storage: open connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("storage: get sql.DB: %w", err)
	}

	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	return db, nil
}

// AutoMigrate는 CNAP 메시지 상태 관리를 위한 스키마를 마이그레이션합니다.
func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("storage: nil database handle")
	}

	if err := db.AutoMigrate(
		&Agent{},
		&Task{},
		&MessageIndex{},
		&RunStep{},
		&Checkpoint{},
	); err != nil {
		return fmt.Errorf("storage: migrate: %w", err)
	}
	return nil
}

// Close는 하부 sql.DB 자원을 해제합니다.
func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("storage: get sql.DB on close: %w", err)
	}
	return sqlDB.Close()
}

// isSQLiteDSN은 DSN이 SQLite 데이터베이스를 가리키는지 확인합니다.
// SQLite DSN 패턴:
// - "file:..." 또는 ":memory:"
// - ".db" 확장자로 끝나는 파일 경로
// - "sqlite:" 프리픽스
func isSQLiteDSN(dsn string) bool {
	dsn = strings.TrimSpace(dsn)
	return strings.HasPrefix(dsn, "file:") ||
		strings.HasPrefix(dsn, "sqlite:") ||
		dsn == ":memory:" ||
		strings.HasSuffix(dsn, ".db")
}
