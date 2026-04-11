package middleware

import (
	"fast-gin/utils/res"
	"fast-gin/utils/validate"

	"github.com/gin-gonic/gin"
)

func ShouldBind[T any](c *gin.Context) {
	var obj T
	if err := c.ShouldBind(&obj); err != nil {
		errMsg := validate.GetValidationErrorMessages(err, &obj)
		res.FailParam(c, errMsg)
		c.Abort()
		return
	}
	c.Set("bound_obj", obj)
}

func GetBind[T any](c *gin.Context) T {
	return c.MustGet("bound_obj").(T)
}
