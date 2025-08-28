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

// TemporaryCredentialInterceptor gRPC临时凭证验证拦截器
// tcStore: 临时凭证存储接口
// 返回: gRPC拦截器函数
func TemporaryCredentialInterceptor(tcStore store.TemporaryCredentialStore) grpc.UnaryServerInterceptor {
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

		// 从metadata获取临时凭证信息
		accessKeyID := vgokit.GetMetadataValue(ctx, "access-key-id")
		sessionToken := vgokit.GetMetadataValue(ctx, "x-amz-security-token")
		sign := vgokit.GetMetadataValue(ctx, "signature")
		timestamp := vgokit.GetMetadataValue(ctx, "x-iam-date")
		requestData := vgokit.GetMetadataValue(ctx, "request-data")

		// 如果没有会话令牌，说明不是临时凭证请求，跳过此拦截器
		if sessionToken == "" {
			return handler(ctx, req)
		}

		// 验证必要参数
		if accessKeyID == "" || sign == "" || timestamp == "" || requestData == "" {
			return nil, status.Error(codes.Unauthenticated, "临时凭证参数不完整")
		}

		// 验证临时凭证
		tc, err := tcStore.GetByAccessKeyID(accessKeyID, cfg.Middleware.MasterKey)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "临时凭证不存在或已过期")
		}

		// 验证会话令牌
		if tc.SessionToken != sessionToken {
			return nil, status.Error(codes.Unauthenticated, "会话令牌不匹配")
		}

		// 验证凭证状态和有效期
		if !tc.IsValid() {
			return nil, status.Error(codes.Unauthenticated, "临时凭证无效或已过期")
		}

		// 验证签名
		valid, err := signature.VerifySignV4(sign, requestData, timestamp, tc.SecretAccessKey)
		if err != nil || !valid {
			return nil, status.Error(codes.Unauthenticated, "临时凭证签名验证失败")
		}

		// 将用户信息添加到上下文
		type userIDKey struct{}
		type credentialTypeKey struct{}
		type roleArnKey struct{}
		ctx = context.WithValue(ctx, userIDKey{}, tc.UserID)
		ctx = context.WithValue(ctx, credentialTypeKey{}, tc.TokenType)
		if tc.RoleArn != "" {
			ctx = context.WithValue(ctx, roleArnKey{}, tc.RoleArn)
		}

		return handler(ctx, req)
	}
}

// CombinedAuthInterceptor 组合认证拦截器，支持访问密钥和临时凭证
// akStore: 访问密钥存储接口
// tcStore: 临时凭证存储接口
// 返回: gRPC拦截器函数
func CombinedAuthInterceptor(akStore store.AccessKeyStore, tcStore store.TemporaryCredentialStore) grpc.UnaryServerInterceptor {
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

		// 从metadata获取认证信息
		accessKeyID := vgokit.GetMetadataValue(ctx, "access-key-id")
		sessionToken := vgokit.GetMetadataValue(ctx, "x-amz-security-token")
		sign := vgokit.GetMetadataValue(ctx, "signature")
		timestamp := vgokit.GetMetadataValue(ctx, "x-iam-date")
		requestData := vgokit.GetMetadataValue(ctx, "request-data")

		// 验证必要参数
		if accessKeyID == "" || sign == "" || timestamp == "" || requestData == "" {
			return nil, status.Error(codes.Unauthenticated, "认证参数不完整")
		}

		// 根据是否有会话令牌判断认证类型
		if sessionToken != "" {
			// 临时凭证认证
			return authenticateWithTemporaryCredential(ctx, req, info, handler, tcStore, accessKeyID, sessionToken, sign, timestamp, requestData, cfg.Middleware.MasterKey)
		} else {
			// 访问密钥认证
			return authenticateWithAccessKey(ctx, req, info, handler, akStore, accessKeyID, sign, timestamp, requestData, cfg.Middleware.MasterKey)
		}
	}
}

// authenticateWithAccessKey 使用访问密钥进行认证
// ctx: 上下文
// req: 请求对象
// info: gRPC服务信息
// handler: 处理函数
// akStore: 访问密钥存储接口
// accessKeyID: 访问密钥ID
// sign: 签名
// timestamp: 时间戳
// requestData: 请求数据
// masterKey: 主密钥
// 返回: 响应和错误信息
func authenticateWithAccessKey(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler, akStore store.AccessKeyStore, accessKeyID, sign, timestamp, requestData, masterKey string) (interface{}, error) {
	// 验证访问密钥
	ak, err := akStore.GetByAccessKeyID(accessKeyID, masterKey)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "访问密钥不存在")
	}

	// 验证密钥状态
	if ak.Status != "active" {
		return nil, status.Error(codes.PermissionDenied, "访问密钥已禁用")
	}

	// 验证签名
	valid, err := signature.VerifySignV4(sign, requestData, timestamp, string(ak.SecretAccessKey))
	if err != nil || !valid {
		return nil, status.Error(codes.Unauthenticated, "访问密钥签名验证失败")
	}

	// 将用户信息添加到上下文
	type userIDKey struct{}
	type credentialTypeKey struct{}
	ctx = context.WithValue(ctx, userIDKey{}, ak.UserID)
	ctx = context.WithValue(ctx, credentialTypeKey{}, "access_key")

	return handler(ctx, req)
}

// authenticateWithTemporaryCredential 使用临时凭证进行认证
// ctx: 上下文
// req: 请求对象
// info: gRPC服务信息
// handler: 处理函数
// tcStore: 临时凭证存储接口
// accessKeyID: 访问密钥ID
// sessionToken: 会话令牌
// sign: 签名
// timestamp: 时间戳
// requestData: 请求数据
// masterKey: 主密钥
// 返回: 响应和错误信息
func authenticateWithTemporaryCredential(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler, tcStore store.TemporaryCredentialStore, accessKeyID, sessionToken, sign, timestamp, requestData, masterKey string) (interface{}, error) {
	// 验证临时凭证
	tc, err := tcStore.GetByAccessKeyID(accessKeyID, masterKey)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "临时凭证不存在")
	}

	// 验证会话令牌
	if tc.SessionToken != sessionToken {
		return nil, status.Error(codes.Unauthenticated, "会话令牌不匹配")
	}

	// 验证凭证状态和有效期
	if !tc.IsValid() {
		return nil, status.Error(codes.Unauthenticated, "临时凭证无效或已过期")
	}

	// 验证签名
	valid, err := signature.VerifySignV4(sign, requestData, timestamp, tc.SecretAccessKey)
	if err != nil || !valid {
		return nil, status.Error(codes.Unauthenticated, "临时凭证签名验证失败")
	}

	// 将用户信息添加到上下文
	type userIDKey struct{}
	type credentialTypeKey struct{}
	type roleArnKey struct{}
	ctx = context.WithValue(ctx, userIDKey{}, tc.UserID)
	ctx = context.WithValue(ctx, credentialTypeKey{}, tc.TokenType)
	if tc.RoleArn != "" {
		ctx = context.WithValue(ctx, roleArnKey{}, tc.RoleArn)
	}

	return handler(ctx, req)
}
