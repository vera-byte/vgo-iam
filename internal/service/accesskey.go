package service

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/vera-byte/vgo-iam/internal/crypto"
	"github.com/vera-byte/vgo-iam/internal/errors"
	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/store"
	"github.com/vera-byte/vgo-iam/internal/util"
	vgokit "github.com/vera-byte/vgo-kit"
	"go.uber.org/zap"
)

// RotationPolicy 轮换策略
type RotationPolicy struct {
	Enabled           bool          `json:"enabled"`
	MaxAge            time.Duration `json:"max_age"`            // 密钥最大使用时间
	WarningThreshold  time.Duration `json:"warning_threshold"`  // 警告阈值
	AutoRotate        bool          `json:"auto_rotate"`        // 是否自动轮换
	GracePeriod       time.Duration `json:"grace_period"`       // 旧密钥保留期
	NotificationEmail string        `json:"notification_email"` // 通知邮箱
}

// DefaultRotationPolicy 默认轮换策略
func DefaultRotationPolicy() *RotationPolicy {
	return &RotationPolicy{
		Enabled:          true,
		MaxAge:           90 * 24 * time.Hour, // 90天
		WarningThreshold: 7 * 24 * time.Hour,  // 7天警告
		AutoRotate:       false,
		GracePeriod:      24 * time.Hour, // 1天保留期
	}
}

// AccessKeyService 访问密钥服务
type AccessKeyService struct {
	accessKeyStore         store.AccessKeyStore
	userStore              store.UserStore
	developerVerifyService DeveloperVerificationService
	applicationService     ApplicationService
	masterKey              string
	rotationPolicy         *RotationPolicy
	rotationMutex          sync.RWMutex
	rotationSchedule       map[string]time.Time // 计划轮换时间
}

// NewAccessKeyService 创建访问密钥服务实例
func NewAccessKeyService(accessKeyStore store.AccessKeyStore, userStore store.UserStore, masterKey string) *AccessKeyService {
	if masterKey == "" {
		err := vgokit.Log.Error("master key is empty", zap.String("masterKey", masterKey))
		panic(err)
	}
	return &AccessKeyService{
		accessKeyStore:   accessKeyStore,
		userStore:        userStore,
		masterKey:        masterKey,
		rotationPolicy:   DefaultRotationPolicy(),
		rotationSchedule: make(map[string]time.Time),
	}
}

// SetDeveloperVerificationService 设置开发者认证服务
func (s *AccessKeyService) SetDeveloperVerificationService(service DeveloperVerificationService) {
	s.developerVerifyService = service
}

// GetDeveloperVerificationService 获取开发者认证服务
func (s *AccessKeyService) GetDeveloperVerificationService() DeveloperVerificationService {
	return s.developerVerifyService
}

// SetApplicationService 设置应用服务
func (s *AccessKeyService) SetApplicationService(service ApplicationService) {
	s.applicationService = service
}

// CreateAccessKey 创建访问密钥
func (s *AccessKeyService) CreateAccessKey(ctx context.Context, userName string) (*model.AccessKey, error) {
	// 获取用户
	user, err := s.userStore.GetByName(userName)
	if err != nil {
		return nil, errors.NewBusinessError(errors.CodeUserNotFound, "user not found")
	}

	// 检查开发者认证状态
	if s.developerVerifyService != nil {
		hasVerification, err := s.checkUserVerificationStatus(ctx, user.ID)
		if err != nil {
			return nil, errors.NewBusinessError(errors.CodeInternalError, "failed to check developer verification status")
		}
		if !hasVerification {
			return nil, errors.NewBusinessError(errors.CodePermissionDenied, "developer verification required before creating access key")
		}
	}

	// 生成密钥
	accessKeyID := util.GenerateAccessKeyID()
	secretKey := util.GenerateSecretAccessKey()

	// 创建访问密钥
	ak := model.NewAccessKey(user.ID, accessKeyID, secretKey, nil, "")
	fmt.Printf("masterKey: %s\n", s.masterKey)
	if err := s.accessKeyStore.Create(ak, s.masterKey); err != nil {
		return nil, errors.NewBusinessError(errors.CodeAccessKeyCreateFailed, "failed to create access key")
	}

	return ak, nil
}

