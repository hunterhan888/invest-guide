package database

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// gormLogger 用 Warn 级别记录 SQL，仅对超过 2s 的慢查询或错误打印。
// 阈值调高避免大批量向量 INSERT 反复打满日志（单条 chunk 含 1024 维向量，日志会爆炸）。
func gormLogger() logger.Interface {
	return logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             2 * time.Second,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      false,
	})
}

// Connect 按连接串前缀分发到对应驱动
//
//	postgres://... 或 postgresql://... → PostgreSQL
//	sqlite://...                       → SQLite（文件或 :memory:）
func Connect(dsn string) (*gorm.DB, error) {
	switch {
	case strings.HasPrefix(dsn, "postgres://"), strings.HasPrefix(dsn, "postgresql://"):
		return gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLogger()})
	case strings.HasPrefix(dsn, "sqlite://"):
		return NewSQLite(strings.TrimPrefix(dsn, "sqlite://"))
	default:
		return nil, fmt.Errorf("unsupported database url: %s", dsn)
	}
}

// NewSQLite 创建内存或文件 SQLite，供测试与本地开发使用
func NewSQLite(path string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger:                                   gormLogger(),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
}

// TransactionWrap 在事务中执行 fn，错误自动回滚
func TransactionWrap(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}

// AutoMigrate 以 GORM AutoMigrate 建表。开发（SQLite）环境使用；
// 生产（PostgreSQL）使用 golang-migrate 迁移文件，见 backend/migrations/。
func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	return db.AutoMigrate(models...)
}
