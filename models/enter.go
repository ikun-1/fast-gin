package models

import (
	"database/sql"
	"time"
)

type Model struct {
	ID        uint         `gorm:"primaryKey;comment:主键ID" json:"id"`
	CreatedAt time.Time    `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt time.Time    `gorm:"comment:更新时间" json:"updatedAt"`
	DeletedAt sql.NullTime `gorm:"index;comment:删除时间" json:"-"`
}

type PageInfo struct {
	Page  int    `form:"page"`
	Limit int    `form:"limit"`
	Key   string `form:"key"`
	Order string `form:"order"`
}