// CreateAccessKeyForApp 为应用创建访问密钥
func (s *AccessKeyService) CreateAccessKeyForApp(ctx context.Context, userName string, appID int64, description string) (*model.AccessKey, error) {
	// 获取用户
	user, err := s.userStore.GetByName(userName)
	if err != nil {
		return nil, errors.NewBusinessError(errors.CodeUserNotFound, "user not found")
	}

	// 检查开发者认证状态
	if s.developerVerifyService != nil {
		hasVerification, err := s.checkUserVerificationStatus(ctx, user.ID)
		if err != nil {
			return nil, errors.NewBusinessError(errors.CodeInternalError, "failed to check developer verification status")
		}
		if !hasVerification {
			return nil, errors.NewBusinessError(errors.CodePermissionDenied, "developer verification required before creating access key")
		}
	}

	// 检查应用所有权
	if s.applicationService != nil {
		isOwner, err := s.applicationService.CheckApplicationOwnership(ctx, appID, user.ID)
		if err != nil {
			return nil, errors.NewBusinessError(errors.CodeInternalError, "failed to check application ownership")
		}
		if !isOwner {
			return nil, errors.NewBusinessError(errors.CodePermissionDenied, "user does not own this application")
		}
	}

	// 生成密钥
	accessKeyID := util.GenerateAccessKeyID()
	secretKey := util.GenerateSecretAccessKey()

	// 创建访问密钥
	ak := model.NewAccessKey(user.ID, accessKeyID, secretKey, &appID, description)
	fmt.Printf("masterKey: %s\n", s.masterKey)
	if err := s.accessKeyStore.Create(ak, s.masterKey); err != nil {
		return nil, errors.NewBusinessError(errors.CodeAccessKeyCreateFailed, "failed to create access key")
	}

	return ak, nil
}

// checkUserVerificationStatus 检查用户是否有通过的开发者认证
func (s *AccessKeyService) checkUserVerificationStatus(ctx context.Context, userID int64) (bool, error) {
	// 检查个人开发者认证
	individualVerified, err := s.developerVerifyService.CheckVerificationStatus(ctx, userID, model.DeveloperTypeIndividual)
	if err != nil {
		return false, err
	}
	if individualVerified {
		return true, nil
	}

	// 检查企业开发者认证
	enterpriseVerified, err := s.developerVerifyService.CheckVerificationStatus(ctx, userID, model.DeveloperTypeEnterprise)
	if err != nil {
		return false, err
	}

	return enterpriseVerified, nil
}

// ListAccessKeys 列出用户所有访问密钥
func (s *AccessKeyService) ListAccessKeys(ctx context.Context, userName string) ([]*model.AccessKey, error) {
	// 获取用户
	user, err := s.userStore.GetByName(userName)
	if err != nil {
		return nil, errors.NewBusinessError(errors.CodeUserNotFound, "user not found")
	}

	aks, err := s.accessKeyStore.ListByUser(user.ID)
	// 解密所有密钥
	for _, ak := range aks {
		if len(ak.EncryptedSecretAccessKey) > 0 && s.masterKey != "" {
			key, err := hex.DecodeString(s.masterKey)
			if err != nil {
				return nil, err
			}
			// 先进行base64解码
			ciphertext, err := base64.StdEncoding.DecodeString(ak.EncryptedSecretAccessKey)
			if err != nil {
				return nil, err
			}
			// 然后解密
			decryptedSecret, err := crypto.DecryptKey(ciphertext, key)
			if err != nil {
				return nil, err
			}
			ak.SecretAccessKey = string(decryptedSecret)
		}
	}
	return aks, err
}

// UpdateStatus 更新访问密钥状态
func (s *AccessKeyService) UpdateStatus(ctx context.Context, accessKeyID, status string) (*model.AccessKey, error) {
	// 验证状态值
	if status != "active" && status != "inactive" {
		return nil, errors.NewBusinessError(errors.CodeInvalidParameter, "status must be either 'active' or 'inactive'")
	}

	// 更新状态
	if err := s.accessKeyStore.UpdateStatus(accessKeyID, status); err != nil {
		return nil, fmt.Errorf("failed to update access key status: %w", err)
	}

	// 获取更新后的密钥信息
	ak, err := s.accessKeyStore.GetByAccessKeyID(accessKeyID, s.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated access key: %w", err)
	}

	return ak, nil
}

// UpdateAccessKeyStatus 更新访问密钥状态
func (s *AccessKeyService) UpdateAccessKeyStatus(ctx context.Context, accessKeyID, status string) (*model.AccessKey, error) {
	if status != "active" && status != "inactive" {
		return nil, errors.NewBusinessError(errors.CodeInvalidParameter, "invalid status")
	}

	// 更新状态
	if err := s.accessKeyStore.UpdateStatus(accessKeyID, status); err != nil {
		return nil, err
	}

	// 获取更新后的密钥
	return s.accessKeyStore.GetByAccessKeyID(accessKeyID, s.masterKey)
}

