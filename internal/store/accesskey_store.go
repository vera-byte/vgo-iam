package store

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/gocraft/dbr/v2"
	"github.com/vera-byte/vgo-iam/internal/crypto"
	"github.com/vera-byte/vgo-iam/internal/model"
)

// AccessKeyStore 访问密钥存储接口
type AccessKeyStore interface {
	Create(ak *model.AccessKey, masterKey string) error
	GetByID(id int64) (*model.AccessKey, error)
	GetByAccessKeyID(accessKeyID string, masterKey string) (*model.AccessKey, error)
	ListByUser(userID int64) ([]*model.AccessKey, error)
	ListAll() ([]*model.AccessKey, error)
	UpdateStatus(accessKeyID, status string) error
	RotateKey(accessKeyID string, masterKey string) (*model.AccessKey, error)
}

// accessKeyStore 访问密钥存储实现
type accessKeyStore struct {
	session *dbr.Session
}

// NewAccessKeyStore 创建访问密钥存储实例
func NewAccessKeyStore(session *dbr.Session) AccessKeyStore {
	return &accessKeyStore{session: session}
}

func (s *accessKeyStore) Create(ak *model.AccessKey, masterKey string) error {
	// 加密密钥
	key, err := hex.DecodeString(masterKey)
	if err != nil {
		return err
	}
	encryptedSecret, err := crypto.EncryptKey([]byte(ak.SecretAccessKey), key)
	if err != nil {
		return err
	}

	insertBuilder := s.session.InsertInto("access_keys").
		Columns(
			"user_id",
			"access_key_id",
			"encrypted_secret_access_key",
			"status",
			"app_id",
			"description",
			"created_at",
			"updated_at",
		).
		Values(
			ak.UserID,
			ak.AccessKeyID,
			base64.StdEncoding.EncodeToString(encryptedSecret),
			ak.Status,
			ak.AppID,
			ak.Description,
			ak.CreatedAt,
			ak.UpdatedAt,
		)

	result, err := insertBuilder.Exec()
	if err != nil {
		return err
	}

	// 获取插入后的ID
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	ak.ID = id

	return nil
}

func (s *accessKeyStore) GetByID(id int64) (*model.AccessKey, error) {
	var ak model.AccessKey
	err := s.session.Select("*").
		From("access_keys").
		Where("id = ?", id).
		LoadOne(&ak)

	return &ak, err
}

func (s *accessKeyStore) GetByAccessKeyID(accessKeyID string, masterKey string) (*model.AccessKey, error) {
	var ak model.AccessKey
	err := s.session.Select(
		"id",
		"user_id",
		"access_key_id",
		"status",
		"created_at",
		"updated_at",
		"encrypted_secret_access_key",
	).From("access_keys").
		Where("access_key_id = ?", accessKeyID).
		LoadOne(&ak)
	if err != nil {
		return nil, err
	}

	// 检查加密密钥是否为空
	if len(ak.EncryptedSecretAccessKey) == 0 {
		return nil, errors.New("encrypted secret access key is empty")
	}

	// 解密密钥
	if masterKey != "" {
		key, err := hex.DecodeString(masterKey)
		if err != nil {
			return nil, err
		}
		ciphertext, err := base64.StdEncoding.DecodeString(ak.EncryptedSecretAccessKey)
		if err != nil {
			return nil, err
		}
		decryptedSecret, err := crypto.DecryptKey(ciphertext, key)
		if err != nil {
			return nil, err
		}
		ak.SecretAccessKey = string(decryptedSecret)
	}

	return &ak, nil
}
func (s *accessKeyStore) ListByUser(userID int64) ([]*model.AccessKey, error) {
	var aks []*model.AccessKey
	_, err := s.session.Select("*").
		From("access_keys").
		Where("user_id = ?", userID).
		Load(&aks)
	return aks, err
}

func (s *accessKeyStore) UpdateStatus(accessKeyID, status string) error {
	_, err := s.session.Update("access_keys").
		Set("status", status).
		Set("updated_at", time.Now()).
		Where("access_key_id = ?", accessKeyID).
		Exec()
	return err
}

func (s *accessKeyStore) RotateKey(accessKeyID string, masterKey string) (*model.AccessKey, error) {
	// 1. 获取现有密钥
	ak, err := s.GetByAccessKeyID(accessKeyID, masterKey)
	if err != nil {
		return nil, err
	}

	// 2. 生成新密钥
	newSecret := generateRandomSecret()

	// 3. 加密新密钥
	key, err := hex.DecodeString(masterKey)
	if err != nil {
		return nil, err
	}
	encryptedSecret, err := crypto.EncryptKey([]byte(newSecret), key)
	if err != nil {
		return nil, err
	}

	// 4. 更新数据库
	_, err = s.session.Update("access_keys").
		Set("encrypted_secret_access_key", encryptedSecret).
		Set("updated_at", time.Now()).
		Where("access_key_id = ?", accessKeyID).
		Exec()

	if err != nil {
		return nil, err
	}

	// 5. 返回新密钥（仅本次返回）
	return &model.AccessKey{
		AccessKeyID:     ak.AccessKeyID,
		SecretAccessKey: newSecret,
		Status:          ak.Status,
	}, nil
}

// ListAll 获取所有访问密钥
func (s *accessKeyStore) ListAll() ([]*model.AccessKey, error) {
	var aks []*model.AccessKey
	_, err := s.session.Select(
		"id",
		"user_id",
		"access_key_id",
		"status",
		"created_at",
		"updated_at",
	).From("access_keys").
		Load(&aks)
	return aks, err
}

// generateRandomSecret 生成随机密钥
func generateRandomSecret() string {
	// 实际实现使用安全随机数生成器
	// 这里简化为示例
	return "new-secret-key-" + time.Now().Format("20060102150405")
}
