package redis_serv

import (
	"context"
	"fast-gin/global"
	"fast-gin/utils/jwts"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// 添加一个prefix，避免与其他缓存数据冲突
const logoutCachePrefix = "logout_"

func Logout(token string) {
	claims, err := jwts.CheckToken(token)
	if err != nil {
		return
	}
	if err := DelUserPermIntSet(claims.UserID); err != nil {
		zap.S().Warnf("删除用户权限缓存失败 userID=%d err=%v", claims.UserID, err)
	}
	key := fmt.Sprintf("%s%s", logoutCachePrefix, token)
	sub := time.Until(claims.ExpiresAt.Time)

	_, err = global.Redis.Set(context.Background(), key, "", sub).Result()
	if err != nil {
		zap.S().Error(err)
	}
}

func HasLogout(token string) (ok bool) {
	key := fmt.Sprintf("%s%s", logoutCachePrefix, token)
	_, err := global.Redis.Get(context.Background(), key).Result()
	if err == nil {
		return true
	}
	return false
}
