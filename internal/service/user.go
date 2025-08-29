package service

import (
	"context"
	"fmt"
	"time"

	"github.com/vera-byte/vgo-iam/internal/errors"
	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/store"
	"github.com/vera-byte/vgo-iam/internal/util"
	vgokit "github.com/vera-byte/vgo-kit"
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
		return nil, errors.NewBusinessError(errors.CodeInvalidUserName, "invalid username format")
	}
	if !util.ValidateEmail(email) {
		return nil, errors.NewBusinessError(errors.CodeInvalidEmail, "invalid email format")
	}
	if displayName == "" {
		return nil, errors.NewBusinessError(errors.CodeInvalidParameter, "display name cannot be empty")
	}

	// 检查用户是否已存在
	if _, err := s.userStore.GetByName(name); err == nil {
		return nil, errors.NewBusinessError(errors.CodeUserAlreadyExists, "username already exists")
	}
	if _, err := s.userStore.GetByEmail(email); err == nil {
		return nil, errors.NewBusinessError(errors.CodeUserAlreadyExists, "email already exists")
	}

	// 创建用户（不再需要密码）
	user := &model.User{
		Name:        name,
		DisplayName: displayName,
		Email:       email,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if userId, err := s.userStore.Create(user); err != nil {
		return nil, errors.NewBusinessError(errors.CodeInternalError, "failed to create user")
	} else {
		user.ID = userId
	}

	// 记录业务指标
	vgokit.Metrics.RecordBusinessMetric("user_created")

	// 缓存用户信息
	cacheKey := fmt.Sprintf("user:name:%s", name)
	vgokit.Cache.Set(ctx, cacheKey, user, 5*time.Minute)

	return user, nil
}

func (s *UserService) GetUserPolicies(ctx context.Context, userID int64) ([]*model.Policy, error) {
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
		return errors.NewBusinessError(errors.CodeUserNotFound, "user not found")
	}

	// 获取策略
	policy, err := s.policyStore.GetByName(policyName)
	if err != nil {
		return errors.NewBusinessError(errors.CodePolicyNotFound, "policy not found")
	}

	// 附加策略
	return s.userStore.AttachPolicy(user.ID, policy.ID)
}

// ListUserPolicies 列出用户所有策略
func (s *UserService) ListUserPolicies(ctx context.Context, userName string) ([]*model.Policy, error) {
	user, err := s.userStore.GetByName(userName)
	if err != nil {
		return nil, errors.NewBusinessError(errors.CodeUserNotFound, "user not found")
	}
	return s.userStore.ListPolicies(user.ID)
}

// UpdateUser 更新用户信息
// 参数: ctx - 上下文, userID - 用户ID, displayName - 显示名称, email - 邮箱地址
// 返回值: 更新后的用户信息, 错误信息
func (s *UserService) UpdateUser(ctx context.Context, userID int64, displayName, email string) (*model.User, error) {
	// 获取用户
	user, err := s.userStore.GetByID(userID)
	if err != nil {
		return nil, errors.NewBusinessError(errors.CodeUserNotFound, "user not found")
	}

	// 验证输入参数
	if displayName != "" {
		if len(displayName) == 0 {
			return nil, errors.NewBusinessError(errors.CodeInvalidParameter, "display name cannot be empty")
		}
		user.DisplayName = displayName
	}

	if email != "" {
		if !util.ValidateEmail(email) {
			return nil, errors.NewBusinessError(errors.CodeInvalidEmail, "invalid email format")
		}
		// 检查邮箱是否已被其他用户使用
		if existingUser, err := s.userStore.GetByEmail(email); err == nil && existingUser.ID != userID {
			return nil, errors.NewBusinessError(errors.CodeUserAlreadyExists, "email already exists")
		}
		user.Email = email
	}

	// 更新时间戳
	user.UpdatedAt = time.Now()

	// 执行更新
	if err := s.userStore.Update(user); err != nil {
		return nil, errors.NewBusinessError(errors.CodeInternalError, "failed to update user")
	}

	// 记录业务指标
	vgokit.Metrics.RecordBusinessMetric("user_updated")

	// 更新缓存
	cacheKey := fmt.Sprintf("user:name:%s", user.Name)
	vgokit.Cache.Set(ctx, cacheKey, user, 5*time.Minute)

	return user, nil
}

// DeleteUser 删除用户
// 参数: ctx - 上下文, userID - 用户ID
// 返回值: 错误信息
func (s *UserService) DeleteUser(ctx context.Context, userID int64) error {
	// 获取用户信息
	user, err := s.userStore.GetByID(userID)
	if err != nil {
		return errors.NewBusinessError(errors.CodeUserNotFound, "user not found")
	}

	// 删除用户
	if err := s.userStore.Delete(userID); err != nil {
		return errors.NewBusinessError(errors.CodeInternalError, "failed to delete user")
	}

	// 记录业务指标
	vgokit.Metrics.RecordBusinessMetric("user_deleted")

	// 清除缓存
	cacheKey := fmt.Sprintf("user:name:%s", user.Name)
	vgokit.Cache.Del(ctx, cacheKey)

	return nil
}

// ListUsers 获取用户列表
// 参数: ctx - 上下文
// 返回值: 用户列表, 错误信息
func (s *UserService) ListUsers(ctx context.Context) ([]*model.User, error) {
	users, err := s.userStore.List()
	if err != nil {
		return nil, errors.NewBusinessError(errors.CodeInternalError, "failed to list users")
	}

	// 记录业务指标
	vgokit.Metrics.RecordBusinessMetric("users_listed")

	return users, nil
}

// GetUserByID 根据ID获取用户
// 参数: ctx - 上下文, userID - 用户ID
// 返回值: 用户信息, 错误信息
func (s *UserService) GetUserByID(ctx context.Context, userID int64) (*model.User, error) {
	user, err := s.userStore.GetByID(userID)
	if err != nil {
		return nil, errors.NewBusinessError(errors.CodeUserNotFound, "user not found")
	}
	return user, nil
}

// UpdateUserPassword 设置用户密码
// 参数: ctx - 上下文, userID - 用户ID, password - 新密码
// 返回值: 错误信息
func (s *UserService) UpdateUserPassword(ctx context.Context, userID int64, password string) error {
	// 获取用户
	user, err := s.userStore.GetByID(userID)
	if err != nil {
		return errors.NewBusinessError(errors.CodeUserNotFound, "user not found")
	}

	// 验证密码强度
	if !util.ValidatePasswordStrength(password) {
		return errors.NewBusinessError(errors.CodeInvalidParameter, "password does not meet strength requirements")
	}

	// 生成密码哈希
	passwordHash, err := util.HashPassword(password)
	if err != nil {
		return errors.NewBusinessError(errors.CodeInternalError, "failed to hash password")
	}

	// 更新用户密码
	user.Password = passwordHash
	user.UpdatedAt = time.Now()
	return s.userStore.Update(user)
}
