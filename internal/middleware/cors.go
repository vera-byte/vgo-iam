package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CORSConfig CORS配置
type CORSConfig struct {
	// AllowOrigins 允许的源列表
	// 支持通配符 "*" 表示允许所有源
	// 支持具体域名如 "https://example.com"
	AllowOrigins []string `json:"allow_origins"`

	// AllowMethods 允许的HTTP方法列表
	AllowMethods []string `json:"allow_methods"`

	// AllowHeaders 允许的请求头列表
	AllowHeaders []string `json:"allow_headers"`

	// ExposeHeaders 暴露给客户端的响应头列表
	ExposeHeaders []string `json:"expose_headers"`

	// AllowCredentials 是否允许发送凭据（cookies、授权头等）
	AllowCredentials bool `json:"allow_credentials"`

	// MaxAge 预检请求的缓存时间（秒）
	MaxAge time.Duration `json:"max_age"`

	// AllowWildcard 是否允许通配符源
	AllowWildcard bool `json:"allow_wildcard"`

	// AllowBrowserExtensions 是否允许浏览器扩展
	AllowBrowserExtensions bool `json:"allow_browser_extensions"`

	// AllowWebSockets 是否允许WebSocket连接
	AllowWebSockets bool `json:"allow_websockets"`

	// AllowFiles 是否允许file://协议
	AllowFiles bool `json:"allow_files"`
}

// DefaultCORSConfig 默认CORS配置
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodHead,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Accept",
			"Accept-Encoding",
			"Accept-Language",
			"Authorization",
			"X-Requested-With",
			"X-Request-ID",
			"X-API-Key",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"X-Request-ID",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},
		AllowCredentials:       false,
		MaxAge:                 12 * time.Hour,
		AllowWildcard:          true,
		AllowBrowserExtensions: false,
		AllowWebSockets:        false,
		AllowFiles:             false,
	}
}

// RestrictiveCORSConfig 限制性CORS配置（用于生产环境）
func RestrictiveCORSConfig(allowedOrigins []string) *CORSConfig {
	return &CORSConfig{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
			"X-Request-ID",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"X-Request-ID",
		},
		AllowCredentials:       true,
		MaxAge:                 1 * time.Hour,
		AllowWildcard:          false,
		AllowBrowserExtensions: false,
		AllowWebSockets:        false,
		AllowFiles:             false,
	}
}

// CORSMiddleware CORS中间件
type CORSMiddleware struct {
	config *CORSConfig
	logger *log.Logger
}

// NewCORSMiddleware 创建CORS中间件
// 参数:
//   - config: CORS配置
//   - logger: 日志记录器
// 返回值:
//   - *CORSMiddleware: CORS中间件实例
func NewCORSMiddleware(config *CORSConfig, logger *log.Logger) *CORSMiddleware {
	if config == nil {
		config = DefaultCORSConfig()
	}

	return &CORSMiddleware{
		config: config,
		logger: logger,
	}
}

// Handler CORS中间件处理函数
// 返回值:
//   - gin.HandlerFunc: Gin中间件函数
func (cm *CORSMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查源是否被允许
		allowedOrigin := cm.getAllowedOrigin(origin)
		if allowedOrigin == "" {
			cm.logger.Printf("CORS: Origin not allowed: %s", origin)
			// 不设置CORS头，让浏览器处理
			c.Next()
			return
		}

		// 设置CORS响应头
		cm.setCORSHeaders(c, allowedOrigin)

		// 处理预检请求
		if c.Request.Method == http.MethodOptions {
			cm.handlePreflightRequest(c)
			return
		}

		cm.logger.Printf("CORS: Request allowed from origin: %s", origin)
		c.Next()
	}
}

// getAllowedOrigin 获取允许的源
func (cm *CORSMiddleware) getAllowedOrigin(origin string) string {
	// 如果没有Origin头，允许请求（可能是同源请求）
	if origin == "" {
		return "*"
	}

	// 检查是否允许通配符
	for _, allowedOrigin := range cm.config.AllowOrigins {
		if allowedOrigin == "*" {
			if cm.config.AllowWildcard {
				return origin // 返回实际的origin而不是*，以支持凭据
			}
			return "*"
		}

		// 精确匹配
		if allowedOrigin == origin {
			return origin
		}

		// 支持子域名匹配（如果配置了通配符域名）
		if cm.isWildcardMatch(allowedOrigin, origin) {
			return origin
		}
	}

	// 检查特殊协议
	if cm.shouldAllowSpecialOrigin(origin) {
		return origin
	}

	return ""
}

// isWildcardMatch 检查通配符匹配
func (cm *CORSMiddleware) isWildcardMatch(pattern, origin string) bool {
	// 简单的通配符匹配，支持 *.example.com 格式
	if strings.HasPrefix(pattern, "*.") {
		domain := pattern[2:] // 去掉 "*."
		return strings.HasSuffix(origin, "."+domain) || origin == domain
	}
	return false
}

// shouldAllowSpecialOrigin 检查是否应该允许特殊源
func (cm *CORSMiddleware) shouldAllowSpecialOrigin(origin string) bool {
	// 浏览器扩展
	if cm.config.AllowBrowserExtensions {
		if strings.HasPrefix(origin, "chrome-extension://") ||
			strings.HasPrefix(origin, "moz-extension://") ||
			strings.HasPrefix(origin, "safari-extension://") {
			return true
		}
	}

	// WebSocket
	if cm.config.AllowWebSockets {
		if strings.HasPrefix(origin, "ws://") || strings.HasPrefix(origin, "wss://") {
			return true
		}
	}

	// 文件协议
	if cm.config.AllowFiles {
		if strings.HasPrefix(origin, "file://") {
			return true
		}
	}

	return false
}

