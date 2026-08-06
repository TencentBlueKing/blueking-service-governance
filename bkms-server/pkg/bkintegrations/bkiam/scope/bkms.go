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

package scope

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/scope/template"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam/types"
)

// BKMSRoleScopesGenerator 是 bkms 角色权限范围生成器
type BKMSRoleScopesGenerator struct {
	WorkspaceID   string
	WorkspaceName string
	TplRoleCode   string
}

// Generate 生成权限范围
func (g BKMSRoleScopesGenerator) Generate() []types.AuthorizationScope {
	return generateFromTemplate(
		template.GetRoleScopeTemplatePath("bkms", g.TplRoleCode),
		map[string]any{
			"BKMSSystemID":  config.G.BkIAMSystemIDs.Bkms,
			"WorkspaceID":   g.WorkspaceID,
			"WorkspaceName": g.WorkspaceName,
		},
	)
}

var _ AuthScopesGenerator = (*BKMSRoleScopesGenerator)(nil)
