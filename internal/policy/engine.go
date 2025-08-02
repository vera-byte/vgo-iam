package policy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/service"
)

// 修改PolicyEngine结构体
type PolicyEngine struct {
	userService *service.UserService
	cache       *cache.Cache // 添加缓存
	mu          sync.RWMutex
}

// 初始化缓存
func NewPolicyEngine(userService *service.UserService) *PolicyEngine {
	return &PolicyEngine{
		userService: userService,
		cache:       cache.New(5*time.Minute, 10*time.Minute), // 5分钟过期，10分钟清理
	}
}

// 修改Evaluate方法添加缓存逻辑
func (e *PolicyEngine) Evaluate(user *model.User, action, resource string) (bool, error) {
	// 创建缓存键
	cacheKey := fmt.Sprintf("%d:%s:%s", user.ID, action, resource)

	// 尝试从缓存获取
	e.mu.RLock()
	cachedResult, found := e.cache.Get(cacheKey)
	e.mu.RUnlock()
	if found {
		return cachedResult.(bool), nil
	}

	// 缓存未命中，执行实际评估
	policies, err := e.userService.GetUserPolicies(context.Background(), user.ID)
	if err != nil {
		return false, err
	}

	result := false
	for _, policy := range policies {
		allowed, match, err := e.evaluateSinglePolicy(policy, action, resource)
		if err != nil {
			return false, err
		}
		if match {
			result = allowed
			break
		}
	}

	// 存入缓存
	e.mu.Lock()
	e.cache.Set(cacheKey, result, cache.DefaultExpiration)
	e.mu.Unlock()

	return result, nil
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
