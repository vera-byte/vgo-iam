package store

import (
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/gocraft/dbr/v2"
	"github.com/vera-byte/vgo-iam/internal/crypto"
	"github.com/vera-byte/vgo-iam/internal/model"
)

// TemporaryCredentialStore 临时凭证存储接口
type TemporaryCredentialStore interface {
	// Create 创建临时凭证
	// tc: 临时凭证实例
	// masterKey: 主密钥用于加密
	// 返回: 错误信息
	Create(tc *model.TemporaryCredential, masterKey string) error

	// GetByID 根据ID获取临时凭证
	// id: 凭证ID
	// 返回: 临时凭证实例和错误信息
	GetByID(id int64) (*model.TemporaryCredential, error)

	// GetByAccessKeyID 根据访问密钥ID获取临时凭证
	// accessKeyID: 访问密钥ID
	// masterKey: 主密钥用于解密
	// 返回: 临时凭证实例和错误信息
	GetByAccessKeyID(accessKeyID string, masterKey string) (*model.TemporaryCredential, error)

	// GetBySessionToken 根据会话令牌获取临时凭证
	// sessionToken: 会话令牌
	// masterKey: 主密钥用于解密
	// 返回: 临时凭证实例和错误信息
	GetBySessionToken(sessionToken string, masterKey string) (*model.TemporaryCredential, error)

	// ListByUser 获取用户的所有临时凭证
	// userID: 用户ID
	// 返回: 临时凭证列表和错误信息
	ListByUser(userID int64) ([]*model.TemporaryCredential, error)

	// ListActive 获取所有活跃的临时凭证
	// 返回: 活跃临时凭证列表和错误信息
	ListActive() ([]*model.TemporaryCredential, error)

	// ListExpired 获取所有过期的临时凭证
	// 返回: 过期临时凭证列表和错误信息
	ListExpired() ([]*model.TemporaryCredential, error)

	// UpdateStatus 更新凭证状态
	// id: 凭证ID
	// status: 新状态
	// 返回: 错误信息
	UpdateStatus(id int64, status string) error

	// Revoke 撤销凭证
	// id: 凭证ID
	// 返回: 错误信息
	Revoke(id int64) error

	// RevokeBySessionToken 根据会话令牌撤销凭证
	// sessionToken: 会话令牌
	// masterKey: 主密钥
	// 返回: 错误信息
	RevokeBySessionToken(sessionToken string, masterKey string) error

	// Refresh 刷新凭证有效期
	// id: 凭证ID
	// newDurationSeconds: 新的有效期（秒）
	// 返回: 更新后的临时凭证实例和错误信息
	Refresh(id int64, newDurationSeconds int32) (*model.TemporaryCredential, error)

	// CleanupExpired 清理过期的凭证
	// 返回: 清理的凭证数量和错误信息
	CleanupExpired() (int64, error)

	// Delete 删除凭证
	// id: 凭证ID
	// 返回: 错误信息
	Delete(id int64) error
}

// temporaryCredentialStore 临时凭证存储实现
type temporaryCredentialStore struct {
	session *dbr.Session
}

// NewTemporaryCredentialStore 创建临时凭证存储实例
// session: 数据库会话
// 返回: 临时凭证存储接口实例
func NewTemporaryCredentialStore(session *dbr.Session) TemporaryCredentialStore {
	return &temporaryCredentialStore{session: session}
}

