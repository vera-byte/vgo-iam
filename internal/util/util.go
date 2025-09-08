package util

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

// Err 返回一个带有错误信息的zap.Field
func Err(err error) zap.Field {
	return zap.Error(err)
}

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
	b := make([]byte, 15) // 增加字节数以确保足够的编码长度
	_, _ = rand.Read(b)
	encoded := base64.RawURLEncoding.EncodeToString(b)
	// 确保不会越界，取前20个字符或全部字符
	if len(encoded) >= 20 {
		return encoded[:20]
	}
	return encoded
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

// PaginationInfo 分页信息结构
type PaginationInfo struct {
	Page  int32 // 当前页码
	Size  int32 // 每页大小
	Total int32 // 总记录数
}

// CalculatePagination 计算分页信息
// page: 页码（从1开始）
// size: 每页大小
// total: 总记录数
// 返回: 标准化的分页信息
func CalculatePagination(page, size int32, total int) PaginationInfo {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	
	return PaginationInfo{
		Page:  page,
		Size:  size,
		Total: int32(total),
	}
}

// CalculateOffset 计算分页偏移量
// page: 页码（从1开始）
// size: 每页大小
// 返回: 起始索引和结束索引
func CalculateOffset(page, size int32) (startIndex, endIndex int32) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	
	startIndex = (page - 1) * size
	endIndex = startIndex + size
	return startIndex, endIndex
}

// SlicePagination 对切片进行分页处理
// data: 原始数据切片
// page: 页码（从1开始）
// size: 每页大小
// 返回: 分页后的数据和分页信息
func SlicePagination[T any](data []T, page, size int32) ([]T, PaginationInfo) {
	total := len(data)
	pagination := CalculatePagination(page, size, total)
	
	startIndex, endIndex := CalculateOffset(page, size)
	
	// 处理边界情况
	if startIndex >= int32(total) {
		// 如果起始索引超出范围，返回空切片
		return []T{}, pagination
	}
	
	if endIndex > int32(total) {
		endIndex = int32(total)
	}
	
	return data[startIndex:endIndex], pagination
}
