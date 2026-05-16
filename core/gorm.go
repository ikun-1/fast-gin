package core

import (
	"fmt"
	"fast-gin/global"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	maxRetries   = 10
	retryBase    = 500 * time.Millisecond
	retryMaxWait = 5 * time.Second
)

func InitGorm() (db *gorm.DB) {
	cfg := global.Config.DB
	var dialector = cfg.Dsn()
	if dialector == nil {
		return
	}

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		db, lastErr = tryConnect(dialector)
		if lastErr == nil {
			break
		}
		wait := retryBase * (1 << i)
		if wait > retryMaxWait {
			wait = retryMaxWait
		}
		zap.S().Warnf("数据库连接失败 (第 %d 次重试, %v 后重试): %v", i+1, wait, lastErr)
		time.Sleep(wait)
	}
	if lastErr != nil {
		zap.S().Fatalf("数据库连接失败，已重试 %d 次: %v", maxRetries, lastErr)
	}

	sqlDB, err := db.DB()
	if err != nil {
		zap.S().Fatalf("获取数据库连接池失败 %v", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	zap.L().Info("数据库连接成功")
	return
}

func tryConnect(dialector gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return db, nil
}
