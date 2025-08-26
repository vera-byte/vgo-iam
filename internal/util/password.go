package util

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword 对密码进行哈希处理
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash 验证密码哈希是否匹配
func CheckPasswordHash(password, hash string) bool {
	error := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return error == nil
}
