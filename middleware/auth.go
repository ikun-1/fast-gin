package middleware

import (
	"fast-gin/service/redis_serv"
	"fast-gin/utils/jwts"
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(c *gin.Context) {
	token := c.GetHeader("token")
	claims, err := jwts.CheckToken(token)
	if err != nil {
		res.FailWithMsg(c, "认证失败")
		c.Abort()
		return
	}
	if redis_serv.HasLogout(token) {
		res.FailWithMsg(c, "当前登录已注销")
		c.Abort()
		return
	}
	c.Set("claims", claims)
	c.Next()
}

func AdminMiddleware(c *gin.Context) {
	token := c.GetHeader("token")
	claims, err := jwts.CheckToken(token)
	if err != nil {
		res.FailWithMsg(c, "认证失败")
		c.Abort()
		return
	}
	if redis_serv.HasLogout(token) {
		res.FailWithMsg(c, "当前登录已注销")
		c.Abort()
		return
	}
	if claims.RoleID != 1 {
		res.FailWithMsg(c, "角色认证失败")
		c.Abort()
		return
	}
	c.Set("claims", claims)
	c.Next()
}

func GetAuth(c *gin.Context) (cl *jwts.MyClaims) {
	cl = new(jwts.MyClaims)
	_claims, ok := c.Get("claims")
	if !ok {
		return
	}
	cl, ok = _claims.(*jwts.MyClaims)
	return
}
