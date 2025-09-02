package model

import (
	"time"
)

// AccessKey 访问密钥模型
type AccessKey struct {
	ID                       int64     `json:"id" db:"id"`
	UserID                   int64     `json:"user_id" db:"user_id"`                     // 关联用户ID
	AccessKeyID              string    `json:"access_key_id" db:"access_key_id"`               // 访问密钥ID
	SecretAccessKey          string    `json:"secret_access_key" db:"-"`           // 密钥（仅创建时返回）
	EncryptedSecretAccessKey string    `json:"encrypted_secret_access_key" db:"encrypted_secret_access_key"` // 加密后的密钥
	Status                   string    `json:"status" db:"status"`                      // 状态: active/inactive
	AppID                    *int64    `json:"app_id" db:"app_id"`                       // 关联应用ID
	Description              string    `json:"description" db:"description"`             // 密钥描述
	CreatedAt                time.Time `json:"created_at" db:"created_at"`                  // 创建时间
	UpdatedAt                time.Time `json:"updated_at" db:"updated_at"`                  // 更新时间
	ExpiresAt                *time.Time `json:"expires_at,omitempty" db:"expires_at"`        // 过期时间
	LastRotatedAt            *time.Time `json:"last_rotated_at,omitempty" db:"last_rotated_at"`   // 最后轮换时间

	// 关联信息（非数据库字段）
	UserName string `json:"user_name,omitempty" db:"user_name"`         // 用户名
	AppName  string `json:"app_name,omitempty" db:"app_name"`           // 应用名称
}

// NewAccessKey 创建新的访问密钥
func NewAccessKey(userID int64, accessKeyID, secretKey string, appID *int64, description string) *AccessKey {
	now := time.Now()
	expiresAt := now.AddDate(0, 3, 0) // 默认3个月过期
	return &AccessKey{
		UserID:          userID,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretKey,
		Status:          "active",
		AppID:           appID,
		Description:     description,
		CreatedAt:       now,
		UpdatedAt:       now,
		ExpiresAt:       &expiresAt,
	}
}

// IsActive 是否为活跃状态
func (ak *AccessKey) IsActive() bool {
	return ak.Status == "active"
}

// IsInactive 是否为非活跃状态
func (ak *AccessKey) IsInactive() bool {
	return ak.Status == "inactive"
}

// IsExpired 是否已过期
func (ak *AccessKey) IsExpired() bool {
	if ak.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*ak.ExpiresAt)
}

// SetStatus 设置状态
func (ak *AccessKey) SetStatus(status string) {
	ak.Status = status
	ak.UpdatedAt = time.Now()
}

// Rotate 轮换密钥
func (ak *AccessKey) Rotate(newSecretKey string) {
	now := time.Now()
	ak.SecretAccessKey = newSecretKey
	ak.LastRotatedAt = &now
	ak.UpdatedAt = now
	// 延长过期时间
	expiresAt := now.AddDate(0, 3, 0)
	ak.ExpiresAt = &expiresAt
}
