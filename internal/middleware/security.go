package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityConfig 安全配置
type SecurityConfig struct {
	// HSTS配置
	HSTS *HSTSConfig `json:"hsts"`

	// CSP配置
	CSP *CSPConfig `json:"csp"`

	// Frame选项配置
	FrameOptions string `json:"frame_options"`

	// Content-Type嗅探保护
	ContentTypeNosniff bool `json:"content_type_nosniff"`

	// XSS保护
	XSSProtection *XSSProtectionConfig `json:"xss_protection"`

	// Referrer策略
	ReferrerPolicy string `json:"referrer_policy"`

	// 权限策略
	PermissionsPolicy string `json:"permissions_policy"`

	// 是否移除Server头
	RemoveServerHeader bool `json:"remove_server_header"`

	// 自定义安全头
	CustomHeaders map[string]string `json:"custom_headers"`

	// 是否启用安全Cookie设置
	SecureCookies bool `json:"secure_cookies"`

	// Cookie SameSite策略
	CookieSameSite string `json:"cookie_same_site"`
}

// HSTSConfig HSTS配置
type HSTSConfig struct {
	// 是否启用HSTS
	Enabled bool `json:"enabled"`

	// 最大存活时间（秒）
	MaxAge int `json:"max_age"`

	// 是否包含子域名
	IncludeSubdomains bool `json:"include_subdomains"`

	// 是否预加载
	Preload bool `json:"preload"`
}

// CSPConfig 内容安全策略配置
type CSPConfig struct {
	// 是否启用CSP
	Enabled bool `json:"enabled"`

	// CSP指令
	Directives map[string][]string `json:"directives"`

	// 是否仅报告模式
	ReportOnly bool `json:"report_only"`

	// 报告URI
	ReportURI string `json:"report_uri"`
}

// XSSProtectionConfig XSS保护配置
type XSSProtectionConfig struct {
	// 是否启用XSS保护
	Enabled bool `json:"enabled"`

	// 保护模式（0=禁用, 1=启用, 1; mode=block=启用并阻止）
	Mode string `json:"mode"`
}

// DefaultSecurityConfig 默认安全配置
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		HSTS: &HSTSConfig{
			Enabled:           true,
			MaxAge:            31536000, // 1年
			IncludeSubdomains: true,
			Preload:           false,
		},
		CSP: &CSPConfig{
			Enabled: true,
			Directives: map[string][]string{
				"default-src": {"'self'"},
				"script-src":  {"'self'", "'unsafe-inline'"},
				"style-src":   {"'self'", "'unsafe-inline'"},
				"img-src":     {"'self'", "data:", "https:"},
				"font-src":    {"'self'", "https:"},
				"connect-src": {"'self'"},
				"media-src":   {"'self'"},
				"object-src":  {"'none'"},
				"frame-src":   {"'none'"},
			},
			ReportOnly: false,
		},
		FrameOptions:       "DENY",
		ContentTypeNosniff: true,
		XSSProtection: &XSSProtectionConfig{
			Enabled: true,
			Mode:    "1; mode=block",
		},
		ReferrerPolicy:     "strict-origin-when-cross-origin",
		PermissionsPolicy:  "geolocation=(), microphone=(), camera=()",
		RemoveServerHeader: true,
		CustomHeaders:      make(map[string]string),
		SecureCookies:      true,
		CookieSameSite:     "Strict",
	}
}

// StrictSecurityConfig 严格安全配置（用于生产环境）
func StrictSecurityConfig() *SecurityConfig {
	config := DefaultSecurityConfig()

	// 更严格的CSP
	config.CSP.Directives = map[string][]string{
		"default-src": {"'none'"},
		"script-src":  {"'self'"},
		"style-src":   {"'self'"},
		"img-src":     {"'self'"},
		"font-src":    {"'self'"},
		"connect-src": {"'self'"},
		"media-src":   {"'none'"},
		"object-src":  {"'none'"},
		"frame-src":   {"'none'"},
		"base-uri":    {"'self'"},
		"form-action": {"'self'"},
	}

	// 更严格的其他设置
	config.FrameOptions = "DENY"
	config.ReferrerPolicy = "no-referrer"
	config.PermissionsPolicy = "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), speaker=()"

	return config
}

