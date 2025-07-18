package model

import (
	"time"
)

// Policy 策略模型
type Policy struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`            // 策略名称（唯一）
	Description    string    `json:"description"`     // 策略描述
	PolicyDocument string    `json:"policy_document"` // JSON格式的策略文档
	CreatedAt      time.Time `json:"created_at"`      // 创建时间
	UpdatedAt      time.Time `json:"updated_at"`      // 更新时间
}

// NewPolicy 创建新策略
func NewPolicy(name, description, policyDocument string) *Policy {
	return &Policy{
		Name:           name,
		Description:    description,
		PolicyDocument: policyDocument,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}
