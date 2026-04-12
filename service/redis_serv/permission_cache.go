package redis_serv

import (
	"context"
	"fast-gin/global"
	bitset "fast-gin/utils/bits"
	"fmt"
	"time"
)

const userPermIntSetCachePrefix = "perm:intset:user:"

func userPermIntSetCacheKey(userID uint) string {
	return fmt.Sprintf("%s%d", userPermIntSetCachePrefix, userID)
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
