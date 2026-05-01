package stats

import (
	"fast-gin/service/stats_serv"
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
)

func (Stats) OverviewView(c *gin.Context) {
	stats, err := stats_serv.GetOverviewStats()
	if err != nil {
		res.FailWithMsg(c, "获取统计失败")
		return
	}
	res.OkWithData(c, stats)
}
