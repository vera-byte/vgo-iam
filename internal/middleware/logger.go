package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// LogLevel 日志级别
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp    time.Time              `json:"timestamp"`     // 时间戳
	Level        string                 `json:"level"`         // 日志级别
	Method       string                 `json:"method"`        // HTTP方法
	Path         string                 `json:"path"`          // 请求路径
	Query        string                 `json:"query"`         // 查询参数
	StatusCode   int                    `json:"status_code"`   // 状态码
	Latency      time.Duration          `json:"latency"`       // 响应时间
	ClientIP     string                 `json:"client_ip"`     // 客户端IP
	UserAgent    string                 `json:"user_agent"`    // 用户代理
	UserID       *int64                 `json:"user_id"`       // 用户ID（如果已认证）
	UserName     *string                `json:"user_name"`     // 用户名（如果已认证）
	RequestID    string                 `json:"request_id"`    // 请求ID
	RequestSize  int64                  `json:"request_size"`  // 请求大小
	ResponseSize int64                  `json:"response_size"` // 响应大小
	Headers      map[string]string      `json:"headers"`       // 重要的请求头
	RequestBody  string                 `json:"request_body"`  // 请求体（敏感信息已脱敏）
	ResponseBody string                 `json:"response_body"` // 响应体（敏感信息已脱敏）
	Error        string                 `json:"error"`         // 错误信息
	Extra        map[string]interface{} `json:"extra"`         // 额外信息
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level            LogLevel      `json:"level"`              // 日志级别
	SkipPaths        []string      `json:"skip_paths"`         // 跳过记录的路径
	LogRequestBody   bool          `json:"log_request_body"`   // 是否记录请求体
	LogResponseBody  bool          `json:"log_response_body"`  // 是否记录响应体
	MaxBodySize      int64         `json:"max_body_size"`      // 最大记录的请求/响应体大小
	SensitiveHeaders []string      `json:"sensitive_headers"`  // 敏感请求头（需要脱敏）
	SensitiveFields  []string      `json:"sensitive_fields"`   // 敏感字段（需要脱敏）
	SlowThreshold    time.Duration `json:"slow_threshold"`     // 慢请求阈值
	EnableColors     bool          `json:"enable_colors"`      // 是否启用颜色输出
}

// DefaultLoggerConfig 默认日志配置
func DefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		Level:         LogLevelInfo,
		SkipPaths:     []string{"/health", "/metrics", "/favicon.ico"},
		LogRequestBody: false,
		LogResponseBody: false,
		MaxBodySize:   1024, // 1KB
		SensitiveHeaders: []string{
			"authorization",
			"cookie",
			"x-api-key",
			"x-auth-token",
		},
		SensitiveFields: []string{
			"password",
			"secret",
			"token",
			"key",
			"credential",
		},
		SlowThreshold: 200 * time.Millisecond,
		EnableColors:  false,
	}
}

// LoggerMiddleware 日志中间件
type LoggerMiddleware struct {
	config *LoggerConfig
	logger *log.Logger
}

// NewLoggerMiddleware 创建日志中间件
// 参数:
//   - config: 日志配置
//   - logger: 日志记录器
// 返回值:
//   - *LoggerMiddleware: 日志中间件实例
func NewLoggerMiddleware(config *LoggerConfig, logger *log.Logger) *LoggerMiddleware {
	if config == nil {
		config = DefaultLoggerConfig()
	}

	return &LoggerMiddleware{
		config: config,
		logger: logger,
	}
}

// Handler 日志中间件处理函数
// 返回值:
//   - gin.HandlerFunc: Gin中间件函数
func (lm *LoggerMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否跳过记录
		if lm.shouldSkip(c.Request.URL.Path) {
			c.Next()
			return
		}

		// 记录开始时间
		start := time.Now()

		// 生成请求ID
		requestID := lm.generateRequestID()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// 读取请求体
		var requestBody string
		var requestSize int64
		if lm.config.LogRequestBody && c.Request.Body != nil {
			requestBody, requestSize = lm.readRequestBody(c)
		}

		// 创建响应写入器包装器
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		// 处理请求
		c.Next()

		// 计算响应时间
		latency := time.Since(start)

		// 创建日志条目
		entry := lm.createLogEntry(c, start, latency, requestID, requestBody, requestSize, writer)

		// 记录日志
		lm.logEntry(entry)
	}
}

