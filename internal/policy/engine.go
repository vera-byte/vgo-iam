package policy

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/service"
)

// PolicyEngine 策略评估引擎，负责权限检查的核心逻辑
type PolicyEngine struct {
	userService *service.UserService // 用户服务用于获取用户关联的策略
}

// NewPolicyEngine 创建新的策略引擎实例
func NewPolicyEngine(userService *service.UserService) *PolicyEngine {
	return &PolicyEngine{
		userService: userService,
	}
}

// Evaluate 评估用户是否有权限执行指定操作
// 参数:
// - user: 要检查的用户对象
// - action: 要执行的操作 (如 "ecs:StartInstance")
// - resource: 要访问的资源ARN (如 "acs:ecs:cn-beijing:123456:instance/i-123456")
// 返回值:
// - bool: 是否允许操作
// - error: 检查过程中发生的错误
func (e *PolicyEngine) Evaluate(user *model.User, action, resource string) (bool, error) {
	// 1. 获取用户关联的所有策略
	policies, err := e.userService.GetUserPolicies(context.Background(), user.ID)
	if err != nil {
		return false, err
	}

	// 2. 检查每个策略是否允许该操作
	for _, policy := range policies {
		allowed, match, err := e.evaluateSinglePolicy(policy, action, resource)
		if err != nil {
			return false, err
		}
		if match {
			return allowed, nil
		}
	}

	// 3. 默认拒绝原则：没有明确允许即视为拒绝
	return false, nil
}

// evaluateSinglePolicy 评估单个策略是否允许指定操作
func (e *PolicyEngine) evaluateSinglePolicy(policy *model.Policy, action, resource string) (allowed, match bool, err error) {
	// 1. 解析策略文档
	var policyDoc model.PolicyDocument
	if err := json.Unmarshal([]byte(policy.PolicyDocument), &policyDoc); err != nil {
		return false, false, errors.New("invalid policy document format")
	}

	// 2. 检查策略中的每个Statement
	for _, statement := range policyDoc.Statement {
		// 检查资源是否匹配
		if !e.matchResource(statement.Resource, resource) {
			continue
		}

		// 检查操作是否匹配
		if !e.matchAction(statement.Action, action) {
			continue
		}

		// 如果匹配，根据Effect返回结果
		return statement.Effect == "Allow", true, nil
	}

	return false, false, nil
}

// matchAction 检查请求的操作是否匹配策略中的操作模式
func (e *PolicyEngine) matchAction(patterns []string, action string) bool {
	for _, pattern := range patterns {
		// 支持通配符 * 匹配所有操作
		if pattern == "*" {
			return true
		}

		// 精确匹配
		if pattern == action {
			return true
		}

		// 支持service:*格式的通配符
		if strings.HasSuffix(pattern, ":*") {
			servicePrefix := strings.TrimSuffix(pattern, ":*")
			if strings.HasPrefix(action, servicePrefix+":") {
				return true
			}
		}
	}
	return false
}

// matchResource 检查请求的资源是否匹配策略中的资源模式
func (e *PolicyEngine) matchResource(patterns []string, resource string) bool {
	for _, pattern := range patterns {
		// 支持通配符 * 匹配所有资源
		if pattern == "*" {
			return true
		}

		// 精确匹配
		if pattern == resource {
			return true
		}

		// 支持ARN通配符匹配 (如 "acs:ecs:*:*:instance/*")
		if e.matchARNPattern(pattern, resource) {
			return true
		}
	}
	return false
}

// matchARNPattern 实现ARN格式的通配符匹配
func (e *PolicyEngine) matchARNPattern(pattern, arn string) bool {
	patternParts := strings.Split(pattern, ":")
	arnParts := strings.Split(arn, ":")

	// ARN格式必须分段匹配
	if len(patternParts) != len(arnParts) {
		return false
	}

	for i := 0; i < len(patternParts); i++ {
		// 当前段允许通配
		if patternParts[i] == "*" {
			continue
		}

		// 当前段需要精确匹配
		if patternParts[i] != arnParts[i] {
			return false
		}
	}

	return true
}
