package flags

import (
	"context"
	"errors"
	"fast-gin/dal/query"
	"fast-gin/models"
	"fast-gin/utils/pwd"
	"fmt"
	"os"

	"go.uber.org/zap"
	"golang.org/x/term"
	"gorm.io/gorm"
)

type User struct {
}

func (User) Create() {
	var user models.User
	var option int
	fmt.Println("请输入角色编码：1.admin 2.user")
	_, err := fmt.Scanln(&option)
	if err != nil {
		fmt.Println("输入错误", err)
		return
	}

	var roleCode string
	switch option {
	case 1:
		roleCode = "admin"
	case 2:
		roleCode = "user"
	default:
		fmt.Println("用户角色输入错误，仅支持：1.admin 2.user")
		return
	}

	ctx := context.Background()
	role, err := query.Role.WithContext(ctx).
		Where(
			query.Role.Code.Eq(roleCode),
			query.Role.Status.Eq(1),
		).
		Take()
	if err != nil {
		fmt.Println("角色不存在或已禁用", err)
		return
	}
	fmt.Println("请输入用户名")
	fmt.Scanln(&user.Username)

	_, err = query.User.WithContext(ctx).Where(query.User.Username.Eq(user.Username)).Take()
	if err == nil {
		fmt.Println("用户名已存在")
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Println("查询用户名失败", err)
		return
	}

	fmt.Println("请输入密码")
	password, err := term.ReadPassword(int(os.Stdin.Fd())) // 读取用户输入的密码
	if err != nil {
		fmt.Println("读取密码时出错:", err)
		return
	}
	fmt.Println("请再次输入密码")
	rePassword, err := term.ReadPassword(int(os.Stdin.Fd())) // 读取用户输入的密码
	if err != nil {
		fmt.Println("读取密码时出错:", err)
		return
	}
	if string(password) != string(rePassword) {
		fmt.Println("两次密码不一致")
		return
	}

	hashPwd := pwd.GenerateFromPassword(string(password))
	newUser := &models.User{
		Username: user.Username,
		Password: hashPwd,
		Status:   1,
	}
	err = query.User.WithContext(ctx).Create(newUser)
	if err != nil {
		zap.S().Errorf("用户创建失败 %s", err)
		return
	}

	err = query.UserRole.WithContext(ctx).Create(&models.UserRole{
		UserID: newUser.ID,
		RoleID: role.ID,
	})
	if err != nil {
		zap.S().Errorf("用户角色关联创建失败 %s", err)
		return
	}
	zap.S().Infof("用户创建成功")

}
func (User) List() {
	userList, err := query.User.WithContext(context.Background()).
		Order(query.User.CreatedAt.Desc()).
		Limit(10).
		Find()
	if err != nil {
		fmt.Println("查询用户列表失败", err)
		return
	}
	for _, model := range userList {
		fmt.Printf("用户id：%d  用户名：%s 用户昵称：%s 创建时间：%s\n",
			model.ID,
			model.Username,
			model.Nickname,
			model.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}
}
