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

package overview

import (
	"context"
	"log/slog"

	"github.com/samber/lo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
)

// clusterNameByID clusterID -> BCS 集群展示名。
type clusterNameByID map[string]string

// queryClusterNames 按环境上的 projectCode 去重后拉取 BCS 集群列表，得到 clusterID -> name。
//
// 任一项目/接口失败只跳过该项目，不阻断总览；无登录用户或无法建客户端时返回空 map。
func queryClusterNames(ctx context.Context, rows []EnvRow) clusterNameByID {
	out := make(clusterNameByID)
	projectCodes := lo.Uniq(lo.FilterMap(rows, func(row EnvRow, _ int) (string, bool) {
		return row.Cluster.ProjectCode, row.Cluster.ProjectCode != "" && row.Cluster.ClusterID != ""
	}))
	if len(projectCodes) == 0 {
		return out
	}

	user, err := auth.GetUser(ctx)
	if err != nil {
		log.WarnAttrs(ctx, "deploy overview skips cluster names, no authenticated user",
			slog.String("error", err.Error()),
		)
		return out
	}
	client, err := bcs.New(user)
	if err != nil {
		log.WarnAttrs(ctx, "deploy overview skips cluster names, init bcs client failed",
			slog.String("error", err.Error()),
		)
		return out
	}

	for _, projectCode := range projectCodes {
		project, err := client.GetProject(ctx, projectCode)
		if err != nil {
			log.WarnAttrs(ctx, "deploy overview skips cluster names for project, get project failed",
				slog.String("project_code", projectCode),
				slog.String("error", err.Error()),
			)
			continue
		}
		clusters, err := client.ListClustersByProject(ctx, project.ID)
		if err != nil {
			log.WarnAttrs(ctx, "deploy overview skips cluster names for project, list clusters failed",
				slog.String("project_code", projectCode),
				slog.String("project_id", project.ID),
				slog.String("error", err.Error()),
			)
			continue
		}
		for _, cluster := range clusters {
			if cluster.ID == "" || cluster.Name == "" {
				continue
			}
			out[cluster.ID] = cluster.Name
		}
	}
	return out
}
