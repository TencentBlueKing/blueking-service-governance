// Package updatecmd provides the 'envvar update' sub-command group.
package update

import "github.com/spf13/cobra"

// NewCmd creates the update sub-command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an environment variable",
		Long: `Update an existing environment variable on the server.

Supports updating public (workspace/envType/env) scoped vars and app-defined vars.`,
		DisableFlagsInUseLine: true,
	}

	// 更新公共环境变量
	cmd.AddCommand(NewPublicCmd())
	// 更新应用环境变量
	cmd.AddCommand(NewAppCmd())
	// 更新环境变量
	cmd.AddCommand(NewEnvCmd())

	return cmd
}
