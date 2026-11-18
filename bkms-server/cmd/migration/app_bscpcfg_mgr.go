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

// Package migration 提供一次性数据迁移与运维命令。
package migration

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod/mods"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	svc "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/service"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// NewAppBscpCfgMgrCmd 创建 app-bscpcfg-mgr 命令，用于初始化应用的 bscp 配置。
//
// 配置管理是否可用由空间级配置开关决定（平台管理员在平台管理中设置），本命令不再负责开关。
func NewAppBscpCfgMgrCmd() *cobra.Command {
	var srvCfg string
	var operator string
	var execute bool

	cmd := &cobra.Command{
		Use:   "app-bscpcfg-mgr",
		Short: "Manage app bscp config",
		Long: "Initialize BSCP config management for apps (post hook and metadata). " +
			"Whether config management takes effect is controlled by the workspace cfg switch, " +
			"which is set by platform administrators.",
	}
	cmd.PersistentFlags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.PersistentFlags().StringVar(&operator, "operator", "", "operator username (bk_username)")
	cmd.PersistentFlags().BoolVar(&execute, "execute", false, "actually execute (default is dry-run)")
	_ = cmd.MarkPersistentFlagRequired("srvCfg")
	_ = cmd.MarkPersistentFlagRequired("operator")

	cmd.AddCommand(newEnableAppBscpCfgCmd(&srvCfg, &operator, &execute))
	return cmd
}

func newEnableAppBscpCfgCmd(srvCfg, operator *string, execute *bool) *cobra.Command {
	var app string
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Initialize bscp config for app(s)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAppBscpCfg(cmd.Context(), *srvCfg, *operator, *execute, app)
		},
	}
	cmd.Flags().StringVar(&app, "app", "", "app ID(s), comma-separated")
	_ = cmd.MarkFlagRequired("app")
	return cmd
}

// runAppBscpCfg 初始化指定应用的 bscp 配置。
func runAppBscpCfg(
	ctx context.Context,
	srvCfg, operator string,
	execute bool,
	app string,
) error {
	cfg, err := config.Load(ctx, srvCfg)
	if err != nil {
		return errors.Wrap(err, "load config")
	}

	// 交互式读取 access token（密文输入，不回显），避免经命令行参数明文传递
	accessToken, err := readAccessToken()
	if err != nil {
		return err
	}

	ctx = auth.WithUser(ctx, auth.User{
		ID:   operator,
		Cred: auth.UserCredential{AccessToken: accessToken},
	})
	if err = log.InitDefaultLogger(cfg.Logging); err != nil {
		return errors.Wrap(err, "init logger")
	}

	database.InitClient(ctx, cfg.Mongo)
	storereg.Init(ctx)
	reg := storereg.G()

	appIDs := parseAppIDs(app)
	if len(appIDs) == 0 {
		return errors.New("--app must contain at least one app ID")
	}

	if !execute {
		log.Infof(ctx, "[DRY-RUN] would initialize bscp config for app(s): %v", appIDs)
		return nil
	}

	user := auth.MustGetUser(ctx)
	for _, id := range appIDs {
		if err := initBscpCfgForApp(ctx, reg, user, id); err != nil {
			return errors.Wrapf(err, "initialize bscp config for app %s", id)
		}
		log.Infof(ctx, "bscp config initialized for app %s", id)
	}
	return nil
}

func parseAppIDs(raw string) []string {
	return lo.FilterMap(strings.Split(raw, ","), func(s string, _ int) (string, bool) {
		s = strings.TrimSpace(s)
		return s, s != ""
	})
}

// initBscpCfgForApp 初始化指定应用的 bscp 配置：初始化 PostHook 与 Metadata（幂等）。
func initBscpCfgForApp(
	ctx context.Context,
	reg *storereg.Registry,
	user auth.User,
	appID string,
) error {
	app, err := store.Of(reg.DB, bkmsapp.AppMeta).Get(ctx, bkmsapp.ByID(appID), store.TenantAll())
	if err != nil {
		return errors.Wrapf(err, "get app %s", appID)
	}
	ctx = store.ScopeToTenant(ctx, app.TenantID)

	ws, err := store.Of(reg.DB, workspace.Meta).Get(ctx, workspace.ByWorkspaceID(app.WorkspaceID))
	if err != nil {
		return errors.Wrapf(err, "get workspace %s", app.WorkspaceID)
	}
	if ws.BkSystems.BkCCBizID == "" {
		return errors.Errorf("workspace %s missing BkCCBizID", ws.ID)
	}
	if ws.BkSystems.BkBSCPProjectID == "" {
		return errors.Errorf("workspace %s missing BkBSCPProjectID, run bind-bscp-project first", ws.ID)
	}

	mgr, err := svc.NewManager(user, reg.DB)
	if err != nil {
		return errors.Wrap(err, "create manager")
	}

	var workloadName, workloadKind string
	runtimeMods := mods.Of(app)
	// 自动填充由能力声明决定；工作负载类型取 AppModel 渲染的默认类型
	if runtimeMods.BSCPWorkloadMeta.Supported {
		workloadName = app.Name
		workloadKind = k8skind.GameDeploy
	}

	if _, initErr := mgr.InitMetadata(ctx, &svc.InitMetadataParams{
		AppID:         app.ID,
		WorkloadName:  workloadName,
		WorkloadKind:  workloadKind,
		BscpBizID:     ws.BkSystems.BkCCBizID,
		BscpProjectID: ws.BkSystems.BkBSCPProjectID,
		Operator:      user.ID,
	}); initErr != nil {
		return errors.Wrapf(initErr, "init metadata for app %s", appID)
	}
	return nil
}
