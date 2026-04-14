package models

type Role struct {
	Model
	Name        string `gorm:"size:50;not null;uniqueIndex;comment:角色名称" json:"name"`
	Code        string `gorm:"size:50;not null;uniqueIndex;comment:角色编码" json:"code"`
	Description string `gorm:"size:200;comment:角色描述" json:"description"`
	Status      int8   `gorm:"default:1;comment:状态 1启用 0禁用" json:"status"` // 1 启用 0 禁用
	PID         *uint  `gorm:"comment:父角色ID 实现单继承" json:"pid"`
}

func init() {
	MigrateModels = append(MigrateModels, &Role{})
}
