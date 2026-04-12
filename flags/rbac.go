package flags

import (
	"context"
	"errors"
	"fast-gin/dal/query"
	"fast-gin/models"
	"fast-gin/permissions"
	"fmt"

	"gorm.io/gorm"
)

type RBAC struct{}

// CreateRole 新建角色
// 交互输入：角色名称、角色编码、角色描述
func (RBAC) CreateRole() {
	ctx := context.Background()
	var name string
	var code string
	var desc string

	fmt.Println("请输入角色名称")
	if _, err := fmt.Scanln(&name); err != nil {
		fmt.Println("输入角色名称失败", err)
		return
	}

	fmt.Println("请输入角色编码，例如：admin / user")
	if _, err := fmt.Scanln(&code); err != nil {
		fmt.Println("输入角色编码失败", err)
		return
	}

	fmt.Println("请输入角色描述（可选，输入-表示空）")
	if _, err := fmt.Scanln(&desc); err != nil {
		fmt.Println("输入角色描述失败", err)
		return
	}
	if desc == "-" {
		desc = ""
	}

	_, err := query.Role.WithContext(ctx).Where(query.Role.Code.Eq(code)).Take()
	if err == nil {
		fmt.Println("角色编码已存在")
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Println("查询角色失败", err)
		return
	}

	err = query.Role.WithContext(ctx).Create(&models.Role{
		Name:        name,
		Code:        code,
		Description: desc,
		Status:      1,
	})
	if err != nil {
		fmt.Println("创建角色失败", err)
		return
	}

	fmt.Println("创建角色成功")
}

// CreatePermission 新建权限
// 交互输入：权限编码、权限名称、模块、类型
func (RBAC) CreatePermission() {
	ctx := context.Background()
	var code string
	var name string
	var module string
	var permType int8

	fmt.Println("请输入权限编码，例如：image:upload")
	if _, err := fmt.Scanln(&code); err != nil {
		fmt.Println("输入权限编码失败", err)
		return
	}

	if _, ok := permissions.PermBit[code]; !ok {
		fmt.Println("权限编码未在代码映射中注册，请先在 permissions 包中定义")
		return
	}

	fmt.Println("请输入权限名称")
	if _, err := fmt.Scanln(&name); err != nil {
		fmt.Println("输入权限名称失败", err)
		return
	}

	fmt.Println("请输入所属模块（可选，输入-表示空）")
	if _, err := fmt.Scanln(&module); err != nil {
		fmt.Println("输入所属模块失败", err)
		return
	}
	if module == "-" {
		module = ""
	}

	fmt.Println("请输入权限类型：1目录 2菜单 3按钮（默认2）")
	if _, err := fmt.Scanln(&permType); err != nil {
		permType = 2
	}
	if permType < 1 || permType > 3 {
		permType = 2
	}

	_, err := query.Permission.WithContext(ctx).Where(query.Permission.Code.Eq(code)).Take()
	if err == nil {
		fmt.Println("权限编码已存在")
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Println("查询权限失败", err)
		return
	}

	err = query.Permission.WithContext(ctx).Create(&models.Permission{
		Code:   code,
		Name:   name,
		Module: module,
		Type:   permType,
	})
	if err != nil {
		fmt.Println("创建权限失败", err)
		return
	}

	fmt.Println("创建权限成功")
}

// GrantUserRole 为用户添加角色关联
// 交互输入：用户名、角色编码
func (RBAC) GrantUserRole() {
	ctx := context.Background()
	var username string
	var roleCode string

	fmt.Println("请输入用户名")
	if _, err := fmt.Scanln(&username); err != nil {
		fmt.Println("输入用户名失败", err)
		return
	}

	fmt.Println("请输入角色编码，例如：admin / user")
	if _, err := fmt.Scanln(&roleCode); err != nil {
		fmt.Println("输入角色编码失败", err)
		return
	}

	user, err := query.User.WithContext(ctx).Where(query.User.Username.Eq(username)).Take()
	if err != nil {
		fmt.Println("用户不存在", err)
		return
	}

	role, err := query.Role.WithContext(ctx).
		Where(query.Role.Code.Eq(roleCode), query.Role.Status.Eq(1)).
		Take()
	if err != nil {
		fmt.Println("角色不存在或已禁用", err)
		return
	}

	_, err = query.UserRole.WithContext(ctx).
		Where(query.UserRole.UserID.Eq(user.ID), query.UserRole.RoleID.Eq(role.ID)).
		Take()
	if err == nil {
		fmt.Println("用户已拥有该角色，无需重复添加")
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Println("查询用户角色关联失败", err)
		return
	}

	err = query.UserRole.WithContext(ctx).Create(&models.UserRole{
		UserID: user.ID,
		RoleID: role.ID,
	})
	if err != nil {
		fmt.Println("添加用户角色关联失败", err)
		return
	}

	fmt.Println("添加用户角色关联成功")
}

