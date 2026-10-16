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
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/scope"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// NewRefreshWorkspaceBkmonitorPermsCmd 批量刷新所有 Ready 状态 workspace 的 bkmonitor
// 权限范围（含 MCP actions）到 IAM grade manager 及用户组。
// 支持 --dry-run 预览待变更内容。
func NewRefreshWorkspaceBkmonitorPermsCmd() *cobra.Command {
	var srvCfg string
	var dryRun bool
	var workspaceIDs []string

	cmd := &cobra.Command{
		Use:   "refresh_workspace_bkmonitor_perms",
		Short: "Refresh bkmonitor permission scopes (including MCP actions) for all workspaces to IAM user groups",
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

			wsStore := store.Of(storereg.G().DB, workspace.Meta)

			if dryRun {
				dryRunRefreshWorkspaceBkmonitorPerms(ctx, wsStore, workspaceIDs)
				return nil
			}

			log.Warn(ctx, "WARNING: This will update grade manager scopes AND grant policies to IAM user groups "+
				"for all Ready workspaces. Only workspaces with complete platform data will be processed.")

			permMgr := perm.NewManager()
			return refreshWorkspaceBkmonitorPerms(ctx, wsStore, permMgr, workspaceIDs)
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.Flags().
		BoolVar(&dryRun, "dry-run", false, "list pending changes without actually executing IAM permission refresh")
	cmd.Flags().StringSliceVar(&workspaceIDs, "workspace-id", nil,
		"only refresh specified workspace IDs (comma-separated), useful for retrying failed ones")
	_ = cmd.MarkFlagRequired("srvCfg")

	return cmd
}

// dryRunRefreshWorkspaceBkmonitorPerms 以 dry-run 模式列出所有待变更的 workspace 及其详细信息。
func dryRunRefreshWorkspaceBkmonitorPerms(
	ctx context.Context,
	wsStore *workspace.WorkspaceStore,
	filterIDs []string,
) {
	workspaces, err := wsStore.List(ctx, workspace.InState(workspace.StateReady))
	if err != nil {
		log.Fatalf("list workspaces: %v", err)
	}

	if len(filterIDs) > 0 {
		filterSet := make(map[string]struct{}, len(filterIDs))
		for _, id := range filterIDs {
			filterSet[id] = struct{}{}
		}
		filtered := workspaces[:0]
		for _, ws := range workspaces {
			if _, ok := filterSet[ws.ID]; ok {
				filtered = append(filtered, ws)
			}
		}
		workspaces = filtered
	}

	fmt.Println("========================================")
	fmt.Println("[DRY-RUN] BKMonitor Permission Refresh Preview")
	fmt.Printf("========================================\n\n")
	fmt.Printf("Total workspaces to process: %d\n\n", len(workspaces))

	var willProcess, willSkip, willError int

	for i, ws := range workspaces {
		fmt.Printf("--- [%d/%d] Workspace: %s (%s) ---\n", i+1, len(workspaces), ws.ID, ws.DisplayName)

		if ws.BkSystems.BkMonitorProjectID == "" {
			fmt.Printf("  Status: SKIP (BkMonitorProjectID is empty)\n\n")
			willSkip++
			continue
		}

		data := buildWorkspaceData(ws)

		if checkErr := hasAllPlatformData(data); checkErr != nil {
			fmt.Printf("  Status: ERROR (platform data incomplete: %v)\n\n", checkErr)
			willError++
			continue
		}

		willProcess++

		fmt.Printf("  Status: WILL_REFRESH\n")
		fmt.Printf("  BKMonitor: SpaceID=%s\n", data.BKMonitor.SpaceID)
		if data.BKLog != nil {
			fmt.Printf("  BKLog:     SpaceID=%s\n", data.BKLog.SpaceID)
		}
		if data.BKCI != nil {
			fmt.Printf("  BKCI:     ProjectID=%s\n", data.BKCI.ProjectID)
		}
		if data.BCS != nil {
			fmt.Printf("  BCS:      ProjectID=%s\n", data.BCS.ProjectID)
		}
		if data.BKRepo != nil {
			fmt.Printf("  BKRepo:   ProjectID=%s\n", data.BKRepo.ProjectID)
		}

		fmt.Println("  BKMonitor actions to grant (by role):")
		for _, roleCode := range append(
			[]string{role.BuiltinRoleCode.Admin},
			role.WorkspaceScopeBuiltinRoles...,
		) {
			g := scope.BKMonitorRoleScopesGenerator{
				SpaceID:     data.BKMonitor.SpaceID,
				SpaceName:   data.BKMonitor.SpaceName,
				TplRoleCode: roleCode,
			}
			scopes := g.Generate()
			fmt.Printf("    [%s] %d scope blocks, actions: ", roleCode, len(scopes))
			for si, s := range scopes {
				if si > 0 {
					fmt.Print(" | ")
				}
				for ai, a := range s.Actions {
					if ai > 0 {
						fmt.Print(", ")
					}
					fmt.Print(a.ID)
				}
			}
			fmt.Println()
		}
		fmt.Println()
	}

	fmt.Println("========================================")
	fmt.Printf("[DRY-RUN] Summary: total=%d, will_process=%d, will_skip=%d, will_error=%d\n",
		len(workspaces), willProcess, willSkip, willError)
	fmt.Println("========================================")

	if willProcess > 0 {
		for _, ws := range workspaces {
			if ws.BkSystems.BkMonitorProjectID == "" {
				continue
			}
			data := buildWorkspaceData(ws)
			jsonBytes, _ := json.MarshalIndent(data, "", "  ")
			fmt.Printf("\n[DRY-RUN] Sample WorkspaceData (workspace=%s):\n%s\n", ws.ID, string(jsonBytes))
			break
		}
	}
}

// refreshWorkspaceBkmonitorPerms 遍历所有 Ready 状态的 workspace，逐个刷新 bkmonitor 权限。
func refreshWorkspaceBkmonitorPerms(
	ctx context.Context,
	wsStore *workspace.WorkspaceStore,
	permMgr perm.Manager,
	filterIDs []string,
) error {
	workspaces, err := wsStore.List(ctx, workspace.InState(workspace.StateReady))
	if err != nil {
		return errors.Wrap(err, "list workspaces")
	}

	// 如果指定了 --workspace-id，则只处理指定的 workspace
	if len(filterIDs) > 0 {
		filterSet := make(map[string]struct{}, len(filterIDs))
		for _, id := range filterIDs {
			filterSet[id] = struct{}{}
		}
		filtered := workspaces[:0]
		for _, ws := range workspaces {
			if _, ok := filterSet[ws.ID]; ok {
				filtered = append(filtered, ws)
			}
		}
		workspaces = filtered
	}

	var total, success, skipped, failed int
	var failedIDs []string
	total = len(workspaces)

	for _, ws := range workspaces {
		if ws.BkSystems.BkMonitorProjectID == "" {
			log.Warnf(ctx, "workspace %s has no BkMonitorProjectID, skipping", ws.ID)
			skipped++
			continue
		}

		data := buildWorkspaceData(ws)

		// 校验所有平台信息是否齐全。UpdateWorkspaceAdmin 调用的 UpdateGradeManager
		// 是 PUT 覆盖语义，如果平台信息不全会导致分级管理员授权范围被意外缩减。
		if err = hasAllPlatformData(data); err != nil {
			log.Errorf(ctx, "workspace %s platform data incomplete: %v, skipping", ws.ID, err)
			skipped++
			continue
		}

		// 1. 先更新分级管理员的授权范围（PUT 覆盖语义），确保所有系统都在授权范围内
		if err = permMgr.UpdateWorkspaceAdmin(ctx, data); err != nil {
			log.Errorf(ctx, "update workspace admin for %s failed: %v", ws.ID, err)
			failed++
			failedIDs = append(failedIDs, ws.ID)
			continue
		}

		// 2. 再刷新用户组 policies（POST 追加语义）
		if err = permMgr.UpdateWorkspaceScopeBuiltinRoles(ctx, data); err != nil {
			log.Errorf(ctx, "update workspace builtin roles for %s failed: %v", ws.ID, err)
			failed++
			failedIDs = append(failedIDs, ws.ID)
			continue
		}

		log.Infof(ctx, "workspace %s bkmonitor perms refreshed successfully", ws.ID)
		success++
	}

	log.Infof(
		ctx,
		"refresh_workspace_bkmonitor_perms completed: total=%d, success=%d, skipped=%d, failed=%d",
		total, success, skipped, failed,
	)

	if failed > 0 {
		log.Errorf(ctx, "failed workspace IDs: %s", strings.Join(failedIDs, ", "))
		log.Infof(
			ctx,
			"to retry failed workspaces, run: refresh_workspace_bkmonitor_perms --srvCfg=<path> --workspace-id=%s",
			strings.Join(failedIDs, ","),
		)
		return errors.Errorf(
			"refresh_workspace_bkmonitor_perms partially failed: %d/%d workspaces failed",
			failed, total,
		)
	}
	return nil
}

// hasAllPlatformData 校验 WorkspaceData 中所有平台信息是否齐全。
func hasAllPlatformData(data bkiam.WorkspaceData) error {
	var missing []string
	if data.BKMonitor == nil || data.BKMonitor.SpaceID == "" {
		missing = append(missing, "BKMonitor")
	}
	if data.BKLog == nil || data.BKLog.SpaceID == "" {
		missing = append(missing, "BKLog")
	}
	if data.BKCI == nil || data.BKCI.ProjectID == "" {
		missing = append(missing, "BKCI")
	}
	if data.BCS == nil || data.BCS.ProjectID == "" {
		missing = append(missing, "BCS")
	}
	if data.BKRepo == nil || data.BKRepo.ProjectID == "" {
		missing = append(missing, "BKRepo")
	}
	if len(missing) > 0 {
		return errors.Errorf("missing: %s", strings.Join(missing, ", "))
	}
	return nil
}

// buildWorkspaceData 根据 workspace 信息构造 WorkspaceData，填充所有平台字段。
func buildWorkspaceData(ws workspace.Workspace) bkiam.WorkspaceData {
	data := bkiam.WorkspaceData{
		WorkspaceID:   ws.ID,
		WorkspaceName: ws.DisplayName,
		BKMonitor: &bkiam.BKMonitorOptions{
			SpaceID:   ws.BkSystems.BkMonitorProjectID,
			SpaceName: ws.DisplayName,
		},
	}

	// 填充 BKLog（与 BKMonitor 共用同一个 project ID）
	if ws.BkSystems.BkLogProjectID != "" {
		data.BKLog = &bkiam.BKLogOptions{
			SpaceID:   ws.BkSystems.BkLogProjectID,
			SpaceName: ws.DisplayName,
		}
	}

	// 填充 BKCI
	if ws.BkSystems.BkCIProjectID != "" {
		data.BKCI = &bkiam.BKCIOptions{
			ProjectID:   ws.BkSystems.BkCIProjectID,
			ProjectName: ws.BkSystems.BkCIProjectID,
		}
	}

	// 填充 BCS
	if ws.BkSystems.BkBCSProjectID != "" {
		data.BCS = &bkiam.BCSOptions{
			ProjectID: ws.BkSystems.BkBCSProjectID,
			// 历史兼容 quirk：BCS.ProjectName 取 bkCIProjectID 而非 bcsProjectID
			ProjectName: ws.BkSystems.BkCIProjectID,
		}
	}

	// 填充 BKRepo
	if ws.BkSystems.BkRepoProjectID != "" {
		data.BKRepo = &bkiam.BKRepoOptions{
			ProjectID:   ws.BkSystems.BkRepoProjectID,
			ProjectName: ws.BkSystems.BkRepoProjectID,
		}
	}

	return data
}
