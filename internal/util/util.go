package util

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/viper"
	"github.com/vera-byte/vgo-iam/internal/config"
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

func LoadConfig(configPath string) (*config.AppConfig, error) {
	v := viper.New()

	// 1. 设置配置类型和文件名
	v.SetConfigType("yaml")
	v.SetConfigName(filepath.Base(configPath)) // 不带扩展名的文件名
	v.AddConfigPath(filepath.Dir(configPath))  // 配置文件所在目录

	// 2. 自动读取环境变量（可选）
	v.AutomaticEnv()
	v.SetEnvPrefix("IAM") // 环境变量前缀 IAM_SERVER_HOST

	// 3. 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// 4. 反序列化到结构体
	var cfg config.AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
