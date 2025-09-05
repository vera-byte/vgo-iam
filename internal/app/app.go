package app

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/gocraft/dbr/v2"
	"github.com/vera-byte/vgo-iam/internal/config"
	"github.com/vera-byte/vgo-iam/internal/container"
	"github.com/vera-byte/vgo-iam/internal/policy"
	"github.com/vera-byte/vgo-iam/internal/router"
	"github.com/vera-byte/vgo-iam/internal/service"
)

// App 应用程序结构体
type App struct {
	config *config.Config
	logger *log.Logger
	engine *gin.Engine
	router *router.Router
}

// NewApp 创建应用程序实例
// 参数:
//   - config: 配置
//   - logger: 日志记录器
//   - session: 数据库会话
// 返回值:
//   - *App: 应用程序实例
//   - error: 错误信息
func NewApp(config *config.Config, logger *log.Logger, session *dbr.Session) (*App, error) {
	// 使用依赖注入容器初始化服务
	container := container.NewContainer(session)

	// 获取服务实例
	userService := container.GetUserService()
	policyService := container.GetPolicyService()

	// 注意：以下服务暂时设为nil，因为它们需要额外的依赖
	// 在实际项目中，应该通过bootstrap.go或其他方式正确初始化
	var accessKeyService *service.AccessKeyService = nil
	var applicationService service.ApplicationService = nil

	// 初始化策略引擎
	policyEngine := policy.NewPolicyEngine(userService)

	// 创建中间件管理器
	middlewareManager := router.NewMiddlewareManager(
		config,
		logger,
		userService,
		policyEngine,
	)

	// 创建控制器管理器
	controllerManager := router.NewControllerManager(
		userService,
		policyService,
		accessKeyService,
		applicationService,
	)

	// 创建路由管理器
	routerManager := router.NewRouter(
		config,
		logger,
		middlewareManager,
		controllerManager,
	)

	// 设置路由
	engine := routerManager.Setup()

	return &App{
		config: config,
		logger: logger,
		engine: engine,
		router: routerManager,
	}, nil
}

// GetEngine 获取Gin引擎
// 返回值:
//   - *gin.Engine: Gin引擎实例
func (a *App) GetEngine() *gin.Engine {
	return a.engine
}

// GetRouter 获取路由管理器
// 返回值:
//   - *router.Router: 路由管理器实例
func (a *App) GetRouter() *router.Router {
	return a.router
}

// PrintRoutes 打印所有路由（用于调试）
func (a *App) PrintRoutes() {
	a.router.PrintRoutes()
}