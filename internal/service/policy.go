package service

import (
	"context"

	"github.com/vera-byte/vgo-iam/internal/errors"
	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/store"
	"github.com/vera-byte/vgo-iam/internal/util"
)

// PolicyService 策略服务
type PolicyService struct {
	policyStore store.PolicyStore
}

// NewPolicyService 创建策略服务实例
func NewPolicyService(policyStore store.PolicyStore) *PolicyService {
	return &PolicyService{policyStore: policyStore}
}

// CreatePolicy 创建策略
func (s *PolicyService) CreatePolicy(ctx context.Context, name, description, policyDocument string) (*model.Policy, error) {
	// 验证输入
	if !util.ValidatePolicyDocument(policyDocument) {
		return nil, errors.NewBusinessError(errors.CodeInvalidPolicy, "invalid policy document")
	}

	// 检查策略是否已存在
	if _, err := s.policyStore.GetByName(name); err == nil {
		return nil, errors.NewBusinessError(errors.CodePolicyAlreadyExists, "policy already exists")
	}

	// 创建策略
	policy := model.NewPolicy(name, description, policyDocument)
	if err := s.policyStore.Create(policy); err != nil {
		return nil, errors.NewBusinessError(errors.CodeInternalError, "failed to create policy")
	}

	return policy, nil
}

// GetStore 返回策略存储
func (s *PolicyService) GetStore() store.PolicyStore {
	return s.policyStore
}

// UpdatePolicy 更新策略
func (s *PolicyService) UpdatePolicy(ctx context.Context, name, description, policyDocument string) (*model.Policy, error) {
	// 获取策略
	policy, err := s.policyStore.GetByName(name)
	if err != nil {
		return nil, errors.NewBusinessError(errors.CodePolicyNotFound, "policy not found")
	}

	// 更新策略
	policy.Description = description
	policy.PolicyDocument = policyDocument
	if err := s.policyStore.Update(policy); err != nil {
		return nil, errors.NewBusinessError(errors.CodeInternalError, "failed to update policy")
	}

	return policy, nil
}
