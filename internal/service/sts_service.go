package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/store"
	pb "github.com/vera-byte/vgo-iam/pkg/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// STSService STS临时授权服务
type STSService struct {
	temporaryCredentialStore store.TemporaryCredentialStore
	userStore                store.UserStore
	policyStore              store.PolicyStore
	masterKey                string
}

// NewSTSService 创建STS服务实例
// temporaryCredentialStore: 临时凭证存储接口
// userStore: 用户存储接口
// policyStore: 策略存储接口
// masterKey: 主密钥用于加密
// 返回: STS服务实例
func NewSTSService(
	temporaryCredentialStore store.TemporaryCredentialStore,
	userStore store.UserStore,
	policyStore store.PolicyStore,
	masterKey string,
) *STSService {
	return &STSService{
		temporaryCredentialStore: temporaryCredentialStore,
		userStore:                userStore,
		policyStore:              policyStore,
		masterKey:                masterKey,
	}
}

// GetStore 返回临时凭证存储接口
// 返回: 临时凭证存储接口
func (s *STSService) GetStore() store.TemporaryCredentialStore {
	return s.temporaryCredentialStore
}

// GetSessionToken 获取会话令牌
// ctx: 上下文
// req: 获取会话令牌请求
// 返回: 获取会话令牌响应和错误信息
func (s *STSService) GetSessionToken(ctx context.Context, req *pb.GetSessionTokenRequest) (*pb.GetSessionTokenResponse, error) {
	// 注意：proto中没有UserId字段，需要从认证上下文获取用户信息
	// 这里暂时使用固定用户ID，实际应该从认证中间件获取
	userID := int64(1) // TODO: 从认证上下文获取用户ID

	// 验证用户是否存在
	user, err := s.userStore.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 验证持续时间
	if req.DurationSeconds < 900 || req.DurationSeconds > 43200 { // 15分钟到12小时
		return nil, fmt.Errorf("持续时间必须在900到43200秒之间")
	}

	// 创建会话令牌
	tc := model.NewSessionToken(
		user.ID,
		generateAccessKeyID(),
		generateSecretAccessKey(),
		generateSessionToken(),
		req.DurationSeconds,
	)

	// 保存到数据库
	if err := s.temporaryCredentialStore.Create(tc, s.masterKey); err != nil {
		return nil, fmt.Errorf("创建临时凭证失败: %w", err)
	}

	// 构造响应
	return &pb.GetSessionTokenResponse{
		Credentials: &pb.TemporaryCredentials{
			AccessKeyId:     tc.AccessKeyID,
			SecretAccessKey: tc.SecretAccessKey,
			SessionToken:    tc.SessionToken,
			Expiration:      timestamppb.New(tc.ExpiresAt),
		},
	}, nil
}

// AssumeRole 扮演角色
// ctx: 上下文
// req: 扮演角色请求
// 返回: 扮演角色响应和错误信息
func (s *STSService) AssumeRole(ctx context.Context, req *pb.AssumeRoleRequest) (*pb.AssumeRoleResponse, error) {
	// 注意：proto中没有UserId字段，需要从认证上下文获取用户信息
	// 这里暂时使用固定用户ID，实际应该从认证中间件获取
	userID := int64(1) // TODO: 从认证上下文获取用户ID

	// 验证用户是否存在
	user, err := s.userStore.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 验证角色ARN格式
	if req.RoleArn == "" {
		return nil, fmt.Errorf("角色ARN不能为空")
	}

	// 验证角色会话名称
	if req.RoleSessionName == "" {
		return nil, fmt.Errorf("角色会话名称不能为空")
	}

	// 验证持续时间
	if req.DurationSeconds < 900 || req.DurationSeconds > 43200 { // 15分钟到12小时
		return nil, fmt.Errorf("持续时间必须在900到43200秒之间")
	}

	// TODO: 这里应该验证用户是否有权限扮演该角色
	// 可以通过检查用户的策略来实现

	// 创建AssumeRole令牌
	tc := model.NewAssumeRoleToken(
		user.ID,
		generateAccessKeyID(),
		generateSecretAccessKey(),
		generateSessionToken(),
		req.RoleArn,
		req.RoleSessionName,
		req.DurationSeconds,
		&req.ExternalId,
		&req.Policy,
		nil, // tags暂时不处理
	)

	// 保存到数据库
	if err := s.temporaryCredentialStore.Create(tc, s.masterKey); err != nil {
		return nil, fmt.Errorf("创建临时凭证失败: %w", err)
	}

	// 构造响应
	return &pb.AssumeRoleResponse{
		Credentials: &pb.TemporaryCredentials{
			AccessKeyId:     tc.AccessKeyID,
			SecretAccessKey: tc.SecretAccessKey,
			SessionToken:    tc.SessionToken,
			Expiration:      timestamppb.New(tc.ExpiresAt),
		},
		AssumedRoleUser: &pb.AssumedRoleUser{
			AssumedRoleId: fmt.Sprintf("%s:%s", tc.RoleArn, tc.RoleSessionName),
			Arn:           tc.RoleArn,
		},
	}, nil
}

