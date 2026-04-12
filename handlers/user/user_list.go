package user

import (
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/service/common"
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
)

// UserListView 获取用户列表
// @Summary      获取用户列表
// @Description  分页获取用户列表，支持按用户名或昵称搜索，支持排序
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        pageInfo  body  models.PageInfo  true  "分页参数和搜索条件"
// @Success      200  {object}  res.Response  "{"code":0,"msg":"success","data":{"list":[...],"count":0}}"
// @Failure      200  {object}  res.Response       "{"code":3,"msg":"认证失败"}"
// @Router       /user/list [post]
func (User) UserListView(c *gin.Context) {
	var cr = middleware.GetBind[models.PageInfo](c)

	list, count, _ := common.QueryList(models.User{}, common.QueryOption{
		PageInfo: cr,
		Likes:    []string{"username", "nickname"},
		Debug:    true,
	})
	res.OkWithList(c, list, count)
}
