// Package export provides the 'envvar export' sub-command group.
package export

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewPublicCmd returns a Command instance for 'envvar export public' sub command.
func NewPublicCmd() *cobra.Command {
	var workspaceID, filePath string

	cmd := &cobra.Command{
		Use:   "public",
		Short: "Export public (workspace/envType) environment variables",
		Long: `Export public environment variables from the server.

The exported content is in dotenv format. By default it is printed to stdout.
Use -f to write it to a file.`,
		Example: `  # Export public env vars to stdout
  bkms-cli envvar export public --workspace <workspaceID>

  # Export public env vars to a file
  bkms-cli envvar export public --workspace <workspaceID> -f vars.env

  # Use default workspace
  bkms-cli envvar export public -f vars.env`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			content, err := client.New().ExportPublicEnvVars(cmd.Context(), workspaceID)
			if err != nil {
				return errors.Wrap(err, "export public env vars")
			}

			return writeExportContent(content, filePath)
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "output file path (default: stdout)")

	return cmd
}

// writeExportContent 将导出内容写入文件或输出到 stdout。
func writeExportContent(content, filePath string) error {
	if filePath == "" {
		fmt.Print(content)
		return nil
	}
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return errors.Wrap(err, "write export file")
	}
	console.Info("Exported to %s\n", filePath)
	return nil
}
