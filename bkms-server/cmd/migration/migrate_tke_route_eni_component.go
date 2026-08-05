// Package migration contains one-off data migration commands.
package migration

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	appspecmigrate "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/migrate"
)

// NewMigrateTkeRouteEniComponentCmd creates the one-off tkeRouteEni component → AppSpec migration command.
func NewMigrateTkeRouteEniComponentCmd() *cobra.Command {
	var srvCfg string
	var apply bool
	cmd := &cobra.Command{
		Use:   "migrate_tke_route_eni_component",
		Short: "Migrate tkeRouteEni component mounts to AppSpec tkeRouteEni section",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			migrator, err := initializeTkeRouteEniMigration(cmd.Context(), srvCfg)
			if err != nil {
				return err
			}
			result, err := migrator.Run(cmd.Context(), !apply)
			data, marshalErr := yaml.Marshal(result)
			if marshalErr != nil {
				return fmt.Errorf("marshal migration result: %w", marshalErr)
			}
			if _, writeErr := cmd.OutOrStdout().Write(data); writeErr != nil {
				return writeErr
			}
			return err
		},
	}
	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.Flags().BoolVar(&apply, "apply", false, "write changes to the database")
	_ = cmd.MarkFlagRequired("srvCfg")
	return cmd
}

func initializeTkeRouteEniMigration(ctx context.Context, srvCfg string) (*appspecmigrate.Migrator, error) {
	cfg, err := config.Load(ctx, srvCfg)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	cfg.Logging.Writers = []config.LoggingWriterConfig{{WriterName: log.WriterStderr}}
	if err = log.InitDefaultLogger(cfg.Logging); err != nil {
		return nil, fmt.Errorf("initialize logger: %w", err)
	}
	database.InitClient(ctx, cfg.Mongo)

	client, dbName := database.Client(), database.Name()
	appStore, err := bkmsapp.NewApplicationStoreMongo(client, dbName)
	if err != nil {
		return nil, err
	}
	appSpecStore, err := appspec.NewAppSpecStoreMongo(client, dbName)
	if err != nil {
		return nil, err
	}
	appModelStore, err := appmodel.NewAppModelStoreMongo(client, dbName)
	if err != nil {
		return nil, err
	}
	wsCompStore, err := workspace.NewWorkspaceCompsStoreMongo(client, dbName)
	if err != nil {
		return nil, err
	}
	compDefStore, err := component.NewComponentDefStoreMongo(client, dbName)
	if err != nil {
		return nil, err
	}
	return appspecmigrate.New(
		client.Database(dbName), appStore, appSpecStore, appModelStore, wsCompStore, compDefStore,
	), nil
}