// RotateAccessKey 轮换访问密钥
func (s *AccessKeyService) RotateAccessKey(ctx context.Context, accessKeyID string) (*model.AccessKey, error) {
	return s.accessKeyStore.RotateKey(accessKeyID, s.masterKey)
}

// GetAccessKey 根据访问密钥ID获取访问密钥

func (s *AccessKeyService) GetAccessKey(ctx context.Context, accessKeyID string) (*model.AccessKey, error) {
	// 参数检查
	if accessKeyID == "" {
		return nil, errors.NewBusinessError(errors.CodeInvalidParameter, "access key ID cannot be empty")
	}

	// 从存储层获取密钥
	ak, err := s.accessKeyStore.GetByAccessKeyID(accessKeyID, s.masterKey)
	if err != nil {
		return nil, errors.NewBusinessError(errors.CodeAccessKeyNotFound, "access key not found")
	}

	// 获取关联用户信息
	user, err := s.userStore.GetByID(ak.UserID)
	if err != nil {
		return nil, errors.NewBusinessError(errors.CodeUserNotFound, "associated user not found")
	}

	// 设置用户名（非数据库字段）
	ak.UserName = user.Name

	return ak, nil
}

// DeleteAccessKey 删除访问密钥
// 参数:
//   - ctx: 上下文
//   - userName: 用户名
//   - accessKeyID: 访问密钥ID
// 返回值:
//   - error: 删除过程中的错误
func (s *AccessKeyService) DeleteAccessKey(ctx context.Context, userName string, accessKeyID string) error {
	// 获取用户
	user, err := s.userStore.GetByName(userName)
	if err != nil {
		return errors.NewBusinessError(errors.CodeUserNotFound, "user not found")
	}

	// 获取访问密钥以验证所有权
	ak, err := s.accessKeyStore.GetByAccessKeyID(accessKeyID, s.masterKey)
	if err != nil {
		return errors.NewBusinessError(errors.CodeAccessKeyNotFound, "access key not found")
	}

	// 检查访问密钥是否属于该用户
	if ak.UserID != user.ID {
		return errors.NewBusinessError(errors.CodePermissionDenied, "access key does not belong to user")
	}

	// 删除访问密钥
	if err := s.accessKeyStore.Delete(accessKeyID); err != nil {
		return errors.NewBusinessError(errors.CodeInternalError, "failed to delete access key")
	}

	return nil
}

// GetStore 返回访问密钥存储实现
func (s *AccessKeyService) GetStore() store.AccessKeyStore {
	return s.accessKeyStore
}

// GetMasterKey 获取主密钥
func (s *AccessKeyService) GetMasterKey() string {
	return s.masterKey
}

// SetRotationPolicy 设置轮换策略
func (s *AccessKeyService) SetRotationPolicy(policy *RotationPolicy) {
	s.rotationMutex.Lock()
	defer s.rotationMutex.Unlock()
	s.rotationPolicy = policy
}

// GetRotationPolicy 获取轮换策略
func (s *AccessKeyService) GetRotationPolicy() *RotationPolicy {
	s.rotationMutex.RLock()
	defer s.rotationMutex.RUnlock()
	return s.rotationPolicy
}

// ScheduleRotation 计划轮换
func (s *AccessKeyService) ScheduleRotation(accessKeyID string, rotationTime time.Time) {
	s.rotationMutex.Lock()
	defer s.rotationMutex.Unlock()
	s.rotationSchedule[accessKeyID] = rotationTime
	vgokit.Log.Info("Scheduled key rotation",
		zap.String("access_key_id", accessKeyID),
		zap.Time("rotation_time", rotationTime))
}

// CheckKeyHealth 检查密钥健康状态
func (s *AccessKeyService) CheckKeyHealth(ctx context.Context, accessKeyID string) (*KeyHealthStatus, error) {
	ak, err := s.accessKeyStore.GetByAccessKeyID(accessKeyID, s.masterKey)
	if err != nil {
		return nil, errors.NewBusinessError(errors.CodeAccessKeyNotFound, "access key not found")
	}

	now := time.Now()
	age := now.Sub(ak.CreatedAt)
	policy := s.GetRotationPolicy()

	status := &KeyHealthStatus{
		AccessKeyID:    accessKeyID,
		Age:            age,
		Status:         ak.Status,
		CreatedAt:      ak.CreatedAt,
		Healthy:        true,
		Warnings:       []string{},
		Recommendations: []string{},
	}

	// 检查是否需要轮换
	if policy.Enabled {
		if age > policy.MaxAge {
			status.Healthy = false
			status.Warnings = append(status.Warnings, "密钥已过期")
			status.Recommendations = append(status.Recommendations, "立即轮换密钥")
		} else if age > policy.MaxAge-policy.WarningThreshold {
			status.Warnings = append(status.Warnings, "密钥即将过期")
			status.Recommendations = append(status.Recommendations, "建议尽快轮换密钥")
		}
	}

	// 检查密钥状态
	if ak.Status != "active" {
		status.Healthy = false
		status.Warnings = append(status.Warnings, "密钥未激活")
	}

	// 检查轮换时间
	if ak.LastRotatedAt != nil && now.Sub(*ak.LastRotatedAt) > 30*24*time.Hour {
		status.Warnings = append(status.Warnings, "密钥长时间未轮换")
		status.Recommendations = append(status.Recommendations, "考虑轮换密钥")
	}

	return status, nil
}

