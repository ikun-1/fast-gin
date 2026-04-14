package redis_serv

import (
	"context"
	"fast-gin/global"
	bitset "fast-gin/utils/bits"
	"fmt"
	"time"
)

const userPermIntSetCachePrefix = "perm:intset:user:"
const rolePermIntSetCachePrefix = "perm:role:"

func userPermIntSetCacheKey(userID uint) string {
	return fmt.Sprintf("%s%d", userPermIntSetCachePrefix, userID)
}

func rolePermIntSetCacheKey(roleID uint) string {
	return fmt.Sprintf("%s%d", rolePermIntSetCachePrefix, roleID)
}

func GetUserPermIntSet(userID uint) (*bitset.IntSet, bool, error) {
	if global.Redis == nil {
		return nil, false, nil
	}

	key := userPermIntSetCacheKey(userID)
	data, err := global.Redis.Get(context.Background(), key).Bytes()
	if err != nil {
		return nil, false, nil
	}

	set := &bitset.IntSet{}
	if err := set.UnmarshalBinary(data); err != nil {
		return nil, false, err
	}
	return set, true, nil
}

func SetUserPermIntSet(userID uint, set *bitset.IntSet) error {
	if global.Redis == nil || set == nil {
		return nil
	}

	data, err := set.MarshalBinary()
	if err != nil {
		return err
	}

	ttl := time.Duration(global.Config.Jwt.Expires) * time.Minute
	if ttl <= 0 {
		ttl = time.Hour
	}

	key := userPermIntSetCacheKey(userID)
	return global.Redis.Set(context.Background(), key, data, ttl).Err()
}

func DelUserPermIntSet(userID uint) error {
	if global.Redis == nil {
		return nil
	}

	key := userPermIntSetCacheKey(userID)
	return global.Redis.Del(context.Background(), key).Err()
}

func GetRolePermIntSet(roleID uint) (*bitset.IntSet, bool, error) { // 获取角色权限缓存，返回 (权限集合, 是否命中, 错误)
	if global.Redis == nil {
		return nil, false, nil
	}

	key := rolePermIntSetCacheKey(roleID)
	data, err := global.Redis.Get(context.Background(), key).Bytes()
	if err != nil {
		return nil, false, nil
	}

	set := &bitset.IntSet{}
	if err := set.UnmarshalBinary(data); err != nil {
		return nil, false, err
	}
	return set, true, nil
}

func SetRolePermIntSet(roleID uint, set *bitset.IntSet) error { // 设置角色权限缓存，过期时间固定为24小时
	if global.Redis == nil || set == nil {
		return nil
	}

	data, err := set.MarshalBinary()
	if err != nil {
		return err
	}

	key := rolePermIntSetCacheKey(roleID)
	return global.Redis.Set(context.Background(), key, data, 24*time.Hour).Err()
}

func DelRolePermIntSet(roleID uint) error { // 删除角色权限缓存
	if global.Redis == nil {
		return nil
	}

	key := rolePermIntSetCacheKey(roleID)
	return global.Redis.Del(context.Background(), key).Err()
}


// 清除所有用户权限缓存和角色权限缓存
func ClearAllPermCaches() error {
	if global.Redis == nil {
		return nil
	}
	ctx := context.Background()

	// 删除用户权限缓存
	userKeys, err := global.Redis.Keys(ctx, userPermIntSetCachePrefix+"*").Result()
	if err != nil {
		return err
	}
	if len(userKeys) > 0 {
		if err := global.Redis.Del(ctx, userKeys...).Err(); err != nil {
			return err
		}
	}
	
	// 删除角色权限缓存
	roleKeys, err := global.Redis.Keys(ctx, rolePermIntSetCachePrefix+"*").Result()
	if err != nil {
		return err
	}
	if len(roleKeys) > 0 {
		if err := global.Redis.Del(ctx, roleKeys...).Err(); err != nil {
			return err
		}
	}

	return nil
}
