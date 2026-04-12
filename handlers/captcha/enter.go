package captcha

import "github.com/mojocn/base64Captcha"

type Captcha struct {
}

var CaptchaStore = base64Captcha.DefaultMemStore
