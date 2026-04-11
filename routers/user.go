package routers

import (
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/views"
	"fast-gin/views/user"

	"github.com/gin-gonic/gin"
)

func UserRouter(g *gin.RouterGroup) {
	User := views.Handlers.User
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
