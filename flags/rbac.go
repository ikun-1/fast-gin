package flags

import (
	"context"
	"errors"
	"fast-gin/dal/query"
	"fast-gin/models"
	"fast-gin/permissions"
	"fmt"
	"sort"

	"go.uber.org/zap"
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

// InitRBAC 非交互式初始化默认 RBAC 数据（角色、权限、关联）
// 在部署时由 docker-entrypoint.sh 调用
func (RBAC) InitRBAC() {
	ctx := context.Background()

	// 1. 创建/获取默认角色
	roleNames := map[string]string{
		"admin": "系统管理员",
		"user":  "普通用户",
	}
	roleIDs := make(map[string]uint)

	for code, name := range roleNames {
		role, err := query.Role.WithContext(ctx).
			Where(query.Role.Code.Eq(code)).
			Take()
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				zap.S().Errorf("查询角色[%s]失败: %v", code, err)
				continue
			}
			// 角色不存在，创建
			role = &models.Role{
				Name:        name,
				Code:        code,
				Description: "自动初始化",
				Status:      1,
			}
			if err := query.Role.WithContext(ctx).Create(role); err != nil {
				zap.S().Errorf("创建角色[%s]失败: %v", code, err)
				continue
			}
			zap.S().Infof("创建默认角色成功 code=%s name=%s id=%d", code, name, role.ID)
		}
		roleIDs[code] = role.ID
	}

	// 2. 遍历已注册的权限代码，创建权限记录
	// PermCode 在 init() 阶段已从各子包收集完毕
	type permEntry struct {
		code string
		bit  permissions.PermissionBit
	}
	var permEntries []permEntry
	for bit, code := range permissions.PermCode {
		permEntries = append(permEntries, permEntry{code: code, bit: bit})
	}
	sort.Slice(permEntries, func(i, j int) bool {
		return permEntries[i].bit < permEntries[j].bit
	})

	permIDs := make(map[permissions.PermissionBit]uint)
	for _, entry := range permEntries {
		perm, err := query.Permission.WithContext(ctx).
			Where(query.Permission.Code.Eq(entry.code)).
			Take()
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				zap.S().Errorf("查询权限[%s]失败: %v", entry.code, err)
				continue
			}
			perm = &models.Permission{
				Code:   entry.code,
				Name:   entry.code,
				Module: "system",
				Type:   3, // 按钮权限
			}
			if err := query.Permission.WithContext(ctx).Create(perm); err != nil {
				zap.S().Errorf("创建权限[%s]失败: %v", entry.code, err)
				continue
			}
			zap.S().Infof("创建默认权限成功 code=%s id=%d", entry.code, perm.ID)
		}
		permIDs[entry.bit] = perm.ID
	}

	// 3. 为 admin 角色绑定所有权限
	if adminRoleID, ok := roleIDs["admin"]; ok {
		for _, entry := range permEntries {
			permID, ok := permIDs[entry.bit]
			if !ok {
				continue
			}
			_, err := query.RolePermission.WithContext(ctx).
				Where(query.RolePermission.RoleID.Eq(adminRoleID), query.RolePermission.PermID.Eq(permID)).
				Take()
			if err == nil {
				continue // 已绑定
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			_ = query.RolePermission.WithContext(ctx).Create(&models.RolePermission{
				RoleID: adminRoleID,
				PermID: permID,
			})
		}
		zap.S().Infof("admin 角色权限绑定完成, 共 %d 个权限", len(permEntries))
	}

	// 4. 为 user 角色绑定基础权限（image:upload, image:delete）
	if userRoleID, ok := roleIDs["user"]; ok {
		userPermCodes := []string{"image:upload", "image:delete"}
		for _, code := range userPermCodes {
			perm, err := query.Permission.WithContext(ctx).
				Where(query.Permission.Code.Eq(code)).
				Take()
			if err != nil {
				zap.S().Warnf("权限[%s]不存在，跳过 user 角色绑定", code)
				continue
			}
			_, err = query.RolePermission.WithContext(ctx).
				Where(query.RolePermission.RoleID.Eq(userRoleID), query.RolePermission.PermID.Eq(perm.ID)).
				Take()
			if err == nil {
				continue
			}
			_ = query.RolePermission.WithContext(ctx).Create(&models.RolePermission{
				RoleID: userRoleID,
				PermID: perm.ID,
			})
		}
		zap.S().Infof("user 角色基础权限绑定完成")
	}

	zap.S().Info("RBAC 初始化完成")
}
