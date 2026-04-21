package routers

import (
	"fast-gin/handlers"
	"fast-gin/handlers/rbac"
	"fast-gin/middleware"
	"fast-gin/models"

	"github.com/gin-gonic/gin"
)

func RBACRouter(g *gin.RouterGroup) {
	RBAC := handlers.Handlers.RBAC

	r := g.Group("rbac")
	r.Use(middleware.AdminMiddleware)

	r.GET("roles", middleware.ShouldBindQuery[models.PageInfo], RBAC.ListRoles)
	r.GET("roles/:id", middleware.ShouldBindUri[models.BindId], RBAC.GetRole)
	r.POST("roles", middleware.ShouldBindJSON[rbac.CreateRoleRequest], RBAC.CreateRole)
	r.PUT("roles/:id", middleware.ShouldBindUri[models.BindId], middleware.ShouldBindJSON[rbac.UpdateRoleRequest], RBAC.UpdateRole)
	r.DELETE("roles/:id", middleware.ShouldBindUri[models.BindId], RBAC.DeleteRole)
	r.POST("permission-cache/rewarm", RBAC.RewarmPermissionCache)

	r.GET("permissions", middleware.ShouldBindQuery[models.PageInfo], RBAC.ListPermissions)
	r.GET("permissions/:id", middleware.ShouldBindUri[models.BindId], RBAC.GetPermission)
	r.POST("permissions", middleware.ShouldBindJSON[rbac.CreatePermissionRequest], RBAC.CreatePermission)
	r.PUT("permissions/:id", middleware.ShouldBindUri[models.BindId], middleware.ShouldBindJSON[rbac.UpdatePermissionRequest], RBAC.UpdatePermission)
	r.DELETE("permissions/:id", middleware.ShouldBindUri[models.BindId], RBAC.DeletePermission)

	r.GET("roles/:id/permissions", middleware.ShouldBindUri[models.BindId], middleware.ShouldBindQuery[models.PageInfo], RBAC.ListRolePermissions)
	r.POST("roles/:id/permissions", middleware.ShouldBindUri[models.BindId], middleware.ShouldBindJSON[rbac.AttachPermissionRequest], RBAC.AttachRolePermission)
	r.DELETE("roles/:id/permissions/:permID", middleware.ShouldBindUri[rbac.RolePermUri], RBAC.DetachRolePermission)

	r.GET("users/:id/roles", middleware.ShouldBindUri[models.BindId], middleware.ShouldBindQuery[models.PageInfo], RBAC.ListUserRoles)
	r.POST("users/:id/roles", middleware.ShouldBindUri[models.BindId], middleware.ShouldBindJSON[rbac.AttachRoleRequest], RBAC.AttachUserRole)
	r.DELETE("users/:id/roles/:roleID", middleware.ShouldBindUri[rbac.UserRoleUri], RBAC.DetachUserRole)
}
