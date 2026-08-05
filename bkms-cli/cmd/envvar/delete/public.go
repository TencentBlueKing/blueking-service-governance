package delete

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewPublicCmd returns a Command instance for 'envvar delete public' sub command.
func NewPublicCmd() *cobra.Command {
	var workspaceID, id string

	cmd := &cobra.Command{
		Use:   "public",
		Short: "Delete a public (workspace/envType/env) scoped environment variable",
		Long:  `Delete an existing public scoped environment variable by its ID.`,
		Example: `  # Delete a public env var
  bkms-cli envvar delete public --id <varID>

  # Delete with explicit workspace
  bkms-cli envvar delete public --workspace <workspaceID> --id <varID>`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)

			if err := client.New().DeleteScopedEnvVar(cmd.Context(), workspaceID, id); err != nil {
				return errors.Wrap(err, "delete public env var")
			}

			fmt.Printf("Deleted env var: id=%s\n", id)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	cmd.Flags().StringVar(&id, "id", "", "scoped env var ID (required)")

	_ = cmd.MarkFlagRequired("id")

	return cmd
}
