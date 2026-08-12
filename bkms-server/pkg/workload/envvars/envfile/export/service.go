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
	"strconv"
	"strings"

	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// Service renders env vars into import-compatible dotenv text.
type Service struct {
	scopedEnvVarStore envvars.ScopedEnvVarStore
	appModelStore     appmodel.AppModelStore
	reader            *envvars.UnifiedEnvVarsReader
}

// NewService creates an export service.
func NewService(
	scopedEnvVarStore envvars.ScopedEnvVarStore,
	appModelStore appmodel.AppModelStore,
	reader *envvars.UnifiedEnvVarsReader,
) *Service {
	return &Service{
		scopedEnvVarStore: scopedEnvVarStore,
		appModelStore:     appModelStore,
		reader:            reader,
	}
}

// ExportPublic exports workspace/envType scoped vars.
func (s *Service) ExportPublic(ctx context.Context, workspaceID string) (string, error) {
	vars, err := s.scopedEnvVarStore.ListPublic(ctx, workspaceID)
	if err != nil {
		return "", err
	}
	vars = filterOutSensitiveScopedVars(vars)

	// 保留显式 scope 元数据，确保导出的文件再次导入时，不会丢失记录原本
	// 属于 workspace 还是 envType 作用域的信息。
	records := make([]renderRecord, 0, len(vars))
	for _, item := range vars {
		records = append(records, renderRecord{
			Key:         item.Key,
			Value:       item.Value,
			Description: item.Description,
			Scope: &envvartypes.ScopedEnvVarScope{
				ScopeType:  item.ScopeType,
				ScopeValue: item.ScopeValue,
			},
		})
	}
	return renderRecords(records), nil
}

// ExportEnv exports vars directly defined in the target env scope.
// 目标环境由页面上下文提供，因此导出的 env file 不携带 scope 元数据。
func (s *Service) ExportEnv(ctx context.Context, environment envmodel.Environment) (string, error) {
	vars, err := s.scopedEnvVarStore.List(
		ctx,
		environment.WorkspaceID,
		envvars.WithScopes(envvartypes.ScopeEnv(environment.Name)),
	)
	if err != nil {
		return "", err
	}
	vars = filterOutSensitiveScopedVars(vars)

	records := make([]renderRecord, 0, len(vars))
	for _, item := range vars {
		records = append(records, renderRecord{
			Key:         item.Key,
			Value:       item.Value,
			Description: item.Description,
		})
	}
	return renderRecords(records), nil
}

// ExportAppDefined exports app-defined env vars only.
func (s *Service) ExportAppDefined(ctx context.Context, appID string) (string, error) {
	vars, err := s.appModelStore.ListAppDefinedEnvVars(ctx, appID)
	if err != nil {
		if errors.Is(err, appmodel.ErrAppModelNotFound) {
			return "", nil
		}
		return "", err
	}
	vars = filterOutSensitiveAppVars(vars)

	records := make([]renderRecord, 0, len(vars))
	for _, item := range vars {
		records = append(records, renderRecord{
			Key:         item.Key,
			Value:       item.Value,
			Description: item.Description,
		})
	}
	return renderRecords(records), nil
}

// ExportEffectiveAppEnv exports the effective env vars for an app in an environment.
func (s *Service) ExportEffectiveAppEnv(
	ctx context.Context,
	app *bkmsapp.Application,
	environment *envmodel.Environment,
) (string, error) {
	am, err := s.appModelStore.GetAppModel(ctx, app.ID)
	if err != nil && !errors.Is(err, appmodel.ErrAppModelNotFound) {
		return "", err
	}
	if errors.Is(err, appmodel.ErrAppModelNotFound) {
		am = nil
	}

	vars, err := envvars.BuildAppEnvVars(ctx, app, am, environment, s.reader)
	if err != nil {
		return "", err
	}
	// BuildAppEnvVars 在拼装最终生效视图时，可能把同一个 key 从多个来源带出来；
	// 这里保留最后一个，代表已经过覆盖决议后的最终结果。
	vars = vars.ToDeduplicatedList()
	vars = filterOutSensitiveEffectiveVars(vars)

	records := make([]renderRecord, 0, len(vars))
	for _, item := range vars {
		records = append(records, renderRecord{
			Key:         item.Key,
			Value:       item.Value,
			Description: item.Description,
		})
	}
	return renderRecords(records), nil
}

type renderRecord struct {
	Key         string
	Value       string
	Description string
	Scope       *envvartypes.ScopedEnvVarScope
}

func renderRecords(records []renderRecord) string {
	if len(records) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, item := range records {
		if i > 0 {
			builder.WriteString("\n")
		}
		// 在 KEY=VALUE 之前输出元数据注释行，既方便人阅读，也能保证再次导入时
		// 仍然可被识别。
		if item.Description != "" {
			builder.WriteString("# desc: ")
			builder.WriteString(renderMetadataValue(item.Description))
			builder.WriteString("\n")
		}
		if item.Scope != nil {
			builder.WriteString("# scopeType: ")
			builder.WriteString(string(item.Scope.ScopeType))
			builder.WriteString("\n")
			if item.Scope.ScopeType != envvartypes.ScopeTypeWorkspace {
				// workspace scope 约定不写 scopeValue；其它 scope 需要把值写出来，
				// 否则导回时无法还原目标作用域。
				builder.WriteString("# scopeValue: ")
				builder.WriteString(item.Scope.ScopeValue)
				builder.WriteString("\n")
			}
		}
		builder.WriteString(item.Key)
		builder.WriteString("=")
		builder.WriteString(renderValue(item.Value))
	}
	builder.WriteString("\n")
	return builder.String()
}

func renderValue(value string) string {
	if value == "" {
		return `""`
	}
	// 遇到空白、注释起始符或引号时，必须显式加双引号，避免后续导入时被 parser
	// 误判成截断、注释或不同的字面量边界。
	if strings.ContainsAny(value, " \t\n\r#\"'") {
		return strconv.Quote(value)
	}
	return value
}

func renderMetadataValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != value || strings.ContainsAny(value, "\n\r#\"'") {
		return strconv.Quote(value)
	}
	return value
}

func filterOutSensitiveScopedVars(vars []envvars.ScopedEnvVar) []envvars.ScopedEnvVar {
	return filterOutSensitive(vars, func(item envvars.ScopedEnvVar) bool {
		return item.IsSensitive
	})
}

func filterOutSensitiveAppVars(vars []appmodel.Variable) []appmodel.Variable {
	return filterOutSensitive(vars, func(item appmodel.Variable) bool {
		return item.IsSensitive
	})
}

func filterOutSensitiveEffectiveVars(vars envvartypes.EnvVariableList) envvartypes.EnvVariableList {
	return filterOutSensitive(vars, func(item envvartypes.EnvVariableObj) bool {
		return item.IsSensitive
	})
}

func filterOutSensitive[T any](vars []T, isSensitive func(T) bool) []T {
	result := make([]T, 0, len(vars))
	for _, item := range vars {
		if isSensitive(item) {
			continue
		}
		result = append(result, item)
	}
	return result
}
