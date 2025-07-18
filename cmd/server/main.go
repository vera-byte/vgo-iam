package main

import (
	"embed"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/vera-byte/vgo-iam/internal/api"
	"github.com/vera-byte/vgo-iam/internal/auth"
	"github.com/vera-byte/vgo-iam/internal/policy"
	"github.com/vera-byte/vgo-iam/internal/service"
	"github.com/vera-byte/vgo-iam/internal/store"
	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"
)

var configFS embed.FS

// Config 应用配置
type Config struct {
	GRPC struct {
		Port string `yaml:"port"`
	} `yaml:"grpc"`
	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`
	Security struct {
		MasterKey string `yaml:"master_key"`
	} `yaml:"security"`
}

func main() {
	// 1. 加载配置
	cfg, err := LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. 初始化数据库连接
	sess, err := store.NewPostgresStore(cfg.Database.DSN)
	if err != nil {
		log.Panicf("failed to connect to database: %v", err)
	}
	// 4. 初始化存储层
	userStore := store.NewUserStore(sess.Session)
	policyStore := store.NewPolicyStore(sess.Session)
	accessKeyStore := store.NewAccessKeyStore(sess.Session)

	// 5. 初始化服务层
	userService := service.NewUserService(userStore, policyStore)
	policyService := service.NewPolicyService(policyStore)
	accessKeyService := service.NewAccessKeyService(accessKeyStore, userStore, []byte(cfg.Security.MasterKey))
	policyEngine := policy.NewPolicyEngine(userService)

	// 6. 创建gRPC服务器
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(auth.AccessKeyInterceptor(accessKeyStore)),
	)
	iamServer := api.NewIAMServer(
		userService,
		policyService,
		accessKeyService,
		policyEngine,
		[]byte(cfg.Security.MasterKey),
	)
	iamv1.RegisterIAMServer(grpcServer, iamServer)

	// 7. 启动gRPC服务
	lis, err := net.Listen("tcp", ":"+cfg.GRPC.Port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		log.Printf("IAM server running on port %s", cfg.GRPC.Port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 8. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	grpcServer.GracefulStop()
	log.Println("Server gracefully stopped")
}
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// 1. 设置配置类型和文件名
	v.SetConfigType("yaml")
	v.SetConfigName(filepath.Base(configPath)) // 不带扩展名的文件名
	v.AddConfigPath(filepath.Dir(configPath))  // 配置文件所在目录

	// 2. 自动读取环境变量（可选）
	v.AutomaticEnv()
	v.SetEnvPrefix("IAM") // 环境变量前缀 IAM_SERVER_HOST

	// 3. 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// 4. 反序列化到结构体
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
