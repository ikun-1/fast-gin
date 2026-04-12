package permission_serv

import (
	"context"
	"fast-gin/dal/query"
	"fast-gin/global"
	"fast-gin/permissions"
	"fast-gin/service/redis_serv"
	bitset "fast-gin/utils/bits"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LoadUserPerms loads a user's permission bitset from role-permission relations.
func LoadUserPerms(db *gorm.DB, userID uint) (*bitset.IntSet, error) {
	if set, ok, err := redis_serv.GetUserPermIntSet(userID); err == nil && ok {
		return set, nil
	}

	var codes []string
	err := query.UserRole.WithContext(context.Background()).
		Distinct(query.Permission.Code).
		Join(query.RolePermission, query.UserRole.RoleID.EqCol(query.RolePermission.RoleID)).
		Join(query.Permission, query.Permission.ID.EqCol(query.RolePermission.PermID)).
		Where(query.UserRole.UserID.Eq(userID)).
		Pluck(query.Permission.Code, &codes)
	if err != nil {
		return nil, err
	}

	s := &bitset.IntSet{}
	for _, code := range codes {
		if bit, ok := permissions.PermBit[code]; ok {
			s.Add(bit)
		}
	}

	// 打印用户权限码列表，方便调试
	zap.S().Debugf("加载用户权限 userID=%d perms=%v", userID, codes)

	// 将结果缓存到Redis，过期时间与JWT一致
	if err := redis_serv.SetUserPermIntSet(userID, s); err != nil {
		zap.S().Warnf("缓存用户权限IntSet失败 userID=%d err=%v", userID, err)
	}
	return s, nil
}

func HasPermissionBit(db *gorm.DB, userID uint, bit permissions.PermissionBit) (bool, error) {
	set, err := LoadUserPerms(db, userID)
	if err != nil {
		return false, err
	}
	return set.Has(bit), nil
}

func WarmUserPerms(userID uint) {
	if _, err := LoadUserPerms(global.DB, userID); err != nil {
		zap.S().Warnf("预热用户权限缓存失败 userID=%d err=%v", userID, err)
	}
}
