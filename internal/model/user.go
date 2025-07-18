package model

import (
	"time"
)

// User 用户模型
type User struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`         // 用户名（唯一）
	DisplayName string    `json:"display_name"` // 显示名称
	Email       string    `json:"email"`        // 邮箱（唯一）
	Password    string    `json:"-"`            // 密码（不导出）
	CreatedAt   time.Time `json:"created_at"`   // 创建时间
	UpdatedAt   time.Time `json:"updated_at"`   // 更新时间
}

// PolicyDocument 策略文档结构
type PolicyDocument struct {
	Version   string      `json:"version"`
	Statement []Statement `json:"statement"`
}

// Statement 策略语句
type Statement struct {
	Effect   string   `json:"effect"`   // Allow/Deny
	Action   []string `json:"action"`   // 操作列表
	Resource []string `json:"resource"` // 资源列表
}

// NewUser 创建新用户
func NewUser(name, displayName, email, password string) *User {
	return &User{
		Name:        name,
		DisplayName: displayName,
		Email:       email,
		Password:    password,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
