// Package importvar provides the 'envvar import' sub-command group.
package importvar

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewAppCmd returns a Command instance for 'envvar import app' sub command.
func NewAppCmd() *cobra.Command {
	var appID, filePath, outputFormat string
	var preview bool

	cmd := &cobra.Command{
		Use:   "app",
		Short: "Import app-defined environment variables",
		Long: `Import app-defined environment variables from a local .env file.

The file will be uploaded to the server and parsed as application-level
environment variables (workload.envVars).

Use --preview to see what would be imported without making any changes.`,
		Example: `  # Import app env vars
  bkms-cli envvar import app --app <appID> -f vars.env

  # Preview import without making changes
  bkms-cli envvar import app --app <appID> -f vars.env --preview`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			if preview {
				previewResult, err := client.New().PreviewAppEnvVars(cmd.Context(), appID, filePath)
				if err != nil {
					return errors.Wrap(err, "preview app env vars")
				}
				return formatPreviewOutput(cmd.Context(), previewResult, outputFormat)
			}

			result, err := client.New().ImportAppEnvVars(cmd.Context(), appID, filePath)
			if err != nil {
				return errors.Wrap(err, "import app env vars")
			}

			fmt.Printf("Import completed: total=%d, new=%d, overwrite=%d\n",
				result.Total, result.New, result.Overwrite)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID (required)")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "path to the .env file to import")
	cmd.Flags().BoolVar(&preview, "preview", false, "preview import without making changes")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}
