package permission_serv

import (
	"context"
	"fast-gin/dal/query"
	"fast-gin/global"
	"fast-gin/permissions"
	"fast-gin/service/redis_serv"
	bitset "fast-gin/utils/bits"
	"fmt"

	"go.uber.org/zap"
)

// LogRolePermissionCache 打印所有角色及其展开后的权限快照。
func LogRolePermissionCache() {
	if global.DB == nil {
		zap.S().Warn("数据库未初始化，跳过角色权限快照打印")
		return
	}

	roles, err := query.Role.WithContext(context.Background()).Order(query.Role.ID).Find()
	if err != nil {
		zap.S().Errorf("打印角色权限快照失败: %v", err)
		return
	}

	for _, role := range roles {
		if role == nil {
			continue
		}

		set, err := GetRoleExpandedPerms(role.ID)
		if err != nil {
			zap.S().Warnf("获取角色权限快照失败 roleID=%d err=%v", role.ID, err)
			continue
		}

		permCodes := make([]string, 0, len(set.Elems()))
		for _, bit := range set.Elems() {
			code, ok := permissions.PermCode[permissions.PermissionBit(bit)]
			if !ok {
				code = fmt.Sprintf("unknown:%d", bit)
			}
			permCodes = append(permCodes, code)
		}

		zap.S().Infof("角色权限快照 roleID=%d name=%s code=%s status=%d perms=%v", role.ID, role.Name, role.Code, role.Status, permCodes)
	}
}

// InitRolePermCache 系统启动时初始化所有角色的展开权限到Redis
func InitRolePermCache() {
	if global.DB == nil {
		zap.S().Warn("数据库未初始化，跳过权限缓存初始化")
		return
	}

	roles, err := query.Role.WithContext(context.Background()).Find()
	if err != nil {
		zap.S().Errorf("查询角色列表失败: %v", err)
		return
	}

	for _, role := range roles {
		if role == nil {
			continue
		}

		if _, err := GetRoleExpandedPerms(role.ID); err != nil {
			zap.S().Warnf("预热角色权限缓存失败 roleID=%d err=%v", role.ID, err)
		}
	}

	LogRolePermissionCache()
	zap.S().Infof("角色权限缓存初始化完成，共处理 %d 个角色", len(roles))
}

// GetRoleExpandedPerms 获取角色的展开权限（包含继承的权限）
// 先查Redis，未命中则动态计算后存入Redis
func GetRoleExpandedPerms(roleID uint) (*bitset.IntSet, error) {
	// 1. 先查Redis缓存
	if set, ok, err := redis_serv.GetRolePermIntSet(roleID); err == nil && ok {
		return set, nil
	}

	// 2. 缓存未命中，执行动态计算
	set, err := calculateRoleExpandedPerms(roleID)
	if err != nil {
		return nil, err
	}

	// 3. 存入Redis缓存（24小时过期）
	if err := redis_serv.SetRolePermIntSet(roleID, set); err != nil {
		zap.S().Warnf("缓存角色权限失败 roleID=%d err=%v", roleID, err)
	}

	return set, nil
}

// calculateRoleExpandedPerms 递归计算角色的所有权限（DFS遍历整个继承链）
// 包括：自己的权限 + 父角色的权限 + 更高层级父角色的权限...
func calculateRoleExpandedPerms(roleID uint) (*bitset.IntSet, error) {
	set := &bitset.IntSet{}
	visited := make(map[uint]bool)

	var dfs func(rid uint) error
	dfs = func(rid uint) error {
		// 避免循环继承
		if visited[rid] {
			return nil
		}
		visited[rid] = true

		// 1. 获取该角色自己的权限
		var codes []string
		err := query.RolePermission.WithContext(context.Background()).
			Distinct(query.Permission.Code).
			Join(query.Permission, query.Permission.ID.EqCol(query.RolePermission.PermID)).
			Where(query.RolePermission.RoleID.Eq(rid)).
			Pluck(query.Permission.Code, &codes)
		if err != nil {
			return err
		}

		for _, code := range codes {
			if bit, ok := permissions.PermBit[code]; ok {
				set.Add(bit)
			}
		}

		// 2. 获取该角色的父角色ID
		var role struct {
			PID *uint
		}
		err = query.Role.WithContext(context.Background()).
			Where(query.Role.ID.Eq(rid)).
			Scan(&role)
		if err != nil {
			return err
		}

		// 3. 递归处理父角色
		if role.PID != nil {
			if err := dfs(*role.PID); err != nil {
				return err
			}
		}

		return nil
	}

	if err := dfs(roleID); err != nil {
		return nil, err
	}

	return set, nil
}

// DelRolePermCache 删除某个角色的权限缓存（权限修改时调用）
func DelRolePermCache(roleID uint) error {
	return redis_serv.DelRolePermIntSet(roleID)
}

func collectRoleAndChildrenIDs(roleID uint) ([]uint, error) {
	visited := make(map[uint]bool)
	ids := make([]uint, 0)

	var dfs func(uint) error
	dfs = func(id uint) error {
		if visited[id] {
			return nil
		}
		visited[id] = true
		ids = append(ids, id)

		var childRoleIDs []uint
		err := query.Role.WithContext(context.Background()).
			Where(query.Role.PID.Eq(id)).
			Pluck(query.Role.ID, &childRoleIDs)
		if err != nil {
			return err
		}

		for _, childID := range childRoleIDs {
			if err := dfs(childID); err != nil {
				return err
			}
		}

		return nil
	}

	if err := dfs(roleID); err != nil {
		return nil, err
	}

	return ids, nil
}

// DelRoleAndChildrenPermCache 删除某角色及其所有子角色的缓存
// 当修改角色继承关系或权限时调用，确保缓存一致性
func DelRoleAndChildrenPermCache(roleID uint) ([]uint, error) {
	if global.Redis == nil {
		return []uint{roleID}, nil
	}

	roleIDs, err := collectRoleAndChildrenIDs(roleID)
	if err != nil {
		return nil, err
	}

	for _, id := range roleIDs {
		if err := DelRolePermCache(id); err != nil {
			return nil, err
		}
	}

	return roleIDs, nil
}

// DelUserAndRolePermCache 删除用户权限缓存以及拥有该角色的所有用户的缓存
// 在修改角色权限时调用，确保下次查询时重新计算
func DelUserAndRolePermCache(roleID uint) error {
	// 1. 删除角色及其子角色的缓存
	affectedRoleIDs, err := DelRoleAndChildrenPermCache(roleID)
	if err != nil {
		return err
	}
	if len(affectedRoleIDs) == 0 {
		return nil
	}

	// 2. 找出所有拥有受影响角色的用户
	var userIDs []uint
	err = query.UserRole.WithContext(context.Background()).
		Distinct(query.UserRole.UserID).
		Where(query.UserRole.RoleID.In(affectedRoleIDs...)).
		Pluck(query.UserRole.UserID, &userIDs)
	if err != nil {
		return err
	}

	// 3. 删除这些用户的权限缓存
	for _, userID := range userIDs {
		if err := redis_serv.DelUserPermIntSet(userID); err != nil {
			zap.S().Warnf("删除用户权限缓存失败 userID=%d roleID=%d err=%v", userID, roleID, err)
		}
	}

	return nil
}
