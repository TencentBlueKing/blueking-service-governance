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

	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	workspaceadmin "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/workspace/admin"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// NewCleanupExpiredWorkspaceTempAdminsCmd 创建一个 Cobra 命令，用于单次回收过期的工作空间临时管理员权限。
//
// 该命令设计为被 Kubernetes CronJob 周期性调用，扫描未回收且已过期的临时管理员授权，
// 必要时同步回收对应的 workspace admin 权限并更新本地记录。
// 命令执行流程：
//  1. 读取业务配置并初始化 MongoDB 连接
//  2. 初始化存储层（WorkspaceStore、TempAdminRecordStore）
//  3. 构造 workspace admin Service 并执行一次 CleanupExpiredGrants() 清理
//  4. 处理完成后退出，退出码反映处理结果（0 成功，非 0 失败）
//
// 必填参数：
//
//	--srvCfg：业务配置文件路径，包含 MongoDB 等连接信息
func NewCleanupExpiredWorkspaceTempAdminsCmd() *cobra.Command {
	var srvCfg string

	cmd := &cobra.Command{
		Use:   "cleanup_expired_workspace_temp_admins",
		Short: "单次回收过期的工作空间临时管理员权限",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			cfg, err := config.Load(ctx, srvCfg)
			if err != nil {
				log.Fatalf("failed to load config: %s", err)
			}
			if err = log.InitDefaultLogger(cfg.Logging); err != nil {
				log.Fatalf("init logger: %v", err)
			}

			database.InitClient(ctx, cfg.Mongo)
			storereg.Init(ctx)

			service := workspaceadmin.NewService(
				storereg.G().WorkspaceStore,
				storereg.G().TempAdminRecordStore,
				perm.NewManager(),
			)
			if err = service.CleanupExpiredGrants(context.Background()); err != nil {
				log.Fatalf("cleanup expired workspace temp admins failed: %v", err)
			}
			log.Info(ctx, "cleanup expired workspace temp admins command completed")
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	_ = cmd.MarkFlagRequired("srvCfg")

	return cmd
}
