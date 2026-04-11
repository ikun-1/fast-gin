package user

import (
	"fast-gin/global"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/utils/jwts"
	"fast-gin/utils/pwd"
	"fast-gin/utils/res"
	"fast-gin/views/captcha"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type LoginRequest struct {
	Username    string `json:"username" form:"username" binding:"required" display:"用户名"`
	Password    string `json:"password" form:"password" binding:"required" display:"密码"`
	RePassword  string `json:"rePassword" form:"rePassword" binding:"eqfield=Password,required" display:"确认密码"`
	CaptchaID   string `json:"captchaID"`
	CaptchaCode string `json:"captchaCode"`
}

// LoginView 用户登录
// @Summary      用户登录
// @Description  用户账号登录，支持可选的图片验证码
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        user  body  LoginRequest  true  "登录参数"
// @Success      200  {object}  res.Response  "{"code":0,"msg":"success","data":"token_string"}"
// @Failure      200  {object}  res.Response       "{"code":1,"msg":"invalid parameters","data":{"rePassword":"确认密码必须等于密码"}}"
// @Router       /user/login [post]
func (User) LoginView(c *gin.Context) {
	cr := middleware.GetBind[LoginRequest](c)

	if global.Config.Site.Login.Captcha {
		if cr.CaptchaID == "" || cr.CaptchaCode == "" {
			res.FailWithMsg(c, "请输入图片验证码")
			return
		}
		if !captcha.CaptchaStore.Verify(cr.CaptchaID, cr.CaptchaCode, true) {
			res.FailWithMsg(c, "图片验证码验证失败")
			return
		}
	}

	var user models.UserModel
	err := global.DB.Take(&user, "username = ?", cr.Username).Error
	if err != nil {
		res.FailWithMsg(c, "用户名或密码错误")
		return
	}

	if !pwd.CompareHashAndPassword(user.Password, cr.Password) {
		res.FailWithMsg(c, "用户名或密码错误")
		return
	}

	token, err := jwts.SetToken(jwts.Claims{
		UserID: user.ID,
		RoleID: user.RoleID,
	})
	if err != nil {
		zap.S().Errorf("生成token失败 %s", err)
		res.FailWithMsg(c, "登录失败")
		return
	}

	res.OkWithData(c, token)
}
