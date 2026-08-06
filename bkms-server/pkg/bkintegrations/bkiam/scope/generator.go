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

// Package scope provides authorization scope generators for various business
// systems (BCS / BKCI / BKLog / BKMonitor / BKMS / BKRepo / BSCP).
//
// Each generator implements the AuthScopesGenerator interface and produces
// a list of types.AuthorizationScope based on the role code and the
// configured IAM system IDs from common/config.
package scope

import (
	"bytes"
	"encoding/json"
	tpl "text/template"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/scope/template"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam/types"
)

// AuthScopesGenerator 权限范围生成器
type AuthScopesGenerator interface {
	// Generate 生成权限范围
	Generate() []types.AuthorizationScope
}

// GenerateAuthScopes 聚合多个 generator 的输出
func GenerateAuthScopes(generators ...AuthScopesGenerator) []types.AuthorizationScope {
	authScopes := make([]types.AuthorizationScope, 0)
	for _, g := range generators {
		authScopes = append(authScopes, g.Generate()...)
	}
	return authScopes
}

// generateFromTemplate 根据模板路径与上下文数据，渲染 JSON 模板并解析为 AuthorizationScope 列表。
// 任一阶段（读模板/解析/渲染/反序列化）失败均直接 panic。
func generateFromTemplate(templatePath string, ctxData map[string]any) []types.AuthorizationScope {
	data, err := template.AuthScopesFS.ReadFile(templatePath)
	if err != nil {
		panic(err)
	}

	tmpl, err := tpl.New(templatePath).Parse(string(data))
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, ctxData); err != nil {
		panic(err)
	}

	var scopes []types.AuthorizationScope
	if err = json.Unmarshal(buf.Bytes(), &scopes); err != nil {
		panic(err)
	}
	return scopes
}
