package list

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/env"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewEnvCmd returns a Command instance for 'env-var list env' sub command.
func NewEnvCmd() *cobra.Command {
	var workspaceID, envName, outputFormat string

	cmd := &cobra.Command{
		Use:   "env",
		Short: "List environment-scoped environment variables",
		Long: `List all environment-scoped environment variables for a specific environment.

The --env flag accepts an environment name (e.g. prod, stag, teamdev).
Displays detailed information including conflict info.`,
		Example: `  # List env-scoped env vars
  bkms-cli env-var list env --env <env-name>

  # Output as JSON
  bkms-cli env-var list env --env <env-name> -o json`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			envID, err := handler.ResolveEnvIDByName(cmd.Context(), client.New(), workspaceID, envName)
			if err != nil {
				return err
			}

			envVars, err := client.New().ListEnvScopedEnvVars(cmd.Context(), envID)
			if err != nil {
				return errors.Wrap(err, "list env scoped env vars")
			}

			// json/yaml 保持原始嵌套结构输出；table（默认）使用扁平结构展示
			var data any
			switch outputFormat {
			case "json", "yaml":
				data = envVars
			default:
				data = handler.ToTableRows(envVars)
			}

			formatted, err := output.FormatData(cmd.Context(), data, outputFormat)
			if err != nil {
				return errors.Wrap(err, "format output")
			}
			fmt.Println(formatted)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("env")

	return cmd
}
