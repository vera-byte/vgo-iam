package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
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
	"google.golang.org/grpc/credentials/insecure"
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

	// 启动gRPC服务协程
	go func() {
		vgokit.Log.Info("Starting gRPC server on port 50051")
		if err := server.Serve(lis); err != nil {
			vgokit.Log.Fatal("Failed to serve gRPC", zap.Error(err))
		}
	}()

	// 创建gRPC Gateway
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 创建gRPC Gateway mux
	mux := runtime.NewServeMux()

	// 连接到gRPC服务器
	grpcServerEndpoint := "localhost:50051"
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// 注册gRPC Gateway处理器
	err := iamv1.RegisterIAMHandlerFromEndpoint(ctx, mux, grpcServerEndpoint, opts)
	if err != nil {
		vgokit.Log.Fatal("Failed to register gateway", zap.Error(err))
	}

	// 启动HTTP服务器协程
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		vgokit.Log.Info("Starting HTTP Gateway server on port 8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			vgokit.Log.Fatal("Failed to serve HTTP", zap.Error(err))
		}
	}()

	vgokit.Log.Info(fmt.Sprintf("gRPC server listening on port 50051"))
	vgokit.Log.Info(fmt.Sprintf("HTTP Gateway server listening on port 8080"))

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	vgokit.Log.Info("Shutting down servers...")

	// 关闭HTTP服务器
	if err := httpServer.Shutdown(ctx); err != nil {
		vgokit.Log.Error("HTTP server shutdown error", zap.Error(err))
	}

	// 关闭gRPC服务器
	server.GracefulStop()
	vgokit.Log.Info("Servers exited")
	vgokit.Log.Close()

}
