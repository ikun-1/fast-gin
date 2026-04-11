package pwd

import (
	"golang.org/x/crypto/bcrypt"
	"go.uber.org/zap"
)

// GenerateFromPassword 加密密码
func GenerateFromPassword(password string) string {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		zap.S().Errorf("加密密码错误 %s", err)
		return ""
	}
	return string(hashedPassword)
}

// CompareHashAndPassword 校验密码
func CompareHashAndPassword(hashedPassword string, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
