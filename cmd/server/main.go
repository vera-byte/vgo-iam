package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/vera-byte/vgo-iam/internal/bootstrap"
	"github.com/vera-byte/vgo-iam/internal/util"
	"github.com/vera-byte/vgo-iam/internal/version"
	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"
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

	// 初始化日志
	logger, err := util.InitLogger(util.DefaultLogConfig())
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// 打印版本信息
	logger.Info("Starting VGO-IAM service")
	logger.Info("Version: " + version.Version)
	logger.Info("Commit: " + version.Commit)
	logger.Info("Build Time: " + version.BuildTime)

	// 启动服务并获取配置
	cfg, lis := bootstrap.Start()

	// 初始化服务
	logger.Info("Initializing services...")
	iamServer, session := bootstrap.InitServices(cfg)
	defer session.Close()
	// 由于InitServices不返回错误，我们不需要错误处理

	// 处理命令行请求
	hasCommand := false

	// 如果没有命令行请求或请求了启动服务器
	if !hasCommand || !noServer {
		// 创建gRPC服务器
		server := grpc.NewServer()
		iamv1.RegisterIAMServer(server, iamServer)

		// 使用从bootstrap.Start()获取的listener
		logger.Info("Using listener from bootstrap.Start()")

		// 启动服务协程
		go func() {
			logger.Info("Starting gRPC server on port 50051")
			if err := server.Serve(lis); err != nil {
				logger.Fatal("Failed to serve", util.Err(err))
			}
		}()

		// 优雅关闭
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Info("Shutting down server...")
		server.GracefulStop()
		logger.Info("Server exiting")
	} else {
		logger.Info("Server not started (--no-server flag set)")
	}

}
