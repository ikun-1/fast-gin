package permission_serv

import (
	"context"
	"fast-gin/dal/query"
	"fast-gin/global"
	"fast-gin/permissions"
	"fast-gin/service/redis_serv"
	bitset "fast-gin/utils/bits"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func formatPermSet(set *bitset.IntSet) string {
	if set == nil {
		return "[]"
	}

	bits := set.Elems()
	labels := make([]string, 0, len(bits))
	for _, bit := range bits {
		if code, ok := permissions.PermCode[permissions.PermissionBit(bit)]; ok {
			labels = append(labels, code)
			continue
		}
		labels = append(labels, fmt.Sprintf("bit:%d", bit))
	}

	sort.Strings(labels)
	return "[" + strings.Join(labels, " ") + "]"
}

// LoadUserPerms 加载用户权限：先查Redis用户缓存，未命中则从角色展开权限合并
// 流程：查询用户角色 → 从Redis获取各角色展开权限 → 合并得到用户权限 → 缓存用户权限
func LoadUserPerms(db *gorm.DB, userID uint) (*bitset.IntSet, error) {
	// 1. 优先查询用户权限缓存
	if set, ok, err := redis_serv.GetUserPermIntSet(userID); err == nil && ok {
		return set, nil
	}

	// 2. 获取用户拥有的所有角色ID
	var roleIDs []uint
	err := query.UserRole.WithContext(context.Background()).
		Where(query.UserRole.UserID.Eq(userID)).
		Pluck(query.UserRole.RoleID, &roleIDs)
	if err != nil {
		return nil, err
	}
	// 打印调试日志，显示用户拥有的角色ID列表
	zap.S().Debugf("加载用户角色 userID=%d roleIDs=%v", userID, roleIDs)

	// 3. 合并所有角色的展开权限
	mergedSet := &bitset.IntSet{}
	for _, roleID := range roleIDs {
		rolePerms, err := GetRoleExpandedPerms(roleID)
		if err != nil {
			zap.S().Warnf("获取角色权限失败 userID=%d roleID=%d err=%v", userID, roleID, err)
			continue
		}
		mergedSet.UnionWith(rolePerms)
	}

	// 4. 缓存用户权限（与JWT过期时间一致，通常为1小时）
	if err := redis_serv.SetUserPermIntSet(userID, mergedSet); err != nil {
		zap.S().Warnf("缓存用户权限失败 userID=%d err=%v", userID, err)
	}

	zap.S().Debugf("加载用户权限 userID=%d perms=%s", userID, formatPermSet(mergedSet))

	return mergedSet, nil
}

func HasPermissionBit(db *gorm.DB, userID uint, bit permissions.PermissionBit) (bool, error) {
	set, err := LoadUserPerms(db, userID)
	if err != nil {
		return false, err
	}
	permCode, ok := permissions.PermCode[bit]
	if !ok {
		permCode = "unknown"
	}
	result := set.Has(bit)
	zap.S().Debugf("检查用户权限 userID=%d checkPerm=%s(%d) userPerms=%s result=%t", userID, permCode, bit, formatPermSet(set), result)
	return result, nil
}

func WarmUserPerms(userID uint) {
	if _, err := LoadUserPerms(global.DB, userID); err != nil {
		zap.S().Warnf("预热用户权限缓存失败 userID=%d err=%v", userID, err)
	}
}
