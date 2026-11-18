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
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bscpcfgmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkcc"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// NewRefreshWorkspaceAppEnvPermsCmd 为初始化完成（就绪、停用）的 workspace 的内置角色补发应用环境联合 action
// （view_app_in_env、edit_app_in_env、deploy_app_to_env）。
//
// 每个 workspace 依次：
//  1. 更新分级管理员的授权范围（PUT 覆盖语义），使其可以授予联合 action；
//  2. 按内置角色模板给管理员、开发者、SRE、运营者用户组追加授权（POST 合并语义）。
//
// 分级管理员是覆盖写入，因此会按本地数据重建所有平台（含 BSCP）的授权范围，平台信息不全的 workspace 会被跳过。
// 已没有绑定记录的 BSCP 服务不再保留在授权范围内，重新绑定时会再次加入。
// 开启按应用环境 action 鉴权前必须执行完成且失败数为 0。支持 --dry-run 预览。
func NewRefreshWorkspaceAppEnvPermsCmd() *cobra.Command {
	var srvCfg string
	var dryRun bool
	var workspaceIDs []string

	cmd := &cobra.Command{
		Use:   "refresh_workspace_app_env_perms",
		Short: "Grant app-env actions to the built-in roles of all workspaces in IAM",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg, err := config.Load(ctx, srvCfg)
			if err != nil {
				return errors.Wrap(err, "load config")
			}
			if err = log.InitDefaultLogger(cfg.Logging); err != nil {
				return errors.Wrap(err, "init logger")
			}

			database.InitClient(ctx, cfg.Mongo)
			storereg.Init(ctx)

			return refreshWorkspaceAppEnvPerms(ctx, storereg.G().DB, perm.NewManager(), workspaceIDs, dryRun)
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "list pending changes without actually granting IAM permissions")
	cmd.Flags().StringSliceVar(&workspaceIDs, "workspace-id", nil,
		"only refresh specified workspace IDs (comma-separated), useful for retrying failed ones")
	_ = cmd.MarkFlagRequired("srvCfg")

	return cmd
}

// refreshWorkspaceAppEnvPerms 遍历初始化完成的 workspace，逐个补发内置角色的应用环境联合 action。
//
// 读取的模型：workspace、app、bscpcfg env binding。
func refreshWorkspaceAppEnvPerms(
	ctx context.Context,
	db *mongo.Database,
	permMgr perm.Manager,
	filterIDs []string,
	dryRun bool,
) error {
	workspaces, err := listWorkspacesToRefresh(ctx, db, filterIDs)
	if err != nil {
		return err
	}

	var success int
	var skippedIDs, failedIDs []string
	for _, ws := range workspaces {
		data := buildWorkspaceData(ws)
		// 分级管理员是覆盖写入，平台信息不全会意外缩减其授权范围
		if err = hasAllPlatformData(data); err != nil {
			log.Errorf(ctx, "workspace %s platform data incomplete: %v, skipping", ws.ID, err)
			skippedIDs = append(skippedIDs, ws.ID)
			continue
		}
		// 应用、BSCP 绑定以及分级管理员与用户组记录按租户存储，切到该 workspace 所属租户
		wsCtx := store.ScopeToTenant(ctx, ws.TenantID)
		if data.BSCP, err = buildBSCPOptions(wsCtx, db, ws); err != nil {
			log.Errorf(ctx, "workspace %s build bscp options failed: %v", ws.ID, err)
			failedIDs = append(failedIDs, ws.ID)
			continue
		}

		if dryRun {
			bscpServices := 0
			if data.BSCP != nil {
				bscpServices = len(data.BSCP.Services)
			}
			fmt.Printf("[DRY-RUN] workspace %s (%s): WILL_REFRESH, bscp services=%d\n",
				ws.ID, ws.DisplayName, bscpServices)
			success++
			continue
		}

		if err = permMgr.UpdateWorkspaceAdmin(wsCtx, data); err != nil {
			log.Errorf(ctx, "update workspace admin for %s failed: %v", ws.ID, err)
			failedIDs = append(failedIDs, ws.ID)
			continue
		}
		if err = permMgr.UpdateWorkspaceScopeBuiltinRoles(wsCtx, data); err != nil {
			log.Errorf(ctx, "update workspace builtin roles for %s failed: %v", ws.ID, err)
			failedIDs = append(failedIDs, ws.ID)
			continue
		}
		log.Infof(ctx, "workspace %s app env perms refreshed successfully", ws.ID)
		success++
	}

	log.Infof(
		ctx, "refresh_workspace_app_env_perms completed: total=%d, success=%d, skipped=%d, failed=%d, dryRun=%v",
		len(workspaces), success, len(skippedIDs), len(failedIDs), dryRun,
	)
	if len(skippedIDs) > 0 || len(failedIDs) > 0 {
		if len(skippedIDs) > 0 {
			log.Warnf(
				ctx,
				"skipped workspaces must be fixed and refreshed before enabling app-env action checks: %s",
				strings.Join(skippedIDs, ","),
			)
		}
		if len(failedIDs) > 0 {
			log.Infof(
				ctx,
				"to retry failed workspaces, run: refresh_workspace_app_env_perms --srvCfg=<path> --workspace-id=%s",
				strings.Join(failedIDs, ","),
			)
		}
		return errors.Errorf(
			"refresh_workspace_app_env_perms incomplete: %d skipped, %d failed of %d workspaces",
			len(skippedIDs), len(failedIDs), len(workspaces),
		)
	}
	return nil
}

