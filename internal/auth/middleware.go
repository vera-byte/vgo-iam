package auth

import (
	"context"
	"slices"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vera-byte/vgo-iam/internal/config"
	"github.com/vera-byte/vgo-iam/internal/store"
	"github.com/vera-byte/vgo-iam/pkg/signature"
	vgokit "github.com/vera-byte/vgo-kit"
)

// AccessKeyInterceptor gRPC访问密钥验证拦截器
func AccessKeyInterceptor(akStore store.AccessKeyStore) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 从上下文获取配置
		cfg := config.LodIAMConfig()
		if cfg == nil {
			return nil, status.Error(codes.Internal, "配置加载失败")
		}
		// 检查是否需要忽略拦截
		if slices.Contains(cfg.Middleware.Ignore, info.FullMethod) {
			return handler(ctx, req)
		}

		// 从metadata获取访问密钥
		accessKeyID := vgokit.GetMetadataValue(ctx, "access-key-id")
		sign := vgokit.GetMetadataValue(ctx, "signature")
		timestamp := vgokit.GetMetadataValue(ctx, "x-iam-date")
		requestData := vgokit.GetMetadataValue(ctx, "request-data")

		if accessKeyID == "" || sign == "" || timestamp == "" || requestData == "" {
			return nil, status.Error(codes.Unauthenticated, "非法访问,请检查请求参数是否正确")
		}

		// 验证访问密钥
		ak, err := akStore.GetByAccessKeyID(accessKeyID, cfg.Middleware.MasterKey)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "非法访问,请检查访问密钥是否正确")
		}

		// 验证密钥状态
		if ak.Status != "active" {
			return nil, status.Error(codes.PermissionDenied, "非法访问,请检查密钥状态是否正确")
		}

		// 验证签名
		valid, err := signature.VerifySignV4(sign, requestData, timestamp, string(ak.SecretAccessKey))
		if err != nil || !valid {
			return nil, status.Error(codes.Unauthenticated, "签名验证失败,请检查签名后再试!")
		}

		// 将用户信息添加到上下文
		type userIDKey struct{}
		ctx = context.WithValue(ctx, userIDKey{}, ak.UserID)

		return handler(ctx, req)
	}
}
