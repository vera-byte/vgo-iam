package router

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vera-byte/vgo-iam/internal/config"
	"github.com/vera-byte/vgo-iam/internal/controller"
	"github.com/vera-byte/vgo-iam/internal/middleware"
)

// Router 路由管理器
type Router struct {
	config     *config.Config
	logger     *log.Logger
	engine     *gin.Engine
	middleware *MiddlewareManager
	controller *ControllerManager
}

// MiddlewareManager 中间件管理器
type MiddlewareManager struct {
	logger      *middleware.LoggerMiddleware
	cors        *middleware.CORSMiddleware
	auth        *middleware.AuthMiddleware
	rateLimiter *middleware.RateLimitMiddleware
	security    *middleware.SecurityMiddleware
	errorHandler *middleware.ErrorHandler
}

// ControllerManager 控制器管理器
type ControllerManager struct {
	user        *controller.UserController
	policy      *controller.PolicyController
	accessKey   *controller.AccessKeyController
	application *controller.ApplicationController
	sts         interface{} // STS控制器，暂未实现
	health      interface{} // 健康检查控制器，暂未实现
}

// NewRouter 创建路由管理器
// 参数:
//   - config: 配置
//   - logger: 日志记录器
//   - middlewareManager: 中间件管理器
//   - controllerManager: 控制器管理器
// 返回值:
//   - *Router: 路由管理器实例
func NewRouter(
	config *config.Config,
	logger *log.Logger,
	middlewareManager *MiddlewareManager,
	controllerManager *ControllerManager,
) *Router {
	// 设置Gin模式
	if config.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else if config.Server.Environment == "test" {
		gin.SetMode(gin.TestMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 创建Gin引擎
	engine := gin.New()

	return &Router{
		config:     config,
		logger:     logger,
		engine:     engine,
		middleware: middlewareManager,
		controller: controllerManager,
	}
}

// Setup 设置路由
// 返回值:
//   - *gin.Engine: Gin引擎实例
func (r *Router) Setup() *gin.Engine {
	// 设置全局中间件
	r.setupGlobalMiddleware()

	// 设置API路由
	r.setupAPIRoutes()

	// 设置健康检查路由
	r.setupHealthRoutes()

	// 设置静态文件路由（如果需要）
	r.setupStaticRoutes()

	// 设置404处理
	r.setupNotFoundHandler()

	r.logger.Println("Router setup completed")
	return r.engine
}

// setupGlobalMiddleware 设置全局中间件
func (r *Router) setupGlobalMiddleware() {
	// 恢复中间件（处理panic）
	r.engine.Use(gin.Recovery())

	// 日志中间件
	if r.middleware.logger != nil {
		r.engine.Use(r.middleware.logger.Handler())
	}

	// CORS中间件
	if r.middleware.cors != nil {
		r.engine.Use(r.middleware.cors.Handler())
	}

	// 安全中间件
	if r.middleware.security != nil {
		r.engine.Use(r.middleware.security.Handler())
	}

	// 限流中间件（全局）
	if r.middleware.rateLimiter != nil {
		r.engine.Use(r.middleware.rateLimiter.Handler())
	}

	r.logger.Println("Global middleware setup completed")
}

// setupAPIRoutes 设置API路由
func (r *Router) setupAPIRoutes() {
	// API v1 路由组
	v1 := r.engine.Group("/api/v1")
	{
		// 用户相关路由
		r.setupUserRoutes(v1)

		// 策略相关路由
		r.setupPolicyRoutes(v1)

		// 访问密钥相关路由
		r.setupAccessKeyRoutes(v1)

		// 应用相关路由
		r.setupApplicationRoutes(v1)

		// STS相关路由
		r.setupSTSRoutes(v1)
	}

	r.logger.Println("API routes setup completed")
}

// setupUserRoutes 设置用户路由
func (r *Router) setupUserRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	{
		// 公开路由（不需要认证）
		users.POST("/register", r.controller.user.Register)
		users.POST("/login", r.controller.user.Login)
		users.POST("/forgot-password", r.controller.user.ForgotPassword)
		users.POST("/reset-password", r.controller.user.ResetPassword)

		// 需要认证的路由
		auth := users.Group("", r.middleware.auth.RequireAuth())
		{
			// 用户信息
			auth.GET("/profile", r.controller.user.GetProfile)
			auth.PUT("/profile", r.controller.user.UpdateProfile)
			auth.POST("/change-password", r.controller.user.ChangePassword)
			auth.DELETE("/account", r.controller.user.DeleteAccount)

			// 用户管理（需要管理员权限）
			admin := auth.Group("", r.middleware.auth.RequirePermission("user", "manage"))
			{
				admin.GET("", r.controller.user.ListUsers)
				admin.POST("", r.controller.user.CreateUser)
				admin.GET("/:id", r.controller.user.GetUser)
				admin.PUT("/:id", r.controller.user.UpdateUser)
				admin.DELETE("/:id", r.controller.user.DeleteUser)
				admin.POST("/:id/enable", r.controller.user.EnableUser)
				admin.POST("/:id/disable", r.controller.user.DisableUser)
			}

			// 用户策略管理
			policies := auth.Group("/:id/policies", r.middleware.auth.RequirePermission("policy", "manage"))
			{
				policies.GET("", r.controller.user.GetUserPolicies)
				policies.POST("/:policy_id", r.controller.user.AttachPolicy)
				policies.DELETE("/:policy_id", r.controller.user.DetachPolicy)
			}
		}
	}
}

// setupPolicyRoutes 设置策略路由
func (r *Router) setupPolicyRoutes(rg *gin.RouterGroup) {
	policies := rg.Group("/policies", r.middleware.auth.RequireAuth())
	{
		// 策略查看（需要读取权限）
		read := policies.Group("", r.middleware.auth.RequirePermission("policy", "read"))
		{
			read.GET("", r.controller.policy.ListPolicies)
			read.GET("/:id", r.controller.policy.GetPolicy)
			read.GET("/:id/versions", r.controller.policy.GetPolicyVersions)
			read.GET("/:id/versions/:version", r.controller.policy.GetPolicyVersion)
		}

		// 策略管理（需要管理权限）
		manage := policies.Group("", r.middleware.auth.RequirePermission("policy", "manage"))
		{
			manage.POST("", r.controller.policy.CreatePolicy)
			manage.PUT("/:id", r.controller.policy.UpdatePolicy)
			manage.DELETE("/:id", r.controller.policy.DeletePolicy)
			manage.POST("/validate", r.controller.policy.ValidatePolicy)
			manage.POST("/:id/versions/:version/set-default", r.controller.policy.SetDefaultPolicyVersion)
			// 以下方法暂未实现
			// manage.POST("/:id/enable", r.controller.policy.EnablePolicy)
			// manage.POST("/:id/disable", r.controller.policy.DisablePolicy)
		}
	}
}

// setupAccessKeyRoutes 设置访问密钥路由
func (r *Router) setupAccessKeyRoutes(rg *gin.RouterGroup) {
	accessKeys := rg.Group("/access-keys", r.middleware.auth.RequireAuth())
	{
		// 个人访问密钥管理
		accessKeys.GET("", r.controller.accessKey.ListAccessKeys)
		accessKeys.POST("", r.controller.accessKey.CreateAccessKey)
		accessKeys.GET("/:id", r.controller.accessKey.GetAccessKey)
		accessKeys.PUT("/:id", r.controller.accessKey.UpdateAccessKey)
		accessKeys.DELETE("/:id", r.controller.accessKey.DeleteAccessKey)
		accessKeys.POST("/:id/rotate", r.controller.accessKey.RotateAccessKey)
		accessKeys.GET("/:id/usage", r.controller.accessKey.GetAccessKeyUsage)
		// 以下方法暂未实现
		// accessKeys.POST("/:id/enable", r.controller.accessKey.EnableAccessKey)
		// accessKeys.POST("/:id/disable", r.controller.accessKey.DisableAccessKey)
		// accessKeys.GET("/:id/metadata", r.controller.accessKey.GetAccessKeyMetadata)
	}
}

// setupApplicationRoutes 设置应用路由
func (r *Router) setupApplicationRoutes(rg *gin.RouterGroup) {
	applications := rg.Group("/applications", r.middleware.auth.RequireAuth())
	{
		// 应用查看（需要读取权限）
		read := applications.Group("", r.middleware.auth.RequirePermission("application", "read"))
		{
			read.GET("", r.controller.application.ListApplications)
			read.GET("/:id", r.controller.application.GetApplication)
		}

		// 应用管理（需要管理权限）
		manage := applications.Group("", r.middleware.auth.RequirePermission("application", "manage"))
		{
			manage.POST("", r.controller.application.CreateApplication)
			manage.PUT("/:id", r.controller.application.UpdateApplication)
			manage.DELETE("/:id", r.controller.application.DeleteApplication)
			// 以下方法暂未实现
			// manage.POST("/:id/enable", r.controller.application.EnableApplication)
			// manage.POST("/:id/disable", r.controller.application.DisableApplication)
			// manage.POST("/:id/rotate-secret", r.controller.application.RotateSecret)
		}
	}
}

// setupSTSRoutes 设置STS路由
func (r *Router) setupSTSRoutes(rg *gin.RouterGroup) {
	// STS控制器暂未实现，设置默认的STS路由，返回未实现错误
	r.logger.Println("STS controller not implemented, skipping STS routes")
	sts := rg.Group("/sts")
	{
		sts.POST("/assume-role", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "STS service not implemented"})
		})
		sts.POST("/get-session-token", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "STS service not implemented"})
		})
		sts.POST("/get-federation-token", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "STS service not implemented"})
		})
		sts.POST("/decode-authorization-message", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "STS service not implemented"})
		})
		sts.GET("/credentials", r.middleware.auth.RequireAuth(), func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "STS service not implemented"})
		})
		sts.DELETE("/credentials/:id", r.middleware.auth.RequireAuth(), func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "STS service not implemented"})
		})
	}

	// 注意：当STS控制器实现后，需要取消注释以下代码并移除上面的默认实现
	// sts := rg.Group("/sts", r.middleware.auth.RequireAuth())
	// {
	// 	// STS令牌操作
	// 	sts.POST("/assume-role", r.controller.sts.AssumeRole)
	// 	sts.POST("/get-session-token", r.controller.sts.GetSessionToken)
	// 	sts.POST("/get-federation-token", r.controller.sts.GetFederationToken)
	// 	sts.POST("/decode-authorization-message", r.controller.sts.DecodeAuthorizationMessage)
	//
	// 	// 临时凭证管理
	// 	sts.GET("/credentials", r.controller.sts.ListCredentials)
	// 	sts.DELETE("/credentials/:id", r.controller.sts.RevokeCredential)
	// }
}