// listWorkspacesToRefresh 返回需要补发的 workspace，指定了 ID 时只返回这些，任一不存在即报错。
//
// 就绪和停用的空间都要补发：停用的空间重新启用时不会重新授权，这里不补就一直缺新 action。
// 初始化中的空间跳过，初始化完成时会按新模板授权。
func listWorkspacesToRefresh(
	ctx context.Context, db *mongo.Database, filterIDs []string,
) ([]workspace.Workspace, error) {
	workspaces, err := store.Of(db, workspace.Meta).List(ctx, store.TenantAll())
	if err != nil {
		return nil, errors.Wrap(err, "list workspaces")
	}
	if len(filterIDs) > 0 {
		allIDs := lo.Map(workspaces, func(ws workspace.Workspace, _ int) string { return ws.ID })
		if missing := lo.Without(filterIDs, allIDs...); len(missing) > 0 {
			return nil, errors.Errorf("workspaces not found: %s", strings.Join(missing, ","))
		}
		workspaces = lo.Filter(workspaces, func(ws workspace.Workspace, _ int) bool {
			return lo.Contains(filterIDs, ws.ID)
		})
	}
	initializing, workspaces := lo.FilterReject(workspaces, func(ws workspace.Workspace, _ int) bool {
		return ws.State == workspace.StateProcessing
	})
	for _, ws := range initializing {
		log.Infof(ctx, "workspace %s is initializing, skipping", ws.ID)
	}
	return workspaces, nil
}

// buildBSCPOptions 根据 workspace 下应用的 BSCP 环境绑定重建 BSCP 授权范围，未接入 BSCP 时返回 nil。
//
// BSCP App 的 IAM 资源 ID 为绑定记录中的 bscpAppID，名称为 bkms 应用 ID。
//
// 读取的模型：app、bscpcfg env binding。
func buildBSCPOptions(ctx context.Context, db *mongo.Database, ws workspace.Workspace) (*bkiam.BSCPOptions, error) {
	if ws.BkSystems.BkCCBizID == "" || ws.BkSystems.BkBSCPProjectID == "" {
		return nil, nil
	}

	apps, err := store.Of(db, bkmsapp.AppMeta).List(ctx, bkmsapp.InWorkspace(ws.ID))
	if err != nil {
		return nil, errors.Wrapf(err, "list apps of workspace %s", ws.ID)
	}
	appIDs := lo.Map(apps, func(app bkmsapp.Application, _ int) string { return app.ID })
	if len(appIDs) == 0 {
		return nil, nil
	}
	bindings, err := store.Of(db, bscpcfgmodel.EnvBindingMeta).
		List(ctx, store.Where("appID", bson.M{"$in": appIDs}))
	if err != nil {
		return nil, errors.Wrapf(err, "list bscp env bindings of workspace %s", ws.ID)
	}
	if len(bindings) == 0 {
		return nil, nil
	}

	services := lo.UniqBy(
		lo.Map(bindings, func(b bscpcfgmodel.EnvBinding, _ int) bkiam.BSCPService {
			return bkiam.BSCPService{ID: b.BscpAppID, Name: b.AppID}
		}),
		func(svc bkiam.BSCPService) string { return svc.ID },
	)
	return &bkiam.BSCPOptions{
		BizID:    ws.BkSystems.BkCCBizID,
		BizName:  getBizName(ctx, ws),
		Services: services,
	}, nil
}

// getBizName 查询业务名称；名称仅用于权限中心展示，查询失败时退化为业务 ID
func getBizName(ctx context.Context, ws workspace.Workspace) string {
	bizID := ws.BkSystems.BkCCBizID
	client, err := bkcc.New(auth.User{ID: ws.Creator})
	if err != nil {
		log.Warnf(ctx, "workspace %s init bkcc client failed, use biz id as name: %v", ws.ID, err)
		return bizID
	}
	business, err := client.GetBusinessByID(ctx, cast.ToInt64(bizID))
	if err != nil {
		log.Warnf(ctx, "workspace %s get business %s failed, use biz id as name: %v", ws.ID, bizID, err)
		return bizID
	}
	return business.BizName
}