// GrantRolePermission 为角色添加权限关联
// 交互输入：角色编码、权限编码
func (RBAC) GrantRolePermission() {
	ctx := context.Background()
	var roleCode string
	var permCode string

	fmt.Println("请输入角色编码，例如：admin / user")
	if _, err := fmt.Scanln(&roleCode); err != nil {
		fmt.Println("输入角色编码失败", err)
		return
	}

	fmt.Println("请输入权限编码，例如：image:upload")
	if _, err := fmt.Scanln(&permCode); err != nil {
		fmt.Println("输入权限编码失败", err)
		return
	}

	role, err := query.Role.WithContext(ctx).
		Where(query.Role.Code.Eq(roleCode), query.Role.Status.Eq(1)).
		Take()
	if err != nil {
		fmt.Println("角色不存在或已禁用", err)
		return
	}

	perm, err := query.Permission.WithContext(ctx).Where(query.Permission.Code.Eq(permCode)).Take()
	if err != nil {
		fmt.Println("权限不存在", err)
		return
	}

	_, err = query.RolePermission.WithContext(ctx).
		Where(query.RolePermission.RoleID.Eq(role.ID), query.RolePermission.PermID.Eq(perm.ID)).
		Take()
	if err == nil {
		fmt.Println("角色已拥有该权限，无需重复添加")
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Println("查询角色权限关联失败", err)
		return
	}

	err = query.RolePermission.WithContext(ctx).Create(&models.RolePermission{
		RoleID: role.ID,
		PermID: perm.ID,
	})
	if err != nil {
		fmt.Println("添加角色权限关联失败", err)
		return
	}

	fmt.Println("添加角色权限关联成功")
}

// RevokeUserRole 为用户删除角色关联
// 交互输入：用户名、角色编码
func (RBAC) RevokeUserRole() {
	ctx := context.Background()
	var username string
	var roleCode string

	fmt.Println("请输入用户名")
	if _, err := fmt.Scanln(&username); err != nil {
		fmt.Println("输入用户名失败", err)
		return
	}

	fmt.Println("请输入角色编码，例如：admin / user")
	if _, err := fmt.Scanln(&roleCode); err != nil {
		fmt.Println("输入角色编码失败", err)
		return
	}

	user, err := query.User.WithContext(ctx).Where(query.User.Username.Eq(username)).Take()
	if err != nil {
		fmt.Println("用户不存在", err)
		return
	}

	role, err := query.Role.WithContext(ctx).Where(query.Role.Code.Eq(roleCode)).Take()
	if err != nil {
		fmt.Println("角色不存在", err)
		return
	}

	_, err = query.UserRole.WithContext(ctx).
		Where(query.UserRole.UserID.Eq(user.ID), query.UserRole.RoleID.Eq(role.ID)).
		Take()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Println("用户未绑定该角色，无需删除")
		return
	}
	if err != nil {
		fmt.Println("查询用户角色关联失败", err)
		return
	}

	_, err = query.UserRole.WithContext(ctx).
		Where(query.UserRole.UserID.Eq(user.ID), query.UserRole.RoleID.Eq(role.ID)).
		Delete()
	if err != nil {
		fmt.Println("删除用户角色关联失败", err)
		return
	}

	fmt.Println("删除用户角色关联成功")
}

// RevokeRolePermission 为角色删除权限关联
// 交互输入：角色编码、权限编码
func (RBAC) RevokeRolePermission() {
	ctx := context.Background()
	var roleCode string
	var permCode string

	fmt.Println("请输入角色编码，例如：admin / user")
	if _, err := fmt.Scanln(&roleCode); err != nil {
		fmt.Println("输入角色编码失败", err)
		return
	}

	fmt.Println("请输入权限编码，例如：image:upload")
	if _, err := fmt.Scanln(&permCode); err != nil {
		fmt.Println("输入权限编码失败", err)
		return
	}

	role, err := query.Role.WithContext(ctx).Where(query.Role.Code.Eq(roleCode)).Take()
	if err != nil {
		fmt.Println("角色不存在", err)
		return
	}

	perm, err := query.Permission.WithContext(ctx).Where(query.Permission.Code.Eq(permCode)).Take()
	if err != nil {
		fmt.Println("权限不存在", err)
		return
	}

	_, err = query.RolePermission.WithContext(ctx).
		Where(query.RolePermission.RoleID.Eq(role.ID), query.RolePermission.PermID.Eq(perm.ID)).
		Take()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Println("角色未绑定该权限，无需删除")
		return
	}
	if err != nil {
		fmt.Println("查询角色权限关联失败", err)
		return
	}

	_, err = query.RolePermission.WithContext(ctx).
		Where(query.RolePermission.RoleID.Eq(role.ID), query.RolePermission.PermID.Eq(perm.ID)).
		Delete()
	if err != nil {
		fmt.Println("删除角色权限关联失败", err)
		return
	}

	fmt.Println("删除角色权限关联成功")
}
