package core

import (
	"fast-gin/global"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func InitGorm() (db *gorm.DB) {
	cfg := global.Config.DB
	var dialector = cfg.Dsn()
	if dialector == nil {
		return
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, // 不生成实体外键
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // use singular table name, table for `User` would be `user` with this option enabled
		},
	})
	if err != nil {
		zap.S().Fatalf("数据库连接失败 %v", err)
	}
	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		zap.S().Fatalf("获取数据库连接失败 %v", err)
		return
	}
	err = sqlDB.Ping()
	if err != nil {
		zap.S().Fatalf("数据库连接失败 %v", err)
		return
	}
	// 设置连接池
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	zap.L().Info("数据库连接成功")
	return
}
