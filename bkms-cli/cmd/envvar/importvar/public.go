package importvar

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewPublicCmd returns a Command instance for 'envvar import public' sub command.
func NewPublicCmd() *cobra.Command {
	var workspaceID, filePath, outputFormat string
	var preview bool

	cmd := &cobra.Command{
		Use:   "public",
		Short: "Import public (workspace/envType) environment variables",
		Long: `Import public environment variables from a local .env file.

The file will be uploaded to the server and parsed as workspace-level
and/or envType-level scoped environment variables.

Use --preview to see what would be imported without making any changes.`,
		Example: `  # Import public env vars from a file
  bkms-cli envvar import public --workspace <workspaceID> -f vars.env

  # Preview import without making changes
  bkms-cli envvar import public --workspace <workspaceID> -f vars.env --preview

  # Preview with JSON output
  bkms-cli envvar import public -f vars.env --preview -o json`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			if preview {
				previewResult, err := client.New().PreviewPublicEnvVars(cmd.Context(), workspaceID, filePath)
				if err != nil {
					return errors.Wrap(err, "preview public env vars")
				}
				return formatPreviewOutput(cmd.Context(), previewResult, outputFormat)
			}

			result, err := client.New().ImportPublicEnvVars(cmd.Context(), workspaceID, filePath)
			if err != nil {
				return errors.Wrap(err, "import public env vars")
			}

			fmt.Printf("Import completed: total=%d, new=%d, overwrite=%d\n",
				result.Total, result.New, result.Overwrite)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "path to the .env file to import")
	cmd.Flags().BoolVar(&preview, "preview", false, "preview import without making changes")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("file")

	return cmd
}