// RefreshToken 刷新令牌
// ctx: 上下文
// req: 刷新令牌请求
// 返回: 刷新令牌响应和错误信息
func (s *STSService) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	// 根据会话令牌获取临时凭证
	tc, err := s.temporaryCredentialStore.GetBySessionToken(req.SessionToken, s.masterKey)
	if err != nil {
		return nil, fmt.Errorf("临时凭证不存在: %w", err)
	}

	// 验证凭证状态
	if !tc.IsValid() {
		return nil, fmt.Errorf("临时凭证无效或已过期")
	}

	// 验证新的持续时间
	if req.DurationSeconds < 900 || req.DurationSeconds > 43200 {
		return nil, fmt.Errorf("持续时间必须在900到43200秒之间")
	}

	// 刷新凭证
	updatedTC, err := s.temporaryCredentialStore.Refresh(tc.ID, req.DurationSeconds)
	if err != nil {
		return nil, fmt.Errorf("刷新临时凭证失败: %w", err)
	}

	// 解密敏感信息用于响应
	if updatedTC.SecretAccessKey == "" || updatedTC.SessionToken == "" {
		// 重新获取解密后的凭证
		updatedTC, err = s.temporaryCredentialStore.GetByID(updatedTC.ID)
		if err != nil {
			return nil, fmt.Errorf("获取更新后的凭证失败: %w", err)
		}
	}

	// 构造响应
	return &pb.RefreshTokenResponse{
		Credentials: &pb.TemporaryCredentials{
			AccessKeyId:     updatedTC.AccessKeyID,
			SecretAccessKey: updatedTC.SecretAccessKey,
			SessionToken:    updatedTC.SessionToken,
			Expiration:      timestamppb.New(updatedTC.ExpiresAt),
		},
	}, nil
}

// RevokeToken 撤销令牌
// ctx: 上下文
// req: 撤销令牌请求
// 返回: 撤销令牌响应和错误信息
func (s *STSService) RevokeToken(ctx context.Context, req *pb.RevokeTokenRequest) (*pb.RevokeTokenResponse, error) {
	// 根据会话令牌撤销凭证
	if err := s.temporaryCredentialStore.RevokeBySessionToken(req.SessionToken); err != nil {
		return nil, fmt.Errorf("撤销临时凭证失败: %w", err)
	}

	return &pb.RevokeTokenResponse{
		Success: true,
		Message: "临时凭证已成功撤销",
	}, nil
}

// ValidateTemporaryCredential 验证临时凭证
// accessKeyID: 访问密钥ID
// sessionToken: 会话令牌
// 返回: 临时凭证实例和错误信息
func (s *STSService) ValidateTemporaryCredential(accessKeyID, sessionToken string) (*model.TemporaryCredential, error) {
	// 根据访问密钥ID获取临时凭证
	tc, err := s.temporaryCredentialStore.GetByAccessKeyID(accessKeyID, s.masterKey)
	if err != nil {
		return nil, fmt.Errorf("临时凭证不存在: %w", err)
	}

	// 验证会话令牌是否匹配
	if tc.SessionToken != sessionToken {
		return nil, fmt.Errorf("会话令牌不匹配")
	}

	// 验证凭证是否有效
	if !tc.IsValid() {
		return nil, fmt.Errorf("临时凭证无效或已过期")
	}

	return tc, nil
}

// ListUserTemporaryCredentials 获取用户的临时凭证列表
// userID: 用户ID
// 返回: 临时凭证列表和错误信息
func (s *STSService) ListUserTemporaryCredentials(userID int64) ([]*model.TemporaryCredential, error) {
	return s.temporaryCredentialStore.ListByUser(userID)
}

// CleanupExpiredCredentials 清理过期的临时凭证
// 返回: 清理的凭证数量和错误信息
func (s *STSService) CleanupExpiredCredentials() (int64, error) {
	return s.temporaryCredentialStore.CleanupExpired()
}

// generateAccessKeyID 生成访问密钥ID
// 返回: 访问密钥ID
func generateAccessKeyID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("AKIA%s", base64.URLEncoding.EncodeToString(b)[:16])
}

// generateSecretAccessKey 生成密钥
// 返回: 密钥
func generateSecretAccessKey() string {
	b := make([]byte, 30)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// generateSessionToken 生成会话令牌
// 返回: 会话令牌
func generateSessionToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("STS%s", base64.URLEncoding.EncodeToString(b))
}