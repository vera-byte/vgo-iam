package auth

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/vera-byte/vgo-iam/internal/store"
)

// verifyTimestamp 验证时间戳是否在允许范围内
func verifyTimestamp(timestamp string) bool {
	// 解析ISO8601时间戳
	reqTime, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return false
	}

	// 检查时间戳是否在允许范围内（±5分钟）
	now := time.Now()
	diff := now.Sub(reqTime)
	if diff.Abs() > 5*time.Minute {
		return false
	}
	return true
}

// AccessKeyInterceptor gRPC访问密钥验证拦截器
func AccessKeyInterceptor(akStore store.AccessKeyStore) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 排除登录和密钥验证方法
		if info.FullMethod == "/iam.v1.IAM/VerifyAccessKey" ||
			info.FullMethod == "/iam.v1.IAM/CreateAccessKey" {
			return handler(ctx, req)
		}

		// 从metadata获取访问密钥
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		accessKeyID := getFirstValue(md, "access-key-id")
		signature := getFirstValue(md, "signature")
		timestamp := getFirstValue(md, "x-iam-date")
		requestData := getFirstValue(md, "request-data")

		if accessKeyID == "" || signature == "" || timestamp == "" || requestData == "" {
			return nil, status.Error(codes.Unauthenticated, "missing authentication parameters")
		}

		// 验证时间戳
		if !verifyTimestamp(timestamp) {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired timestamp")
		}

		// 验证访问密钥
		ak, err := akStore.GetByAccessKeyID(accessKeyID)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid access key")
		}

		// 验证密钥状态
		if ak.Status != "active" {
			return nil, status.Error(codes.PermissionDenied, "access key is inactive")
		}

		// 验证签名
		valid, err := VerifySignatureV4(signature, requestData, timestamp, ak.SecretAccessKey)
		if err != nil || !valid {
			return nil, status.Error(codes.Unauthenticated, "signature verification failed")
		}

		// 将用户信息添加到上下文
		ctx = context.WithValue(ctx, "user_id", ak.UserID)

		return handler(ctx, req)
	}
}

func getFirstValue(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}
