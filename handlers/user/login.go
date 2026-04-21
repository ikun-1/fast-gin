package user

import (
	"errors"
	"fast-gin/dal/query"
	"fast-gin/global"
	"fast-gin/handlers/captcha"
	"fast-gin/middleware"
	"fast-gin/permissions"
	"fast-gin/service/permission_serv"
	"fast-gin/utils/jwts"
	"fast-gin/utils/pwd"
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type LoginRequest struct {
	Username    string `json:"username" form:"username" binding:"required" display:"用户名" example:"admin"`
	Password    string `json:"password" form:"password" binding:"required" display:"密码" example:"123456"`
	CaptchaID   string `json:"captchaId" form:"captchaId" example:"captcha-id-123"`
	CaptchaCode string `json:"captchaCode" form:"captchaCode" example:"1234"`
}

type LoginUserVO struct {
	ID       uint     `json:"id"`
	Username string   `json:"username"`
	Nickname string   `json:"nickname,omitempty"`
	AvatarID *uint    `json:"avatarId,omitempty"`
	Avatar   string   `json:"avatar,omitempty"`
	Roles    []string `json:"roles,omitempty"`
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  LoginUserVO `json:"user"`
}

// LoginView 用户登录
// @Summary      用户登录
// @Description  用户账号登录，支持可选的图片验证码
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        user  body  LoginRequest  true  "登录参数"
// @Success      200  {object}  res.Response  "{"code":0,"msg":"success","data":{"token":"token_string","user":{"id":1,"username":"admin"}}}"
// @Failure      200  {object}  res.Response       "{"code":1,"msg":"invalid parameters","data":{"rePassword":"确认密码必须等于密码"}}"
// @Router       /auth/login [post]
func (User) LoginView(c *gin.Context) {
	cr := middleware.GetJSON[LoginRequest](c)

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

	user, err := query.User.WithContext(c).
		Where(query.User.Username.Eq(cr.Username)).
		Take()

	if err != nil {
		res.FailWithMsg(c, "用户名或密码错误")
		return
	}

	if !pwd.CompareHashAndPassword(user.Password, cr.Password) {
		res.FailWithMsg(c, "用户名或密码错误")
		return
	}

	var adminRoleCount int64
	adminRoleCount, err = query.UserRole.WithContext(c).
		Join(query.Role, query.UserRole.RoleID.EqCol(query.Role.ID)).
		Where(
			query.UserRole.UserID.Eq(user.ID),
			query.Role.Code.Eq(permissions.RoleCode[permissions.RoleAdmin]),
		).
		Count()
	if err != nil {
		zap.S().Warnf("查询用户管理员角色失败 userID=%d err=%v", user.ID, err)
	}

	token, err := jwts.SetToken(jwts.Claims{
		UserID:  user.ID,
		IsAdmin: adminRoleCount > 0,
	})
	if err != nil {
		res.FailWithMsg(c, "登录失败")
		return
	}

	permission_serv.WarmUserPerms(user.ID)

	roles := make([]string, 0)
	if adminRoleCount > 0 {
		roles = append(roles, permissions.RoleCode[permissions.RoleAdmin])
	}

	avatarURL := ""
	if user.AvatarID != nil {
		imageModel, avatarErr := query.Image.WithContext(c).
			Where(query.Image.ID.Eq(*user.AvatarID)).
			Take()
		if avatarErr == nil {
			avatarURL = imageModel.Address
		} else if !errors.Is(avatarErr, gorm.ErrRecordNotFound) {
			zap.S().Warnf("查询用户头像失败 userID=%d avatarID=%d err=%v", user.ID, *user.AvatarID, avatarErr)
		}
	}

	res.OkWithData(c, LoginResponse{
		Token: token,
		User: LoginUserVO{
			ID:       user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			AvatarID: user.AvatarID,
			Avatar:   avatarURL,
			Roles:    roles,
		},
	})
}
