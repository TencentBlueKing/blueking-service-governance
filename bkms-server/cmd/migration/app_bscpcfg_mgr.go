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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
	svc "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/service"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// NewAppBscpCfgMgrCmd 创建 app-bscpcfg-mgr 命令，用于启用或停用应用的 bscp 配置。
func NewAppBscpCfgMgrCmd() *cobra.Command {
	var srvCfg string
	var operator string
	var accessToken string
	var execute bool

	cmd := &cobra.Command{
		Use:   "app-bscpcfg-mgr",
		Short: "Manage app bscp config",
	}
	cmd.PersistentFlags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.PersistentFlags().StringVar(&operator, "operator", "", "operator username (bk_username)")
	cmd.PersistentFlags().StringVar(&accessToken, "access-token", "", "operator access token")
	cmd.PersistentFlags().BoolVar(&execute, "execute", false, "actually execute (default is dry-run)")
	_ = cmd.MarkPersistentFlagRequired("srvCfg")
	_ = cmd.MarkPersistentFlagRequired("operator")
	_ = cmd.MarkPersistentFlagRequired("access-token")

	cmd.AddCommand(newEnableAppBscpCfgCmd(&srvCfg, &operator, &accessToken, &execute))
	cmd.AddCommand(newDisableAppBscpCfgCmd(&srvCfg, &operator, &accessToken, &execute))
	return cmd
}

func newEnableAppBscpCfgCmd(srvCfg, operator, accessToken *string, execute *bool) *cobra.Command {
	var app string
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable bscp config for app(s)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAppBscpCfg(cmd.Context(), *srvCfg, *operator, *accessToken, *execute, app, true)
		},
	}
	cmd.Flags().StringVar(&app, "app", "", "app ID(s), comma-separated")
	_ = cmd.MarkFlagRequired("app")
	return cmd
}

func newDisableAppBscpCfgCmd(srvCfg, operator, accessToken *string, execute *bool) *cobra.Command {
	var app string
	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable bscp config for app(s)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAppBscpCfg(cmd.Context(), *srvCfg, *operator, *accessToken, *execute, app, false)
		},
	}
	cmd.Flags().StringVar(&app, "app", "", "app ID(s), comma-separated")
	_ = cmd.MarkFlagRequired("app")
	return cmd
}

// runAppBscpCfg 启用或停用指定应用的 bscp 配置。
func runAppBscpCfg(
	ctx context.Context,
	srvCfg, operator, accessToken string,
	execute bool,
	app string,
	enable bool,
) error {
	cfg, err := config.Load(ctx, srvCfg)
	if err != nil {
		return errors.Wrap(err, "load config")
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

	action := "disable"
	if enable {
		action = "enable"
	}
	if !execute {
		log.Infof(ctx, "[DRY-RUN] would %s bscp config for app(s): %v", action, appIDs)
		return nil
	}

	user := auth.MustGetUser(ctx)
	for _, id := range appIDs {
		if err := setBscpCfgForApp(ctx, reg, user, id, enable); err != nil {
			return errors.Wrapf(err, "%s bscp config for app %s", action, id)
		}
		log.Infof(ctx, "bscp config %sd for app %s", action, id)
	}
	return nil
}

func parseAppIDs(raw string) []string {
	return lo.FilterMap(strings.Split(raw, ","), func(s string, _ int) (string, bool) {
		s = strings.TrimSpace(s)
		return s, s != ""
	})
}

// setBscpCfgForApp 启用或停用指定 appID 的 bscp 配置。
func setBscpCfgForApp(
	ctx context.Context,
	reg *storereg.Registry,
	user auth.User,
	appID string,
	enable bool,
) error {
	if enable {
		return enableBscpCfgForApp(ctx, reg, user, appID)
	}

	if err := reg.BscpCfgStore.UpsertFeatureFlag(ctx, &model.FeatureFlag{
		AppID:    appID,
		Enabled:  false,
		Operator: user.ID,
	}); err != nil {
		return errors.Wrapf(err, "upsert feature flag for app %s", appID)
	}
	return nil
}

// enableBscpCfgForApp 启用指定 appID 的 bscp 配置：初始化 Credential、PostHook、Metadata 与 FeatureFlag。
func enableBscpCfgForApp(
	ctx context.Context,
	reg *storereg.Registry,
	user auth.User,
	appID string,
) error {
	app, err := reg.AppStore.GetApp(ctx, appID)
	if err != nil {
		return errors.Wrapf(err, "get app %s", appID)
	}

	ws, err := reg.WorkspaceStore.Get(ctx, app.WorkspaceID)
	if err != nil {
		return errors.Wrapf(err, "get workspace %s", app.WorkspaceID)
	}
	if ws.BkSystems.BkCCBizID == "" {
		return errors.Errorf("workspace %s missing BkCCBizID", ws.ID)
	}
	if ws.BkSystems.BkBSCPProjectID == "" {
		return errors.Errorf("workspace %s missing BkBSCPProjectID, run bind-bscp-project first", ws.ID)
	}

	mgr, err := svc.NewManager(user, reg.BscpCfgStore)
	if err != nil {
		return errors.Wrap(err, "create manager")
	}

	var workloadName, workloadKind string
	if app.Type == bkmsapp.AppTypeTRPC {
		workloadName = app.Name
		workloadKind = k8skind.GameDeploy
	}

	if _, initErr := mgr.InitMetadata(ctx, &svc.InitMetadataParams{
		AppID:          app.ID,
		WorkloadName:   workloadName,
		WorkloadKind:   workloadKind,
		BscpBizID:      ws.BkSystems.BkCCBizID,
		BscpProjectID:  ws.BkSystems.BkBSCPProjectID,
		BscpProjectKey: ws.BkSystems.BkBSCPProjectKey,
		Operator:       user.ID,
	}); initErr != nil {
		return errors.Wrapf(initErr, "init metadata for app %s", appID)
	}

	if upsertErr := reg.BscpCfgStore.UpsertFeatureFlag(ctx, &model.FeatureFlag{
		AppID:    appID,
		Enabled:  true,
		Operator: user.ID,
	}); upsertErr != nil {
		return errors.Wrapf(upsertErr, "upsert feature flag for app %s", appID)
	}

	return nil
}
