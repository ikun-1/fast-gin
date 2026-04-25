package routers

import (
	"fast-gin/handlers"
	"fast-gin/middleware"
	"fast-gin/models"

	"github.com/gin-gonic/gin"
)

func RecordingRouter(g *gin.RouterGroup) {
	h := handlers.Handlers.Recording

	g.GET("recordings",
		middleware.AuthMiddleware,
		middleware.ShouldBindQuery[models.PageInfo],
		h.ListView)
	g.GET("recordings/:id",
		middleware.AuthMiddleware,
		middleware.ShouldBindUri[models.BindId],
		h.DetailView)
	g.GET("recordings/:id/files/:fileId/download",
		middleware.AuthMiddleware,
		middleware.ShouldBindUri[models.BindFileId],
		h.FileDownloadView)
	g.GET("recordings/:id/files/:fileId/play",
		middleware.AuthMiddleware,
		middleware.ShouldBindUri[models.BindFileId],
		h.FilePlayView)
	g.DELETE("recordings/:id",
		middleware.AuthMiddleware,
		middleware.ShouldBindUri[models.BindId],
		h.DeleteView)
}
