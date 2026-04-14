package models

type Permission struct {
	Model
	Code      string `gorm:"size:100;not null;uniqueIndex;comment:权限标识" json:"code"`
	Name      string `gorm:"size:50;not null;comment:权限名称" json:"name"`
	Module    string `gorm:"size:50;comment:所属模块" json:"module"`
	Type      int8   `gorm:"default:2;comment:类型 1目录 2菜单 3按钮" json:"type"`
	PID       uint   `gorm:"default:0;comment:父权限ID" json:"pid"`
	Path      string `gorm:"size:200;comment:前端路由路径" json:"path"`
	Component string `gorm:"size:200;comment:前端组件路径" json:"component"`
	Icon      string `gorm:"size:50;comment:图标" json:"icon"`
	SortOrder int    `gorm:"default:0;comment:排序" json:"sortOrder"`
}

func init() {
	MigrateModels = append(MigrateModels, &Permission{})
}
