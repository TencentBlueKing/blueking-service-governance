package startcommand

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/appspec"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewEditCmd returns a Command instance for 'appspec start-command edit' sub command.
func NewEditCmd() *cobra.Command {
	var appID, specFile string

	cmd := &cobra.Command{
		Use:    "edit",
		Short:  "Edit application start command",
		PreRun: cmdutil.CommonPreRun,
		Long: `Edit the application start command and arguments from a YAML file.

The YAML file should contain 'command' and/or 'args' fields as string arrays.
For trpc/taf apps, the current trpcSpec/tafSpec will be automatically preserved
unless explicitly overridden in the YAML file.`,
		Example: `  # Set start command from YAML file
  bkms-cli app appspec start-command edit --app my-app -f start-command.yaml

  # Example YAML file content (basic):
  #   command:
  #     - /usr/local/trpc/bin/container-start.sh
  #   args:
  #     - -conf
  #     - /usr/local/trpc/bin/trpc-go.yaml
  #
  # Example YAML file content (with trpcSpec override):
  #   command:
  #     - ./server
  #   args:
  #     - --config
  #     - /usr/local/trpc/conf/trpc_go.yaml
  #   trpcSpec:
  #     language: go
  #     fileName: trpc_go.yaml
  #     filePath: /usr/local/trpc/conf`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := appspec.EditStartCommandHandler(cmd.Context(), appID, specFile); err != nil {
				return errors.Wrap(err, "edit start command")
			}

			fmt.Printf("Successfully updated start command for app %s\n", appID)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID (required)")
	cmd.Flags().StringVarP(&specFile, "file", "f", "", "path to YAML spec file (required)")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}
