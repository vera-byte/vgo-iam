package cmd

import (
	"github.com/spf13/cobra"
)

// 根命令
var RootCmd = &cobra.Command{
	Use:   "vgo-iam",
	Short: "VGO IAM Service",
	Long:  "Identity and Access Management service for VGO ecosystem",
}
