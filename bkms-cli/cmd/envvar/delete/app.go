package delete

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewAppCmd returns a Command instance for 'env-var delete app' sub command.
func NewAppCmd() *cobra.Command {
	var appID, key string

	cmd := &cobra.Command{
		Use:   "app",
		Short: "Delete an app-defined environment variable",
		Long:  `Delete an existing app-defined environment variable by its key.`,
		Example: `  # Delete an app env var
  bkms-cli env-var delete app --app <appID> --key MY_VAR`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client.New().DeleteAppDefinedEnvVar(cmd.Context(), appID, key); err != nil {
				return errors.Wrap(err, "delete app env var")
			}

			fmt.Printf("Deleted app env var: key=%s\n", key)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID (required)")
	cmd.Flags().StringVar(&key, "key", "", "environment variable key (required)")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("key")

	return cmd
}
