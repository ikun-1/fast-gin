package models

import "time"

type RolePermission struct {
	RoleID    uint      `gorm:"primaryKey;index;comment:角色ID" json:"roleID"`
	PermID    uint      `gorm:"primaryKey;index;comment:权限ID" json:"permID"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"createdAt"`
}
