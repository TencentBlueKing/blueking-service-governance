// Package delete provides the 'envvar delete' sub-command group.
package delete

import "github.com/spf13/cobra"

// NewCmd creates the delete sub-command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an environment variable",
		Long: `Delete an environment variable from the server.

Supports deleting public (workspace/envType/env) scoped vars and app-defined vars.`,
		DisableFlagsInUseLine: true,
	}

	// 删除公共环境变量
	cmd.AddCommand(NewPublicCmd())
	// 删除应用环境变量
	cmd.AddCommand(NewAppCmd())
	// 删除环境某个变量
	cmd.AddCommand(NewEnvCmd())

	return cmd
}
