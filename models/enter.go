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
	Page    int    `form:"page" json:"page" default:"1" example:"1"`
	Limit   int    `form:"limit" json:"limit" default:"10" example:"10"`
	Key     string `form:"key" json:"key" example:""`
	SortBy  string `form:"sortBy" json:"sortBy" default:"created_at" example:"created_at"`
	SortDir string `form:"sortDir" json:"sortDir" example:"desc" default:"desc"`
}

// MigrateModels stores all models that need schema migration.
var MigrateModels = make([]any, 0)

type BindId struct {
	ID uint `uri:"id" binding:"required"`
}
