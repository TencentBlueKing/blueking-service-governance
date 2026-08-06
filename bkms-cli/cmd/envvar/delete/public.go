package delete

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	envhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/env"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewPublicCmd returns a Command instance for 'envvar delete public' sub command.
func NewPublicCmd() *cobra.Command {
	var workspaceID, key, scopeType, scopeValue string

	cmd := &cobra.Command{
		Use:   "public",
		Short: "Delete a public (workspace/envType) scoped environment variable",
		Long: `Delete an existing public scoped environment variable by key and scope.

The --key flag specifies the variable key to delete.
The --scope-type flag specifies the scope type (workspace or envType).
The --scope-value flag is required when scope-type is envType.`,
		Example: `  # Delete a workspace-scoped public env var
  bkms-cli envvar delete public --key MY_VAR --scope-type workspace

  # Delete an envType-scoped public env var
  bkms-cli envvar delete public --key MY_VAR --scope-type envType --scope-value test`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 校验 scope-type / scope-value 组合
			if scopeType == "envType" && scopeValue == "" {
				return errors.New("--scope-value is required when --scope-type is envType")
			}
			if scopeType == "workspace" {
				scopeValue = "" // workspace 级别忽略 scope-value
			}

			workspaceID = cmdutil.GetWorkspaceID(workspaceID)
			cli := client.New()

			varID, err := envhandler.ResolveScopedEnvVarID(cmd.Context(), cli, workspaceID, key, scopeType, scopeValue)
			if err != nil {
				return err
			}

			if err = cli.DeleteScopedEnvVar(cmd.Context(), workspaceID, varID); err != nil {
				return errors.Wrap(err, "delete public env var")
			}

			fmt.Printf("Deleted env var: key=%s, scopeType=%s, scopeValue=%s\n", key, scopeType, scopeValue)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVar(&key, "key", "", "environment variable key (required)")
	cmd.Flags().StringVar(&scopeType, "scope-type", "", "scope type: workspace or envType (required)")
	cmd.Flags().StringVar(&scopeValue, "scope-value", "", "scope value (required for envType)")

	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("scope-type")

	return cmd
}
