// Package exportcmd provides the 'env-var export' sub-command group.
package export

import "github.com/spf13/cobra"

// NewCmd creates the export sub-command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export environment variables to a local .env file",
		Long: `Export environment variables from the server to stdout or a local .env file.

Supports exporting public (workspace/envType), env-scoped, and app-defined env vars.`,
		DisableFlagsInUseLine: true,
	}

	// 导出公共环境变量
	cmd.AddCommand(NewPublicCmd())
	// 导出环境变量
	cmd.AddCommand(NewEnvCmd())
	// 导出应用环境变量
	cmd.AddCommand(NewAppCmd())

	return cmd
}
