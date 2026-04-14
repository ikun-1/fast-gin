package models

import "time"

type UserRole struct {
	UserID    uint      `gorm:"primaryKey;index;comment:用户ID" json:"userID"`
	RoleID    uint      `gorm:"primaryKey;index;comment:角色ID" json:"roleID"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"createdAt"`
}

func init() {
	MigrateModels = append(MigrateModels, &UserRole{})
}
