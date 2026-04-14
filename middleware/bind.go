package middleware

import (
	"fast-gin/utils/res"
	"fast-gin/utils/validate"
	"github.com/gin-gonic/gin"
)

func bindAndSet[T any](c *gin.Context, binder func(any) error, key string) {
    var obj T
    if err := binder(&obj); err != nil {
        errMsg := validate.GetValidationErrorMessages(err, &obj)
        res.FailParam(c, errMsg)
        c.Abort()
        return
    }
    c.Set(key, obj)
}

// 分别绑定到不同的 key
func ShouldBindUri[T any](c *gin.Context) {
    bindAndSet[T](c, c.ShouldBindUri, "uri")
}

func ShouldBindQuery[T any](c *gin.Context) {
    bindAndSet[T](c, c.ShouldBindQuery, "query")
}

func ShouldBindJSON[T any](c *gin.Context) {
    bindAndSet[T](c, c.ShouldBindJSON, "json")
}

func ShouldBindForm[T any](c *gin.Context) {
    bindAndSet[T](c, c.ShouldBind, "form")
}

func ShouldBind[T any](c *gin.Context) {
	bindAndSet[T](c, c.ShouldBind, "req")
}

// 获取方法
func GetUri[T any](c *gin.Context) T {
    return c.MustGet("uri").(T)
}

func GetQuery[T any](c *gin.Context) T {
    return c.MustGet("query").(T)
}

func GetJSON[T any](c *gin.Context) T {
    return c.MustGet("json").(T)
}

func GetForm[T any](c *gin.Context) T {
    return c.MustGet("form").(T)
}

func GetReq[T any](c *gin.Context) T {
	return c.MustGet("req").(T)
}