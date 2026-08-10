// Package list provides the 'envvar list' sub-command group.
package list

import "github.com/spf13/cobra"

// NewCmd creates the list sub-command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List environment variables",
		Long: `List environment variables from the server.

Supports listing scoped (workspace/envType/env) vars, app-defined vars,
and effective vars for an app in a specific environment.`,
		DisableFlagsInUseLine: true,
	}

	// 查询 scoped 环境变量（workspace/envType）
	cmd.AddCommand(NewPublicCmd())
	// 查询应用环境变量
	cmd.AddCommand(NewAppCmd())
	// 查询环境变量
	cmd.AddCommand(NewEnvCmd())

	return cmd
}
