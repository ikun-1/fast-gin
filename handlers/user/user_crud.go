package user

import (
	"errors"
	"fast-gin/dal/query"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/service/redis_serv"
	"fast-gin/utils/pwd"
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CreateUserRequest struct {
	Username string `json:"username" binding:"required" display:"用户名" example:"tom"`
	Nickname string `json:"nickname" example:"Tom"`
	Password string `json:"password" binding:"required,min=6" display:"密码" example:"1234567"`
	RealName string `json:"realName" example:"Tom Jerry"`
	Email    string `json:"email" example:"tom@example.com"`
	Phone    string `json:"phone" example:"13800138000"`
	Status   int8   `json:"status" default:"1" example:"1"`
}

type UpdateUserRequest struct {
	Nickname string `json:"nickname" example:"Tom Updated"`
	Password string `json:"password" example:"654321"`
	RealName string `json:"realName" example:"Tom Updated"`
	Email    string `json:"email" example:"tom.updated@example.com"`
	Phone    string `json:"phone" example:"13900139000"`
	AvatarID *uint  `json:"avatarId" example:"1"`
	Status   int8   `json:"status" default:"1" example:"1"`
}

type UserProfileVO struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname,omitempty"`
	RealName string `json:"realName,omitempty"`
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	AvatarID *uint  `json:"avatarId,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
	Status   int8   `json:"status"`
}

// GetUserView 用户详情
// @Summary      用户详情
// @Description  根据用户ID获取用户详情
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        id  path  int  true  "用户ID"
// @Success      200  {object}  res.Response
// @Failure      200  {object}  res.Response
// @Router       /users/{id} [get]
func (User) GetUserView(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)

	user, err := query.User.WithContext(c).Where(query.User.ID.Eq(uri.ID)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			res.FailNotFound(c)
			return
		}
		res.FailWithCode(c, res.DatabaseErr)
		return
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

	res.OkWithData(c, UserProfileVO{
		ID:       user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
		RealName: user.RealName,
		Email:    user.Email,
		Phone:    user.Phone,
		AvatarID: user.AvatarID,
		Avatar:   avatarURL,
		Status:   user.Status,
	})
}

// CreateUserView 创建用户
// @Summary      创建用户
// @Description  创建新用户
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        user  body  CreateUserRequest  true  "用户信息"
// @Success      200  {object}  res.Response
// @Failure      200  {object}  res.Response
// @Router       /users [post]
func (User) CreateUserView(c *gin.Context) {
	req := middleware.GetJSON[CreateUserRequest](c)

	count, err := query.User.WithContext(c).Where(query.User.Username.Eq(req.Username)).Count()
	if err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}
	if count > 0 {
		res.FailWithMsg(c, "用户名已存在")
		return
	}

	status := req.Status
	if status == 0 {
		status = 1
	}

	hash := pwd.GenerateFromPassword(req.Password)
	if hash == "" {
		res.FailWithMsg(c, "密码加密失败")
		return
	}

	user := &models.User{
		Username: req.Username,
		Nickname: req.Nickname,
		Password: hash,
		RealName: req.RealName,
		Email:    req.Email,
		Phone:    req.Phone,
		Status:   status,
	}

	if err := query.User.WithContext(c).Create(user); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	res.OkWithData(c, user)
}

// UpdateUserView 更新用户
// @Summary      更新用户
// @Description  根据用户ID更新用户信息
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        id    path  int                true  "用户ID"
// @Param        user  body  UpdateUserRequest  true  "用户信息"
// @Success      200  {object}  res.Response
// @Failure      200  {object}  res.Response
// @Router       /users/{id} [put]
func (User) UpdateUserView(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)
	req := middleware.GetJSON[UpdateUserRequest](c)

	user, err := query.User.WithContext(c).Where(query.User.ID.Eq(uri.ID)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			res.FailNotFound(c)
			return
		}
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	user.Nickname = req.Nickname
	user.RealName = req.RealName
	user.Email = req.Email
	user.Phone = req.Phone
	if req.AvatarID != nil {
		user.AvatarID = req.AvatarID
	}
	if req.Status == 0 || req.Status == 1 {
		user.Status = req.Status
	}
	if req.Password != "" {
		hash := pwd.GenerateFromPassword(req.Password)
		if hash == "" {
			res.FailWithMsg(c, "密码加密失败")
			return
		}
		user.Password = hash
	}

	if _, err = query.User.WithContext(c).Where(query.User.ID.Eq(uri.ID)).Updates(user); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	res.OkWithData(c, user)
}

// DeleteUserView 删除用户
// @Summary      删除用户
// @Description  根据用户ID删除用户
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        id  path  int  true  "用户ID"
// @Success      200  {object}  res.Response
// @Failure      200  {object}  res.Response
// @Router       /users/{id} [delete]
func (User) DeleteUserView(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)

	user, err := query.User.WithContext(c).Where(query.User.ID.Eq(uri.ID)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			res.FailNotFound(c)
			return
		}
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	if _, err := query.UserRole.WithContext(c).Where(query.UserRole.UserID.Eq(uri.ID)).Delete(); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}
	if _, err := query.User.WithContext(c).Delete(user); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}
	_ = redis_serv.DelUserPermIntSet(uri.ID)

	res.OkSuccess(c)
}