// Create 创建临时凭证
func (s *temporaryCredentialStore) Create(tc *model.TemporaryCredential, masterKey string) error {
	// 加密密钥和会话令牌
	key, err := hex.DecodeString(masterKey)
	if err != nil {
		return err
	}

	// 加密SecretAccessKey
	encryptedSecret, err := crypto.EncryptKey([]byte(tc.SecretAccessKey), key)
	if err != nil {
		return err
	}

	// 加密SessionToken
	encryptedSessionToken, err := crypto.EncryptKey([]byte(tc.SessionToken), key)
	if err != nil {
		return err
	}

	// 使用PostgreSQL的RETURNING子句获取插入的ID
	var id int64
	err = s.session.InsertInto("temporary_credentials").
		Columns(
			"user_id",
			"access_key_id",
			"encrypted_secret_access_key",
			"encrypted_session_token",
			"status",
			"token_type",
			"role_arn",
			"role_session_name",
			"external_id",
			"session_policy",
			"tags",
			"duration_seconds",
			"created_at",
			"expires_at",
		).
		Values(
			tc.UserID,
			tc.AccessKeyID,
			base64.StdEncoding.EncodeToString(encryptedSecret),
			base64.StdEncoding.EncodeToString(encryptedSessionToken),
			tc.Status,
			tc.TokenType,
			tc.RoleArn,
			tc.RoleSessionName,
			tc.ExternalID,
			tc.SessionPolicy,
			tc.Tags,
			tc.DurationSeconds,
			tc.CreatedAt,
			tc.ExpiresAt,
		).
		Returning("id").
		Load(&id)

	if err != nil {
		return err
	}

	tc.ID = id
	return nil
}

// GetByID 根据ID获取临时凭证
func (s *temporaryCredentialStore) GetByID(id int64) (*model.TemporaryCredential, error) {
	var tc model.TemporaryCredential
	err := s.session.Select("*").
		From("temporary_credentials").
		Where("id = ?", id).
		LoadOne(&tc)

	return &tc, err
}

// GetByAccessKeyID 根据访问密钥ID获取临时凭证
func (s *temporaryCredentialStore) GetByAccessKeyID(accessKeyID string, masterKey string) (*model.TemporaryCredential, error) {
	var tc model.TemporaryCredential
	err := s.session.Select("*").
		From("temporary_credentials").
		Where("access_key_id = ?", accessKeyID).
		LoadOne(&tc)
	if err != nil {
		return nil, err
	}

	// 解密密钥和会话令牌
	if err := s.decryptCredential(&tc, masterKey); err != nil {
		return nil, err
	}

	return &tc, nil
}

// GetBySessionToken 根据会话令牌获取临时凭证
func (s *temporaryCredentialStore) GetBySessionToken(sessionToken string, masterKey string) (*model.TemporaryCredential, error) {
	// 由于AES-GCM加密每次都会产生不同的密文（随机nonce），
	// 我们不能直接比较加密后的token，需要获取所有活跃凭证然后逐一解密比较
	var tcs []*model.TemporaryCredential
	_, err := s.session.Select("*").
		From("temporary_credentials").
		Where("status = ? AND expires_at > ?", model.CredentialStatusActive, time.Now()).
		Load(&tcs)
	if err != nil {
		return nil, err
	}

	// 逐一解密并比较SessionToken
	for _, tc := range tcs {
		if err := s.decryptCredential(tc, masterKey); err != nil {
			continue // 解密失败，跳过这个凭证
		}
		if tc.SessionToken == sessionToken {
			return tc, nil
		}
	}

	return nil, dbr.ErrNotFound
}

// ListByUser 获取用户的所有临时凭证
func (s *temporaryCredentialStore) ListByUser(userID int64) ([]*model.TemporaryCredential, error) {
	var tcs []*model.TemporaryCredential
	_, err := s.session.Select("*").
		From("temporary_credentials").
		Where("user_id = ?", userID).
		OrderBy("created_at DESC").
		Load(&tcs)
	return tcs, err
}

// ListActive 获取所有活跃的临时凭证
func (s *temporaryCredentialStore) ListActive() ([]*model.TemporaryCredential, error) {
	var tcs []*model.TemporaryCredential
	_, err := s.session.Select("*").
		From("temporary_credentials").
		Where("status = ? AND expires_at > ?", model.CredentialStatusActive, time.Now()).
		Load(&tcs)
	return tcs, err
}

// ListExpired 获取所有过期的临时凭证
func (s *temporaryCredentialStore) ListExpired() ([]*model.TemporaryCredential, error) {
	var tcs []*model.TemporaryCredential
	_, err := s.session.Select("*").
		From("temporary_credentials").
		Where("expires_at <= ? OR status = ?", time.Now(), model.CredentialStatusExpired).
		Load(&tcs)
	return tcs, err
}

