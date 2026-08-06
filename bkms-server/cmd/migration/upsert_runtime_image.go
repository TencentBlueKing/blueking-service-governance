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
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	workloadruntime "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/runtime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// descriptionFlag description flag 名称，用于判断用户是否显式传入该参数
const descriptionFlag = "description"

// NewUpsertRuntimeImageCmd 创建运行时镜像 upsert 命令
// 该命令仅用于平台管理功能补齐前的临时管理操作：
//   - 若指定 type + name 的镜像不存在，则新建记录后触发一次 snapshot 同步；
//   - 若已存在，则跳过创建（按需更新 description），并同样触发一次 snapshot 同步，实现幂等重跑
//
// 示例：
//
//	bkms-server upsert-runtime-image \
//		--srvCfg ./biz_cfg.yaml \
//		--type builder \
//		--name mirrors.example.com/build/go \
//		--description "Go builder image"
//
//	bkms-server upsert-runtime-image \
//		--srvCfg ./biz_cfg.yaml \
//		--type runner \
//		--name mirrors.example.com/runtime/node
//
// 执行完成后会按镜像仓库名刷新 tag 快照，并在标准输出打印新增、移除 tag 数量和同步状态
func NewUpsertRuntimeImageCmd() *cobra.Command {
	var srvCfg string
	var image workloadruntime.Image
	var imageType string

	cmd := &cobra.Command{
		Use:   "upsert-runtime-image",
		Short: "Upsert a runtime image (create or sync) and trigger snapshot synchronization",
		Run: func(cmd *cobra.Command, _ []string) {
			ctx := cmd.Context()
			cfg, err := config.Load(ctx, srvCfg)
			if err != nil {
				log.Fatalf("failed to load config: %s", err)
			}
			ctx = auth.WithMaintenanceUser(ctx)
			if err = log.InitDefaultLogger(cfg.Logging); err != nil {
				log.Fatalf("init logger: %v", err)
			}

			database.InitClient(ctx, cfg.Mongo)

			store, err := workloadruntime.NewStoreMongo(database.Client(), database.Name())
			if err != nil {
				log.Fatalf("init runtime image store failed: %v", err)
			}
			snapshotStore, err := snapshot.NewSnapshotStoreMongo(database.Client(), database.Name())
			if err != nil {
				log.Fatalf("init snapshot store failed: %v", err)
			}

			image.Type = workloadruntime.ImageType(imageType)

			// 判断是否已存在，决定走新增还是幂等分支
			_, err = store.GetByTypeAndName(ctx, image.Type, image.Name)
			switch {
			case err == nil:
				// 已存在：跳过创建，按需更新 description
				_, _ = fmt.Fprintf(
					cmd.OutOrStdout(),
					"runtime image already exists, skip creation and continue sync: type=%s name=%s\n",
					image.Type, image.Name,
				)
				// 仅当用户显式传入了非空 --description 时才更新描述
				if cmd.Flags().Changed(descriptionFlag) && image.Description != "" {
					if err = store.UpdateDescription(ctx, image.Type, image.Name, image.Description); err != nil {
						log.Fatalf("update runtime image description failed: %v", err)
					}
					_, _ = fmt.Fprintf(
						cmd.OutOrStdout(),
						"updated %s runtime image description: %s -> %q\n",
						image.Type, image.Name, image.Description,
					)
				}
			case errors.Is(err, workloadruntime.ErrRuntimeImageNotFound):
				// 不存在：走原有新增分支
				if err = store.Create(ctx, &image); err != nil {
					log.Fatalf("add runtime image failed: %v", err)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "added %s runtime image: %s\n", image.Type, image.Name)
			default:
				log.Fatalf("check runtime image existence failed: %v", err)
			}

			result, err := snapshot.NewService(snapshotStore, nil, nil).RefreshRepositorySnapshots(ctx, image.Name)
			if err != nil {
				log.Fatalf("sync runtime image tags failed: %v", err)
			}
			_, _ = fmt.Fprintf(
				cmd.OutOrStdout(),
				"synced runtime image tags: status=%s added=%d removed=%d message=%s\n",
				result.Status,
				result.AddedTagCnt,
				result.RemovedTagCnt,
				result.Message,
			)
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.Flags().StringVar(&imageType, "type", "", "image type: builder or runner")
	cmd.Flags().StringVar(&image.Name, "name", "", "image repository name without tag")
	cmd.Flags().StringVar(&image.Description, descriptionFlag, "", "description")
	_ = cmd.MarkFlagRequired("srvCfg")
	_ = cmd.MarkFlagRequired("type")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}