// DevelopmentSecurityConfig 开发环境安全配置（较宽松）
func DevelopmentSecurityConfig() *SecurityConfig {
	config := DefaultSecurityConfig()

	// 开发环境不启用HSTS
	config.HSTS.Enabled = false

	// 更宽松的CSP
	config.CSP.Directives = map[string][]string{
		"default-src": {"'self'", "'unsafe-inline'", "'unsafe-eval'"},
		"script-src":  {"'self'", "'unsafe-inline'", "'unsafe-eval'"},
		"style-src":   {"'self'", "'unsafe-inline'"},
		"img-src":     {"'self'", "data:", "https:", "http:"},
		"font-src":    {"'self'", "https:", "http:"},
		"connect-src": {"'self'", "ws:", "wss:"},
		"media-src":   {"'self'"},
		"object-src":  {"'self'"},
		"frame-src":   {"'self'"},
	}

	// 开发环境允许iframe
	config.FrameOptions = "SAMEORIGIN"

	// 不强制安全Cookie
	config.SecureCookies = false
	config.CookieSameSite = "Lax"

	return config
}

// SecurityMiddleware 安全中间件
type SecurityMiddleware struct {
	config *SecurityConfig
	logger *log.Logger
}

// NewSecurityMiddleware 创建安全中间件
// 参数:
//   - config: 安全配置
//   - logger: 日志记录器
// 返回值:
//   - *SecurityMiddleware: 安全中间件实例
func NewSecurityMiddleware(config *SecurityConfig, logger *log.Logger) *SecurityMiddleware {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return &SecurityMiddleware{
		config: config,
		logger: logger,
	}
}

// Handler 安全中间件处理函数
// 返回值:
//   - gin.HandlerFunc: Gin中间件函数
func (sm *SecurityMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置安全头
		sm.setSecurityHeaders(c)

		// 处理Cookie安全设置
		if sm.config.SecureCookies {
			sm.setSecureCookies(c)
		}

		// 移除敏感头
		if sm.config.RemoveServerHeader {
			c.Header("Server", "")
		}

		sm.logger.Printf("Security headers applied for request: %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
	}
}

// setSecurityHeaders 设置安全头
func (sm *SecurityMiddleware) setSecurityHeaders(c *gin.Context) {
	// HSTS
	if sm.config.HSTS != nil && sm.config.HSTS.Enabled {
		hstsValue := fmt.Sprintf("max-age=%d", sm.config.HSTS.MaxAge)
		if sm.config.HSTS.IncludeSubdomains {
			hstsValue += "; includeSubDomains"
		}
		if sm.config.HSTS.Preload {
			hstsValue += "; preload"
		}
		c.Header("Strict-Transport-Security", hstsValue)
	}

	// CSP
	if sm.config.CSP != nil && sm.config.CSP.Enabled {
		cspValue := sm.buildCSPValue()
		if sm.config.CSP.ReportOnly {
			c.Header("Content-Security-Policy-Report-Only", cspValue)
		} else {
			c.Header("Content-Security-Policy", cspValue)
		}
	}

	// X-Frame-Options
	if sm.config.FrameOptions != "" {
		c.Header("X-Frame-Options", sm.config.FrameOptions)
	}

	// X-Content-Type-Options
	if sm.config.ContentTypeNosniff {
		c.Header("X-Content-Type-Options", "nosniff")
	}

	// X-XSS-Protection
	if sm.config.XSSProtection != nil && sm.config.XSSProtection.Enabled {
		c.Header("X-XSS-Protection", sm.config.XSSProtection.Mode)
	}

	// Referrer-Policy
	if sm.config.ReferrerPolicy != "" {
		c.Header("Referrer-Policy", sm.config.ReferrerPolicy)
	}

	// Permissions-Policy
	if sm.config.PermissionsPolicy != "" {
		c.Header("Permissions-Policy", sm.config.PermissionsPolicy)
	}

	// 自定义头
	for key, value := range sm.config.CustomHeaders {
		c.Header(key, value)
	}
}

