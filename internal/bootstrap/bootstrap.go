package bootstrap

import (
	"fmt"
	"net"
	"path/filepath"
	"time"

	"github.com/gocraft/dbr/v2"
	"github.com/vera-byte/vgo-iam/internal/api"
	"github.com/vera-byte/vgo-iam/internal/config"
	"github.com/vera-byte/vgo-iam/internal/policy"
	"github.com/vera-byte/vgo-iam/internal/service"
	"github.com/vera-byte/vgo-iam/internal/store"
	"github.com/vera-byte/vgo-iam/internal/version"
	vgokit "github.com/vera-byte/vgo-kit"
	"github.com/vera-byte/vgo-kit/cache"
	"github.com/vera-byte/vgo-kit/db"
	"github.com/vera-byte/vgo-kit/i18n"
	"github.com/vera-byte/vgo-kit/metrics"
	"github.com/vera-byte/vgo-kit/ratelimit"
	"go.uber.org/zap"
)

// 全局轮换调度器
var rotationScheduler *service.RotationScheduler

// 全局翻译器
var globalTranslator i18n.Translator

func Banner() {
	art := `
 __      ___      _        ___ ___    _    __  __ 
 \ \    / (_)    | |      |_ _/ _ \  / \  |  \/  |
  \ \/\/ / _  ___| | __    | | | | |/ _ \ | |\/| |
   \_/\_/ | |/ __| |/ /    | | |_| / ___ \| |  | |
          |_|\___|_|\_\   |___\___/_/   \_\_|  |_|
`
	fmt.Println(art)

}

// InitServices 初始化服务层和API层
// 返回IAMServer实例，用于gRPC服务和命令行操作
func InitServices(cfg *config.AppConfig) (*api.IAMServer, *dbr.Session) {
	// 初始化指标收集器
	if vgokit.Metrics == nil {
		vgokit.Metrics = metrics.NewMetrics("vgo_iam")
		vgokit.Log.Info("metrics initialized successfully")
	}

	// 初始化缓存
	if cfg.Cache == nil {
		vgokit.Cache = cache.NewNoOpCache()
		vgokit.Log.Info("cache initialized successfully")
	} else {
		cache, err := cache.NewRedisCache(cfg.Cache)
		if err != nil {
			vgokit.Log.Error("failed to connect to redis", zap.Error(err))
			panic(err)
		}
		vgokit.Cache = cache
	}

	// 初始化数据库连接
	sess, err := db.NewPostgresStore(cfg.Database.DSN)
	if err != nil {
		vgokit.Log.Error("failed to connect to database", zap.Error(err))
		// 为了调试，先打印详细错误信息
		fmt.Printf("Database connection failed: %v\n", err)
		fmt.Printf("DSN: %s\n", cfg.Database.DSN)
		panic(err)
	}
	vgokit.Log.Info("database connected successfully", zap.String("dsn", cfg.Database.DSN))

	// 初始化存储层
	userStore := store.NewUserStore(sess.Session)
	policyStore := store.NewPolicyStore(sess.Session)
	accessKeyStore := store.NewAccessKeyStore(sess.Session)

	// 初始化服务层
	userService := service.NewUserService(userStore, policyStore)
	policyService := service.NewPolicyService(policyStore)
	accessKeyService := service.NewAccessKeyService(accessKeyStore, userStore, cfg.Middleware.MasterKey)
	policyEngine := policy.NewPolicyEngine(userService)

	// 初始化开发者认证和应用服务
	developerVerificationStore := store.NewDeveloperVerificationStore(sess.DB)
	applicationStore := store.NewApplicationStore(sess.DB)
	developerVerificationService := service.NewDeveloperVerificationService(developerVerificationStore, userStore)
	applicationService := service.NewApplicationService(applicationStore, userStore, developerVerificationService)

	// 设置访问密钥服务的依赖
	accessKeyService.SetDeveloperVerificationService(developerVerificationService)
	accessKeyService.SetApplicationService(applicationService)

	// 初始化轮换调度器
	initRotationScheduler(accessKeyService)

	// 初始化API层
	server := api.NewIAMServer(
		userService,
		policyService,
		accessKeyService,
		developerVerificationService,
		applicationService,
		policyEngine,
		[]byte(cfg.Middleware.MasterKey),
		globalTranslator,
	)

	return server, sess.Session
}

