package routers

import (
	"fast-gin/handlers"
	"fast-gin/models"

	"github.com/gin-gonic/gin"
	"fast-gin/middleware"
	"fast-gin/handlers/meeting"
)

func MeetingRouter(g *gin.RouterGroup) {
	h := handlers.Handlers.Meeting

	g.POST("meetings",
		middleware.AuthMiddleware,
		middleware.ShouldBindJSON[meeting.CreateMeetingRequest],
		h.CreateView)
	g.GET("meetings/:roomNo",
		middleware.ShouldBindUri[models.BindRoomNo],
		h.InfoView)
	g.POST("meetings/:roomNo/join",
		middleware.AuthMiddleware,
		middleware.ShouldBindJSON[meeting.JoinMeetingRequest],
		middleware.ShouldBindUri[models.BindRoomNo],
		h.JoinView)
	g.DELETE("meetings/:roomNo",
		middleware.AuthMiddleware,
		middleware.ShouldBindUri[models.BindRoomNo],
		h.EndView)
	g.GET("meetings",
		middleware.AuthMiddleware,
		middleware.ShouldBindQuery[models.PageInfo],
		h.ListView)
}
