package core

import (
	"fast-gin/config"
	"fast-gin/flags"
	"fast-gin/global"
	"os"

	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"
)

func ReadConfig() (cfg *config.Config) {
	cfg = new(config.Config)
	byteData, err := os.ReadFile(flags.Options.File)
	if err != nil {
		zap.S().Fatalf("配置文件读取错误 %v", err)
		return
	}
	err = yaml.Unmarshal(byteData, cfg)
	if err != nil {
		zap.S().Fatalf("配置文件格式错误 %v", err)
		return
	}
	zap.S().Infof("%s 配置文件读取成功", flags.Options.File)
	return
}

func DumpConfig() {
	byteData, err := yaml.Marshal(global.Config)
	if err != nil {
		zap.S().Errorf("配置文件转换错误 %v", err)
		return
	}
	err = os.WriteFile(flags.Options.File, byteData, 0666)
	if err != nil {
		zap.S().Errorf("配置文件写入错误 %v", err)
		return
	}
	zap.S().Infof("%s 配置文件写入成功", flags.Options.File)
}
