/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

// Package migration contains one-off data migration commands.
package migration

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	componentmigrate "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/migrate"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

// NewMigrateComponentPatchCmd creates the component patch storage migration command.
func NewMigrateComponentPatchCmd() *cobra.Command {
	var srvCfg string
	var apply bool
	cmd := &cobra.Command{
		Use:   "migrate_component_patch",
		Short: "Migrate component definitions from output to patchers/specs arrays",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			migrator, err := initializeComponentPatchMigration(cmd.Context(), srvCfg)
			if err != nil {
				return err
			}
			dryRun := !apply
			result, runErr := migrator.Run(cmd.Context(), dryRun)
			if dryRun || runErr != nil {
				data, marshalErr := yaml.Marshal(result)
				if marshalErr != nil {
					return fmt.Errorf("marshal component patch migration result: %w", marshalErr)
				}
				if _, err = cmd.OutOrStdout().Write(data); err != nil {
					return fmt.Errorf("write component patch migration result: %w", err)
				}
			}
			if runErr != nil {
				return runErr
			}
			log.Infof(cmd.Context(),
				"component patch migration completed: dryRun=%t migrated=%d skipped=%d",
				dryRun, result.Summary.Migrated, result.Summary.Skipped)
			return nil
		},
	}
	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.Flags().BoolVar(&apply, "apply", false, "write changes to the database")
	_ = cmd.MarkFlagRequired("srvCfg")
	return cmd
}

func initializeComponentPatchMigration(ctx context.Context, srvCfg string) (*componentmigrate.Migrator, error) {
	cfg, err := config.Load(ctx, srvCfg)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	// stdout 专用于输出可解析的迁移结果，命令日志统一写入 stderr。
	cfg.Logging.Writers = []config.LoggingWriterConfig{{WriterName: log.WriterStderr}}
	if err = log.InitDefaultLogger(cfg.Logging); err != nil {
		return nil, fmt.Errorf("initialize logger: %w", err)
	}
	database.InitClient(ctx, cfg.Mongo)
	return componentmigrate.New(database.Client().Database(database.Name())), nil
}
