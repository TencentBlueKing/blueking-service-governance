// Package create provides the 'envvar create' sub-command group.
package create

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewPublicCmd returns a Command instance for 'envvar create for public' sub command.
func NewPublicCmd() *cobra.Command {
	var workspaceID, key, value, scopeType, scopeValue, description string
	var sensitive bool

	cmd := &cobra.Command{
		Use:   "public",
		Short: "Create a public (workspace/envType) scoped environment variable",
		Long: `Create a new public scoped environment variable in a workspace.

The --scope-type flag determines the scope level:
  - workspace: applies to all environments
  - envType: applies to a specific environment type (requires --scope-value)`,
		Example: `  # Create a workspace-level env var
  bkms-cli envvar create public --key MY_VAR --value my-value --scope-type workspace

  # Create an envType-level env var
  bkms-cli envvar create public --key MY_VAR --value my-value --scope-type envType --scope-value <env-type>

  # Create a sensitive env var
  bkms-cli envvar create public --key MY_VAR --value my-value --scope-type workspace --sensitive`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			key = strings.TrimSpace(key)

			// 校验：scope-type 仅支持 workspace 和 envType
			if scopeType == "env" {
				return errors.New(
					"scope-type 'env' is not supported here, please use 'bkms-cli envvar create env' instead",
				)
			}
			// scope-type 为 envType 时必须提供 scope-value
			if scopeType == "envType" && scopeValue == "" {
				return errors.New("--scope-value is required when --scope-type is envType")
			}

			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			result, err := client.New().
				CreateScopedEnvVar(cmd.Context(), workspaceID, client.CreateScopedEnvVarOptions{
					ScopeType:   scopeType,
					ScopeValue:  scopeValue,
					Key:         key,
					Value:       value,
					Description: description,
					IsSensitive: sensitive,
				})
			if err != nil {
				return errors.Wrap(err, "create public env var")
			}

			console.Info("Created env var: key=%s, scopeType=%s, scopeValue=%s, id=%s\n",
				result.Key, result.ScopeType, result.ScopeValue, result.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVar(&key, "key", "", "environment variable key (required)")
	cmd.Flags().StringVar(&value, "value", "", "environment variable value")
	cmd.Flags().StringVar(&scopeType, "scope-type", "", "scope type: workspace or envType (required)")
	cmd.Flags().StringVar(&scopeValue, "scope-value", "", "scope value (required for envType)")
	cmd.Flags().StringVar(&description, "description", "", "variable description")
	cmd.Flags().BoolVar(&sensitive, "sensitive", false, "mark as sensitive variable")

	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("scope-type")

	return cmd
}
