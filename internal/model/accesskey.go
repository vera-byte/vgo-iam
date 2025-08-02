package model

import (
	"time"
)

// AccessKey 访问密钥模型
// 修改AccessKey结构体
type AccessKey struct {
	ID                 int       `json:"id"`
	UserID             int       `json:"user_id"`                   // 关联用户ID
	AccessKeyID        string    `json:"access_key_id"`             // 访问密钥ID
	SecretAccessKey    string    `json:"secret_access_key"`         // 密钥（仅创建时返回）
	EncryptedSecretKey []byte    `json:"encrypted_secret_key"`      // 加密后的密钥
	Status             string    `json:"status"`                    // 状态: active/inactive
	CreatedAt          time.Time `json:"created_at"`                // 创建时间
	UpdatedAt          time.Time `json:"updated_at"`                // 更新时间
	UserName           string    `json:"user_name,omitempty"`       // 用户名（非数据库字段，仅用于返回）
	ExpiresAt          time.Time `json:"expires_at,omitempty"`      // 添加过期时间
	LastRotatedAt      time.Time `json:"last_rotated_at,omitempty"` // 添加最后轮换时间
}

// 修改NewAccessKey方法
func NewAccessKey(userID int, accessKeyID, secretKey string) *AccessKey {
	return &AccessKey{
		UserID:          userID,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretKey,
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		ExpiresAt:       time.Now().AddDate(0, 3, 0), // 默认3个月过期
	}
}
