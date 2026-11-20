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

package migration

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// NewGrantPlatformAccountCmd 把平台级虚拟账号加入存量空间的管理员角色。
//
// 服务启用虚拟账号前，用开启了虚拟账号的配置执行；启用后新建的空间会自动加入。
// 处理所有状态的空间，可重复执行，支持 --dry-run 预览。
func NewGrantPlatformAccountCmd() *cobra.Command {
	var srvCfg string
	var dryRun bool
	var workspaceIDs []string

	cmd := &cobra.Command{
		Use:   "grant_platform_account",
		Short: "Grant permissions of the platform account in all workspaces",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg, err := config.Load(ctx, srvCfg)
			if err != nil {
				return errors.Wrap(err, "load config")
			}
			if err = log.InitDefaultLogger(cfg.Logging); err != nil {
				return errors.Wrap(err, "init logger")
			}
			if !cfg.PlatformAccount.Enabled {
				return errors.New("platform account is not enabled")
			}

			database.InitClient(ctx, cfg.Mongo)
			storereg.Init(ctx)

			return grantPlatformAccount(ctx, storereg.G().DB, perm.NewManager(), workspaceIDs, dryRun)
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "list pending changes without actually granting IAM permissions")
	cmd.Flags().StringSliceVar(&workspaceIDs, "workspace-id", nil,
		"only grant specified workspace IDs (comma-separated), useful for retrying failed ones")
	_ = cmd.MarkFlagRequired("srvCfg")

	return cmd
}

// grantPlatformAccount 遍历空间，逐个把虚拟账号加入管理员角色。
func grantPlatformAccount(
	ctx context.Context,
	db *mongo.Database,
	permMgr perm.Manager,
	filterIDs []string,
	dryRun bool,
) error {
	workspaces, err := store.Of(db, workspace.Meta).List(ctx, store.TenantAll())
	if err != nil {
		return errors.Wrap(err, "list workspaces")
	}
	if len(filterIDs) > 0 {
		allIDs := lo.Map(workspaces, func(ws workspace.Workspace, _ int) string { return ws.ID })
		if missing := lo.Without(filterIDs, allIDs...); len(missing) > 0 {
			return errors.Errorf("workspaces not found: %s", strings.Join(missing, ","))
		}
		workspaces = lo.Filter(workspaces, func(ws workspace.Workspace, _ int) bool {
			return lo.Contains(filterIDs, ws.ID)
		})
	}
	var failedIDs []string
	for _, ws := range workspaces {
		if dryRun {
			fmt.Printf("[DRY-RUN] workspace %s (%s, %s): WILL_GRANT\n", ws.ID, ws.DisplayName, ws.State)
			continue
		}

		// 角色记录按租户存储，切到该 workspace 所属租户
		if err = workspace.AddPlatformAccount(store.ScopeToTenant(ctx, ws.TenantID), permMgr, ws.ID); err != nil {
			log.Errorf(ctx, "add platform account to workspace %s failed: %v", ws.ID, err)
			failedIDs = append(failedIDs, ws.ID)
			continue
		}
		log.Infof(ctx, "workspace %s platform account granted", ws.ID)
	}

	log.Infof(
		ctx, "grant_platform_account completed: total=%d, failed=%d, dryRun=%v",
		len(workspaces), len(failedIDs), dryRun,
	)
	if len(failedIDs) > 0 {
		log.Infof(
			ctx, "to retry failed workspaces, run: grant_platform_account --srvCfg=<path> --workspace-id=%s",
			strings.Join(failedIDs, ","),
		)
		return errors.Errorf("grant_platform_account incomplete: %d failed of %d workspaces",
			len(failedIDs), len(workspaces))
	}
	return nil
}
