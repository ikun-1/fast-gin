package routers

import (
	"fast-gin/handlers"
	"fast-gin/handlers/user"
	"fast-gin/middleware"
	"fast-gin/models"

	"github.com/gin-gonic/gin"
)

func UserRouter(g *gin.RouterGroup) {
	User := handlers.Handlers.User
	g.POST("user/login",
		middleware.LimitMiddleware(2),
		middleware.ShouldBind[user.LoginRequest],
		User.LoginView)
	g.POST("user/list",
		middleware.LimitMiddleware(10),
		middleware.AuthMiddleware,
		middleware.ShouldBind[models.PageInfo],
		User.UserListView)
	g.POST("user/logout",
		middleware.AuthMiddleware,
		User.LogoutView)
}
