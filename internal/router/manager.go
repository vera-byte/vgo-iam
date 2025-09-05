package router

import (
	"log"

	"github.com/vera-byte/vgo-iam/internal/config"
	"github.com/vera-byte/vgo-iam/internal/controller"
	"github.com/vera-byte/vgo-iam/internal/middleware"
	"github.com/vera-byte/vgo-iam/internal/policy"
	"github.com/vera-byte/vgo-iam/internal/service"
)

// NewMiddlewareManager 创建中间件管理器
// 参数:
//   - config: 配置
//   - logger: 日志记录器
//   - userService: 用户服务
//   - policyEngine: 策略引擎
// 返回值:
//   - *MiddlewareManager: 中间件管理器实例
func NewMiddlewareManager(
	config *config.Config,
	logger *log.Logger,
	userService *service.UserService,
	policyEngine *policy.PolicyEngine,
) *MiddlewareManager {
	// 创建错误处理器
	errorHandler := middleware.NewErrorHandler(logger)

	// 创建各种中间件
	loggerMiddleware := middleware.NewLoggerMiddleware(nil, logger)
	corsMiddleware := middleware.NewCORSMiddleware(nil, logger)
	authMiddleware := middleware.NewAuthMiddleware(userService, policyEngine, errorHandler, logger)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(nil, errorHandler, logger, nil)
	securityMiddleware := middleware.NewSecurityMiddleware(nil, logger)

	return &MiddlewareManager{
		logger:       loggerMiddleware,
		cors:         corsMiddleware,
		auth:         authMiddleware,
		rateLimiter:  rateLimitMiddleware,
		security:     securityMiddleware,
		errorHandler: errorHandler,
	}
}

// NewControllerManager 创建控制器管理器
// 参数:
//   - userService: 用户服务
//   - policyService: 策略服务
//   - accessKeyService: 访问密钥服务
//   - applicationService: 应用服务
// 返回值:
//   - *ControllerManager: 控制器管理器实例
func NewControllerManager(
	userService *service.UserService,
	policyService *service.PolicyService,
	accessKeyService *service.AccessKeyService,
	applicationService service.ApplicationService,
) *ControllerManager {
	return &ControllerManager{
		user:        controller.NewUserController(userService),
		policy:      controller.NewPolicyController(policyService),
		accessKey:   controller.NewAccessKeyController(accessKeyService),
		application: controller.NewApplicationController(applicationService),
		sts:         nil, // STS控制器暂未实现
		health:      nil, // 健康检查控制器暂未实现
	}
}