package config

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type DBMode string

const (
	DBMysqlMode  = "mysql"
	DBSqliteMode = "sqlite"
)

type DB struct {
	Mode     DBMode `yaml:"mode"` // 模式 mysql sqlite
	DBName   string `yaml:"db_name"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func (db DB) Dsn() gorm.Dialector {
	switch db.Mode {
	case DBMysqlMode:
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			db.User,
			db.Password,
			db.Host,
			db.Port,
			db.DBName,
		)
		return mysql.Open(dsn)
	case DBSqliteMode:
		return sqlite.Open(db.DBName)
	default:
		zap.L().Fatal("未配置数据库连接")
		return nil
	}
}
