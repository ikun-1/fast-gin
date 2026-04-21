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
	g.POST("auth/login",
		middleware.LimitMiddleware(2),
		middleware.ShouldBindJSON[user.LoginRequest],
		User.LoginView)
	g.POST("auth/logout",
		middleware.AuthMiddleware,
		User.LogoutView)

	g.GET("users",
		middleware.LimitMiddleware(10),
		// middleware.AuthMiddleware,
		middleware.ShouldBindQuery[models.PageInfo],
		User.UserListView)
	g.GET("users/:id",
		middleware.AuthMiddleware,
		middleware.ShouldBindUri[models.BindId],
		User.GetUserView)
	g.POST("users",
		middleware.AdminMiddleware,
		middleware.ShouldBindJSON[user.CreateUserRequest],
		User.CreateUserView)
	g.PUT("users/:id",
		middleware.ShouldBindUri[models.BindId],
		middleware.SelfOrAdminMiddleware,
		middleware.ShouldBindJSON[user.UpdateUserRequest],
		User.UpdateUserView)
	g.DELETE("users/:id",
		middleware.AdminMiddleware,
		middleware.ShouldBindUri[models.BindId],
		User.DeleteUserView)
}
