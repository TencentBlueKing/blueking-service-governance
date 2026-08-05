package create

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewAppCmd returns a Command instance for 'env-var create app' sub command.
func NewAppCmd() *cobra.Command {
	var appID, key, value, description string
	var sensitive bool

	cmd := &cobra.Command{
		Use:   "app",
		Short: "Create an app-defined environment variable",
		Long: `Create a new environment variable directly defined by an application.

The key must be unique within the application.`,
		Example: `  # Create an app env var
  bkms-cli env-var create app --app <appID> --key MY_VAR --value my-value

  # Create a sensitive app env var
  bkms-cli env-var create app --app <appID> --key MY_VAR --value my-value --sensitive

  # Create with description
  bkms-cli env-var create app --app <appID> --key MY_VAR --value my-value --description "My variable"`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := client.New().
				CreateAppDefinedEnvVar(cmd.Context(), appID, client.CreateAppDefinedEnvVarOptions{
					Key:         key,
					Value:       value,
					Description: description,
					IsSensitive: sensitive,
				})
			if err != nil {
				return errors.Wrap(err, "create app env var")
			}

			fmt.Printf("Created app env var: key=%s\n", result.Key)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID (required)")
	cmd.Flags().StringVar(&key, "key", "", "environment variable key (required)")
	cmd.Flags().StringVar(&value, "value", "", "environment variable value")
	cmd.Flags().StringVar(&description, "description", "", "variable description")
	cmd.Flags().BoolVar(&sensitive, "sensitive", false, "mark as sensitive variable")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("key")

	return cmd
}