// KeyHealthStatus 密钥健康状态
type KeyHealthStatus struct {
	AccessKeyID     string        `json:"access_key_id"`
	Age             time.Duration `json:"age"`
	Status          string        `json:"status"`
	CreatedAt       time.Time     `json:"created_at"`
	Healthy         bool          `json:"healthy"`
	Warnings        []string      `json:"warnings"`
	Recommendations []string      `json:"recommendations"`
}

// CheckAndRotateExpiredKeys 检查并轮换过期密钥
func (s *AccessKeyService) CheckAndRotateExpiredKeys(ctx context.Context) error {
	policy := s.GetRotationPolicy()
	if !policy.Enabled {
		return nil
	}

	// 获取所有访问密钥
	allKeys, err := s.accessKeyStore.ListAll()
	if err != nil {
		return err
	}

	now := time.Now()
	var rotatedCount, warningCount int

	for _, key := range allKeys {
		age := now.Sub(key.CreatedAt)
		
		// 检查是否需要自动轮换
		if policy.AutoRotate && age > policy.MaxAge {
			if _, err := s.RotateAccessKey(ctx, key.AccessKeyID); err != nil {
				vgokit.Log.Error("Failed to auto-rotate expired key",
					zap.String("access_key_id", key.AccessKeyID),
					zap.Error(err))
			} else {
				rotatedCount++
				vgokit.Log.Info("Auto-rotated expired key",
					zap.String("access_key_id", key.AccessKeyID))
			}
		} else if age > policy.MaxAge-policy.WarningThreshold {
			// 发送警告
			warningCount++
			vgokit.Log.Warn("Access key approaching expiration",
				zap.String("access_key_id", key.AccessKeyID),
				zap.Duration("age", age),
				zap.Duration("max_age", policy.MaxAge))
			
			// 这里可以添加邮件通知逻辑
			if policy.NotificationEmail != "" {
				s.sendExpirationWarning(key, policy.NotificationEmail)
			}
		}
	}

	vgokit.Log.Info("Key rotation check completed",
		zap.Int("rotated_count", rotatedCount),
		zap.Int("warning_count", warningCount))

	return nil
}

// sendExpirationWarning 发送过期警告（占位符实现）
func (s *AccessKeyService) sendExpirationWarning(key *model.AccessKey, email string) {
	// 这里可以集成邮件服务发送警告
	vgokit.Log.Info("Sending expiration warning",
		zap.String("access_key_id", key.AccessKeyID),
		zap.String("email", email))
}

// GetRotationSchedule 获取轮换计划
func (s *AccessKeyService) GetRotationSchedule() map[string]time.Time {
	s.rotationMutex.RLock()
	defer s.rotationMutex.RUnlock()
	schedule := make(map[string]time.Time)
	for k, v := range s.rotationSchedule {
		schedule[k] = v
	}
	return schedule
}

// ProcessScheduledRotations 处理计划的轮换
func (s *AccessKeyService) ProcessScheduledRotations(ctx context.Context) error {
	s.rotationMutex.Lock()
	defer s.rotationMutex.Unlock()

	now := time.Now()
	var processedKeys []string

	for accessKeyID, rotationTime := range s.rotationSchedule {
		if now.After(rotationTime) {
			if _, err := s.RotateAccessKey(ctx, accessKeyID); err != nil {
				vgokit.Log.Error("Failed to process scheduled rotation",
					zap.String("access_key_id", accessKeyID),
					zap.Error(err))
			} else {
				vgokit.Log.Info("Processed scheduled rotation",
					zap.String("access_key_id", accessKeyID))
				processedKeys = append(processedKeys, accessKeyID)
			}
		}
	}

	// 清理已处理的计划
	for _, keyID := range processedKeys {
		delete(s.rotationSchedule, keyID)
	}

	return nil
}
