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
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

const (
	skipReasonNotTrpcOrTaf        = "not trpc/taf"
	skipReasonNoFrameworkDef      = "no framework def"
	skipReasonNoPrimaryDefault    = "no framework def with a default file"
	skipReasonAlreadyHealthy      = "already one framework def and no env-mode repair"
	skipReasonConflicts           = "conflicts; skip apply for this app"
	leftoverReasonDefaultOnOrphan = "default file on non-primary framework def"
	leftoverReasonMissingDefID    = "file missing defID"
)

// repairTrpcTafFrameworkDefsOptions 描述本次治理任务的执行参数。
type repairTrpcTafFrameworkDefsOptions struct {
	AppID          string
	WorkspaceID    string
	Execute        bool
	FailOnConflict bool
}

// repairFrameworkDefMove 一条可安全改挂的环境 file：从孤儿 def 挪到主 def。
// 只改 defID，不改 content / overlayContent。
type repairFrameworkDefMove struct {
	FileID      string `json:"fileId"`
	EnvName     string `json:"envName"`
	Type        string `json:"type"`
	FromDefID   string `json:"fromDefId"`
	FromDefName string `json:"fromDefName"`
	ToDefID     string `json:"toDefId"`
	ContentLen  int    `json:"contentLen"`
	OverlayLen  int    `json:"overlayLen"`
}

// repairFrameworkDefConflict 无法自动改挂：主 def（或另一条孤儿）已占用同一 envName。
// 有 conflict 的应用整单不落库，避免撞 defID+envName 唯一索引。
type repairFrameworkDefConflict struct {
	FileID      string `json:"fileId"`
	EnvName     string `json:"envName"`
	FromDefID   string `json:"fromDefId"`
	FromDefName string `json:"fromDefName"`
	Reason      string `json:"reason"`
}

// repairFrameworkDefLeftover 需要人工看、命令不会自动处理的 file。
// 典型：孤儿 def 上还有默认文件（可能是第二条真文件），或 file 缺 defID。
type repairFrameworkDefLeftover struct {
	FileID  string `json:"fileId"`
	DefID   string `json:"defId"`
	EnvName string `json:"envName"`
	Reason  string `json:"reason"`
}

// repairTrpcTafFrameworkDefsAppPlan 单个应用的预览 / 执行计划。
type repairTrpcTafFrameworkDefsAppPlan struct {
	AppID          string                       `json:"appId"`
	AppName        string                       `json:"appName"`
	AppType        string                       `json:"appType"`
	Skipped        bool                         `json:"skipped"`
	SkipReason     string                       `json:"skipReason,omitempty"`
	PrimaryDefID   string                       `json:"primaryDefId,omitempty"`
	PrimaryDefName string                       `json:"primaryDefName,omitempty"`
	SetIndependent bool                         `json:"setIndependent"`
	Moves          []repairFrameworkDefMove     `json:"moves"`
	Conflicts      []repairFrameworkDefConflict `json:"conflicts"`
	Leftovers      []repairFrameworkDefLeftover `json:"leftovers"`
	DeleteDefIDs   []string                     `json:"deleteDefIds"`
	DeleteDefNames []string                     `json:"deleteDefNames"`
	Applied        bool                         `json:"applied"`
}

// repairTrpcTafFrameworkDefsSummary 全量预览 / 执行结果，命令 stdout 打成 JSON。
type repairTrpcTafFrameworkDefsSummary struct {
	Execute      bool                                `json:"execute"`
	AppCount     int                                 `json:"appCount"`
	Repairable   int                                 `json:"repairable"`
	Skipped      int                                 `json:"skipped"`
	ConflictApps int                                 `json:"conflictApps"`
	Applied      int                                 `json:"applied"`
	Plans        []repairTrpcTafFrameworkDefsAppPlan `json:"plans"`
}

// repairTrpcTafFrameworkDefsDeps 命令依赖的 store，便于单测注入。
type repairTrpcTafFrameworkDefsDeps struct {
	appStore  bkmsapp.ApplicationStore
	defStore  appcfg.AppConfigFileDefStore
	fileStore appcfg.AppConfigFileStore
}

