package update

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewPublicCmd returns a Command instance for 'envvar update public' sub command.
func NewPublicCmd() *cobra.Command {
	var workspaceID, id, key, value, description string
	var sensitive, noSensitive bool

	cmd := &cobra.Command{
		Use:   "public",
		Short: "Update a public (workspace/envType/env) scoped environment variable",
		Long: `Update an existing public scoped environment variable.

Use --sensitive to mark as sensitive, or --no-sensitive to unmark.
Only specified fields will be updated.`,
		Example: `  # Update value
  bkms-cli envvar update public --id <varID> --key MY_VAR --value new-value

  # Update description
  bkms-cli envvar update public --id <varID> --key MY_VAR --description "New description"

  # Mark as sensitive
  bkms-cli envvar update public --id <varID> --key MY_VAR --sensitive`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sensitive && noSensitive {
				return errors.New("--sensitive and --no-sensitive cannot be used together")
			}

			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			opts := client.UpdateScopedEnvVarOptions{
				Key:         key,
				Description: description,
			}
			if cmd.Flags().Changed("value") {
				opts.Value = &value
			}
			if sensitive {
				t := true
				opts.IsSensitive = &t
			} else if noSensitive {
				f := false
				opts.IsSensitive = &f
			}

			result, err := client.New().UpdateScopedEnvVar(cmd.Context(), workspaceID, id, opts)
			if err != nil {
				return errors.Wrap(err, "update public env var")
			}

			fmt.Printf("Updated env var: key=%s, id=%s\n", result.Key, result.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVar(&id, "id", "", "scoped env var ID (required)")
	cmd.Flags().StringVar(&key, "key", "", "environment variable key (required)")
	cmd.Flags().StringVar(&value, "value", "", "environment variable value")
	cmd.Flags().StringVar(&description, "description", "", "variable description")
	cmd.Flags().BoolVar(&sensitive, "sensitive", false, "mark as sensitive variable")
	cmd.Flags().BoolVar(&noSensitive, "no-sensitive", false, "unmark sensitive variable")

	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("key")

	return cmd
}
