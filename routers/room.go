package routers

import (
	"fast-gin/handlers"
	"fast-gin/handlers/room"
	"fast-gin/middleware"
	"fast-gin/models"

	"github.com/gin-gonic/gin"
)

func RoomRouter(g *gin.RouterGroup) {
	Room := handlers.Handlers.Room
	g.POST("rooms", middleware.AuthMiddleware, middleware.ShouldBindJSON[room.CreateRoomRequest], Room.CreateRoomView)
	g.POST("rooms/join", middleware.AuthMiddleware, middleware.ShouldBindJSON[room.JoinRoomRequest], Room.JoinRoomView)
	g.GET("rooms/:id", middleware.AuthMiddleware, middleware.ShouldBindUri[models.BindId], Room.RoomDetailView)
	g.GET("rooms/:id/members", middleware.AuthMiddleware, middleware.ShouldBindUri[models.BindId], Room.RoomMembersView)
}
