package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/vera-byte/vgo-iam/internal/auth"
	"github.com/vera-byte/vgo-iam/internal/bootstrap"
	"github.com/vera-byte/vgo-iam/internal/config"
	"github.com/vera-byte/vgo-iam/internal/service"
	"github.com/vera-byte/vgo-iam/internal/version"
	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"

	vgokit "github.com/vera-byte/vgo-kit"
	"go.uber.org/zap"
	"golang.org/x/term"
)

// 根命令
var rootCmd = &cobra.Command{
	Use:   "vgo-iam",
	Short: "VGO IAM Service",
	Long:  "Identity and Access Management service for VGO ecosystem",
}

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

// InitAdminCmd 初始化管理员用户命令
var InitAdminCmd = &cobra.Command{
	Use:   "init admin",
	Short: "Initialize admin user and access key",
	Long:  "Create initial admin user, policy and access key if they don't exist",
	Run: func(cmd *cobra.Command, args []string) {
		// 交互式输入email
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("请输入管理员邮箱: ")
		email, _ := reader.ReadString('\n')
		email = strings.TrimSpace(email)
		if email == "" {
			fmt.Println("邮箱不能为空，请重新输入")
			os.Exit(1)
		}

		// 交互式输入密码并二次确认
		var password string
		var confirmPassword string
		var err error

		for {
			fmt.Print("请输入管理员密码: ")
			password, err = readPassword()
			if err != nil {
				fmt.Println("读取密码失败: ", err)
				os.Exit(1)
			}

			fmt.Print("请再次输入密码进行确认: ")
			confirmPassword, err = readPassword()
			if err != nil {
				fmt.Println("读取确认密码失败: ", err)
				os.Exit(1)
			}

			if password != confirmPassword {
				fmt.Println("两次输入的密码不一致，请重新输入")
			} else if len(password) < 8 {
				fmt.Println("密码长度不能少于8位，请重新输入")
			} else {
				break
			}
		}

		fmt.Println("正在初始化管理员用户...")

		// 初始化配置
		cfg := config.LodIAMConfig()
		// 初始化服务
		iamServer, session := bootstrap.InitServices(cfg)
		defer session.Close()

		// 获取服务实例
		userService := iamServer.UserService()
		policyService := iamServer.PolicyService()
		accessKeyService := iamServer.AccessKeyService()

		// 调用初始化函数
		err = initAdminUser(userService, policyService, accessKeyService, email, password)
		if err != nil {
			fmt.Printf("创建管理员用户失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("管理员用户初始化成功")
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
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
	// 添加子命令
	rootCmd.AddCommand(ServerCmd)
	rootCmd.AddCommand(InitAdminCmd)

	// 添加服务器命令行标志
	ServerCmd.Flags().StringVar(&createUser, "create-user", "", "Create a new user")
	ServerCmd.Flags().StringVar(&getUser, "get-user", "", "Get user information")
	ServerCmd.Flags().StringVar(&getPolicies, "get-policies", "", "Get policies for a user")
	ServerCmd.Flags().BoolVar(&noServer, "no-server", false, "Run command without starting server")

}

// 读取密码（不回显）
func readPassword() (string, error) {
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	fmt.Println() // 输入完成后换行
	return string(bytePassword), nil
}

// 初始化管理员用户
func initAdminUser(userService *service.UserService, policyService *service.PolicyService, accessKeyService *service.AccessKeyService, email string, password string) error {
	var (
		ctx = context.Background()
	)

	user, err := userService.CreateUser(ctx, "admin", "System Administrator", email)
	if err != nil {
		vgokit.Log.Error("创建管理员用户失败", zap.Error(err))
		return err
	}
	vgokit.Log.Info("管理员用户创建成功", zap.Int64("user_id", user.ID))

	// 设置管理员用户密码
	vgokit.Log.Info("正在设置管理员用户密码...", zap.Int64("user_id", user.ID))
	if err = userService.UpdateUserPassword(ctx, user.ID, password); err != nil {
		vgokit.Log.Error("设置管理员密码失败", zap.Error(err))
		return err
	}
	vgokit.Log.Info("管理员用户密码设置成功")

	// 创建管理员策略（如果不存在）
	_, err = policyService.GetStore().GetByName("admin-policy")

	if err != nil {
		vgokit.Log.Info("Admin policy not found, creating...")
		adminPolicyDoc := `{"Version":"2025-08-01","Statement":[{"Effect":"Allow","Action":"*","Resource":"*"}]}`
		_, err = policyService.CreatePolicy(ctx, "admin-policy", "Administrator policy with full permissions", adminPolicyDoc)
		if err != nil {
			vgokit.Log.Error("Failed to create admin policy", zap.Error(err))
			return err
		}
		vgokit.Log.Info("Admin policy created successfully")
	}

	attachPolicyErr := userService.AttachPolicy(ctx, "admin", "admin-policy")
	if attachPolicyErr != nil {
		// 忽略已存在的错误s
		if attachPolicyErr.Error() != "user policy already exists" {
			vgokit.Log.Error("Failed to attach policy to admin user", zap.Error(attachPolicyErr))
			return attachPolicyErr
		}
	}

	// 检查是否已有访问密钥
	accessKeys, err := accessKeyService.ListAccessKeys(ctx, "admin")
	if err != nil {
		vgokit.Log.Error("Failed to list admin access keys", zap.Error(err))
		return err
	}

	// 如果没有访问密钥，创建一个
	if len(accessKeys) == 0 {
		vgokit.Log.Info("No access keys found for admin, creating...")
		accessKey, err := accessKeyService.CreateAccessKey(ctx, "admin")
		if err != nil {
			vgokit.Log.Error("Failed to create admin access key", zap.Error(err))
			return err
		}
		vgokit.Log.Info("Admin access key created",
			zap.String("access_key_id", accessKey.AccessKeyID),
			zap.String("secret_access_key", accessKey.SecretAccessKey))

		// 注意：在生产环境中，应该将密钥保存到安全的地方，而不是仅记录日志
	} else {
		vgokit.Log.Info("Admin access keys already exist")
	}
	return nil
}

func startServer(cmd *cobra.Command) {
	// 从命令行获取参数值
	createUser, _ = cmd.Flags().GetString("create-user")
	getUser, _ = cmd.Flags().GetString("get-user")
	getPolicies, _ = cmd.Flags().GetString("get-policies")
	noServer, _ = cmd.Flags().GetBool("no-server")

	// 检查是否有命令行请求
	hasCommand := createUser != "" || getUser != "" || getPolicies != ""

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

	// 处理命令行请求
	hasCommand = createUser != "" || getUser != "" || getPolicies != ""

	// 如果没有命令行请求或请求了启动服务器
	if !hasCommand || !noServer {
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
