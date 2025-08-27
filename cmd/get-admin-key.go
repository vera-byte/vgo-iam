package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vera-byte/vgo-iam/internal/bootstrap"
	"github.com/vera-byte/vgo-iam/internal/config"
	vgokit "github.com/vera-byte/vgo-kit"
	"go.uber.org/zap"
)

func init() {
	RootCmd.AddCommand(GetAdminKeyCmd)
}

// GetAdminKeyCmd 获取admin用户访问密钥命令
var GetAdminKeyCmd = &cobra.Command{
	Use:   "get-admin-key",
	Short: "Get admin user access keys",
	Long:  "Display all access keys for the admin user",
	Run: func(cmd *cobra.Command, args []string) {
		// 初始化配置
		cfg := config.LodIAMConfig()
		// 初始化服务
		iamServer, session := bootstrap.InitServices(cfg)
		defer session.Close()

		// 获取访问密钥服务
		accessKeyService := iamServer.AccessKeyService()

		// 获取admin用户的访问密钥
		ctx := context.Background()
		accessKeys, err := accessKeyService.ListAccessKeys(ctx, "admin")
		if err != nil {
			vgokit.Log.Error("获取admin访问密钥失败", zap.Error(err))
			return
		}

		if len(accessKeys) == 0 {
			fmt.Println("admin用户没有访问密钥")
			return
		}

		fmt.Printf("admin用户的访问密钥:\n")
		for i, ak := range accessKeys {
			fmt.Printf("密钥 %d:\n", i+1)
			fmt.Printf("  AccessKeyID: %s\n", ak.AccessKeyID)
			fmt.Printf("  SecretAccessKey: %s\n", ak.SecretAccessKey)
			fmt.Printf("  Status: %s\n", ak.Status)
			fmt.Printf("  AppID: %v\n", ak.AppID)
			fmt.Printf("  Description: %s\n", ak.Description)
			fmt.Println()
		}
	},
}
