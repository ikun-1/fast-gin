package redis_serv

import (
	"context"
	"fast-gin/global"
	"fast-gin/utils/jwts"
	"fmt"
	"time"

	"go.uber.org/zap"
)

func Logout(token string) {
	claims, err := jwts.CheckToken(token)
	if err != nil {
		return
	}
	key := fmt.Sprintf("logout_%s", token)
	sub := time.Until(claims.ExpiresAt.Time)

	_, err = global.Redis.Set(context.Background(), key, "", sub).Result()
	if err != nil {
		zap.S().Error(err)
	}
}

func HasLogout(token string) (ok bool) {
	key := fmt.Sprintf("logout_%s", token)
	_, err := global.Redis.Get(context.Background(), key).Result()
	if err == nil {
		return true
	}
	return false
}
