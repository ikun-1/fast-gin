package middleware

import (
	"fast-gin/models"
	"fast-gin/service/redis_serv"
	"fast-gin/utils/jwts"
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func getAuthorizationToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return c.GetHeader("token")
	}
	const bearerPrefix = "Bearer "
	if len(authHeader) > len(bearerPrefix) && authHeader[:len(bearerPrefix)] == bearerPrefix {
		return authHeader[len(bearerPrefix):]
	}
	return authHeader
}

func AuthMiddleware(c *gin.Context) {
	token := getAuthorizationToken(c)
	claims, err := jwts.CheckToken(token)
	if err != nil {
		res.FailAuth(c)
		c.Abort()
		return
	}
	if redis_serv.HasLogout(token) {
		zap.S().Infof("用户已注销 token=%s userID=%d", token, claims.UserID)
		res.FailNotLogin(c)
		c.Abort()
		return
	}
	c.Set("claims", claims)
	c.Next()
}

func AdminMiddleware(c *gin.Context) {
	token := getAuthorizationToken(c)
	claims, err := jwts.CheckToken(token)
	if err != nil {
		res.FailAuth(c)
		c.Abort()
		return
	}
	if redis_serv.HasLogout(token) {
		res.FailNotLogin(c)
		c.Abort()
		return
	}
	c.Set("claims", claims)
	c.Next()
}

// SelfOrAdminMiddleware allows access when requester is admin or matches URI user id.
func SelfOrAdminMiddleware(c *gin.Context) {
	token := getAuthorizationToken(c)
	claims, err := jwts.CheckToken(token)
	if err != nil {
		res.FailAuth(c)
		c.Abort()
		return
	}
	if redis_serv.HasLogout(token) {
		res.FailNotLogin(c)
		c.Abort()
		return
	}

	if claims.IsAdmin {
		c.Set("claims", claims)
		c.Next()
		return
	}

	uri := GetUri[models.BindId](c)

	if claims.UserID != uri.ID {
		res.FailPermission(c)
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
