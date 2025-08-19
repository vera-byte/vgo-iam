package bootstrap

import (
	"fmt"
	"net"

	"github.com/gocraft/dbr/v2"
	"github.com/vera-byte/vgo-iam/internal/api"
	"github.com/vera-byte/vgo-iam/internal/config"
	"github.com/vera-byte/vgo-iam/internal/policy"
	"github.com/vera-byte/vgo-iam/internal/service"
	"github.com/vera-byte/vgo-iam/internal/store"
	"github.com/vera-byte/vgo-iam/internal/version"
	vgokit "github.com/vera-byte/vgo-kit"
	"go.uber.org/zap"
)

func Banner(cfg *config.LogConfig) string {
	art := `
 __      ___      _        ___ ___    _    __  __ 
 \ \    / (_)    | |      |_ _/ _ \  / \  |  \/  |
  \ \/\/ / _  ___| | __    | | | | |/ _ \ | |\/| |
   \_/\_/ | |/ __| |/ /    | | |_| / ___ \| |  | |
          |_|\___|_|\_\   |___\___/_/   \_\_|  |_|
`
	if cfg.ToStdout {
		fmt.Println(art)
	}
	return art
}

// InitServices 初始化服务层和API层
// 返回IAMServer实例，用于gRPC服务和命令行操作
func InitServices(cfg *config.AppConfig) (*api.IAMServer, *dbr.Session) {
	// 初始化数据库连接
	sess, err := store.NewPostgresStore(cfg.Database.DSN)
	if err != nil {
		vgokit.Log.Error("failed to connect to database", zap.Error(err))
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
	accessKeyService := service.NewAccessKeyService(accessKeyStore, userStore, []byte(cfg.Security.MasterKey))
	policyEngine := policy.NewPolicyEngine(userService)

	// 初始化API层
	server := api.NewIAMServer(
		userService,
		policyService,
		accessKeyService,
		policyEngine,
		[]byte(cfg.Security.MasterKey),
	)

	return server, sess.Session
}

func Start() (*config.AppConfig, net.Listener) {

	var cfg *config.AppConfig
	if err := vgokit.Cfg.Unmarshal(&cfg); err != nil {
		panic(err)
	}

	Banner(&cfg.Log)

	vgokit.Log.Info("VGO-IAM 服务启动",
		zap.String("version", version.Version),
		zap.String("commit", version.Commit),
		zap.String("build_time", version.BuildTime),
	)

	vgokit.Log.Info("config loaded successfully")
	vgokit.Log.Info("logger initialized successfully")

	listenAddr := ":" + cfg.GRPC.Port
	vgokit.Log.Info("gRPC server will listen on", zap.String("address", listenAddr))
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		vgokit.Log.Error("failed to listen", zap.Error(err))
		panic(err)
	}

	return cfg, lis
}