// NewRepairTrpcTafFrameworkDefsCmd 创建一个一次性 Mongo 数据治理命令，
// 把 tRPC/TAF 上被旧接口拆散的 framework 环境 overlay 收回到主 def。
//
// 这不是 db/migrations 里的 JSON migrate：golang-migrate 无法按应用选型、处理冲突，
// 也不会在发版时自动执行。必须人工 dry-run 确认后再加 --execute。
//
// 背景：
//   - 框架页第一次按环境保存曾走旧 POST /apps/{appID}/app-config-files
//   - Create 每次新建一条 def，name 用环境名，环境 overlay 成了独立 framework def
//   - 部署 GetFrameworkMountableFile 只认「带默认文件的 def + 同 def 环境实例」
//   - 孤儿 overlay def 没有 envName="" 的默认文件，整组被跳过，ConfigMap 只剩默认 {} + 北极星注入
//
// 处理策略：
//   - 默认仅预览：输出每个应用的 plan（moves / conflicts / leftovers / deleteDefIds）
//   - --execute 才改库：改挂 file.defID、必要时把主 def 设为独立配置、删除迁空的孤儿 def
//   - 只处理 applications.type=trpc/taf；Helm 多 values / overlay values 是正常多 def，一律跳过
//   - 不改 file 正文，不更新集群 ConfigMap；刷完需对该环境重新部署才会生效
//   - 主 def 已有同 envName、或两条孤儿抢同一环境 → conflict，该应用整单不落库
//   - 不要重跑 000011 / 000014：那两次不会删除 name=环境名 的后建 def
//
// 示例：
//
//	bkms-server repair_trpc_taf_framework_defs --srvCfg ./biz.yaml
//	bkms-server repair_trpc_taf_framework_defs --srvCfg ./biz.yaml --app-id porter-taf-helloworld-uwpdse
//	bkms-server repair_trpc_taf_framework_defs --srvCfg ./biz.yaml --execute
func NewRepairTrpcTafFrameworkDefsCmd() *cobra.Command {
	var srvCfg string
	opts := repairTrpcTafFrameworkDefsOptions{}

	cmd := &cobra.Command{
		Use:   "repair_trpc_taf_framework_defs",
		Short: "Preview or repair orphan tRPC/TAF framework defs created by the old env overlay API",
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
			storereg.Init(ctx)

			summary, err := runRepairTrpcTafFrameworkDefs(ctx, repairTrpcTafFrameworkDefsDeps{
				appStore:  storereg.G().AppStore,
				defStore:  storereg.G().AppConfigFileDefStore,
				fileStore: storereg.G().AppConfigFileStore,
			}, opts)
			if err != nil {
				return err
			}
			return writeRepairTrpcTafFrameworkDefsOutput(cmd.OutOrStdout(), summary)
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.Flags().StringVar(&opts.AppID, "app-id", "", "only process a specific app (applications.id)")
	cmd.Flags().StringVar(&opts.WorkspaceID, "workspace-id", "", "only process apps in this workspace")
	cmd.Flags().BoolVar(&opts.Execute, "execute", false, "apply the plan; default is dry-run")
	cmd.Flags().BoolVar(
		&opts.FailOnConflict,
		"fail-on-conflict",
		false,
		"return error when any app has a defID+envName conflict",
	)
	_ = cmd.MarkFlagRequired("srvCfg")
	return cmd
}

// runRepairTrpcTafFrameworkDefs 先为每个目标应用生成 plan，再按开关决定是否落库。
func runRepairTrpcTafFrameworkDefs(
	ctx context.Context,
	deps repairTrpcTafFrameworkDefsDeps,
	opts repairTrpcTafFrameworkDefsOptions,
) (*repairTrpcTafFrameworkDefsSummary, error) {
	apps, err := listTrpcTafAppsForRepair(ctx, deps.appStore, opts)
	if err != nil {
		return nil, err
	}

	summary := &repairTrpcTafFrameworkDefsSummary{
		Execute: opts.Execute,
		Plans:   make([]repairTrpcTafFrameworkDefsAppPlan, 0, len(apps)),
	}
	for _, app := range apps {
		plan, err := buildRepairTrpcTafFrameworkDefsAppPlan(ctx, deps, app)
		if err != nil {
			return nil, errors.Wrapf(err, "plan repair for app %s", app.ID)
		}
		if opts.Execute && plan.isApplyable() {
			if err = applyRepairTrpcTafFrameworkDefsAppPlan(ctx, deps, plan); err != nil {
				return nil, errors.Wrapf(err, "apply repair for app %s", app.ID)
			}
			plan.Applied = true
		}
		summary.Plans = append(summary.Plans, *plan)
	}
	summarizeRepairTrpcTafFrameworkDefs(summary)

	if opts.FailOnConflict && summary.ConflictApps > 0 {
		return summary, errors.Errorf("%d app(s) have conflicts", summary.ConflictApps)
	}
	return summary, nil
}

// listTrpcTafAppsForRepair 解析要处理的应用列表。
// 未指定 --app-id 时只扫 trpc/taf，Helm 不会进入后续 plan。
// 指定了 --app-id 时仍会装载该应用，由 build 阶段按 type 跳过非 trpc/taf。
func listTrpcTafAppsForRepair(
	ctx context.Context,
	appStore bkmsapp.ApplicationStore,
	opts repairTrpcTafFrameworkDefsOptions,
) ([]*bkmsapp.Application, error) {
	if opts.AppID != "" {
		app, err := appStore.GetApp(ctx, opts.AppID)
		if err != nil {
			return nil, errors.Wrap(err, "get app")
		}
		if opts.WorkspaceID != "" && app.WorkspaceID != opts.WorkspaceID {
			return nil, errors.Errorf("app %s does not belong to workspace %s", app.ID, opts.WorkspaceID)
		}
		return []*bkmsapp.Application{app}, nil
	}

	var apps []*bkmsapp.Application
	for _, appType := range []string{bkmsapp.AppTypeTRPC, bkmsapp.AppTypeTAF} {
		listed, err := appStore.ListApps(ctx, &bkmsapp.ListOpts{
			WorkspaceID: opts.WorkspaceID,
			AppType:     appType,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "list %s apps", appType)
		}
		apps = append(apps, listed...)
	}
	slices.SortFunc(apps, func(a, b *bkmsapp.Application) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return apps, nil
}

// buildRepairTrpcTafFrameworkDefsAppPlan 读取该应用的 framework def / file，交给纯函数算 plan。
func buildRepairTrpcTafFrameworkDefsAppPlan(
	ctx context.Context,
	deps repairTrpcTafFrameworkDefsDeps,
	app *bkmsapp.Application,
) (*repairTrpcTafFrameworkDefsAppPlan, error) {
	plan := &repairTrpcTafFrameworkDefsAppPlan{
		AppID:     app.ID,
		AppName:   app.Name,
		AppType:   app.Type,
		Moves:     []repairFrameworkDefMove{},
		Conflicts: []repairFrameworkDefConflict{},
		Leftovers: []repairFrameworkDefLeftover{},
	}
	if app.Type != bkmsapp.AppTypeTRPC && app.Type != bkmsapp.AppTypeTAF {
		plan.Skipped = true
		plan.SkipReason = skipReasonNotTrpcOrTaf
		return plan, nil
	}

	defs, err := deps.defStore.ListByApp(ctx, app.ID, appcfg.DefFilterConfigKind(appcfg.ConfigKindFramework))
	if err != nil {
		return nil, errors.Wrap(err, "list framework defs")
	}
	if len(defs) == 0 {
		plan.Skipped = true
		plan.SkipReason = skipReasonNoFrameworkDef
		return plan, nil
	}

	defIDs := loMapDefIDs(defs)
	files, err := deps.fileStore.List(ctx, app.ID, appcfg.AcfFilterDefIDs(defIDs))
	if err != nil {
		return nil, errors.Wrap(err, "list framework files")
	}

	allFiles, err := deps.fileStore.List(ctx, app.ID)
	if err != nil {
		return nil, errors.Wrap(err, "list app config files")
	}
	appendMissingDefIDLeftovers(plan, allFiles)

	return planRepairTrpcTafFrameworkDefs(plan, defs, files), nil
}

// applyRepairTrpcTafFrameworkDefsAppPlan 按已确认的 plan 写库。
// 调用方必须先保证 isApplyable()：无 conflict、有实际变更。
// 删除孤儿 def 前再查一次剩余 file，避免 leftover 默认文件被误删。
func applyRepairTrpcTafFrameworkDefsAppPlan(
	ctx context.Context,
	deps repairTrpcTafFrameworkDefsDeps,
	plan *repairTrpcTafFrameworkDefsAppPlan,
) error {
	primaryID, err := bson.ObjectIDFromHex(plan.PrimaryDefID)
	if err != nil {
		return errors.Wrap(err, "parse primary def id")
	}

	for _, move := range plan.Moves {
		fileID, err := bson.ObjectIDFromHex(move.FileID)
		if err != nil {
			return errors.Wrapf(err, "parse file id %s", move.FileID)
		}
		file, err := deps.fileStore.GetByID(ctx, fileID)
		if err != nil {
			return errors.Wrapf(err, "load file %s", move.FileID)
		}
		file.DefID = primaryID
		if _, err = deps.fileStore.Update(ctx, *file); err != nil {
			return errors.Wrapf(err, "repoint file %s to primary def", move.FileID)
		}
	}

	if plan.SetIndependent {
		def, err := deps.defStore.GetByID(ctx, primaryID)
		if err != nil {
			return errors.Wrap(err, "load primary def")
		}
		def.EnvConfigMode.IsUnifiedConfig = false
		if _, err = deps.defStore.Update(ctx, *def); err != nil {
			return errors.Wrap(err, "set primary def independent")
		}
	}

	for _, defIDHex := range plan.DeleteDefIDs {
		defID, err := bson.ObjectIDFromHex(defIDHex)
		if err != nil {
			return errors.Wrapf(err, "parse orphan def id %s", defIDHex)
		}
		remaining, err := deps.fileStore.ListByDefID(ctx, defID)
		if err != nil {
			return errors.Wrapf(err, "list files of orphan def %s", defIDHex)
		}
		if len(remaining) > 0 {
			continue
		}
		if _, err = deps.defStore.DeleteByID(ctx, defID); err != nil {
			return errors.Wrapf(err, "delete orphan def %s", defIDHex)
		}
	}
	return nil
}

// isApplyable 表示 --execute 时可以安全落库：未跳过、无 conflict，且确有 move / 切模式 / 删空 def。
func (p *repairTrpcTafFrameworkDefsAppPlan) isApplyable() bool {
	return !p.Skipped && len(p.Conflicts) == 0 && (len(p.Moves) > 0 || p.SetIndependent || len(p.DeleteDefIDs) > 0)
}

func summarizeRepairTrpcTafFrameworkDefs(summary *repairTrpcTafFrameworkDefsSummary) {
	summary.AppCount = len(summary.Plans)
	for _, plan := range summary.Plans {
		if plan.Skipped {
			summary.Skipped++
			continue
		}
		if len(plan.Conflicts) > 0 {
			summary.ConflictApps++
			continue
		}
		if plan.isApplyable() || plan.Applied {
			summary.Repairable++
		} else {
			summary.Skipped++
		}
		if plan.Applied {
			summary.Applied++
		}
	}
}

func writeRepairTrpcTafFrameworkDefsOutput(w io.Writer, summary *repairTrpcTafFrameworkDefsSummary) error {
	payload, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshal repair summary")
	}
	_, err = fmt.Fprintln(w, string(payload))
	if !summary.Execute {
		_, _ = fmt.Fprintln(w, "dry-run only. rerun with --execute to apply apps that have no conflicts.")
	}
	return err
}

func loMapDefIDs(defs []appcfg.AppConfigFileDef) []bson.ObjectID {
	ids := make([]bson.ObjectID, 0, len(defs))
	for _, def := range defs {
		ids = append(ids, def.ID)
	}
	return ids
}

// appendMissingDefIDLeftovers 把缺 defID 的 file 记入 leftovers，供人工补 000014 类回填，本命令不改挂。
func appendMissingDefIDLeftovers(plan *repairTrpcTafFrameworkDefsAppPlan, files []appcfg.AppConfigFile) {
	for _, file := range files {
		if file.DefID != bson.NilObjectID {
			continue
		}
		plan.Leftovers = append(plan.Leftovers, repairFrameworkDefLeftover{
			FileID:  file.ID.Hex(),
			EnvName: file.EnvName,
			Reason:  leftoverReasonMissingDefID,
		})
	}
}
