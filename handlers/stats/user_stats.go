package stats

import (
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/service/stats_serv"
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
)

func (Stats) UserStatsView(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)

	stats, err := stats_serv.GetUserStats(uri.ID)
	if err != nil {
		res.FailNotFound(c)
		return
	}
	res.OkWithData(c, stats)
}
