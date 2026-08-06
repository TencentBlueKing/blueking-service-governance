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

// 实现检查
var _ AuthScopesGenerator = (*BSCPRoleScopesGenerator)(nil)

// BSCPService 表示一个 BSCP 服务（包含 ID 和 Name）
type BSCPService struct {
	ID   string
	Name string
}

// BSCPRoleScopesGenerator 是 bk-bscp 角色权限范围生成器
type BSCPRoleScopesGenerator struct {
	BizID       string
	BizName     string
	TplRoleCode string
	Services    []BSCPService
}

// Generate 生成权限范围
func (g BSCPRoleScopesGenerator) Generate() []types.AuthorizationScope {
	return generateFromTemplate(
		template.GetRoleScopeTemplatePath("bscp", g.TplRoleCode),
		map[string]any{
			"BKBSCPSystemID": config.G.BkIAMSystemIDs.BSCP,
			"BKCCSystemID":   config.G.BkIAMSystemIDs.BkCC,
			"BizID":          g.BizID,
			"BizName":        g.BizName,
			"Services":       g.Services,
		},
	)
}
