// Package update provides the bkms-cli self-update command.
package update

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/updater"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// updateCheckTimeout bounds metadata-only checks; an installation continues to
// use the command context because downloading and replacing may take longer.
const updateCheckTimeout = 15 * time.Second

// NewCmd creates the self-update command.
func NewCmd() *cobra.Command {
	var checkOnly bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for and install bkms-cli updates.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				info updater.Info
				err  error
			)
			if checkOnly {
				checkContext, cancel := context.WithTimeout(cmd.Context(), updateCheckTimeout)
				defer cancel()
				info, err = updater.Check(checkContext)
			} else {
				info, err = updater.Update(cmd.Context())
			}
			if err != nil {
				return err
			}

			// Print the result of the update check or installation.
			switch {
			case !info.Available:
				console.Info("bkms-cli v%s is up to date", info.CurrentVersion)
			case checkOnly:
				console.Info("bkms-cli v%s is available (current: v%s)", info.LatestVersion, info.CurrentVersion)
			default:
				console.Info("bkms-cli updated from v%s to v%s", info.CurrentVersion, info.LatestVersion)
			}
			return nil
		},
		Annotations: map[string]string{
			cmdutil.SkipAuthAnnotationKey: "true",
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "check for updates without installing")
	return cmd
}
