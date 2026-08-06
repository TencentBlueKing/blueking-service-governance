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

package workspace

import (
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

// BuildDefaultEnvs 构建工作空间默认环境
// 参考产品定义:
// | 环境名称   | 环境 ID    | 环境分类 | BKMS_NAMESPACE | BKMS_ENV | BKMS_ENV_NAME |
// |------------|------------|----------|----------------|----------|---------------|
// | 正式环境   | production | 生产     | production     | prod     | production    |
// | 预发布环境 | staging    | 预发布   | staging        | stag     | staging       |
// | 测试环境   | test       | 测试     | development    | test     | test          |
func BuildDefaultEnvs(creator, workspaceID, bcsProjectCode string) []envmodel.Environment {
	// Note: 默认环境没有集群信息
	return []envmodel.Environment{
		{
			Name:        bkmsenv.TypeTest.String(),
			DisplayName: "测试环境",
			Type:        bkmsenv.TypeTest.String(),
			WorkspaceID: workspaceID,
			Cluster: envmodel.BizCluster{
				ProjectCode: bcsProjectCode,
			},
			Creator: creator,
		},
		{
			Name:        bkmsenv.TypeStaging.String(),
			DisplayName: "预发布环境",
			Type:        bkmsenv.TypeStaging.String(),
			WorkspaceID: workspaceID,
			Cluster: envmodel.BizCluster{
				ProjectCode: bcsProjectCode,
			},
			Creator: creator,
		},
		{
			Name:        bkmsenv.TypeProduction.String(),
			DisplayName: "正式环境",
			Type:        bkmsenv.TypeProduction.String(),
			WorkspaceID: workspaceID,
			Cluster: envmodel.BizCluster{
				ProjectCode: bcsProjectCode,
			},
			Creator: creator,
		},
	}
}
