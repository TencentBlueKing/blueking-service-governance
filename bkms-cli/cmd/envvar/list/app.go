package list

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewAppCmd returns a Command instance for 'envvar list app' sub command.
func NewAppCmd() *cobra.Command {
	var appID, envName, outputFormat string

	cmd := &cobra.Command{
		Use:   "app",
		Short: "List application environment variables",
		Long: `List environment variables for an application.

Without --env: lists only the app-defined environment variables.
With --env: lists all effective environment variables for the app in the
specified environment (including inherited public vars, env-scoped vars,
and app-defined vars after priority-based deduplication).

Sensitive values are masked with '******'.`,
		Example: `  # List app-defined env vars only
  bkms-cli envvar list app --app <appID>

  # List all effective env vars for an app in a specific environment
  bkms-cli envvar list app --app <appID> --env <env-name>

  # Output as JSON
  bkms-cli envvar list app --app <appID> --env <env-name> -o json`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			var data any
			var err error

			if envName != "" {
				// 带 --env：查询应用在某环境下最终生效的全部环境变量
				data, err = client.New().ListAppEnvVars(cmd.Context(), appID, envName)
			} else {
				// 不带 --env：查询应用直接定义的环境变量
				data, err = client.New().ListAppDefinedEnvVars(cmd.Context(), appID)
			}
			if err != nil {
				return errors.Wrap(err, "list app env vars")
			}

			formatted, err := output.FormatData(cmd.Context(), data, outputFormat)
			if err != nil {
				return errors.Wrap(err, "format output")
			}
			fmt.Println(formatted)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID (required)")
	cmd.Flags().StringVar(&envName, "env", "", "environment name (optional, lists effective vars when specified)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("app")

	return cmd
}
