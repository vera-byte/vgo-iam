package model

import (
	"time"
)

// TemporaryCredential STS临时凭证模型
type TemporaryCredential struct {
	ID                        int64      `json:"id" db:"id"`                                                   // 凭证ID
	UserID                    int64      `json:"user_id" db:"user_id"`                                         // 用户ID
	AccessKeyID               string     `json:"access_key_id" db:"access_key_id"`                             // 访问密钥ID
	SecretAccessKey           string     `json:"secret_access_key" db:"-"`                                     // 密钥（敏感信息，不直接存储）
	EncryptedSecretAccessKey  string     `json:"-" db:"encrypted_secret_access_key"`                          // 加密的密钥
	SessionToken              string     `json:"session_token" db:"-"`                                         // 会话令牌（敏感信息，不直接存储）
	EncryptedSessionToken     string     `json:"-" db:"encrypted_session_token"`                              // 加密的会话令牌
	Status                    string     `json:"status" db:"status"`                                           // 状态：active, expired, revoked
	TokenType                 string     `json:"token_type" db:"token_type"`                                   // 令牌类型：session_token, assume_role
	RoleArn                   string     `json:"role_arn,omitempty" db:"role_arn"`                              // 角色ARN（AssumeRole时使用）
	RoleSessionName           string     `json:"role_session_name,omitempty" db:"role_session_name"`           // 角色会话名称
	ExternalID                string     `json:"external_id,omitempty" db:"external_id"`                       // 外部ID
	SessionPolicy             string     `json:"session_policy,omitempty" db:"session_policy"`                 // 会话策略
	Tags                      string     `json:"tags,omitempty" db:"tags"`                                     // 标签（JSON格式）
	DurationSeconds           int32      `json:"duration_seconds" db:"duration_seconds"`                       // 有效期（秒）
	CreatedAt                 time.Time  `json:"created_at" db:"created_at"`                                   // 创建时间
	ExpiresAt                 time.Time  `json:"expires_at" db:"expires_at"`                                   // 过期时间
	RevokedAt                 *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`                         // 撤销时间
}

// TokenType 令牌类型常量
const (
	TokenTypeSession    = "session"     // 会话令牌
	TokenTypeAssumeRole = "assume_role" // 角色假设令牌
)

// CredentialStatus 凭证状态常量
const (
	CredentialStatusActive  = "active"  // 活跃状态
	CredentialStatusExpired = "expired" // 已过期
	CredentialStatusRevoked = "revoked" // 已撤销
)

// NewSessionToken 创建新的会话令牌
// userName: 用户名
// accessKeyID: 临时访问密钥ID
// secretKey: 临时密钥
// sessionToken: 会话令牌
// durationSeconds: 有效期（秒）
// 返回: 临时凭证实例
func NewSessionToken(userID int64, accessKeyID, secretKey, sessionToken string, durationSeconds int32) *TemporaryCredential {
	now := time.Now()
	return &TemporaryCredential{
		UserID:          userID,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretKey,
		SessionToken:    sessionToken,
		Status:          CredentialStatusActive,
		TokenType:       TokenTypeSession,
		DurationSeconds: durationSeconds,
		CreatedAt:       now,
		ExpiresAt:       now.Add(time.Duration(durationSeconds) * time.Second),
	}
}

// NewAssumeRoleToken 创建新的角色假设令牌
// userID: 用户ID
// accessKeyID: 临时访问密钥ID
// secretKey: 临时密钥
// sessionToken: 会话令牌
// roleArn: 角色ARN
// roleSessionName: 角色会话名称
// durationSeconds: 有效期（秒）
// externalID: 外部ID（可选）
// sessionPolicy: 会话策略（可选）
// tags: 会话标签（可选）
// 返回: 临时凭证实例
func NewAssumeRoleToken(userID int64, accessKeyID, secretKey, sessionToken, roleArn, roleSessionName string, durationSeconds int32, externalID, sessionPolicy, tags *string) *TemporaryCredential {
	now := time.Now()
	tc := &TemporaryCredential{
		UserID:          userID,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretKey,
		SessionToken:    sessionToken,
		Status:          CredentialStatusActive,
		TokenType:       TokenTypeAssumeRole,
		RoleArn:         roleArn,
		RoleSessionName: roleSessionName,
		DurationSeconds: durationSeconds,
		CreatedAt:       now,
		ExpiresAt:       now.Add(time.Duration(durationSeconds) * time.Second),
	}

	// 处理可选字段
	if externalID != nil {
		tc.ExternalID = *externalID
	}
	if sessionPolicy != nil {
		tc.SessionPolicy = *sessionPolicy
	}
	if tags != nil {
		tc.Tags = *tags
	}

	return tc
}

// IsActive 检查凭证是否为活跃状态
// 返回: true表示活跃，false表示非活跃
func (tc *TemporaryCredential) IsActive() bool {
	return tc.Status == CredentialStatusActive
}

// IsExpired 检查凭证是否已过期
// 返回: true表示已过期，false表示未过期
func (tc *TemporaryCredential) IsExpired() bool {
	return time.Now().After(tc.ExpiresAt) || tc.Status == CredentialStatusExpired
}

// IsRevoked 检查凭证是否已撤销
// 返回: true表示已撤销，false表示未撤销
func (tc *TemporaryCredential) IsRevoked() bool {
	return tc.Status == CredentialStatusRevoked
}

// IsValid 检查凭证是否有效（活跃且未过期且未撤销）
// 返回: true表示有效，false表示无效
func (tc *TemporaryCredential) IsValid() bool {
	return tc.IsActive() && !tc.IsExpired() && !tc.IsRevoked()
}

// Revoke 撤销凭证
// 功能: 将凭证状态设置为已撤销，并记录撤销时间
func (tc *TemporaryCredential) Revoke() {
	now := time.Now()
	tc.Status = CredentialStatusRevoked
	tc.RevokedAt = &now
}

// MarkExpired 标记凭证为已过期
// 功能: 将凭证状态设置为已过期
func (tc *TemporaryCredential) MarkExpired() {
	tc.Status = CredentialStatusExpired
}

// Refresh 刷新凭证有效期
// newDurationSeconds: 新的有效期（秒）
// 功能: 延长凭证的有效期
func (tc *TemporaryCredential) Refresh(newDurationSeconds int32) {
	now := time.Now()
	tc.DurationSeconds = newDurationSeconds
	tc.ExpiresAt = now.Add(time.Duration(newDurationSeconds) * time.Second)
	if tc.Status == CredentialStatusExpired {
		tc.Status = CredentialStatusActive
	}
}

// GetRemainingSeconds 获取剩余有效时间（秒）
// 返回: 剩余秒数，如果已过期则返回0
func (tc *TemporaryCredential) GetRemainingSeconds() int64 {
	if tc.IsExpired() {
		return 0
	}
	remaining := tc.ExpiresAt.Sub(time.Now()).Seconds()
	if remaining < 0 {
		return 0
	}
	return int64(remaining)
}