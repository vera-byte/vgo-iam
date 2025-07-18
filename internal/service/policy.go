package service

import (
	"context"
	"errors"

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
		return nil, errors.New("invalid policy document")
	}

	// 检查策略是否已存在
	if _, err := s.policyStore.GetByName(name); err == nil {
		return nil, errors.New("policy already exists")
	}

	// 创建策略
	policy := model.NewPolicy(name, description, policyDocument)
	if err := s.policyStore.Create(policy); err != nil {
		return nil, err
	}

	return policy, nil
}

// UpdatePolicy 更新策略
func (s *PolicyService) UpdatePolicy(ctx context.Context, name, description, policyDocument string) (*model.Policy, error) {
	// 获取策略
	policy, err := s.policyStore.GetByName(name)
	if err != nil {
		return nil, errors.New("policy not found")
	}

	// 更新策略
	policy.Description = description
	policy.PolicyDocument = policyDocument
	if err := s.policyStore.Update(policy); err != nil {
		return nil, err
	}

	return policy, nil
}
