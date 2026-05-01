package stats

import (
	"fast-gin/service/stats_serv"
	"fast-gin/utils/res"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (Stats) TrendView(c *gin.Context) {
	days := 7
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 90 {
			days = parsed
		}
	}

	stats, err := stats_serv.GetTrendStats(days)
	if err != nil {
		res.FailWithMsg(c, "获取趋势数据失败")
		return
	}
	res.OkWithData(c, stats)
}
