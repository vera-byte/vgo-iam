package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/vera-byte/vgo-iam/internal/api"
	"github.com/vera-byte/vgo-iam/internal/auth"
	"github.com/vera-byte/vgo-iam/internal/bootstrap"
	"github.com/vera-byte/vgo-iam/internal/policy"
	"github.com/vera-byte/vgo-iam/internal/service"
	"github.com/vera-byte/vgo-iam/internal/store"
	"github.com/vera-byte/vgo-iam/internal/util"
	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"
	"google.golang.org/grpc"
)

func main() {
	cfg, lis, session := bootstrap.Start()

	// 初始化存储层
	userStore := store.NewUserStore(session)
	policyStore := store.NewPolicyStore(session)
	accessKeyStore := store.NewAccessKeyStore(session)

	// 初始化服务层
	userService := service.NewUserService(userStore, policyStore)
	policyService := service.NewPolicyService(policyStore)
	accessKeyService := service.NewAccessKeyService(accessKeyStore, userStore, []byte(cfg.Security.MasterKey))
	policyEngine := policy.NewPolicyEngine(userService)

	// 创建gRPC服务器
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

	// 启动gRPC服务
	go func() {
		util.Logger.Info("IAM server running")
		if err := grpcServer.Serve(lis); err != nil {
			util.Logger.Error("failed to serve")
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	util.Logger.Info("Shutting down server...")
	grpcServer.GracefulStop()
	util.Logger.Info("Server gracefully stopped")
}
