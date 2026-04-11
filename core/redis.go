package core

import (
	"context"
	"fast-gin/global"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func InitRedis() (client *redis.Client) {
	cfg := global.Config.Redis
	if cfg.Host == "" {
		zap.L().Fatal("未配置redis连接地址")
		return
	}
	if cfg.Port == 0 {
		zap.L().Fatal("未配置redis连接端口")
		return
	}
	client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		zap.S().Fatal("连接redis失败 %s", err)
		return
	}
	zap.L().Info("成功连接redis")
	return
}
