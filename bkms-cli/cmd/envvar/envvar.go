// Package envvar provide envvar command group for environment variable import/export.
package envvar

import (
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/envvar/create"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/envvar/delete"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/envvar/export"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/envvar/importvar"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/envvar/list"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/envvar/update"
)

// NewCmd creates the envvar command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "envvar",
		Short: "Manage environment variables",
		Long: `Manage environment variables import/export and CRUD operations.

Use this command to import/export environment variables from/to local .env files,
or list, create, update, delete individual environment variables.`,
		DisableFlagsInUseLine: true,
	}

	// 导入环境变量
	cmd.AddCommand(importvar.NewCmd())
	// 导出环境变量
	cmd.AddCommand(export.NewCmd())
	// 查询环境变量
	cmd.AddCommand(list.NewCmd())
	// 创建环境变量
	cmd.AddCommand(create.NewCmd())
	// 更新环境变量
	cmd.AddCommand(update.NewCmd())
	// 删除环境变量
	cmd.AddCommand(delete.NewCmd())

	return cmd
}
