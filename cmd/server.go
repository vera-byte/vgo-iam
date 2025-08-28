package cmd

import (
	"os"
	"os/signal"
	"syscall"

	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"

	"github.com/spf13/cobra"
	"github.com/vera-byte/vgo-iam/internal/auth"
	"github.com/vera-byte/vgo-iam/internal/bootstrap"
	"github.com/vera-byte/vgo-iam/internal/middleware"
	"github.com/vera-byte/vgo-iam/internal/version"
	vgokit "github.com/vera-byte/vgo-kit"
	"github.com/vera-byte/vgo-kit/i18n"
	"github.com/vera-byte/vgo-kit/ratelimit"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func init() {
	RootCmd.AddCommand(ServerCmd)
}

// ServerCmd 代表server命令
var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the IAM server and handle command line requests",
	Long: `Start the IAM server and handle command line requests such as creating users,
getting user information, and getting user policies.`,
	Run: func(cmd *cobra.Command, args []string) {
		startServer()
	},
}

func startServer() {
	// 打印版本信息
	vgokit.Log.Info("Starting VGO-IAM service")
	vgokit.Log.Info("Version: " + version.Version)
	vgokit.Log.Info("Commit: " + version.Commit)
	vgokit.Log.Info("Build Time: " + version.BuildTime)

	// 启动服务并获取配置
	cfg, lis := bootstrap.Start()

	// 初始化服务
	vgokit.Log.Info("Initializing services...")
	iamServer, session := bootstrap.InitServices(cfg)
	defer session.Close()
	vgokit.Log.Info("Services initialized successfully")

	// 获取服务实例
	accessKeyService := iamServer.AccessKeyService()
	accessKeyStore := accessKeyService.GetStore()
	stsService := iamServer.STSService()
	temporaryCredentialStore := stsService.GetStore()

	// 处理命令行请求

	// 创建速率限制拦截器配置
	rateLimitConfig := &ratelimit.InterceptorConfig{
		RateLimiter: vgokit.RateLimiter,
		KeyFunc:     ratelimit.DefaultKeyFunc,
		SkipFunc:    ratelimit.HealthCheckSkipFunc,
	}

	// 创建国际化拦截器配置
	i18nConfig := &i18n.InterceptorConfig{
		Translator:      bootstrap.GetTranslator(),
		DefaultLanguage: i18n.DefaultLanguage,
		LanguageHeader:  "accept-language",
	}

	// 创建gRPC服务器并添加拦截器链
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.LoggingUnaryInterceptor(),
			i18n.UnaryServerInterceptor(i18nConfig),
			ratelimit.UnaryServerInterceptor(rateLimitConfig),
			auth.CombinedAuthInterceptor(accessKeyStore, temporaryCredentialStore),
		),
		grpc.ChainStreamInterceptor(
			middleware.LoggingStreamInterceptor(),
			i18n.StreamServerInterceptor(i18nConfig),
		),
	)
	iamv1.RegisterIAMServer(server, iamServer)

	// 使用从bootstrap.Start()获取的listener
	vgokit.Log.Info("Using listener from bootstrap.Start()")

	// 启动服务协程
	go func() {
		vgokit.Log.Info("Starting gRPC server on port 50051")
		if err := server.Serve(lis); err != nil {
			vgokit.Log.Fatal("Failed to serve", zap.Error(err))
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	vgokit.Log.Info("Shutting down server...")
	server.GracefulStop()
	vgokit.Log.Info("Server exiting")
	vgokit.Log.Close()

}