// setCORSHeaders 设置CORS响应头
func (cm *CORSMiddleware) setCORSHeaders(c *gin.Context, allowedOrigin string) {
	// Access-Control-Allow-Origin
	c.Header("Access-Control-Allow-Origin", allowedOrigin)

	// Access-Control-Allow-Methods
	if len(cm.config.AllowMethods) > 0 {
		c.Header("Access-Control-Allow-Methods", strings.Join(cm.config.AllowMethods, ", "))
	}

	// Access-Control-Allow-Headers
	if len(cm.config.AllowHeaders) > 0 {
		c.Header("Access-Control-Allow-Headers", strings.Join(cm.config.AllowHeaders, ", "))
	}

	// Access-Control-Expose-Headers
	if len(cm.config.ExposeHeaders) > 0 {
		c.Header("Access-Control-Expose-Headers", strings.Join(cm.config.ExposeHeaders, ", "))
	}

	// Access-Control-Allow-Credentials
	if cm.config.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}

	// Access-Control-Max-Age
	if cm.config.MaxAge > 0 {
		c.Header("Access-Control-Max-Age", strconv.Itoa(int(cm.config.MaxAge.Seconds())))
	}
}

// handlePreflightRequest 处理预检请求
func (cm *CORSMiddleware) handlePreflightRequest(c *gin.Context) {
	// 检查请求的方法是否被允许
	requestedMethod := c.Request.Header.Get("Access-Control-Request-Method")
	if requestedMethod != "" && !cm.isMethodAllowed(requestedMethod) {
		cm.logger.Printf("CORS: Method not allowed in preflight: %s", requestedMethod)
		c.AbortWithStatus(http.StatusMethodNotAllowed)
		return
	}

	// 检查请求的头是否被允许
	requestedHeaders := c.Request.Header.Get("Access-Control-Request-Headers")
	if requestedHeaders != "" && !cm.areHeadersAllowed(requestedHeaders) {
		cm.logger.Printf("CORS: Headers not allowed in preflight: %s", requestedHeaders)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	cm.logger.Printf("CORS: Preflight request approved for method: %s, headers: %s", requestedMethod, requestedHeaders)
	c.AbortWithStatus(http.StatusNoContent)
}

// isMethodAllowed 检查方法是否被允许
func (cm *CORSMiddleware) isMethodAllowed(method string) bool {
	for _, allowedMethod := range cm.config.AllowMethods {
		if strings.EqualFold(allowedMethod, method) {
			return true
		}
	}
	return false
}

// areHeadersAllowed 检查头是否被允许
func (cm *CORSMiddleware) areHeadersAllowed(requestedHeaders string) bool {
	if requestedHeaders == "" {
		return true
	}

	headers := strings.Split(requestedHeaders, ",")
	for _, header := range headers {
		header = strings.TrimSpace(header)
		if !cm.isHeaderAllowed(header) {
			return false
		}
	}
	return true
}

// isHeaderAllowed 检查单个头是否被允许
func (cm *CORSMiddleware) isHeaderAllowed(header string) bool {
	// 简单头总是被允许
	simpleHeaders := []string{
		"Accept",
		"Accept-Language",
		"Content-Language",
		"Content-Type",
	}

	for _, simpleHeader := range simpleHeaders {
		if strings.EqualFold(simpleHeader, header) {
			return true
		}
	}

	// 检查配置的允许头
	for _, allowedHeader := range cm.config.AllowHeaders {
		if strings.EqualFold(allowedHeader, header) {
			return true
		}
	}

	return false
}

// ValidateConfig 验证CORS配置
// 参数:
//   - config: CORS配置
// 返回值:
//   - error: 验证错误
func ValidateConfig(config *CORSConfig) error {
	if config == nil {
		return nil
	}

	// 检查凭据和通配符的冲突
	if config.AllowCredentials {
		for _, origin := range config.AllowOrigins {
			if origin == "*" {
				return fmt.Errorf("cannot use wildcard origin '*' with AllowCredentials=true")
			}
		}
	}

	// 检查最大缓存时间
	if config.MaxAge < 0 {
		return fmt.Errorf("MaxAge cannot be negative")
	}

	return nil
}

// GetCORSInfo 获取CORS信息（用于调试）
// 参数:
//   - c: Gin上下文
// 返回值:
//   - map[string]interface{}: CORS信息
func (cm *CORSMiddleware) GetCORSInfo(c *gin.Context) map[string]interface{} {
	origin := c.Request.Header.Get("Origin")
	allowedOrigin := cm.getAllowedOrigin(origin)

	return map[string]interface{}{
		"origin":          origin,
		"allowed_origin":  allowedOrigin,
		"is_allowed":      allowedOrigin != "",
		"is_preflight":    c.Request.Method == http.MethodOptions,
		"requested_method": c.Request.Header.Get("Access-Control-Request-Method"),
		"requested_headers": c.Request.Header.Get("Access-Control-Request-Headers"),
		"config": map[string]interface{}{
			"allow_origins":      cm.config.AllowOrigins,
			"allow_methods":      cm.config.AllowMethods,
			"allow_headers":      cm.config.AllowHeaders,
			"allow_credentials":  cm.config.AllowCredentials,
			"max_age_seconds":    int(cm.config.MaxAge.Seconds()),
		},
	}
}