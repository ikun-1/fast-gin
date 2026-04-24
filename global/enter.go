package global

import (
	"fast-gin/config"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const Version = "1.0.0"

var (
	Config *config.Config
	DB     *gorm.DB
	Redis  *redis.Client
)
