package models

type User struct {
	Model
	Username string `gorm:"size:50;not null;uniqueIndex;comment:用户名" json:"username"`
	Nickname string `gorm:"size:32;comment:昵称" json:"nickname"`
	Password string `gorm:"size:255;not null;comment:密码哈希" json:"-"`
	RealName string `gorm:"size:50;comment:真实姓名" json:"realName"`
	Email    string `gorm:"size:100;comment:邮箱" json:"email"`
	Phone    string `gorm:"size:20;comment:手机号" json:"phone"`
	AvatarID *uint  `gorm:"comment:头像图片ID" json:"avatarId"`
	Status   int8   `gorm:"default:1;comment:状态 1启用 0禁用" json:"status"`
}

func init() {
	MigrateModels = append(MigrateModels, &User{})
}