// shouldSkip 检查是否应该跳过记录
func (lm *LoggerMiddleware) shouldSkip(path string) bool {
	for _, skipPath := range lm.config.SkipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// generateRequestID 生成请求ID
func (lm *LoggerMiddleware) generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// readRequestBody 读取请求体
func (lm *LoggerMiddleware) readRequestBody(c *gin.Context) (string, int64) {
	if c.Request.Body == nil {
		return "", 0
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return "", 0
	}

	// 重新设置请求体，以便后续处理器可以读取
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// 限制记录的大小
	if int64(len(body)) > lm.config.MaxBodySize {
		body = body[:lm.config.MaxBodySize]
	}

	// 脱敏处理
	bodyStr := lm.sanitizeBody(string(body))

	return bodyStr, int64(len(body))
}

// createLogEntry 创建日志条目
func (lm *LoggerMiddleware) createLogEntry(
	c *gin.Context,
	start time.Time,
	latency time.Duration,
	requestID string,
	requestBody string,
	requestSize int64,
	writer *responseWriter,
) *LogEntry {
	entry := &LogEntry{
		Timestamp:    start,
		Level:        lm.getLogLevel(c.Writer.Status(), latency).String(),
		Method:       c.Request.Method,
		Path:         c.Request.URL.Path,
		Query:        c.Request.URL.RawQuery,
		StatusCode:   c.Writer.Status(),
		Latency:      latency,
		ClientIP:     c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		RequestID:    requestID,
		RequestSize:  requestSize,
		ResponseSize: int64(writer.body.Len()),
		Headers:      lm.extractHeaders(c),
		RequestBody:  requestBody,
		Extra:        make(map[string]interface{}),
	}

	// 获取用户信息
	if userID, exists := GetCurrentUserID(c); exists {
		entry.UserID = &userID
	}
	if user, exists := GetCurrentUser(c); exists {
		entry.UserName = &user.Name
	}

	// 记录响应体
	if lm.config.LogResponseBody && writer.body.Len() > 0 {
		responseBody := writer.body.String()
		if int64(len(responseBody)) > lm.config.MaxBodySize {
			responseBody = responseBody[:lm.config.MaxBodySize]
		}
		entry.ResponseBody = lm.sanitizeBody(responseBody)
	}

	// 记录错误信息
	if len(c.Errors) > 0 {
		entry.Error = c.Errors.String()
	}

	// 添加额外信息
	if latency > lm.config.SlowThreshold {
		entry.Extra["slow_request"] = true
	}

	return entry
}

// getLogLevel 根据状态码和响应时间确定日志级别
func (lm *LoggerMiddleware) getLogLevel(statusCode int, latency time.Duration) LogLevel {
	if statusCode >= 500 {
		return LogLevelError
	}
	if statusCode >= 400 {
		return LogLevelWarn
	}
	if latency > lm.config.SlowThreshold {
		return LogLevelWarn
	}
	return LogLevelInfo
}

// extractHeaders 提取重要的请求头
func (lm *LoggerMiddleware) extractHeaders(c *gin.Context) map[string]string {
	headers := make(map[string]string)

	// 提取常见的重要请求头
	importantHeaders := []string{
		"Content-Type",
		"Accept",
		"Accept-Language",
		"Accept-Encoding",
		"X-Forwarded-For",
		"X-Real-IP",
		"X-Request-ID",
		"User-Agent",
	}

	for _, header := range importantHeaders {
		if value := c.GetHeader(header); value != "" {
			headers[strings.ToLower(header)] = value
		}
	}

	// 脱敏敏感请求头
	for _, sensitiveHeader := range lm.config.SensitiveHeaders {
		if _, exists := headers[strings.ToLower(sensitiveHeader)]; exists {
			headers[strings.ToLower(sensitiveHeader)] = "[REDACTED]"
		}
	}

	return headers
}

// sanitizeBody 脱敏请求/响应体
func (lm *LoggerMiddleware) sanitizeBody(body string) string {
	// 尝试解析为JSON
	var data interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		// 不是JSON，直接返回（可能需要其他脱敏处理）
		return body
	}

	// 递归脱敏JSON数据
	sanitized := lm.sanitizeJSONData(data)

	// 重新序列化
	sanitizedBytes, err := json.Marshal(sanitized)
	if err != nil {
		return body // 序列化失败，返回原始数据
	}

	return string(sanitizedBytes)
}

// sanitizeJSONData 递归脱敏JSON数据
func (lm *LoggerMiddleware) sanitizeJSONData(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			if lm.isSensitiveField(key) {
				result[key] = "[REDACTED]"
			} else {
				result[key] = lm.sanitizeJSONData(value)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = lm.sanitizeJSONData(item)
		}
		return result
	default:
		return v
	}
}

// isSensitiveField 检查字段是否敏感
func (lm *LoggerMiddleware) isSensitiveField(field string) bool {
	lowerField := strings.ToLower(field)
	for _, sensitiveField := range lm.config.SensitiveFields {
		if strings.Contains(lowerField, strings.ToLower(sensitiveField)) {
			return true
		}
	}
	return false
}

// logEntry 记录日志条目
func (lm *LoggerMiddleware) logEntry(entry *LogEntry) {
	// 检查日志级别
	if lm.getLogLevelFromString(entry.Level) < lm.config.Level {
		return
	}

	// 序列化日志条目
	logData, err := json.Marshal(entry)
	if err != nil {
		lm.logger.Printf("Failed to marshal log entry: %v", err)
		return
	}

	// 输出日志
	if lm.config.EnableColors {
		lm.logWithColors(entry, string(logData))
	} else {
		lm.logger.Println(string(logData))
	}
}

// getLogLevelFromString 从字符串获取日志级别
func (lm *LoggerMiddleware) getLogLevelFromString(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return LogLevelDebug
	case "INFO":
		return LogLevelInfo
	case "WARN":
		return LogLevelWarn
	case "ERROR":
		return LogLevelError
	default:
		return LogLevelInfo
	}
}

// logWithColors 带颜色输出日志
func (lm *LoggerMiddleware) logWithColors(entry *LogEntry, logData string) {
	var color string
	switch entry.Level {
	case "DEBUG":
		color = "\033[36m" // 青色
	case "INFO":
		color = "\033[32m" // 绿色
	case "WARN":
		color = "\033[33m" // 黄色
	case "ERROR":
		color = "\033[31m" // 红色
	default:
		color = "\033[0m" // 默认颜色
	}

	reset := "\033[0m"
	lm.logger.Printf("%s%s%s", color, logData, reset)
}

// responseWriter 响应写入器包装器
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write 写入响应数据
func (w *responseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

// WriteString 写入字符串响应数据
func (w *responseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// GetRequestID 从上下文中获取请求ID
// 参数:
//   - c: Gin上下文
// 返回值:
//   - string: 请求ID
//   - bool: 是否找到请求ID
func GetRequestID(c *gin.Context) (string, bool) {
	requestID, exists := c.Get("request_id")
	if !exists {
		return "", false
	}

	requestIDStr, ok := requestID.(string)
	return requestIDStr, ok
}