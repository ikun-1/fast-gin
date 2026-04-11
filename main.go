package main

import (
	"fast-gin/core"
	"fast-gin/flags"
	"fast-gin/global"
	"fast-gin/routers"
	"fast-gin/service/cron_serv"
	"fast-gin/utils/validate"
)

// @title           Fast-Gin API
// @version         1.0.0
// @description     Fast-Gin 后端服务 API 文档
// @host            0.0.0.0:8080
// @basePath        /api
// @schemes         http https
// @securityDefinitions.apikey Bearer
// @in              header
// @name            token
// @description     Enter a valid token with the Bearer scheme
func main() {
	core.InitLogger()
	flags.Parse()
	global.Config = core.ReadConfig()
	global.DB = core.InitGorm()
	global.Redis = core.InitRedis()
	flags.Run()

	validate.InitValidator()
	cron_serv.InitCron()

	routers.Run()
}
