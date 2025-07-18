package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/store"
	"github.com/vera-byte/vgo-iam/internal/util"
)

// AccessKeyService 访问密钥服务
type AccessKeyService struct {
	accessKeyStore store.AccessKeyStore
	userStore      store.UserStore
	masterKey      []byte
}

// NewAccessKeyService 创建访问密钥服务实例
func NewAccessKeyService(accessKeyStore store.AccessKeyStore, userStore store.UserStore, masterKey []byte) *AccessKeyService {
	return &AccessKeyService{
		accessKeyStore: accessKeyStore,
		userStore:      userStore,
		masterKey:      masterKey,
	}
}

// CreateAccessKey 创建访问密钥
func (s *AccessKeyService) CreateAccessKey(ctx context.Context, userName string) (*model.AccessKey, error) {
	// 获取用户
	user, err := s.userStore.GetByName(userName)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// 生成密钥
	accessKeyID := util.GenerateAccessKeyID()
	secretKey := util.GenerateSecretAccessKey()

	// 创建访问密钥
	ak := model.NewAccessKey(user.ID, accessKeyID, secretKey)
	if err := s.accessKeyStore.Create(ak, s.masterKey); err != nil {
		return nil, fmt.Errorf("failed to create access key: %w", err)
	}

	return ak, nil
}

// ListAccessKeys 列出用户所有访问密钥
func (s *AccessKeyService) ListAccessKeys(ctx context.Context, userName string) ([]*model.AccessKey, error) {
	// 获取用户
	user, err := s.userStore.GetByName(userName)
	if err != nil {
		return nil, errors.New("user not found")
	}

	return s.accessKeyStore.ListByUser(user.ID)
}

// UpdateStatus 更新访问密钥状态
func (s *AccessKeyService) UpdateStatus(ctx context.Context, accessKeyID, status string) (*model.AccessKey, error) {
	// 验证状态值
	if status != "active" && status != "inactive" {
		return nil, errors.New("status must be either 'active' or 'inactive'")
	}

	// 更新状态
	if err := s.accessKeyStore.UpdateStatus(accessKeyID, status); err != nil {
		return nil, fmt.Errorf("failed to update access key status: %w", err)
	}

	// 获取更新后的密钥信息
	ak, err := s.accessKeyStore.GetByAccessKeyID(accessKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated access key: %w", err)
	}

	return ak, nil
}

// UpdateAccessKeyStatus 更新访问密钥状态
func (s *AccessKeyService) UpdateAccessKeyStatus(ctx context.Context, accessKeyID, status string) (*model.AccessKey, error) {
	if status != "active" && status != "inactive" {
		return nil, errors.New("invalid status")
	}

	// 更新状态
	if err := s.accessKeyStore.UpdateStatus(accessKeyID, status); err != nil {
		return nil, err
	}

	// 获取更新后的密钥
	return s.accessKeyStore.GetByAccessKeyID(accessKeyID)
}

// RotateAccessKey 轮换访问密钥
func (s *AccessKeyService) RotateAccessKey(ctx context.Context, accessKeyID string) (*model.AccessKey, error) {
	return s.accessKeyStore.RotateKey(accessKeyID, s.masterKey)
}

// GetAccessKey 根据访问密钥ID获取访问密钥

func (s *AccessKeyService) GetAccessKey(ctx context.Context, accessKeyID string) (*model.AccessKey, error) {
	// 参数检查
	if accessKeyID == "" {
		return nil, errors.New("access key ID cannot be empty")
	}

	// 从存储层获取密钥
	ak, err := s.accessKeyStore.GetByAccessKeyID(accessKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get access key: %w", err)
	}

	// 获取关联用户信息
	user, err := s.userStore.GetByID(ak.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get associated user: %w", err)
	}

	// 设置用户名（非数据库字段）
	ak.UserName = user.Name

	return ak, nil
}
