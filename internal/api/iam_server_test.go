package api

import (
	"context"
	"net"
	"testing"

	"github.com/vera-byte/vgo-iam/internal/policy"
	"github.com/vera-byte/vgo-iam/internal/service"
	"github.com/vera-byte/vgo-iam/internal/store"
	"github.com/vera-byte/vgo-iam/internal/util"
	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func bufDialer(lis *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
}

func initServer(t *testing.T) iamv1.IAMClient {
	lis := bufconn.Listen(bufSize)

	cfg, err := util.LoadConfig("/Users/mac/workspace/vgo-iam/config/config.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	sess, err := store.NewPostgresStore(cfg.Database.DSN)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	userStore := store.NewUserStore(sess.Session)
	policyStore := store.NewPolicyStore(sess.Session)
	accessKeyStore := store.NewAccessKeyStore(sess.Session)
	s := grpc.NewServer(
	// 可以在这里插入 mock 授权中间件
	// grpc.UnaryInterceptor(auth.AccessKeyInterceptor(accessKeyStore)),
	)

	userService := service.NewUserService(userStore, policyStore)
	policyService := service.NewPolicyService(policyStore)
	accessKeyService := service.NewAccessKeyService(accessKeyStore, userStore, []byte(cfg.Security.MasterKey))
	policyEngine := policy.NewPolicyEngine(userService)
	iamv1.RegisterIAMServer(s, NewIAMServer(
		// 传入 mock userService, policyService, accessKeyService, policyEngine, masterKey
		userService, policyService, accessKeyService, policyEngine, []byte(cfg.Security.MasterKey),
	))

	errChan := make(chan error, 1)
	go func() {
		if err := s.Serve(lis); err != nil {
			errChan <- err
		}
		close(errChan)
	}()
	defer s.Stop()

	if err := <-errChan; err != nil {
		t.Fatalf("Server exited with error: %v", err)
	}

	conn, err := grpc.NewClient(
		"bufnet",
		grpc.WithContextDialer(bufDialer(lis)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := iamv1.NewIAMClient(conn)
	return client
}

func TestCreateUser(t *testing.T) {
	server := initServer(t)
	// 构造请求
	req := &iamv1.CreateUserRequest{
		Name:        "testuser",
		DisplayName: "testpass",
		Email:       "test@test.com",
	}
	ctx := context.Background()

	resp, err := server.CreateUser(ctx, req)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if resp.Name == "" {
		t.Errorf("unexpected response: %+v", resp)
	}
}