func Start() (*config.AppConfig, net.Listener) {

	cfg := config.LodIAMConfig()

	Banner()

	vgokit.Log.Info("VGO-IAM 服务启动",
		zap.String("version", version.Version),
		zap.String("commit", version.Commit),
		zap.String("build_time", version.BuildTime),
	)

	vgokit.Log.Info("config loaded successfully")
	vgokit.Log.Info("logger initialized successfully")

	// 初始化指标收集器
	vgokit.Metrics = metrics.NewMetrics("vgo_iam")
	vgokit.Log.Info("metrics initialized successfully")

	// 初始化国际化
	if err := initI18n(); err != nil {
		vgokit.Log.Error("failed to initialize i18n", zap.Error(err))
		panic(err)
	}
	vgokit.Log.Info("i18n initialized successfully")

	// 初始化速率限制器
	if err := initRateLimiter(&cfg.RateLimit); err != nil {
		vgokit.Log.Error("failed to initialize rate limiter", zap.Error(err))
		panic(err)
	}
	vgokit.Log.Info("rate limiter initialized successfully")

	listenAddr := ":" + cfg.GRPC.Port
	vgokit.Log.Info("gRPC server will listen on", zap.String("address", listenAddr))
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		vgokit.Log.Error("failed to listen", zap.Error(err))
		panic(err)
	}

	return cfg, lis
}

// initRateLimiter 初始化速率限制器
func initRateLimiter(cfg *config.RateLimitConfig) error {
	rlConfig := &ratelimit.RateLimitConfig{
		Enabled:   cfg.Enabled,
		Type:      cfg.Type,
		Limit:     cfg.Limit,
		Window:    cfg.Window,
		Prefix:    cfg.Prefix,
		RedisAddr: cfg.RedisAddr,
		RedisDB:   cfg.RedisDB,
		RedisPass: cfg.RedisPass,
	}

	var err error
	vgokit.RateLimiter, err = ratelimit.NewRateLimiter(rlConfig)
	if err != nil {
		return err
	}

	return nil
}

// initRotationScheduler 初始化轮换调度器
func initRotationScheduler(accessKeyService *service.AccessKeyService) {
	// 设置默认轮换策略
	defaultPolicy := service.DefaultRotationPolicy()
	accessKeyService.SetRotationPolicy(defaultPolicy)

	// 创建并启动轮换调度器（每小时检查一次）
	rotationScheduler = service.NewRotationScheduler(accessKeyService, time.Hour)
	rotationScheduler.Start()

	vgokit.Log.Info("访问密钥轮换调度器已初始化")
}

// GetRotationScheduler 获取轮换调度器
func GetRotationScheduler() *service.RotationScheduler {
	return rotationScheduler
}

// StopRotationScheduler 停止轮换调度器
func StopRotationScheduler() {
	if rotationScheduler != nil {
		rotationScheduler.Stop()
	}
}

// initI18n 初始化国际化
// 返回值:
//   - error: 错误信息
func initI18n() error {
	// 创建翻译器实例
	globalTranslator = i18n.NewTranslator(i18n.DefaultLanguage)

	// 加载翻译文件
	localesDir := filepath.Join(".", "locales")
	if err := globalTranslator.LoadTranslations(localesDir); err != nil {
		vgokit.Log.Warn("failed to load translation files", zap.Error(err))
		// 不返回错误，允许服务继续运行
	}

	return nil
}

// GetTranslator 获取全局翻译器
// 返回值:
//   - i18n.Translator: 翻译器实例
func GetTranslator() i18n.Translator {
	return globalTranslator
}
