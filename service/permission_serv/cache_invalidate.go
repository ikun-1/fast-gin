package permission_serv

import (
	"fast-gin/service/redis_serv"

	"go.uber.org/zap"
)

// CacheInvalidateOps 缓存失效操作

// OnRolePermissionChanged 当修改角色权限关系时调用
// 会清除该角色及其所有子角色的缓存，以及拥有这些角色的所有用户的缓存
// 调用示例：
//
//	err := permission_serv.OnRolePermissionChanged(roleID)
func OnRolePermissionChanged(roleID uint) error {
	if err := DelUserAndRolePermCache(roleID); err != nil {
		zap.S().Errorf("清除角色权限缓存失败 roleID=%d err=%v", roleID, err)
		return err
	}
	zap.S().Infof("已清除角色权限缓存 roleID=%d", roleID)
	return nil
}

// OnRoleInheritanceChanged 当修改角色继承关系时调用
// 例如修改了某角色的父角色ID
// 会清除该角色及其所有子角色的缓存，以及相关用户的缓存
// 调用示例：
//
//	err := permission_serv.OnRoleInheritanceChanged(roleID)
func OnRoleInheritanceChanged(roleID uint) error {
	if err := DelUserAndRolePermCache(roleID); err != nil {
		zap.S().Errorf("清除角色继承缓存失败 roleID=%d err=%v", roleID, err)
		return err
	}
	zap.S().Infof("已清除角色继承缓存 roleID=%d", roleID)
	return nil
}

// OnUserRoleChanged 当修改用户角色关系时调用
// 例如给用户添加或移除某个角色
// 调用示例：
//
//	err := permission_serv.OnUserRoleChanged(userID)
func OnUserRoleChanged(userID uint) error {
	if err := redis_serv.DelUserPermIntSet(userID); err != nil {
		zap.S().Errorf("清除用户权限缓存失败 userID=%d err=%v", userID, err)
		return err
	}
	zap.S().Infof("已清除用户权限缓存 userID=%d", userID)
	return nil
}

// RewarmAllPermCache 重新预热所有权限缓存
// 用于权限数据被外部修改（如数据库直接修改）后的恢复
func RewarmAllPermCache() {

	err := redis_serv.ClearAllPermCaches()
	if err != nil {
		zap.S().Errorf("清除权限缓存失败: %v", err)
		return
	}
	zap.S().Info("已清除所有权限缓存")

	// 重新初始化角色权限缓存
	InitRolePermCache()
}