// setupHealthRoutes 设置健康检查路由
func (r *Router) setupHealthRoutes() {
	// 健康检查控制器暂未实现，设置简单的健康检查路由
	health := r.engine.Group("/health")
	{
		health.GET("/live", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "service is alive"})
		})
		health.GET("/ready", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "service is ready"})
		})
		health.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "service is healthy"})
		})
	}

	// 系统信息路由（需要管理员权限）- 返回未实现错误
	system := r.engine.Group("/system", r.middleware.auth.RequireAuth(), r.middleware.auth.RequireRole("admin"))
	{
		system.GET("/info", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Health service not implemented"})
		})
		system.GET("/metrics", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Health service not implemented"})
		})
		system.GET("/debug/pprof", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Health service not implemented"})
		})
	}

	r.logger.Println("Basic health routes setup completed")

	// 注意：当健康检查控制器实现后，需要取消注释以下代码并移除上面的默认实现
	// health := r.engine.Group("/health")
	// {
	// 	health.GET("/live", r.controller.health.LivenessProbe)
	// 	health.GET("/ready", r.controller.health.ReadinessProbe)
	// 	health.GET("/status", r.controller.health.HealthStatus)
	// }
	//
	// // 系统信息路由（需要管理员权限）
	// system := r.engine.Group("/system", r.middleware.auth.RequireAuth(), r.middleware.auth.RequireRole("admin"))
	// {
	// 	system.GET("/info", r.controller.health.SystemInfo)
	// 	system.GET("/metrics", r.controller.health.Metrics)
	// 	system.GET("/debug/pprof", r.controller.health.PProf)
	// }
	//
	// r.logger.Println("Health routes setup completed")
}