// UpdateStatus 更新凭证状态
func (s *temporaryCredentialStore) UpdateStatus(id int64, status string) error {
	updateBuilder := s.session.Update("temporary_credentials").
		Set("status", status)

	if status == model.CredentialStatusRevoked {
		updateBuilder = updateBuilder.Set("revoked_at", time.Now())
	}

	_, err := updateBuilder.Where("id = ?", id).Exec()
	return err
}

// Revoke 撤销凭证
func (s *temporaryCredentialStore) Revoke(id int64) error {
	return s.UpdateStatus(id, model.CredentialStatusRevoked)
}

// RevokeBySessionToken 根据会话令牌撤销凭证
func (s *temporaryCredentialStore) RevokeBySessionToken(sessionToken string, masterKey string) error {
	// 由于AES-GCM加密每次都会产生不同的密文（随机nonce），
	// 我们需要先通过GetBySessionToken找到凭证，然后通过ID撤销
	tc, err := s.GetBySessionToken(sessionToken, masterKey)
	if err != nil {
		return err
	}

	_, err = s.session.Update("temporary_credentials").
		Set("status", model.CredentialStatusRevoked).
		Set("revoked_at", time.Now()).
		Where("id = ?", tc.ID).
		Exec()
	return err
}

// Refresh 刷新凭证有效期
func (s *temporaryCredentialStore) Refresh(id int64, newDurationSeconds int32) (*model.TemporaryCredential, error) {
	now := time.Now()
	newExpiresAt := now.Add(time.Duration(newDurationSeconds) * time.Second)

	_, err := s.session.Update("temporary_credentials").
		Set("duration_seconds", newDurationSeconds).
		Set("expires_at", newExpiresAt).
		Set("status", model.CredentialStatusActive). // 重新激活
		Where("id = ?", id).
		Exec()

	if err != nil {
		return nil, err
	}

	// 返回更新后的凭证
	return s.GetByID(id)
}

// CleanupExpired 清理过期的凭证
// 返回: 清理的凭证数量和错误信息
func (s *temporaryCredentialStore) CleanupExpired() (int64, error) {
	// 删除已过期的凭证（expires_at <= 当前时间）
	// 不管状态如何，只要过期就删除
	result, err := s.session.DeleteFrom("temporary_credentials").
		Where("expires_at <= ?", time.Now()).
		Exec()

	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	return rowsAffected, err
}

// Delete 删除凭证
func (s *temporaryCredentialStore) Delete(id int64) error {
	_, err := s.session.DeleteFrom("temporary_credentials").
		Where("id = ?", id).
		Exec()
	return err
}

// decryptCredential 解密凭证中的敏感信息
// tc: 临时凭证实例
// masterKey: 主密钥
// 返回: 错误信息
func (s *temporaryCredentialStore) decryptCredential(tc *model.TemporaryCredential, masterKey string) error {
	if masterKey == "" {
		return nil // 如果没有主密钥，跳过解密
	}

	key, err := hex.DecodeString(masterKey)
	if err != nil {
		return err
	}

	// 解密SecretAccessKey
	if tc.EncryptedSecretAccessKey != "" {
		ciphertext, err := base64.StdEncoding.DecodeString(tc.EncryptedSecretAccessKey)
		if err != nil {
			return err
		}
		decryptedSecret, err := crypto.DecryptKey(ciphertext, key)
		if err != nil {
			return err
		}
		tc.SecretAccessKey = string(decryptedSecret)
	}

	// 解密SessionToken
	if tc.EncryptedSessionToken != "" {
		ciphertext, err := base64.StdEncoding.DecodeString(tc.EncryptedSessionToken)
		if err != nil {
			return err
		}
		decryptedToken, err := crypto.DecryptKey(ciphertext, key)
		if err != nil {
			return err
		}
		tc.SessionToken = string(decryptedToken)
	}

	return nil
}