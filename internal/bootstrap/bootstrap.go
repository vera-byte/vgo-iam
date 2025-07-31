package bootstrap

import (
	"fmt"
	"net"

	"github.com/gocraft/dbr/v2"
	"github.com/vera-byte/vgo-iam/internal/config"
	"github.com/vera-byte/vgo-iam/internal/store"
	"github.com/vera-byte/vgo-iam/internal/util"
	"github.com/vera-byte/vgo-iam/internal/version"
	"go.uber.org/zap"
)

func Banner(cfg *config.BannerConfig) string {
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
	if cfg.ToLog {
		// 用 Info 或 Warn 级别写入日志文件
		util.Logger.Info("\n" + art)
	}
	return art
}

func Start() (*config.AppConfig, net.Listener, *dbr.Session) {
	cfg, err := util.LoadConfig("config/config.yaml")
	if err != nil {
		panic(err)
	}

	if err := util.InitLogger(cfg.Log); err != nil {
		panic(err)
	}
	defer util.Logger.Sync()

	Banner(&cfg.Banner)

	util.Logger.Info("VGO-IAM 服务启动",
		zap.String("version", version.Version),
		zap.String("commit", version.Commit),
		zap.String("build_time", version.BuildTime),
	)

	util.Logger.Info("config loaded successfully")
	util.Logger.Info("logger initialized successfully")

	sess, err := store.NewPostgresStore(cfg.Database.DSN)
	if err != nil {
		util.Logger.Error("failed to connect to database", zap.Error(err))
		panic(err)
	}
	util.Logger.Info("database connected successfully", zap.String("dsn", cfg.Database.DSN))

	listenAddr := ":" + cfg.GRPC.Port
	util.Logger.Info("gRPC server will listen on", zap.String("address", listenAddr))
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		util.Logger.Error("failed to listen", zap.Error(err))
		panic(err)
	}

	return cfg, lis, sess.Session // 返回 *dbr.Session
}
