package appcfgfile

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/appcfgfile"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewRollbackVersionCmd returns a Command instance for 'app app-cfg-file rollback-version' sub command.
func NewRollbackVersionCmd() *cobra.Command {
	var appID, envName, cfgFileName, versionID, description string
	var version int64

	cmd := &cobra.Command{
		Use:    "rollback-version",
		Short:  "Rollback an application config file to one history version",
		PreRun: cmdutil.CommonPreRun,
		Long: `Rollback the application config file selected by app and environment to one history version.

Use exactly one of --version or --version-id to identify the target history version.
When --env is omitted, this command rolls back the default application-level config.
When --env is provided, this command rolls back that environment's overlay config.
When an application has multiple config files in the same environment, use --name to select one.`,
		Example: `  # Roll back the default config file to version 7
  bkms-cli app app-cfg-file rollback-version --app demo --version 7

  # Roll back one version by version record ID
  bkms-cli app app-cfg-file rollback-version --app demo --env prod --version-id <record-id>

  # Roll back one Helm config file version by name
  bkms-cli app app-cfg-file rollback-version --app demo --name values --version 3

  # Roll back and record a description
  bkms-cli app app-cfg-file rollback-version --app demo --env prod --version 7 --description "rollback prod values"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			versionRef, err := parseVersionRefOptions(cmd, version, versionID)
			if err != nil {
				return err
			}

			result, err := handler.RollbackVersion(
				cmd.Context(),
				client.New(),
				appID,
				envName,
				cfgFileName,
				handler.RollbackVersionOptions{
					VersionRef:  versionRef,
					Description: description,
				},
			)
			if err != nil {
				return errors.Wrap(err, "rollback app config file version")
			}

			console.Info(
				"app config file %s rolled back successfully from version %d using version record %s",
				result.File.ID,
				result.CurrentVersion,
				result.VersionID,
			)
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
	cmd.Flags().StringVar(&description, "description", "", "rollback version description")

	_ = cmd.MarkFlagRequired("app")

	return cmd
}
