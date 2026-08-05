// Package cmd defines the commands.
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/cmd/migration"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

var rootCmd = &cobra.Command{
	Use:   "bkms-server",
	Short: "bkms server",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info(cmd.Context(), "welcome to use bkms-server, use `bkms-server -h` for help")
	},
}

func init() {
	rootCmd.AddCommand(NewMigrateCmd())
	rootCmd.AddCommand(migration.NewLoadBuiltinComponentCmd())
	rootCmd.AddCommand(migration.NewMigrateComponentPatchCmd())
	rootCmd.AddCommand(migration.NewMigrateTkeRouteEniComponentCmd())
	rootCmd.AddCommand(migration.NewMigrateIAMSystemModelCmd())
	rootCmd.AddCommand(migration.NewCleanupExpiredWorkspaceTempAdminsCmd())
	rootCmd.AddCommand(migration.NewCleanupOrphanAppConfigFileVersionsCmd())
	rootCmd.AddCommand(migration.NewUpsertRuntimeImageCmd())
}

// Execute ...
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err.Error())
	}
}
