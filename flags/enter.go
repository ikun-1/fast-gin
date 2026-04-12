package flags

import (
	"fast-gin/global"
	"flag"
	"fmt"
	"os"
)

type FlagOptions struct {
	File    string
	Version bool
	DB      bool
	Menu    string // 菜单 user rbac
	Type    string // 类型 create list create-role create-perm grant-user-role grant-role-perm revoke-user-role revoke-role-perm
}

var Options FlagOptions

func Parse() {
	flag.StringVar(&Options.File, "f", "settings-dev.yaml", "配置文件路径")
	flag.StringVar(&Options.Menu, "m", "", "菜单 user rbac")
	flag.StringVar(&Options.Type, "t", "", "类型 create list create-role create-perm grant-user-role grant-role-perm revoke-user-role revoke-role-perm")
	flag.BoolVar(&Options.Version, "v", false, "打印当前版本")
	flag.BoolVar(&Options.DB, "db", false, "迁移表结构")
	flag.Parse()
}

func Run() {
	if Options.DB {
		MigrateDB()
		os.Exit(0)
	}
	if Options.Version {
		fmt.Println("当前后端版本", global.Version)
		os.Exit(0)
	}
	if Options.Menu == "user" {
		var user User
		switch Options.Type {
		case "create":
			user.Create()
		case "list":
			user.List()
		}
		os.Exit(0)
	}
	if Options.Menu == "rbac" {
		var rbac RBAC
		switch Options.Type {
		case "create-role":
			rbac.CreateRole()
		case "create-perm":
			rbac.CreatePermission()
		case "grant-user-role":
			rbac.GrantUserRole()
		case "grant-role-perm":
			rbac.GrantRolePermission()
		case "revoke-user-role":
			rbac.RevokeUserRole()
		case "revoke-role-perm":
			rbac.RevokeRolePermission()
		}
		os.Exit(0)
	}
}
