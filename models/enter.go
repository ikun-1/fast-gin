package models

import (
	"database/sql"
	"time"
)

type Model struct {
	ID        uint         `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
	DeletedAt sql.NullTime `gorm:"index" json:"-"`
}

type PageInfo struct {
	Page  int    `form:"page"`
	Limit int    `form:"limit"`
	Key   string `form:"key"`
	Order string `form:"order"`
}
