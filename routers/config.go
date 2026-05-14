package routers

import (
	"fast-gin/handlers"

	"github.com/gin-gonic/gin"
)

func ConfigRouter(g *gin.RouterGroup) {
	h := handlers.Handlers.Config

	g.GET("ice-servers", h.GetIceServers)
}
