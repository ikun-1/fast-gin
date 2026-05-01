package routers

import (
	"fast-gin/handlers"
	"fast-gin/middleware"
	"fast-gin/models"

	"github.com/gin-gonic/gin"
)

func StatsRouter(g *gin.RouterGroup) {
	h := handlers.Handlers.Stats

	g.GET("stats/overview",
		middleware.AuthMiddleware,
		h.OverviewView)

	g.GET("stats/meetings/:id",
		middleware.AuthMiddleware,
		middleware.ShouldBindUri[models.BindId],
		h.MeetingStatsView)

	g.GET("stats/users/:id",
		middleware.AuthMiddleware,
		middleware.ShouldBindUri[models.BindId],
		h.UserStatsView)

	g.GET("stats/trend",
		middleware.AuthMiddleware,
		h.TrendView)

	g.GET("stats/quality/:id",
		middleware.AuthMiddleware,
		middleware.ShouldBindUri[models.BindId],
		h.QualityReportView)
}
