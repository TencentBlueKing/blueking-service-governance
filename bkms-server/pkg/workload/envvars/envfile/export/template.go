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

package export

import (
	"context"
	"strings"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// RenderScopedImportTemplate renders the sample dotenv template for scoped env var import.
func RenderScopedImportTemplate(ctx context.Context) string {
	return renderTemplate(
		[]string{
			i18n.LocalizeOr(
				ctx, "workload.envvars.exportTemplate.header.keyValueFormat", nil,
				"# - Each variable uses KEY=VALUE format.",
			),
			i18n.LocalizeOr(
				ctx, "workload.envvars.exportTemplate.header.optionalDesc", nil,
				"# - Optional # desc: applies to the next variable.",
			),
			i18n.LocalizeOr(
				ctx, "workload.envvars.exportTemplate.header.scopedRequired", nil,
				"# - Scoped import requires # scopeType: workspace or envType.",
			),
			i18n.LocalizeOr(
				ctx, "workload.envvars.exportTemplate.header.scopeValueRequired", nil,
				"# - # scopeValue is required when scopeType=envType.",
			),
			i18n.LocalizeOr(
				ctx, "workload.envvars.exportTemplate.header.scopeValueOptions", nil,
				"# - scopeValue for envType can be: development, test, staging, production.",
			),
		},
		[]renderRecord{
			{
				Key:   "BKMS_SHARED_NAMESPACE",
				Value: "bk-shared",
				Description: i18n.LocalizeOr(
					ctx, "workload.envvars.exportTemplate.desc.sharedAcrossEnvs", nil, "shared across all envs",
				),
				Scope: &envvartypes.ScopedEnvVarScope{
					ScopeType: envvartypes.ScopeTypeWorkspace,
				},
			},
			{
				Key:   "FEATURE_FLAG",
				Value: "true",
				Description: i18n.LocalizeOr(
					ctx, "workload.envvars.exportTemplate.desc.onlyForDevelopment", nil, "only for development",
				),
				Scope: &envvartypes.ScopedEnvVarScope{
					ScopeType:  envvartypes.ScopeTypeEnvType,
					ScopeValue: "development",
				},
			},
			{
				Key:   "JAVA_OPTS",
				Value: "-Xms1024m -Xmx2048m",
				Description: i18n.LocalizeOr(
					ctx, "workload.envvars.exportTemplate.desc.onlyForProduction", nil, "only for production",
				),
				Scope: &envvartypes.ScopedEnvVarScope{
					ScopeType:  envvartypes.ScopeTypeEnvType,
					ScopeValue: "production",
				},
			},
		},
	)
}

// RenderEnvAppImportTemplate renders the shared sample dotenv template for env/app import.
func RenderEnvAppImportTemplate(ctx context.Context) string {
	return renderTemplate(
		[]string{
			i18n.LocalizeOr(
				ctx, "workload.envvars.exportTemplate.header.keyValueFormat", nil,
				"# - Each variable uses KEY=VALUE format.",
			),
			i18n.LocalizeOr(
				ctx, "workload.envvars.exportTemplate.header.optionalDesc", nil,
				"# - Optional # desc: applies to the next variable.",
			),
			i18n.LocalizeOr(
				ctx, "workload.envvars.exportTemplate.header.noScopeInEnvApp", nil,
				"# - Do not add # scopeType or # scopeValue in env/app import.",
			),
		},
		[]renderRecord{
			{
				Key:   "APP_NAME",
				Value: "demo-service",
				Description: i18n.LocalizeOr(
					ctx, "workload.envvars.exportTemplate.desc.plainStringValue", nil, "plain string value",
				),
			},
			{
				Key:   "FEATURE_FLAG",
				Value: "true",
				Description: i18n.LocalizeOr(
					ctx, "workload.envvars.exportTemplate.desc.booleanStyleValue", nil, "boolean-style value",
				),
			},
			{
				Key:   "APP_PORT",
				Value: "8080",
				Description: i18n.LocalizeOr(
					ctx, "workload.envvars.exportTemplate.desc.numberStyleValue", nil, "number-style value",
				),
			},
			{
				Key:   "WELCOME_MESSAGE",
				Value: "hello bkms",
				Description: i18n.LocalizeOr(
					ctx, "workload.envvars.exportTemplate.desc.valueWithSpaces", nil, "value with spaces",
				),
			},
			{
				Key:   "EMPTY_VALUE",
				Value: "",
				Description: i18n.LocalizeOr(
					ctx, "workload.envvars.exportTemplate.desc.emptyStringValue", nil, "empty string value",
				),
			},
		},
	)
}

func renderTemplate(headerLines []string, records []renderRecord) string {
	var builder strings.Builder
	for _, line := range headerLines {
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	if len(records) > 0 {
		builder.WriteString("\n")
		builder.WriteString(renderRecords(records))
	}
	return builder.String()
}
