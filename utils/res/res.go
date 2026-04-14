package res

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code int
	Msg  string
	Data any `json:"data,omitempty"`
}

const (
	Success = iota
	InternalErr
	NotFoundErr
	AuthErr
	PermissionErr
	ParamErr
	DatabaseErr
	NotLoginErr
)

var CodeMsgMap = map[int]string{
	Success:       "成功",
	InternalErr:   "内部服务器错误",
	NotFoundErr:   "资源未找到",
	AuthErr:       "身份验证失败",
	PermissionErr: "权限不足",
	ParamErr:      "参数无效",
	DatabaseErr:   "数据库错误",
	NotLoginErr:   "用户未登录",
}

func response(c *gin.Context, code int, msg string, data any) {
	c.JSON(http.StatusOK, Response{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}

func Ok(c *gin.Context, data any, msg string) {
	response(c, Success, msg, data)
}

func OkWithMsg(c *gin.Context, msg string) {
	response(c, Success, msg, gin.H{})
}

func OkWithData(c *gin.Context, data any) {
	response(c, Success, CodeMsgMap[Success], data)
}

func OkWithList(c *gin.Context, list any, count int64) {
	response(c, Success, CodeMsgMap[Success], gin.H{
		"list":  list,
		"count": count,
	})
}

func OkSuccess(c *gin.Context) {
	response(c, Success, CodeMsgMap[Success], gin.H{})
}

func Fail(c *gin.Context, code int, msg string) {
	response(c, code, msg, gin.H{})
}

func FailWithMsg(c *gin.Context, msg string) {
	response(c, InternalErr, msg, gin.H{})
}

func FailWithCode(c *gin.Context, code int) {
	if _, exists := CodeMsgMap[code]; !exists {
		code = InternalErr
	}
	response(c, code, CodeMsgMap[code], gin.H{})
}

func FailInternal(c *gin.Context) {
	FailWithCode(c, InternalErr)
}

func FailNotFound(c *gin.Context) {
	FailWithCode(c, NotFoundErr)
}

func FailAuth(c *gin.Context) {
	FailWithCode(c, AuthErr)
}

func FailPermission(c *gin.Context) {
	FailWithCode(c, PermissionErr)
}

func FailParam(c *gin.Context, data any) {
	response(c, ParamErr, CodeMsgMap[ParamErr], data)
}

func FailNotLogin(c *gin.Context) {
	FailWithCode(c, NotLoginErr)
}
