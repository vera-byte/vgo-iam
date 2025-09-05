package middleware

import (
	"context"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vera-byte/vgo-iam/internal/errors"
	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/policy"
	"github.com/vera-byte/vgo-iam/internal/service"
)

// AuthMiddleware 认证和授权中间件
type AuthMiddleware struct {
	userService   *service.UserService
	policyEngine  *policy.PolicyEngine
	errorHandler  *ErrorHandler
	logger        *log.Logger
}

// NewAuthMiddleware 创建认证中间件实例
// 参数:
//   - userService: 用户服务
//   - policyEngine: 策略引擎
//   - errorHandler: 错误处理器
//   - logger: 日志记录器
// 返回值:
//   - *AuthMiddleware: 认证中间件实例
func NewAuthMiddleware(
	userService *service.UserService,
	policyEngine *policy.PolicyEngine,
	errorHandler *ErrorHandler,
	logger *log.Logger,
) *AuthMiddleware {
	return &AuthMiddleware{
		userService:  userService,
		policyEngine: policyEngine,
		errorHandler: errorHandler,
		logger:       logger,
	}
}

// RequireAuth 要求用户认证的中间件
// 返回值:
//   - gin.HandlerFunc: Gin中间件函数
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取认证信息
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			m.errorHandler.HandleError(c, errors.NewBusinessError(errors.CodeUnauthorized, "missing authorization header"))
			c.Abort()
			return
		}

		// 解析Bearer token
		token := m.extractBearerToken(authHeader)
		if token == "" {
			m.errorHandler.HandleError(c, errors.NewBusinessError(errors.CodeUnauthorized, "invalid authorization header format"))
			c.Abort()
			return
		}

		// 验证token并获取用户信息
		user, err := m.validateToken(c.Request.Context(), token)
		if err != nil {
			m.errorHandler.HandleError(c, err)
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("user_name", user.Name)

		m.logger.Printf("User authenticated: %s (ID: %d)", user.Name, user.ID)
		c.Next()
	}
}

// RequirePermission 要求特定权限的中间件
// 参数:
//   - action: 操作名称
//   - resource: 资源名称
// 返回值:
//   - gin.HandlerFunc: Gin中间件函数
func (m *AuthMiddleware) RequirePermission(action, resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		userID, exists := c.Get("user_id")
		if !exists {
			m.errorHandler.HandleError(c, errors.NewBusinessError(errors.CodeUnauthorized, "user not authenticated"))
			c.Abort()
			return
		}

		userIDInt, ok := userID.(int64)
		if !ok {
			m.errorHandler.HandleError(c, errors.NewBusinessError(errors.CodeInternalError, "invalid user ID type"))
			c.Abort()
			return
		}

		// 获取用户信息用于权限检查
		user, exists := c.Get("user")
		if !exists {
			m.errorHandler.HandleError(c, errors.NewBusinessError(errors.CodeUnauthorized, "user not authenticated"))
			c.Abort()
			return
		}

		userModel, ok := user.(*model.User)
		if !ok {
			m.errorHandler.HandleError(c, errors.NewBusinessError(errors.CodeInternalError, "invalid user type"))
			c.Abort()
			return
		}

		// 检查权限
		allowed, err := m.policyEngine.Evaluate(userModel, action, resource)
		if err != nil {
			m.logger.Printf("Permission check failed for user %d: %v", userIDInt, err)
			m.errorHandler.HandleError(c, errors.NewBusinessError(errors.CodeInternalError, "permission check failed"))
			c.Abort()
			return
		}

		if !allowed {
			m.logger.Printf("Permission denied for user %d: action=%s, resource=%s", userIDInt, action, resource)
			m.errorHandler.HandleError(c, errors.NewBusinessError(errors.CodeForbidden, "insufficient permissions"))
			c.Abort()
			return
		}

		m.logger.Printf("Permission granted for user %d: action=%s, resource=%s", userIDInt, action, resource)
		c.Next()
	}
}

// RequireRole 要求特定角色的中间件
// 参数:
//   - roles: 允许的角色列表
// 返回值:
//   - gin.HandlerFunc: Gin中间件函数
func (m *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		userID, exists := c.Get("user_id")
		if !exists {
			m.errorHandler.HandleError(c, errors.NewBusinessError(errors.CodeUnauthorized, "user not authenticated"))
			c.Abort()
			return
		}

		userIDInt, ok := userID.(int64)
		if !ok {
			m.errorHandler.HandleError(c, errors.NewBusinessError(errors.CodeInternalError, "invalid user ID type"))
			c.Abort()
			return
		}

		// 获取用户策略
		policies, err := m.userService.GetUserPolicies(c.Request.Context(), userIDInt)
		if err != nil {
			m.logger.Printf("Failed to get user policies for user %d: %v", userIDInt, err)
			m.errorHandler.HandleError(c, errors.NewBusinessError(errors.CodeInternalError, "failed to check user roles"))
			c.Abort()
			return
		}

		// 检查用户是否具有所需角色
		hasRole := false
		for _, policy := range policies {
			for _, requiredRole := range roles {
				if policy.Name == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			m.logger.Printf("Role check failed for user %d: required roles=%v", userIDInt, roles)
			m.errorHandler.HandleError(c, errors.NewBusinessError(errors.CodeForbidden, "insufficient role permissions"))
			c.Abort()
			return
		}

		m.logger.Printf("Role check passed for user %d: roles=%v", userIDInt, roles)
		c.Next()
	}
}

// OptionalAuth 可选认证中间件（不强制要求认证）
// 返回值:
//   - gin.HandlerFunc: Gin中间件函数
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取认证信息
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// 没有认证信息，继续处理请求
			c.Next()
			return
		}

		// 解析Bearer token
		token := m.extractBearerToken(authHeader)
		if token == "" {
			// token格式无效，继续处理请求
			c.Next()
			return
		}

		// 验证token并获取用户信息
		user, err := m.validateToken(c.Request.Context(), token)
		if err != nil {
			// token验证失败，继续处理请求
			m.logger.Printf("Optional auth failed: %v", err)
			c.Next()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("user_name", user.Name)

		m.logger.Printf("Optional auth successful: %s (ID: %d)", user.Name, user.ID)
		c.Next()
	}
}

// extractBearerToken 从Authorization头中提取Bearer token
func (m *AuthMiddleware) extractBearerToken(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}

// validateToken 验证token并返回用户信息
// 注意：这里是一个简化的实现，实际项目中应该使用JWT或其他token验证机制
func (m *AuthMiddleware) validateToken(ctx context.Context, token string) (*model.User, error) {
	// TODO: 实现真正的token验证逻辑
	// 这里只是一个示例实现
	if token == "invalid" {
		return nil, errors.NewBusinessError(errors.CodeUnauthorized, "invalid token")
	}

	// 模拟从token中解析用户信息
	// 实际实现中应该解析JWT或查询数据库
	return &model.User{
		ID:   1,
		Name: "test_user",
	}, nil
}

// GetCurrentUser 从上下文中获取当前用户信息
// 参数:
//   - c: Gin上下文
// 返回值:
//   - *model.User: 用户信息
//   - bool: 是否找到用户信息
func GetCurrentUser(c *gin.Context) (*model.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}

	userModel, ok := user.(*model.User)
	return userModel, ok
}

// GetCurrentUserID 从上下文中获取当前用户ID
// 参数:
//   - c: Gin上下文
// 返回值:
//   - int64: 用户ID
//   - bool: 是否找到用户ID
func GetCurrentUserID(c *gin.Context) (int64, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	userIDInt, ok := userID.(int64)
	return userIDInt, ok
}