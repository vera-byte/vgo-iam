package service

import (
	"context"
	"errors"
	"time"

	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/store"
	"github.com/vera-byte/vgo-iam/internal/util"
)

// UserService 用户服务
type UserService struct {
	userStore   store.UserStore
	policyStore store.PolicyStore
}

// NewUserService 创建用户服务实例
func NewUserService(userStore store.UserStore, policyStore store.PolicyStore) *UserService {
	return &UserService{
		userStore:   userStore,
		policyStore: policyStore,
	}
}

func (s *UserService) CreateUser(ctx context.Context, name, displayName, email string) (*model.User, error) {
	// 验证输入
	if !util.ValidateUserName(name) {
		return nil, errors.New("invalid username format")
	}
	if !util.ValidateEmail(email) {
		return nil, errors.New("invalid email format")
	}

	// 检查用户是否已存在
	if _, err := s.userStore.GetByName(name); err == nil {
		return nil, errors.New("username already exists")
	}
	if _, err := s.userStore.GetByEmail(email); err == nil {
		return nil, errors.New("email already exists")
	}

	// 创建用户（不再需要密码）
	user := &model.User{
		Name:        name,
		DisplayName: displayName,
		Email:       email,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.userStore.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetUserPolicies(ctx context.Context, userID int) ([]*model.Policy, error) {
	return s.userStore.ListPolicies(userID)
}
func (s *UserService) GetUser(ctx context.Context, name string) (*model.User, error) {
	return s.userStore.GetByName(name)
}

// AttachPolicy 为用户附加策略
func (s *UserService) AttachPolicy(ctx context.Context, userName, policyName string) error {
	// 获取用户
	user, err := s.userStore.GetByName(userName)
	if err != nil {
		return errors.New("user not found")
	}

	// 获取策略
	policy, err := s.policyStore.GetByName(policyName)
	if err != nil {
		return errors.New("policy not found")
	}

	// 附加策略
	return s.userStore.AttachPolicy(user.ID, policy.ID)
}

// ListUserPolicies 列出用户所有策略
func (s *UserService) ListUserPolicies(ctx context.Context, userName string) ([]*model.Policy, error) {
	user, err := s.userStore.GetByName(userName)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return s.userStore.ListPolicies(user.ID)
}