// setupStaticRoutes 设置静态文件路由
func (r *Router) setupStaticRoutes() {
	// 如果有静态文件目录配置
	if r.config.Server.StaticDir != "" {
		r.engine.Static("/static", r.config.Server.StaticDir)
		r.logger.Printf("Static routes setup completed: %s", r.config.Server.StaticDir)
	}

	// 上传文件目录
	if r.config.Server.UploadDir != "" {
		r.engine.Static("/uploads", r.config.Server.UploadDir)
		r.logger.Printf("Upload routes setup completed: %s", r.config.Server.UploadDir)
	}
}

// setupNotFoundHandler 设置404处理
func (r *Router) setupNotFoundHandler() {
	r.engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "endpoint not found",
			"path":  c.Request.URL.Path,
			"method": c.Request.Method,
		})
	})

	r.engine.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error": "method not allowed",
			"path":  c.Request.URL.Path,
			"method": c.Request.Method,
		})
	})

	r.logger.Println("404/405 handlers setup completed")
}

// GetEngine 获取Gin引擎
// 返回值:
//   - *gin.Engine: Gin引擎实例
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

// PrintRoutes 打印所有路由（用于调试）
func (r *Router) PrintRoutes() {
	routes := r.engine.Routes()
	r.logger.Printf("Total routes: %d", len(routes))

	for _, route := range routes {
		r.logger.Printf("[%s] %s -> %s", route.Method, route.Path, route.Handler)
	}
}

// GetRouteInfo 获取路由信息
// 返回值:
//   - []gin.RouteInfo: 路由信息列表
func (r *Router) GetRouteInfo() []gin.RouteInfo {
	return r.engine.Routes()
}

// AddCustomRoute 添加自定义路由
// 参数:
//   - method: HTTP方法
//   - path: 路径
//   - handlers: 处理函数列表
func (r *Router) AddCustomRoute(method, path string, handlers ...gin.HandlerFunc) {
	switch method {
	case http.MethodGet:
		r.engine.GET(path, handlers...)
	case http.MethodPost:
		r.engine.POST(path, handlers...)
	case http.MethodPut:
		r.engine.PUT(path, handlers...)
	case http.MethodPatch:
		r.engine.PATCH(path, handlers...)
	case http.MethodDelete:
		r.engine.DELETE(path, handlers...)
	case http.MethodHead:
		r.engine.HEAD(path, handlers...)
	case http.MethodOptions:
		r.engine.OPTIONS(path, handlers...)
	default:
		r.logger.Printf("Unsupported HTTP method: %s", method)
	}

	r.logger.Printf("Custom route added: [%s] %s", method, path)
}