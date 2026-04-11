package validate

import (
	"github.com/go-playground/validator/v10"
)

func init() {
	validations["notAdmin"] = ValidateConfig{
		Tag:         "notAdmin",
		Validate:    notAdmin,
		Translation: "{0}不能为admin",
		Override:    false,
	}
	validations["strongPwd"] = ValidateConfig{
		Tag:         "strongPwd",
		Validate:    strongPwd,
		Translation: "{0}必须包含大写字母、小写字母、数字和特殊字符(!@#$%^&*)",
		Override:    false,
	}
}

func strongPwd(fl validator.FieldLevel) bool {
	// 强密码验证：必须包含大小写字母和数字
	password := fl.Field().String()
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, ch := range password {
		switch {
		case 'A' <= ch && ch <= 'Z':
			hasUpper = true
		case 'a' <= ch && ch <= 'z':
			hasLower = true
		case '0' <= ch && ch <= '9':
			hasDigit = true
		}
		switch ch {
		case '!', '@', '#', '$', '%', '^', '&', '*':
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSpecial
}

func notAdmin(fl validator.FieldLevel) bool {
	if fl.Field().String() == "admin" {
		return false
	}
	return true
}
