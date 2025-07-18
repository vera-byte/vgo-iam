package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// KeyRotationManager 密钥轮换管理器
type KeyRotationManager struct {
	currentKey   []byte
	previousKeys [][]byte
}

// NewKeyRotationManager 创建密钥轮换管理器
func NewKeyRotationManager(initialKey []byte) *KeyRotationManager {
	return &KeyRotationManager{
		currentKey:   initialKey,
		previousKeys: [][]byte{},
	}
}

// RotateKey 轮换主密钥
func (m *KeyRotationManager) RotateKey(newKey []byte) {
	m.previousKeys = append(m.previousKeys, m.currentKey)
	m.currentKey = newKey

	// 只保留最近的两个密钥
	if len(m.previousKeys) > 2 {
		m.previousKeys = m.previousKeys[len(m.previousKeys)-2:]
	}
}

// ReEncryptKeys 使用新密钥重新加密所有密钥
func (m *KeyRotationManager) ReEncryptKeys(encryptedKeys [][]byte) ([][]byte, error) {
	var reencrypted [][]byte

	for _, encKey := range encryptedKeys {
		// 尝试用之前的密钥解密
		var decrypted []byte
		var err error

		// 先尝试当前密钥
		decrypted, err = DecryptKey(encKey, m.currentKey)
		if err != nil {
			// 尝试之前的密钥
			for _, oldKey := range m.previousKeys {
				decrypted, err = DecryptKey(encKey, oldKey)
				if err == nil {
					break
				}
			}
			if err != nil {
				return nil, err
			}
		}

		// 用新密钥重新加密
		newEncKey, err := EncryptKey(decrypted, m.currentKey)
		if err != nil {
			return nil, err
		}
		reencrypted = append(reencrypted, newEncKey)
	}

	return reencrypted, nil
}

// GenerateNewMasterKey 生成新的主密钥
func GenerateNewMasterKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptWithCurrentKey 使用当前密钥加密
func (m *KeyRotationManager) EncryptWithCurrentKey(plaintext []byte) ([]byte, error) {
	return EncryptKey(plaintext, m.currentKey)
}

// DecryptWithAnyKey 使用可用密钥解密
func (m *KeyRotationManager) DecryptWithAnyKey(ciphertext []byte) ([]byte, error) {
	// 先尝试当前密钥
	decrypted, err := DecryptKey(ciphertext, m.currentKey)
	if err == nil {
		return decrypted, nil
	}

	// 尝试之前的密钥
	for _, key := range m.previousKeys {
		decrypted, err = DecryptKey(ciphertext, key)
		if err == nil {
			return decrypted, nil
		}
	}

	return nil, errors.New("failed to decrypt with any available key")
}

// EncryptKey encrypts plaintext with the given key using AES-GCM.
func EncryptKey(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext := aesgcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptKey decrypts ciphertext with the given key using AES-GCM.
func DecryptKey(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := aesgcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
