package util

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	match, _ := regexp.MatchString(pattern, email)
	return match
}

// ValidateUserName 验证用户名格式
func ValidateUserName(username string) bool {
	if len(username) < 3 || len(username) > 32 {
		return false
	}
	pattern := `^[a-zA-Z0-9_-]+$`
	match, _ := regexp.MatchString(pattern, username)
	return match
}

// ValidatePasswordStrength 验证密码强度
func ValidatePasswordStrength(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+{}|:"<>?~\-=[\]\\;',./]`).MatchString(password)

	return hasUpper && hasLower && hasDigit && hasSpecial
}

// ValidatePolicyDocument 验证策略文档格式
func ValidatePolicyDocument(policyDoc string) bool {
	// 简化的验证逻辑
	// 实际实现应解析JSON并验证结构
	return strings.Contains(policyDoc, `"Statement"`) &&
		(strings.Contains(policyDoc, `"Allow"`) || strings.Contains(policyDoc, `"Deny"`))
}

// GenerateAccessKeyID 生成访问密钥ID
func GenerateAccessKeyID() string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)[:20]
}

// GenerateSecretAccessKey 生成密钥
func GenerateSecretAccessKey() string {
	b := make([]byte, 30)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// SerializeRequest 序列化请求数据用于签名
func SerializeRequest(method, path, query, body string) string {
	return fmt.Sprintf("%s\n%s\n%s\n%s", method, path, query, body)
}

// ParsePolicyResource 解析策略资源
func ParsePolicyResource(resource string) (service, resourceType, resourceID string) {
	parts := strings.Split(resource, ":")
	if len(parts) < 3 {
		return "", "", ""
	}
	return parts[0], parts[1], parts[2]
}