// buildCSPValue 构建CSP值
func (sm *SecurityMiddleware) buildCSPValue() string {
	var parts []string

	for directive, sources := range sm.config.CSP.Directives {
		if len(sources) > 0 {
			part := fmt.Sprintf("%s %s", directive, strings.Join(sources, " "))
			parts = append(parts, part)
		}
	}

	cspValue := strings.Join(parts, "; ")

	// 添加报告URI
	if sm.config.CSP.ReportURI != "" {
		cspValue += fmt.Sprintf("; report-uri %s", sm.config.CSP.ReportURI)
	}

	return cspValue
}

// setSecureCookies 设置安全Cookie
func (sm *SecurityMiddleware) setSecureCookies(c *gin.Context) {
	// 这里可以拦截Set-Cookie头并添加安全属性
	// 由于Gin的限制，我们通过包装ResponseWriter来实现
	originalWriter := c.Writer
	c.Writer = &secureResponseWriter{
		ResponseWriter: originalWriter,
		config:         sm.config,
	}
}

// secureResponseWriter 安全响应写入器
type secureResponseWriter struct {
	gin.ResponseWriter
	config *SecurityConfig
}

// Header 重写Header方法以处理Set-Cookie
func (w *secureResponseWriter) Header() http.Header {
	header := w.ResponseWriter.Header()

	// 处理Set-Cookie头
	cookies := header["Set-Cookie"]
	if len(cookies) > 0 {
		for i, cookie := range cookies {
			cookies[i] = w.addSecureCookieAttributes(cookie)
		}
		header["Set-Cookie"] = cookies
	}

	return header
}

// addSecureCookieAttributes 添加安全Cookie属性
func (w *secureResponseWriter) addSecureCookieAttributes(cookie string) string {
	// 如果已经包含安全属性，不重复添加
	if strings.Contains(strings.ToLower(cookie), "secure") {
		return cookie
	}

	// 添加Secure属性
	if w.config.SecureCookies {
		cookie += "; Secure"
	}

	// 添加HttpOnly属性（如果还没有）
	if !strings.Contains(strings.ToLower(cookie), "httponly") {
		cookie += "; HttpOnly"
	}

	// 添加SameSite属性
	if w.config.CookieSameSite != "" && !strings.Contains(strings.ToLower(cookie), "samesite") {
		cookie += fmt.Sprintf("; SameSite=%s", w.config.CookieSameSite)
	}

	return cookie
}

// SecurityHeadersInfo 获取安全头信息（用于调试）
// 参数:
//   - c: Gin上下文
// 返回值:
//   - map[string]interface{}: 安全头信息
func (sm *SecurityMiddleware) SecurityHeadersInfo(c *gin.Context) map[string]interface{} {
	headers := make(map[string]string)

	// 收集所有安全相关的响应头
	securityHeaders := []string{
		"Strict-Transport-Security",
		"Content-Security-Policy",
		"Content-Security-Policy-Report-Only",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"X-XSS-Protection",
		"Referrer-Policy",
		"Permissions-Policy",
	}

	for _, header := range securityHeaders {
		if value := c.GetHeader(header); value != "" {
			headers[header] = value
		}
	}

	// 添加自定义头
	for key, value := range sm.config.CustomHeaders {
		headers[key] = value
	}

	return map[string]interface{}{
		"applied_headers": headers,
		"config": map[string]interface{}{
			"hsts_enabled":           sm.config.HSTS != nil && sm.config.HSTS.Enabled,
			"csp_enabled":            sm.config.CSP != nil && sm.config.CSP.Enabled,
			"frame_options":          sm.config.FrameOptions,
			"content_type_nosniff":   sm.config.ContentTypeNosniff,
			"xss_protection_enabled": sm.config.XSSProtection != nil && sm.config.XSSProtection.Enabled,
			"referrer_policy":        sm.config.ReferrerPolicy,
			"secure_cookies":         sm.config.SecureCookies,
			"cookie_same_site":       sm.config.CookieSameSite,
		},
	}
}

