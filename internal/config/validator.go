package config

import (
	"fmt"
	"strings"
)

// ValidateMasterKey 验证主密钥配置
// 参数:
//   - masterKey: 主密钥字符串
//
// 返回值:
//   - error: 验证失败时返回错误信息
func ValidateMasterKey(masterKey string) error {
	// 检查主密钥是否为空
	if strings.TrimSpace(masterKey) == "" {
		return fmt.Errorf("主密钥不能为空，请在配置文件中设置 middleware.master_key")
	}

	// 检查主密钥长度（至少32个字符）
	if len(masterKey) < 32 {
		return fmt.Errorf("主密钥长度不能少于32个字符，当前长度: %d", len(masterKey))
	}

	// 避免使用默认/示例值
	defaultKeys := []string{
		"your-secret-key-here",
		"change-me",
		"default-key",
		"example-key",
		"test-key",
		"12345678901234567890123456789012",
	}
	for _, defaultKey := range defaultKeys {
		if masterKey == defaultKey {
			return fmt.Errorf("不能使用默认或示例主密钥值")
		}
	}

	// 检查主密钥只包含字母和数字
	for _, char := range masterKey {
		if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
			return fmt.Errorf("主密钥只能包含字母和数字")
		}
	}

	return nil
}

// ValidateConfig 验证应用配置
// 参数:
//   - cfg: 应用配置结构体
//
// 返回值:
//   - error: 验证失败时返回错误信息
func ValidateConfig(cfg *AppConfig) error {
	// 验证主密钥
	if err := ValidateMasterKey(cfg.Middleware.MasterKey); err != nil {
		return fmt.Errorf("主密钥验证失败: %w", err)
	}

	// 验证gRPC配置
	if cfg.GRPC.Server.Host == "" {
		return fmt.Errorf("gRPC服务器主机不能为空")
	}
	if cfg.GRPC.Server.Port == 0 {
		return fmt.Errorf("gRPC服务器端口不能为空")
	}

	// 验证数据库DSN
	if cfg.Database.DSN == "" {
		return fmt.Errorf("数据库连接字符串不能为空")
	}

	return nil
}
