package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gocraft/dbr/v2"
	"github.com/spf13/cobra"
	"github.com/vera-byte/vgo-iam/internal/bootstrap"
	"github.com/vera-byte/vgo-iam/internal/config"
	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/service"
	"github.com/vera-byte/vgo-iam/internal/util"
	vgokit "github.com/vera-byte/vgo-kit"
	"go.uber.org/zap"
	"golang.org/x/term"
)

func init() {
	RootCmd.AddCommand(InitAdminCmd)

}

// InitAdminCmd 初始化管理员用户命令
var InitAdminCmd = &cobra.Command{
	Use:   "init admin",
	Short: "Initialize admin user and access key",
	Long:  "Create initial admin user, policy and access key if they don't exist",
	Run: func(cmd *cobra.Command, args []string) {
		// 交互式输入email
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("请输入管理员邮箱: ")
		email, _ := reader.ReadString('\n')
		email = strings.TrimSpace(email)
		if email == "" {
			fmt.Println("邮箱不能为空，请重新输入")
			os.Exit(1)
		}

		// 交互式输入密码并二次确认
		var password string
		var confirmPassword string
		var err error

		for {
			fmt.Print("请输入管理员密码: ")
			password, err = readPassword()
			if err != nil {
				fmt.Println("读取密码失败: ", err)
				os.Exit(1)
			}

			fmt.Print("请再次输入密码进行确认: ")
			confirmPassword, err = readPassword()
			if err != nil {
				fmt.Println("读取确认密码失败: ", err)
				os.Exit(1)
			}

			if password != confirmPassword {
				fmt.Println("两次输入的密码不一致，请重新输入")
			} else if len(password) < 8 {
				fmt.Println("密码长度不能少于8位，请重新输入")
			} else {
				break
			}
		}

		fmt.Println("正在初始化管理员用户...")

		// 初始化配置
		cfg := config.LodIAMConfig()
		// 初始化服务
		iamServer, session, _, _ := bootstrap.InitServices(cfg)
		defer session.Close()

		// 获取服务实例
		userService := iamServer.UserService()
		policyService := iamServer.PolicyService()
		accessKeyService := iamServer.AccessKeyService()

		// 调用初始化函数
	err = initAdminUser(userService, policyService, accessKeyService, session, email, password)
		if err != nil {
			fmt.Printf("创建管理员用户失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("管理员用户初始化成功")
	},
}

// 读取密码（不回显）
func readPassword() (string, error) {
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	fmt.Println() // 输入完成后换行
	return string(bytePassword), nil
}

// 初始化管理员用户
func initAdminUser(userService *service.UserService, policyService *service.PolicyService, accessKeyService *service.AccessKeyService, session *dbr.Session, email string, password string) error {
	var (
		ctx = context.Background()
	)

	// 开始数据库事务
	tx, err := session.BeginTx(ctx, nil)
	if err != nil {
		vgokit.Log.Error("开始事务失败", zap.Error(err))
		return err
	}

	// 用于标记是否需要回滚
	var shouldCommit = false
	defer func() {
		if r := recover(); r != nil {
			tx.RollbackUnlessCommitted()
			vgokit.Log.Error("事务回滚 - panic", zap.Any("panic", r))
			panic(r)
		}
		if shouldCommit {
			if commitErr := tx.Commit(); commitErr != nil {
				vgokit.Log.Error("提交事务失败", zap.Error(commitErr))
			} else {
				vgokit.Log.Info("事务提交成功")
			}
		} else {
			tx.RollbackUnlessCommitted()
			vgokit.Log.Info("事务已回滚")
		}
	}()

	// 在事务中创建或获取管理员用户
	var userID int64
	err = tx.QueryRow("SELECT id FROM users WHERE name = $1", "admin").Scan(&userID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			// 用户不存在，创建新用户
			err = tx.QueryRow(`
				INSERT INTO users (name, display_name, email, created_at, updated_at) 
				VALUES ($1, $2, $3, NOW(), NOW()) 
				RETURNING id`,
				"admin", "System Administrator", email).Scan(&userID)
			if err != nil {
				vgokit.Log.Error("创建管理员用户失败", zap.Error(err))
				return err
			}
			vgokit.Log.Info("管理员用户创建成功", zap.Int64("user_id", userID))
		} else {
			vgokit.Log.Error("查询管理员用户失败", zap.Error(err))
			return err
		}
	} else {
		vgokit.Log.Info("管理员用户已存在", zap.Int64("user_id", userID))
	}

	// 设置管理员用户密码（在事务内部直接更新）
	vgokit.Log.Info("正在设置管理员用户密码...", zap.Int64("user_id", userID))
	
	// 验证密码强度
	if !util.ValidatePasswordStrength(password) {
		vgokit.Log.Error("密码不符合强度要求")
		return fmt.Errorf("password does not meet strength requirements")
	}
	
	// 生成密码哈希
	passwordHash, err := util.HashPassword(password)
	if err != nil {
		vgokit.Log.Error("密码哈希生成失败", zap.Error(err))
		return err
	}
	
	// 在事务内部直接更新密码
	_, err = tx.Update("users").
		Set("password_hash", passwordHash).
		Set("updated_at", time.Now()).
		Where("id = ?", userID).
		Exec()
	if err != nil {
		vgokit.Log.Error("设置管理员密码失败", zap.Error(err))
		return err
	}
	vgokit.Log.Info("管理员用户密码设置成功")

	// 创建管理员策略（如果不存在）
	_, err = policyService.GetStore().GetByName("admin-policy")

	if err != nil {
		vgokit.Log.Info("Admin policy not found, creating...")
		adminPolicyDoc := `{"Version":"2025-08-01","Statement":[{"Effect":"Allow","Action":"*","Resource":"*"}]}`
		_, err = policyService.CreatePolicy(ctx, "admin-policy", "Administrator policy with full permissions", adminPolicyDoc)
		if err != nil {
			vgokit.Log.Error("Failed to create admin policy", zap.Error(err))
			// 错误时不设置shouldCommit，将触发回滚
			return err
		}
		vgokit.Log.Info("Admin policy created successfully")
	}

	// 在事务内部直接绑定策略到用户
	// 首先获取策略ID
	var policyID int
	err = tx.Select("id").From("policies").Where("name = ?", "admin-policy").LoadOne(&policyID)
	if err != nil {
		vgokit.Log.Error("Failed to find admin policy", zap.Error(err))
		return err
	}

	// 检查策略绑定是否已存在
	var existingBinding int
	err = tx.Select("COUNT(*)").From("user_policies").Where("user_id = ? AND policy_id = ?", userID, policyID).LoadOne(&existingBinding)
	if err != nil {
		vgokit.Log.Error("Failed to check existing policy binding", zap.Error(err))
		return err
	}

	// 如果策略绑定不存在，则创建
	if existingBinding == 0 {
		_, err = tx.InsertInto("user_policies").Columns("user_id", "policy_id").Values(userID, policyID).Exec()
		if err != nil {
			vgokit.Log.Error("Failed to attach policy to admin user", zap.Error(err))
			return err
		}
		vgokit.Log.Info("Admin policy attached successfully")
	} else {
		vgokit.Log.Info("管理员策略绑定已存在，跳过")
	}

	// 在事务内部直接检查是否已有访问密钥
	var accessKeyCount int
	err = tx.Select("COUNT(*)").From("access_keys").Where("user_id = ?", userID).LoadOne(&accessKeyCount)
	if err != nil {
		vgokit.Log.Error("Failed to count admin access keys", zap.Error(err))
		return err
	}

	// 如果没有访问密钥，创建一个
	if accessKeyCount == 0 {
		vgokit.Log.Info("No access keys found for admin, creating...")
		// 为admin用户创建一个不绑定应用的访问密钥，跳过开发者认证检查
		accessKey, err := createAdminAccessKey(ctx, accessKeyService, userID)
		if err != nil {
			vgokit.Log.Error("Failed to create admin access key", zap.Error(err))
			return err
		}
		vgokit.Log.Info("Admin access key created",
			zap.String("access_key_id", accessKey.AccessKeyID),
			zap.String("secret_access_key", accessKey.SecretAccessKey))

		// 注意：在生产环境中，应该将密钥保存到安全的地方，而不是仅记录日志
	} else {
		vgokit.Log.Info("Admin access keys already exist")
	}

	// 所有操作成功，标记为可以提交
	shouldCommit = true
	return nil
}

// createAdminAccessKey 为admin用户创建访问密钥，跳过开发者认证检查
func createAdminAccessKey(ctx context.Context, accessKeyService *service.AccessKeyService, userID int64) (*model.AccessKey, error) {
	// 生成访问密钥ID和密钥
	accessKeyID := util.GenerateAccessKeyID()
	secretKey := util.GenerateSecretAccessKey()
	
	// 创建访问密钥模型
	accessKey := model.NewAccessKey(userID, accessKeyID, secretKey, nil, "Admin access key")
	
	// 直接使用AccessKeyStore创建访问密钥
	accessKeyStore := accessKeyService.GetStore()
	masterKey := accessKeyService.GetMasterKey()
	
	err := accessKeyStore.Create(accessKey, masterKey)
	if err != nil {
		return nil, err
	}
	
	return accessKey, nil
}
