package util

import (
	"regexp"
	"strings"
)

// SensitivePatterns 定义敏感信息的正则表达式模式
var SensitivePatterns = map[string]*regexp.Regexp{
	"password":    regexp.MustCompile(`(?i)(password[=:]\s*)[^\s&]+`),
	"secret":      regexp.MustCompile(`(?i)(secret[_\w]*[=:]\s*)[^\s&]+`),
	"key":         regexp.MustCompile(`(?i)(key[=:]\s*)[^\s&]+`),
	"token":       regexp.MustCompile(`(?i)(token[=:]\s*)[^\s&]+`),
	"dsn":         regexp.MustCompile(`(?i)(postgres://[^:]+:)[^@]+(@.+)`),
	"redis_pass":  regexp.MustCompile(`(?i)(redis_pass[=:]\s*)[^\s&]+`),
	"access_key":  regexp.MustCompile(`(?i)(access_key[_\w]*[=:]\s*)[^\s&]+`),
	"secret_key":  regexp.MustCompile(`(?i)(secret_key[_\w]*[=:]\s*)[^\s&]+`),
	"master_key":  regexp.MustCompile(`(?i)(master_key[=:]\s*)[^\s&]+`),
	"jwt_secret":  regexp.MustCompile(`(?i)(jwt_secret[=:]\s*)[^\s&]+`),
}

// SanitizeString 对字符串中的敏感信息进行脱敏处理
// 参数:
//   - input: 需要脱敏的原始字符串
// 返回值:
//   - string: 脱敏后的字符串
func SanitizeString(input string) string {
	result := input
	
	// 对每种敏感信息模式进行脱敏
	for _, pattern := range SensitivePatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// 查找等号或冒号的位置
			if idx := strings.IndexAny(match, "=:"); idx != -1 {
				prefix := match[:idx+1]
				return prefix + "***"
			}
			return "***"
		})
	}
	
	// 特殊处理数据库DSN
	if dsnPattern := SensitivePatterns["dsn"]; dsnPattern.MatchString(result) {
		result = dsnPattern.ReplaceAllString(result, "${1}***${2}")
	}
	
	return result
}

// SanitizeMap 对map中的敏感信息进行脱敏处理
// 参数:
//   - data: 需要脱敏的map数据
// 返回值:
//   - map[string]interface{}: 脱敏后的map数据
func SanitizeMap(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for key, value := range data {
		// 检查键名是否包含敏感信息
		lowerKey := strings.ToLower(key)
		if isSensitiveKey(lowerKey) {
			result[key] = "***"
			continue
		}
		
		// 递归处理嵌套的map
		switch v := value.(type) {
		case map[string]interface{}:
			result[key] = SanitizeMap(v)
		case string:
			result[key] = SanitizeString(v)
		default:
			result[key] = value
		}
	}
	
	return result
}

// isSensitiveKey 检查键名是否为敏感信息
// 参数:
//   - key: 键名（小写）
// 返回值:
//   - bool: 是否为敏感键名
func isSensitiveKey(key string) bool {
	sensitiveKeys := []string{
		"password", "passwd", "pwd",
		"secret", "secret_key", "secret_access_key",
		"key", "private_key", "public_key", "master_key",
		"token", "access_token", "refresh_token", "jwt_secret",
		"dsn", "database_url", "db_password",
		"redis_pass", "redis_password",
		"api_key", "auth_key",
	}
	
	for _, sensitiveKey := range sensitiveKeys {
		if strings.Contains(key, sensitiveKey) {
			return true
		}
	}
	
	return false
}

// SanitizeLogMessage 专门用于日志消息的脱敏处理
// 参数:
//   - message: 日志消息
//   - fields: 日志字段
// 返回值:
//   - string: 脱敏后的消息
//   - map[string]interface{}: 脱敏后的字段
func SanitizeLogMessage(message string, fields map[string]interface{}) (string, map[string]interface{}) {
	sanitizedMessage := SanitizeString(message)
	sanitizedFields := SanitizeMap(fields)
	
	return sanitizedMessage, sanitizedFields
}