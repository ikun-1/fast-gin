package user

import (
	"fast-gin/global"
	"fast-gin/service/redis_serv"
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
)

// LogoutView 用户注销
// @Summary      用户注销
// @Description  注销用户登录状态，token将被加入黑名单
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  res.Response  "{"code":0,"msg":"注销成功"}"
// @Failure      200  {object}  res.Response  "{"code":3,"msg":"认证失败"}"
// @Router       /user/logout [post]
func (User) LogoutView(c *gin.Context) {
	token := c.GetHeader("token")
	if global.Redis == nil {
		res.OkWithMsg(c, "注销成功")
		return
	}
	redis_serv.Logout(token)
	res.OkWithMsg(c, "注销成功")
}
