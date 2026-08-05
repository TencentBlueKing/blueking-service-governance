package create

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewEnvCmd returns a Command instance for 'env-var create env' sub command.
func NewEnvCmd() *cobra.Command {
	var workspaceID, key, value, scopeValue, description string
	var sensitive bool

	cmd := &cobra.Command{
		Use:   "env",
		Short: "Create an environment-scoped environment variable",
		Long: `Create a new environment-scoped environment variable.

The variable will be scoped to a specific environment (scopeType=env).
The --scope-value flag specifies the target environment name.`,
		Example: `  # Create an env-scoped env var
  bkms-cli env-var create env --key MY_VAR --value my-value --scope-value <env-name>

  # Create a sensitive env-scoped env var
  bkms-cli env-var create env --key MY_SECRET --value my-value --scope-value <env-name> --sensitive`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			result, err := client.New().
				CreateScopedEnvVar(cmd.Context(), workspaceID, client.CreateScopedEnvVarOptions{
					ScopeType:   "env",
					ScopeValue:  scopeValue,
					Key:         key,
					Value:       value,
					Description: description,
					IsSensitive: sensitive,
				})
			if err != nil {
				return errors.Wrap(err, "create env scoped env var")
			}

			fmt.Printf("Created env var: key=%s, scopeType=env, scopeValue=%s, id=%s\n",
				result.Key, result.ScopeValue, result.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVar(&key, "key", "", "environment variable key (required)")
	cmd.Flags().StringVar(&value, "value", "", "environment variable value")
	cmd.Flags().StringVar(&scopeValue, "scope-value", "", "environment name (required)")
	cmd.Flags().StringVar(&description, "description", "", "variable description")
	cmd.Flags().BoolVar(&sensitive, "sensitive", false, "mark as sensitive variable")

	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("scope-value")

	return cmd
}
