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

// GetPolicy 根据策略名称获取策略
func (s *PolicyService) GetPolicy(ctx context.Context, name string) (*model.Policy, error) {
	policy, err := s.policyStore.GetByName(name)
	if err != nil {
		// 对于策略查找，优先返回具体的策略不存在错误
		if bizErr := errors.HandleDBError(err); bizErr != nil {
			// 如果是 "暂无数据" 错误，转换为更具体的 "策略不存在" 错误
			if bizErr.Code == errors.CodeNoData {
				return nil, errors.NewBusinessError(errors.CodePolicyNotFound, "policy not found")
			}
			return nil, bizErr
		}
		return nil, errors.NewBusinessError(errors.CodePolicyNotFound, "policy not found")
	}
	return policy, nil
}

// DeletePolicy 删除策略
func (s *PolicyService) DeletePolicy(ctx context.Context, name string) error {
	// 检查策略是否存在
	policy, err := s.policyStore.GetByName(name)
	if err != nil {
		if bizErr := errors.HandleDBError(err); bizErr != nil {
			return bizErr
		}
		return errors.NewBusinessError(errors.CodePolicyNotFound, "policy not found")
	}

	// 删除策略
	if err := s.policyStore.Delete(policy.ID); err != nil {
		return errors.NewBusinessError(errors.CodeInternalError, "failed to delete policy")
	}

	return nil
}

// ListPolicies 列出所有策略
func (s *PolicyService) ListPolicies(ctx context.Context) ([]*model.Policy, error) {
	policies, err := s.policyStore.List()
	if err != nil {
		if bizErr := errors.HandleDBError(err); bizErr != nil {
			return nil, bizErr
		}
		return nil, errors.NewBusinessError(errors.CodeInternalError, "failed to list policies")
	}
	return policies, nil
}

// GetPoliciesCount 获取策略总数
// 返回系统中策略的总数量
func (s *PolicyService) GetPoliciesCount(ctx context.Context) (int64, error) {
	policies, err := s.policyStore.List()
	if err != nil {
		return 0, errors.NewBusinessError(errors.CodeInternalError, "failed to get policies count")
	}
	return int64(len(policies)), nil
}
