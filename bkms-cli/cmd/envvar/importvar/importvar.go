// Package importvar provides the 'envvar import' sub-command group.
package importvar

import "github.com/spf13/cobra"

// NewCmd creates the import sub-command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import environment variables from a local .env file",
		Long: `Import environment variables from a local .env file to the server.

Supports importing scoped (workspace/envType), env-scoped, and app-defined env vars.`,
		DisableFlagsInUseLine: true,
	}

	// 导入 scoped 环境变量（workspace/envType）
	cmd.AddCommand(NewPublicCmd())
	// 导入环境变量
	cmd.AddCommand(NewEnvCmd())
	// 导入应用环境变量
	cmd.AddCommand(NewAppCmd())

	return cmd
}
