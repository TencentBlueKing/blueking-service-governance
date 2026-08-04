package appcfgfile

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/appcfgfile"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewDeleteVersionCmd returns a Command instance for 'app app-cfg-file delete-version' sub command.
func NewDeleteVersionCmd() *cobra.Command {
	var appID, envName, cfgFileName, versionID string
	var version int64

	cmd := &cobra.Command{
		Use:    "delete-version",
		Short:  "Delete one history version of an application config file",
		PreRun: cmdutil.CommonPreRun,
		Long: `Delete one history version of the application config file selected by app and environment.

Use exactly one of --version or --version-id to identify the target history version.
When --env is omitted, this command deletes a version of the default application-level config.
When --env is provided, this command deletes a version of that environment's overlay config.
When an application has multiple config files in the same environment, use --name to select one.`,
		Example: `  # Delete version 7 of the default config file
  bkms-cli app app-cfg-file delete-version --app demo --version 7

  # Delete one version by version record ID
  bkms-cli app app-cfg-file delete-version --app demo --env prod --version-id <record-id>

  # Delete one Helm config file version by name
  bkms-cli app app-cfg-file delete-version --app demo --name values --version 3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := parseVersionRefOptions(cmd, version, versionID)
			if err != nil {
				return err
			}

			result, err := handler.DeleteVersion(cmd.Context(), client.New(), appID, envName, cfgFileName, opts)
			if err != nil {
				return errors.Wrap(err, "delete app config file version")
			}

			console.Info("app config file version %s deleted successfully", result.VersionID)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().StringVar(&envName, "env", "", "environment name")
	cmd.Flags().StringVar(
		&cfgFileName,
		"name",
		"",
		"config file name; useful for Helm apps with multiple app-level config files",
	)
	registerVersionRefFlags(cmd, &version, &versionID)

	_ = cmd.MarkFlagRequired("app")

	return cmd
}