// ValidateSecurityConfig 验证安全配置
// 参数:
//   - config: 安全配置
// 返回值:
//   - error: 验证错误
func ValidateSecurityConfig(config *SecurityConfig) error {
	if config == nil {
		return nil
	}

	// 验证HSTS配置
	if config.HSTS != nil && config.HSTS.Enabled {
		if config.HSTS.MaxAge < 0 {
			return fmt.Errorf("HSTS MaxAge cannot be negative")
		}
		if config.HSTS.MaxAge < 300 {
			return fmt.Errorf("HSTS MaxAge should be at least 300 seconds")
		}
	}

	// 验证Frame Options
	validFrameOptions := []string{"DENY", "SAMEORIGIN", ""}
	if config.FrameOptions != "" {
		valid := false
		for _, option := range validFrameOptions {
			if config.FrameOptions == option {
				valid = true
				break
			}
		}
		if !valid && !strings.HasPrefix(config.FrameOptions, "ALLOW-FROM ") {
			return fmt.Errorf("invalid FrameOptions value: %s", config.FrameOptions)
		}
	}

	// 验证SameSite策略
	validSameSite := []string{"Strict", "Lax", "None", ""}
	if config.CookieSameSite != "" {
		valid := false
		for _, policy := range validSameSite {
			if config.CookieSameSite == policy {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid CookieSameSite value: %s", config.CookieSameSite)
		}
	}

	return nil
}

// GetSecurityScore 获取安全评分（用于安全审计）
// 参数:
//   - config: 安全配置
// 返回值:
//   - int: 安全评分（0-100）
//   - []string: 改进建议
func GetSecurityScore(config *SecurityConfig) (int, []string) {
	if config == nil {
		return 0, []string{"No security configuration provided"}
	}

	score := 0
	var suggestions []string

	// HSTS检查 (20分)
	if config.HSTS != nil && config.HSTS.Enabled {
		score += 15
		if config.HSTS.IncludeSubdomains {
			score += 3
		} else {
			suggestions = append(suggestions, "Enable HSTS includeSubDomains")
		}
		if config.HSTS.Preload {
			score += 2
		} else {
			suggestions = append(suggestions, "Consider enabling HSTS preload")
		}
	} else {
		suggestions = append(suggestions, "Enable HSTS for better security")
	}

	// CSP检查 (25分)
	if config.CSP != nil && config.CSP.Enabled {
		score += 20
		if !config.CSP.ReportOnly {
			score += 5
		} else {
			suggestions = append(suggestions, "Switch CSP from report-only to enforcement mode")
		}
	} else {
		suggestions = append(suggestions, "Enable Content Security Policy")
	}

	// Frame Options检查 (15分)
	if config.FrameOptions == "DENY" {
		score += 15
	} else if config.FrameOptions == "SAMEORIGIN" {
		score += 10
		suggestions = append(suggestions, "Consider using DENY for X-Frame-Options")
	} else {
		suggestions = append(suggestions, "Set X-Frame-Options to DENY or SAMEORIGIN")
	}

	// Content Type Nosniff检查 (10分)
	if config.ContentTypeNosniff {
		score += 10
	} else {
		suggestions = append(suggestions, "Enable X-Content-Type-Options: nosniff")
	}

	// XSS Protection检查 (10分)
	if config.XSSProtection != nil && config.XSSProtection.Enabled {
		score += 10
	} else {
		suggestions = append(suggestions, "Enable X-XSS-Protection")
	}

	// Referrer Policy检查 (10分)
	if config.ReferrerPolicy != "" {
		score += 10
	} else {
		suggestions = append(suggestions, "Set Referrer-Policy header")
	}

	// Secure Cookies检查 (10分)
	if config.SecureCookies {
		score += 10
	} else {
		suggestions = append(suggestions, "Enable secure cookie settings")
	}

	return score, suggestions
}