package flags

import (
	"fast-gin/global"
	"fast-gin/models"

	"go.uber.org/zap"
)

func MigrateDB() {
	err := global.DB.AutoMigrate(
		&models.UserModel{},
		&models.ImageModel{},
	)
	if err != nil {
		zap.S().Errorf("表结构迁移失败 %s", err)
		return
	}
	zap.S().Infof("表结构迁移成功")
}
