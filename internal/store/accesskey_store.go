package store

import (
	"time"

	"github.com/gocraft/dbr/v2"
	"github.com/vera-byte/vgo-iam/internal/crypto"
	"github.com/vera-byte/vgo-iam/internal/model"
)

// AccessKeyStore 访问密钥存储接口
type AccessKeyStore interface {
	Create(ak *model.AccessKey, masterKey []byte) error
	GetByID(id int) (*model.AccessKey, error)
	GetByAccessKeyID(accessKeyID string) (*model.AccessKey, error)
	ListByUser(userID int) ([]*model.AccessKey, error)
	UpdateStatus(accessKeyID, status string) error
	RotateKey(accessKeyID string, masterKey []byte) (*model.AccessKey, error)
}

// accessKeyStore 访问密钥存储实现
type accessKeyStore struct {
	session *dbr.Session
}

// NewAccessKeyStore 创建访问密钥存储实例
func NewAccessKeyStore(session *dbr.Session) AccessKeyStore {
	return &accessKeyStore{session: session}
}

func (s *accessKeyStore) Create(ak *model.AccessKey, masterKey []byte) error {
	// 加密密钥
	encryptedSecret, err := crypto.EncryptKey([]byte(ak.SecretAccessKey), masterKey)
	if err != nil {
		return err
	}

	_, err = s.session.InsertInto("access_keys").
		Columns(
			"user_id",
			"access_key_id",
			"encrypted_secret_access_key",
			"status",
		).
		Values(
			ak.UserID,
			ak.AccessKeyID,
			encryptedSecret,
			ak.Status,
		).Exec()

	return err
}

func (s *accessKeyStore) GetByID(id int) (*model.AccessKey, error) {
	var ak model.AccessKey
	err := s.session.Select("*").
		From("access_keys").
		Where("id = ?", id).
		LoadOne(&ak)

	return &ak, err
}

func (s *accessKeyStore) GetByAccessKeyID(accessKeyID string) (*model.AccessKey, error) {
	var ak model.AccessKey
	err := s.session.Select(
		"id",
		"user_id",
		"access_key_id",
		"status",
		"created_at",
		"updated_at",
	).From("access_keys").
		Where("access_key_id = ?", accessKeyID).
		LoadOne(&ak)
	return &ak, err
}
func (s *accessKeyStore) ListByUser(userID int) ([]*model.AccessKey, error) {
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

func (s *accessKeyStore) RotateKey(accessKeyID string, masterKey []byte) (*model.AccessKey, error) {
	// 1. 获取现有密钥
	ak, err := s.GetByAccessKeyID(accessKeyID)
	if err != nil {
		return nil, err
	}

	// 2. 生成新密钥
	newSecret := generateRandomSecret()

	// 3. 加密新密钥
	encryptedSecret, err := crypto.EncryptKey([]byte(newSecret), masterKey)
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

// generateRandomSecret 生成随机密钥
func generateRandomSecret() string {
	// 实际实现使用安全随机数生成器
	// 这里简化为示例
	return "new-secret-key-" + time.Now().Format("20060102150405")
}
