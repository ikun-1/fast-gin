package routers

import (
	"fast-gin/global"

	_ "fast-gin/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

func Run() {
	gin.SetMode(global.Config.System.Mode)
	r := gin.Default()
	r.Static("/uploads", "uploads")

	if global.Config.System.Swagger.Enabled {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		zap.S().Infof("Swagger 已启用，访问地址: http://%s/swagger/index.html", global.Config.System.Addr())
	}

	g := r.Group("api")

	UserRouter(g)
	CaptchaRouter(g)
	ImageRouter(g)
	RBACRouter(g)
	RoomRouter(g)
	SignalRouter(g)

	addr := global.Config.System.Addr()
	if global.Config.System.Mode == "release" {
		zap.S().Infof("后端服务运行在 %s", addr)
	}

	r.Run(addr)
}
