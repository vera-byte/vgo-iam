package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"

	"github.com/spf13/cobra"
	"github.com/vera-byte/vgo-iam/internal/api"
	"github.com/vera-byte/vgo-iam/internal/auth"
	"github.com/vera-byte/vgo-iam/internal/bootstrap"
	"github.com/vera-byte/vgo-iam/internal/version"
	vgokit "github.com/vera-byte/vgo-kit"
	vgogrpc "github.com/vera-byte/vgo-kit/grpc"
	"github.com/vera-byte/vgo-kit/i18n"
	"github.com/vera-byte/vgo-kit/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	RootCmd.AddCommand(ServerCmd)
}



// corsMiddleware 创建CORS中间件
func corsMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置CORS头部
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, access-key-id, signature, x-iam-date, request-data, x-amz-security-token")
		w.Header().Set("Access-Control-Expose-Headers", "*")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 调用下一个处理器
		handler.ServeHTTP(w, r)
	})
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
	cfg, _ := bootstrap.Start()

	// 初始化服务
	vgokit.Log.Info("Initializing services...")
	iamServer, session, accessKeyStore, temporaryCredentialStore := bootstrap.InitServices(cfg)
	defer session.Close()
	vgokit.Log.Info("Services initialized successfully")



	// 创建 gRPC 配置
	grpcConfig := &vgogrpc.Config{
		Server: cfg.GRPC.Server,
		Clients: cfg.GRPC.Client.Connections,
	}

	// 获取 zap logger 实例
	var zapLogger *zap.Logger
	if zapLog, ok := vgokit.Log.(*logger.ZapLogger); ok {
		zapLogger = zapLog.Logger
	} else {
		// 如果不是 ZapLogger，创建一个新的
		zapLogger = zap.NewNop()
	}

	// 创建 gRPC 管理器
	grpcManager, err := vgogrpc.NewManager(grpcConfig, zapLogger)
	if err != nil {
		vgokit.Log.Fatal("Failed to create gRPC manager", zap.Error(err))
	}

	// 配置拦截器
	interceptorOpts := []vgogrpc.InterceptorOption{
		vgogrpc.WithLoggingInterceptor(zapLogger),
		vgogrpc.WithRecoveryInterceptor(zapLogger),
		// 添加i18n拦截器
		vgogrpc.WithCustomUnaryInterceptor(i18n.UnaryServerInterceptor(i18n.DefaultInterceptorConfig())),
		// 添加认证拦截器
		vgogrpc.WithCustomUnaryInterceptor(auth.CombinedAuthInterceptor(accessKeyStore, temporaryCredentialStore)),
	}

	// 初始化服务器
	err = grpcManager.InitServer(interceptorOpts...)
	if err != nil {
		vgokit.Log.Fatal("Failed to initialize gRPC server", zap.Error(err))
	}

	// 获取底层 gRPC 服务器并注册服务
	grpcServer := grpcManager.GetServer()
	if grpcServer != nil {
		// 注册服务
		iamv1.RegisterIAMServer(grpcServer.GetServer(), iamServer)
	}

	// 启动gRPC服务协程
	go func() {
		if err := grpcManager.StartServer(); err != nil {
			vgokit.Log.Fatal("Failed to serve gRPC", zap.Error(err))
		}
	}()

	// 创建gRPC Gateway
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 创建gRPC Gateway mux，配置metadata处理器和错误处理器
	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			// 检查头部是否在配置的允许列表中
			lowerKey := strings.ToLower(key)
			for _, allowedHeader := range cfg.Middleware.AllowedHeaders {
				if lowerKey == strings.ToLower(allowedHeader) {
					return lowerKey, true
				}
			}
			// 支持标准Authorization头
			if lowerKey == "authorization" {
				return "authorization", true
			}
			return "", false // 其他头部不传递
		}),
		runtime.WithOutgoingHeaderMatcher(func(key string) (string, bool) {
			return key, true
		}),
		// 自定义错误处理器，使用标准响应格式
		runtime.WithErrorHandler(func(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
			// 使用标准错误响应处理器
			api.WriteErrorResponse(w, err)
		}),
	)

	// 连接到gRPC服务器
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// 注册gRPC Gateway处理器
	grpcAddr := fmt.Sprintf("%s:%d", cfg.GRPC.Server.Host, cfg.GRPC.Server.Port)
	err = iamv1.RegisterIAMHandlerFromEndpoint(ctx, mux, grpcAddr, opts)
	if err != nil {
		vgokit.Log.Fatal("Failed to register gateway", zap.Error(err))
	}

	// 获取翻译器
	translator := bootstrap.GetTranslator()
	
	// 创建中间件链：CORS -> StandardResponseWithI18n -> mux
	standardResponseHandler := api.StandardResponseMiddlewareWithI18n(translator)(mux)
	corsHandler := corsMiddleware(standardResponseHandler)

	// 启动HTTP服务器协程
	httpServer := &http.Server{
		Addr:    cfg.HTTP.Addr,
		Handler: corsHandler,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			vgokit.Log.Fatal("Failed to serve HTTP", zap.Error(err))
		}
	}()

	vgokit.Log.Info("gRPC server listening on port ", zap.String("addr", fmt.Sprintf("%s:%d", cfg.GRPC.Server.Host, cfg.GRPC.Server.Port)))
	vgokit.Log.Info("HTTP Gateway server listening on port ", zap.Any("port", cfg.HTTP.Addr))

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
	grpcManager.StopServer(context.Background())
	vgokit.Log.Info("Servers exited")
	vgokit.Log.Close()

}
