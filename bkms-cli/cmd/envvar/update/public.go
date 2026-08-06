package update

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	envhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/env"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewPublicCmd returns a Command instance for 'envvar update public' sub command.
func NewPublicCmd() *cobra.Command {
	var workspaceID, key, updatedKey, scopeType, scopeValue, value, description string
	var sensitive, noSensitive bool

	cmd := &cobra.Command{
		Use:   "public",
		Short: "Update a public (workspace/envType) scoped environment variable",
		Long: `Update an existing public scoped environment variable.

The --key flag specifies the variable key to update.
The --scope-type flag specifies the scope type (workspace or envType).
The --scope-value flag is required when scope-type is envType.
The --updated-key flag allows renaming the variable key (optional, defaults to --key).
Use --sensitive to mark as sensitive, or --no-sensitive to unmark.
Only specified fields will be updated.`,
		Example: `  # Update value (workspace scope)
  bkms-cli envvar update public --key MY_VAR --scope-type workspace --value new-value

  # Update value (envType scope)
  bkms-cli envvar update public --key MY_VAR --scope-type envType --scope-value test --value new-value

  # Rename key
  bkms-cli envvar update public --key MY_VAR --scope-type workspace --updated-key NEW_KEY

  # Mark as sensitive
  bkms-cli envvar update public --key MY_VAR --scope-type workspace --sensitive`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			key = strings.TrimSpace(key)
			updatedKey = strings.TrimSpace(updatedKey)

			if sensitive && noSensitive {
				return errors.New("--sensitive and --no-sensitive cannot be used together")
			}
			// 校验 scope-type / scope-value 组合
			if scopeType == "envType" && scopeValue == "" {
				return errors.New("--scope-value is required when --scope-type is envType")
			}
			if scopeType == "workspace" {
				scopeValue = "" // workspace 级别忽略 scope-value
			}

			workspaceID = cmdutil.GetWorkspaceID(workspaceID)
			cli := client.New()

			varID, err := envhandler.ResolveScopedEnvVarID(cmd.Context(), cli, workspaceID, key, scopeType, scopeValue)
			if err != nil {
				return err
			}

			// --updated-key 不传时默认使用 --key 的值（不改名）
			effectiveKey := updatedKey
			if effectiveKey == "" {
				effectiveKey = key
			}

			opts := client.UpdateScopedEnvVarOptions{
				Key:         effectiveKey,
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

			result, err := cli.UpdateScopedEnvVar(cmd.Context(), workspaceID, varID, opts)
			if err != nil {
				return errors.Wrap(err, "update public env var")
			}

			fmt.Printf("Updated env var: key=%s, id=%s\n", result.Key, result.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVar(&key, "key", "", "current environment variable key (required)")
	cmd.Flags().StringVar(&updatedKey, "updated-key", "", "new environment variable key (optional, defaults to --key)")
	cmd.Flags().StringVar(&scopeType, "scope-type", "", "scope type: workspace or envType (required)")
	cmd.Flags().StringVar(&scopeValue, "scope-value", "", "scope value (required for envType)")
	cmd.Flags().StringVar(&value, "value", "", "environment variable value")
	cmd.Flags().StringVar(&description, "description", "", "variable description")
	cmd.Flags().BoolVar(&sensitive, "sensitive", false, "mark as sensitive variable")
	cmd.Flags().BoolVar(&noSensitive, "no-sensitive", false, "unmark sensitive variable")

	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("scope-type")

	return cmd
}
