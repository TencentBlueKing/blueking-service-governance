// Package create provides the 'envvar create' sub-command group.
package create

import "github.com/spf13/cobra"

// NewCmd creates the create sub-command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an environment variable",
		Long: `Create a new environment variable on the server.

Supports creating public (workspace/envType/env) scoped vars and app-defined vars.`,
		DisableFlagsInUseLine: true,
	}

	// 创建公共环境变量
	cmd.AddCommand(NewPublicCmd())
	// 创建应用环境变量
	cmd.AddCommand(NewAppCmd())
	// 创建环境变量
	cmd.AddCommand(NewEnvCmd())

	return cmd
}
