package middleware

import (
	"fast-gin/global"
	"fast-gin/permissions"
	"fast-gin/service/permission_serv"
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
)

func PermissionMiddleware(permissionBit permissions.PermissionBit) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetAuth(c)
		if claims == nil || claims.UserID == 0 {
			res.FailAuth(c)
			c.Abort()
			return
		}

		if claims.IsAdmin {
			c.Next()
			return
		}

		ok, err := permission_serv.HasPermissionBit(global.DB, claims.UserID, permissionBit)
		if err != nil || !ok {
			res.FailPermission(c)
			c.Abort()
			return
		}

		c.Next()
	}
}
