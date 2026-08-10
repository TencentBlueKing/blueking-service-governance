// Package list provides the 'envvar list' sub-command group.
package list

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewPublicCmd returns a Command instance for 'envvar list scoped' sub command.
func NewPublicCmd() *cobra.Command {
	var workspaceID, outputFormat string

	cmd := &cobra.Command{
		Use:   "scoped",
		Short: "List scoped (workspace/envType/env) environment variables",
		Long: `List all scoped environment variables in a workspace.

Displays key, value, scopeType, scopeValue, description, and isSensitive fields.
Sensitive values are masked with '******'.`,
		Example: `  # List scoped env vars in default workspace
  bkms-cli envvar list scoped

  # List scoped env vars in a specific workspace
  bkms-cli envvar list scoped --workspace <workspaceID>

  # Output as JSON
  bkms-cli envvar list scoped --workspace <workspaceID> -o json`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			envVars, err := client.New().ListPublicEnvVars(cmd.Context(), workspaceID)
			if err != nil {
				return errors.Wrap(err, "list scoped env vars")
			}

			formatted, err := output.FormatData(cmd.Context(), envVars, outputFormat)
			if err != nil {
				return errors.Wrap(err, "format output")
			}
			fmt.Println(formatted)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	return cmd
}
