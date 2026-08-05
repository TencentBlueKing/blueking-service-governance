package export

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

const (
	exportScopeAppDefined     = "appDefined"
	exportScopeEffectiveByEnv = "effectiveByEnv"
)

// NewAppCmd returns a Command instance for 'env-var export app' sub command.
func NewAppCmd() *cobra.Command {
	var appID, scope, envName, filePath string

	cmd := &cobra.Command{
		Use:   "app",
		Short: "Export app environment variables",
		Long: `Export app environment variables from the server.

Supports two export scopes:
  - appDefined: export only app-defined variables (default)
  - effectiveByEnv: export all effective variables for a specific environment
    (requires --env flag)

The exported content is in dotenv format. By default it is printed to stdout.
Use -f to write it to a file.`,
		Example: `  # Export app-defined env vars to stdout
  bkms-cli env-var export app --app <appID>

  # Export app-defined env vars to a file
  bkms-cli env-var export app --app <appID> -f vars.env

  # Export effective env vars for a specific environment
  bkms-cli env-var export app --app <appID> --scope effectiveByEnv --env <env-name>

  # Export effective env vars to a file
  bkms-cli env-var export app --app <appID> --scope effectiveByEnv --env <env-name> -f vars.env`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			if scope == exportScopeEffectiveByEnv && envName == "" {
				return errors.New("--env is required when --scope is effectiveByEnv")
			}

			content, err := client.New().ExportAppEnvVars(cmd.Context(), appID, client.ExportAppEnvVarsOptions{
				Scope:   scope,
				EnvName: envName,
			})
			if err != nil {
				return errors.Wrap(err, "export app env vars")
			}

			return writeExportContent(content, filePath)
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID (required)")
	cmd.Flags().StringVar(&scope, "scope", exportScopeAppDefined,
		"export scope: appDefined (default) or effectiveByEnv")
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required when scope=effectiveByEnv)")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "output file path (default: stdout)")

	_ = cmd.MarkFlagRequired("app")

	return cmd
}
