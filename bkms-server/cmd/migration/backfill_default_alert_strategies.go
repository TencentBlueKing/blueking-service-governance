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
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/redis"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/usergroup"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

const skipReasonExistingStrategies = "existing alert strategies found"

type backfillDefaultAlertStrategiesOptions struct {
	WorkspaceID string
	AppID       string
	Execute     bool
}

type backfillDefaultAlertStrategiesSummary struct {
	WorkspaceID           string
	AppID                 string
	AppName               string
	Executed              bool
	Skipped               bool
	SkipReason            string
	ExistingStrategyCount int
	NoticeGroupIDs        []int64
}

type backfillDefaultAlertStrategiesDeps struct {
	appStore              bkmsapp.ApplicationStore
	workspaceStore        workspace.WorkspaceStore
	strategyStore         alertstrategy.Store
	resolveNoticeGroupIDs func(ctx context.Context, ws *workspace.Workspace) ([]int64, error)
	initDefaultStrategies func(ctx context.Context, ws *workspace.Workspace, app *bkmsapp.Application, noticeGroupIDs []int64) error
}

// NewBackfillDefaultAlertStrategiesCmd 创建一个用于定向补刷默认告警策略的一次性命令。
func NewBackfillDefaultAlertStrategiesCmd() *cobra.Command {
	var srvCfg string
	opts := backfillDefaultAlertStrategiesOptions{}

	cmd := &cobra.Command{
		Use:   "backfill_default_alert_strategies",
		Short: "Preview or backfill default alert strategies for one workspace/app pair",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := auth.WithMaintenanceUser(cmd.Context())
			cfg, err := config.Load(ctx, srvCfg)
			if err != nil {
				return errors.Wrap(err, "load config")
			}
			if err = log.InitDefaultLogger(cfg.Logging); err != nil {
				return errors.Wrap(err, "init logger")
			}

			database.InitClient(ctx, cfg.Mongo)
			redis.InitClient(ctx, cfg.Redis)
			storereg.Init(ctx)

			operator := auth.MustGetUser(ctx).ID
			summary, err := runBackfillDefaultAlertStrategies(ctx, backfillDefaultAlertStrategiesDeps{
				appStore:       storereg.G().AppStore,
				workspaceStore: storereg.G().WorkspaceStore,
				strategyStore:  storereg.G().AlertStrategyStore,
				resolveNoticeGroupIDs: func(ctx context.Context, ws *workspace.Workspace) ([]int64, error) {
					return usergroup.ResolveDefaultAlertNoticeGroupIDs(
						ctx,
						ws,
						usergroup.New(),
						perm.NewManager(),
						operator,
					)
				},
				initDefaultStrategies: func(
					ctx context.Context,
					_ *workspace.Workspace,
					app *bkmsapp.Application,
					noticeGroupIDs []int64,
				) error {
					return alertstrategy.NewService(
						storereg.G().AlertStrategyStore,
						storereg.G().EnvStore,
						storereg.G().AppStore,
						storereg.G().ResourceSnapshotStore,
					).InitDefaultAlertStrategiesForApp(
						ctx,
						app.WorkspaceID,
						app.ID,
						app.Name,
						operator,
						noticeGroupIDs,
					)
				},
			}, opts)
			if err != nil {
				return err
			}
			writeBackfillDefaultAlertStrategiesOutput(cmd.OutOrStdout(), summary)
			return nil
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.Flags().StringVar(&opts.WorkspaceID, "workspace-id", "", "target workspace ID")
	cmd.Flags().StringVar(&opts.AppID, "app-id", "", "target app ID")
	cmd.Flags().
		BoolVar(&opts.Execute, "execute", false, "actually write default alert strategies; default is preview only")
	_ = cmd.MarkFlagRequired("srvCfg")
	_ = cmd.MarkFlagRequired("workspace-id")
	_ = cmd.MarkFlagRequired("app-id")

	return cmd
}

func runBackfillDefaultAlertStrategies(
	ctx context.Context,
	deps backfillDefaultAlertStrategiesDeps,
	opts backfillDefaultAlertStrategiesOptions,
) (*backfillDefaultAlertStrategiesSummary, error) {
	if opts.WorkspaceID == "" {
		return nil, errors.New("workspace-id is required")
	}
	if opts.AppID == "" {
		return nil, errors.New("app-id is required")
	}

	app, err := deps.appStore.GetApp(ctx, opts.AppID)
	if err != nil {
		return nil, errors.Wrap(err, "get app")
	}
	if app.WorkspaceID != opts.WorkspaceID {
		return nil, errors.Errorf("app %s does not belong to workspace %s", app.ID, opts.WorkspaceID)
	}

	ws, err := deps.workspaceStore.Get(ctx, opts.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "get workspace")
	}

	existingStrategies, err := deps.strategyStore.ListByApp(ctx, opts.WorkspaceID, opts.AppID)
	if err != nil {
		return nil, errors.Wrap(err, "list existing alert strategies by app")
	}

	summary := &backfillDefaultAlertStrategiesSummary{
		WorkspaceID:           opts.WorkspaceID,
		AppID:                 opts.AppID,
		AppName:               app.Name,
		ExistingStrategyCount: len(existingStrategies),
	}
	if len(existingStrategies) > 0 {
		summary.Skipped = true
		summary.SkipReason = skipReasonExistingStrategies
		return summary, nil
	}
	if !opts.Execute {
		return summary, nil
	}

	noticeGroupIDs, err := deps.resolveNoticeGroupIDs(ctx, ws)
	if err != nil {
		return nil, errors.Wrap(err, "resolve default alert notice group IDs")
	}
	if err = deps.initDefaultStrategies(ctx, ws, app, noticeGroupIDs); err != nil {
		return nil, errors.Wrap(err, "init default alert strategies for app")
	}

	summary.Executed = true
	summary.NoticeGroupIDs = append([]int64(nil), noticeGroupIDs...)
	return summary, nil
}

func writeBackfillDefaultAlertStrategiesOutput(w io.Writer, summary *backfillDefaultAlertStrategiesSummary) {
	_, _ = fmt.Fprintf(w, "workspace: %s\n", summary.WorkspaceID)
	_, _ = fmt.Fprintf(w, "app: %s\n", summary.AppID)
	_, _ = fmt.Fprintf(w, "appName: %s\n", summary.AppName)
	_, _ = fmt.Fprintf(w, "existingStrategyCount: %d\n", summary.ExistingStrategyCount)
	if summary.Skipped {
		_, _ = fmt.Fprintf(w, "status: skipped (%s)\n", summary.SkipReason)
		return
	}
	if !summary.Executed {
		_, _ = fmt.Fprintln(w, "status: preview-only")
		return
	}
	_, _ = fmt.Fprintf(w, "noticeGroupIDs: %v\n", summary.NoticeGroupIDs)
	_, _ = fmt.Fprintln(w, "status: executed")
}
