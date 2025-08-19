package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/vera-byte/vgo-iam/internal/auth"
	"github.com/vera-byte/vgo-iam/internal/bootstrap"
	"github.com/vera-byte/vgo-iam/internal/version"
	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"
	_ "github.com/vera-byte/vgo-kit"
	vgokit "github.com/vera-byte/vgo-kit"
	"go.uber.org/zap"
)

// ServerCmd 代表server命令
var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the IAM server and handle command line requests",
	Long: `Start the IAM server and handle command line requests such as creating users,
getting user information, and getting user policies.`,
	Run: func(cmd *cobra.Command, args []string) {
		startServer(cmd)
	},
}

func main() {
	if err := ServerCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

var (
	createUser  string
	getUser     string
	getPolicies string
	noServer    bool
)

func init() {
	// 添加标志
	ServerCmd.Flags().StringVar(&createUser, "create-user", "", "Create a new user")
	ServerCmd.Flags().StringVar(&getUser, "get-user", "", "Get user information by username")
	ServerCmd.Flags().StringVar(&getPolicies, "get-policies", "", "Get policies for a user")
	ServerCmd.Flags().BoolVar(&noServer, "no-server", false, "Run command without starting server")
}

func startServer(cmd *cobra.Command) {
	// 从命令行获取参数值
	createUser, _ = cmd.Flags().GetString("create-user")
	getUser, _ = cmd.Flags().GetString("get-user")
	getPolicies, _ = cmd.Flags().GetString("get-policies")
	noServer, _ = cmd.Flags().GetBool("no-server")

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
	// 由于InitServices不返回错误，我们不需要错误处理

	// 获取accessKeyService并获取其存储实现
	accessKeyService := iamServer.AccessKeyService()
	accessKeyStore := accessKeyService.GetStore()

	// 处理命令行请求
	hasCommand := createUser != "" || getUser != "" || getPolicies != ""

	// 如果没有命令行请求或请求了启动服务器
	if !hasCommand || !noServer {
		// 创建gRPC服务器并添加认证中间件
		// 创建gRPC服务器并添加认证中间件
		server := grpc.NewServer(
			grpc.UnaryInterceptor(auth.AccessKeyInterceptor(accessKeyStore)),
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
	} else {
		vgokit.Log.Info("Server not started (--no-server flag set)")
	}

}
