// Package importvar provides the 'envvar import' sub-command group.
package importvar

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/env"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewEnvCmd returns a Command instance for 'envvar import env' sub command.
func NewEnvCmd() *cobra.Command {
	var workspaceID, envName, filePath, outputFormat string
	var preview bool

	cmd := &cobra.Command{
		Use:   "env",
		Short: "Import environment-scoped environment variables",
		Long: `Import environment-scoped environment variables from a local .env file.

The --env flag accepts an environment name (e.g. prod, stag, teamdev).
The file will be uploaded to the server and parsed as env-level scoped
environment variables for the specified environment.

Use --preview to see what would be imported without making any changes.`,
		Example: `  # Import env-scoped env vars
  bkms-cli envvar import env --env <env-name> -f vars.env

  # Preview import without making changes
  bkms-cli envvar import env --env <env-name> -f vars.env --preview`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			envID, err := handler.ResolveEnvIDByName(cmd.Context(), client.New(), workspaceID, envName)
			if err != nil {
				return err
			}

			if preview {
				previewResult, previewErr := client.New().PreviewEnvScopedEnvVars(cmd.Context(), envID, filePath)
				if previewErr != nil {
					return errors.Wrap(previewErr, "preview env scoped env vars")
				}
				return formatPreviewOutput(cmd.Context(), previewResult, outputFormat)
			}

			result, err := client.New().ImportEnvScopedEnvVars(cmd.Context(), envID, filePath)
			if err != nil {
				return errors.Wrap(err, "import env scoped env vars")
			}

			fmt.Printf("Import completed: total=%d, new=%d, overwrite=%d\n",
				result.Total, result.New, result.Overwrite)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "path to the .env file to import")
	cmd.Flags().BoolVar(&preview, "preview", false, "preview import without making changes")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}
