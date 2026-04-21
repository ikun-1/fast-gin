package captcha

import (
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"
	"go.uber.org/zap"
)

type GenerateResponse struct {
	CaptchaID  string `json:"captchaId"`
	Captcha    string `json:"captcha"`
	CaptchaAns string `json:"captchaAns"`
}

// GenerateView 生成图片验证码
// @Summary      生成图片验证码
// @Description  生成4位数字图片验证码，返回验证码ID、Base64编码的图片和验证答案
// @Tags         captcha
// @Accept       json
// @Produce      json
// @Success      200  {object}  res.Response  "{"code":0,"msg":"success","data":{"captchaId":"xxx","captcha":"data:image/png;base64,...","captchaAns":"1234"}}"
// @Failure      200  {object}  res.Response       "{"code":1,"msg":"图片验证码生成失败"}"
// @Router       /captchas [post]
func (Captcha) GenerateView(c *gin.Context) {
	var driver = base64Captcha.DriverString{
		Width:           200,
		Height:          60,
		NoiseCount:      2,
		ShowLineOptions: 4,
		Length:          4,
		Source:          "0123456789",
	}
	cp := base64Captcha.NewCaptcha(&driver, CaptchaStore)
	id, b64s, ans, err := cp.Generate()
	if err != nil {
		zap.S().Errorf("图片验证码生成失败 %s", err)
		res.FailWithMsg(c, "图片验证码生成失败")
		return
	}
	res.OkWithData(c, GenerateResponse{
		CaptchaID:  id,
		CaptchaAns: ans,
		Captcha:    b64s,
	})
}
