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

package client

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// ==========================================================================
// 类型定义
// ==========================================================================

// ---------- 环境变量导入导出类型 ----------

// EnvVarImportResult 环境变量导入结果汇总。
type EnvVarImportResult struct {
	// Total 导入条目总数。
	Total int `json:"total"`
	// New 新增条数。
	New int `json:"new"`
	// Overwrite 覆盖条数。
	Overwrite int `json:"overwrite"`
}

// ExportAppEnvVarsOptions 应用环境变量导出选项。
type ExportAppEnvVarsOptions struct {
	// Scope 导出范围：appDefined 或 effectiveByEnv。
	Scope string
	// EnvName 环境名称；Scope 为 effectiveByEnv 时必填。
	EnvName string
}

// ---------- 环境变量导入预览类型 ----------

// EnvVarImportPreviewScope 预览结果中的 scope 信息。
type EnvVarImportPreviewScope struct {
	// Type scope 类型（workspace / envType / env）
	Type string `json:"type" yaml:"type"`
	// Value scope 值；workspace 时省略
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
}

// EnvVarImportPreviewItem 单条导入环境变量预览结果。
type EnvVarImportPreviewItem struct {
	// Key 环境变量 Key
	Key string `json:"key" yaml:"key"`
	// Value 环境变量 Value（导入值）
	Value string `json:"value" yaml:"value"`
	// OriginalValue 被覆盖变量的原值，仅当 action 为 overwrite 时返回
	OriginalValue string `json:"originalValue,omitempty" yaml:"originalValue,omitempty"`
	// Description 描述信息
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// DeclaredScope 输入中显式声明的原始 scope 信息
	DeclaredScope *EnvVarImportPreviewScope `json:"declaredScope,omitempty" yaml:"declaredScope,omitempty"`
	// EffectiveScope 预览后实际生效的 scope 信息
	EffectiveScope *EnvVarImportPreviewScope `json:"effectiveScope,omitempty" yaml:"effectiveScope,omitempty"`
	// Action 导入动作：new（新增）/ overwrite（覆盖）
	Action string `json:"action" yaml:"action"`
	// EffectScope scope 生效状态：none / applied
	EffectScope string `json:"effectScope" yaml:"effectScope"`
	// Messages 额外提示信息
	Messages []string `json:"messages,omitempty" yaml:"messages,omitempty"`
}

// EnvVarImportPreviewSummary 导入预览汇总统计。
type EnvVarImportPreviewSummary struct {
	// Total 导入条目总数
	Total int `json:"total" yaml:"total"`
	// New 新增条数
	New int `json:"new" yaml:"new"`
	// Overwrite 覆盖条数
	Overwrite int `json:"overwrite" yaml:"overwrite"`
}

// EnvVarImportPreview 导入预览完整结果。
type EnvVarImportPreview struct {
	// Items 逐条预览结果
	Items []EnvVarImportPreviewItem `json:"items" yaml:"items"`
	// Summary 汇总统计
	Summary EnvVarImportPreviewSummary `json:"summary" yaml:"summary"`
}

// ---------- 环境变量 CRUD 类型 ----------

// ScopedEnvVar 公共作用域环境变量。
type ScopedEnvVar struct {
	ID          string `json:"id" yaml:"id"`
	WorkspaceID string `json:"workspaceID" yaml:"workspaceID" table:"-"`
	ScopeType   string `json:"scopeType" yaml:"scopeType"`
	ScopeValue  string `json:"scopeValue" yaml:"scopeValue"`
	Key         string `json:"key" yaml:"key"`
	Value       string `json:"value" yaml:"value"`
	Description string `json:"description" yaml:"description"`
	IsSensitive bool   `json:"isSensitive" yaml:"isSensitive"`
	CreatedAt   string `json:"createdAt" yaml:"createdAt" table:"-"`
	UpdatedAt   string `json:"updatedAt" yaml:"updatedAt" table:"-"`
}

// AppDefinedEnvVar 应用直接定义的环境变量。
type AppDefinedEnvVar struct {
	Key         string `json:"key" yaml:"key"`
	Value       string `json:"value" yaml:"value"`
	Description string `json:"description" yaml:"description"`
	IsSensitive bool   `json:"isSensitive" yaml:"isSensitive"`
	CreatedAt   string `json:"createdAt" yaml:"createdAt" table:"-"`
	UpdatedAt   string `json:"updatedAt" yaml:"updatedAt" table:"-"`
}

// AppEnvVar 应用在某环境下生效的环境变量。
type AppEnvVar struct {
	Key         string `json:"key" yaml:"key"`
	Value       string `json:"value" yaml:"value"`
	Description string `json:"description" yaml:"description"`
	IsBuiltin   bool   `json:"isBuiltin" yaml:"isBuiltin"`
	IsSensitive bool   `json:"isSensitive" yaml:"isSensitive"`
}

// EnvVarConflictedSource 环境变量冲突来源。
type EnvVarConflictedSource struct {
	Source      string `json:"source" yaml:"source"`
	SourceValue string `json:"sourceValue" yaml:"sourceValue"`
}

// EnvVarConflictedInfo 环境变量冲突信息。
type EnvVarConflictedInfo struct {
	ConflictedSources  []EnvVarConflictedSource `json:"conflictedSources" yaml:"conflictedSources"`
	OverrideConflicted bool                     `json:"overrideConflicted" yaml:"overrideConflicted"`
	ConflictedDetail   string                   `json:"conflictedDetail" yaml:"conflictedDetail"`
}

// ScopedEnvVarDetailed 带冲突信息的作用域环境变量详情。
type ScopedEnvVarDetailed struct {
	ScopedEnvVar   *ScopedEnvVar         `json:"scopedEnvVar" yaml:"scopedEnvVar"`
	ConflictedInfo *EnvVarConflictedInfo `json:"conflictedInfo,omitempty" yaml:"conflictedInfo,omitempty"`
}

// CreateScopedEnvVarOptions 创建公共作用域环境变量的选项。
type CreateScopedEnvVarOptions struct {
	ScopeType   string `json:"scopeType"`
	ScopeValue  string `json:"scopeValue"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	IsSensitive bool   `json:"isSensitive"`
}

// UpdateScopedEnvVarOptions 更新公共作用域环境变量的选项。
type UpdateScopedEnvVarOptions struct {
	Key         string  `json:"key"`
	Value       *string `json:"value,omitempty"`
	Description *string `json:"description,omitempty"`
	IsSensitive *bool   `json:"isSensitive,omitempty"`
}

// CreateAppDefinedEnvVarOptions 创建应用环境变量的选项。
type CreateAppDefinedEnvVarOptions struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	IsSensitive bool   `json:"isSensitive"`
}

// UpdateAppDefinedEnvVarOptions 更新应用环境变量的选项。
type UpdateAppDefinedEnvVarOptions struct {
	UpdatedKey  string  `json:"updatedKey"`
	Value       *string `json:"value,omitempty"`
	Description *string `json:"description,omitempty"`
	IsSensitive *bool   `json:"isSensitive,omitempty"`
}

// ---------- 内部响应结构 ----------

type envVarImportResp struct {
	Data *EnvVarImportResult `json:"data"`
}

type envVarPreviewResp struct {
	Data *EnvVarImportPreview `json:"data"`
}

type listPublicEnvVarsResp struct {
	Data []ScopedEnvVar `json:"data"`
}

type createScopedEnvVarResp struct {
	Data *ScopedEnvVar `json:"data"`
}

type updateScopedEnvVarResp struct {
	Data *ScopedEnvVar `json:"data"`
}

type listAppDefinedEnvVarsResp struct {
	Data []AppDefinedEnvVar `json:"data"`
}

type createAppDefinedEnvVarResp struct {
	Data *AppDefinedEnvVar `json:"data"`
}

type updateAppDefinedEnvVarResp struct {
	Data *AppDefinedEnvVar `json:"data"`
}

type listAppEnvVarsResp struct {
	Data []AppEnvVar `json:"data"`
}

type listEnvScopedEnvVarsResp struct {
	Data []ScopedEnvVarDetailed `json:"data"`
}

// ==========================================================================
// 方法实现
// ==========================================================================

// ---------- 环境变量导入导出方法 ----------

// ImportPublicEnvVars 导入公共环境变量。
func (c *SvcBasedClient) ImportPublicEnvVars(
	ctx context.Context,
	workspaceID, filePath string,
) (*EnvVarImportResult, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/scoped-env-vars/public-vars/import", workspaceID)
	return c.uploadEnvFile(ctx, url, filePath)
}

// ExportPublicEnvVars 导出公共环境变量。
func (c *SvcBasedClient) ExportPublicEnvVars(ctx context.Context, workspaceID string) (string, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/scoped-env-vars/public-vars/export", workspaceID)
	return c.downloadEnvFile(ctx, url, nil)
}

// ImportEnvScopedEnvVars 导入单环境环境变量。
func (c *SvcBasedClient) ImportEnvScopedEnvVars(
	ctx context.Context,
	envID, filePath string,
) (*EnvVarImportResult, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/scoped-env-vars/import/%s", envID)
	return c.uploadEnvFile(ctx, url, filePath)
}

// ExportEnvScopedEnvVars 导出单环境环境变量。
func (c *SvcBasedClient) ExportEnvScopedEnvVars(ctx context.Context, envID string) (string, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/scoped-env-vars/export/%s", envID)
	return c.downloadEnvFile(ctx, url, nil)
}

// ImportAppEnvVars 导入应用直接定义的环境变量。
func (c *SvcBasedClient) ImportAppEnvVars(ctx context.Context, appID, filePath string) (*EnvVarImportResult, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/env-vars/import", appID)
	return c.uploadEnvFile(ctx, url, filePath)
}

// ExportAppEnvVars 导出应用环境变量。
func (c *SvcBasedClient) ExportAppEnvVars(
	ctx context.Context,
	appID string,
	opts ExportAppEnvVarsOptions,
) (string, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/env-vars/export", appID)
	params := map[string]string{
		"scope": opts.Scope,
	}
	if opts.EnvName != "" {
		params["envName"] = opts.EnvName
	}
	return c.downloadEnvFile(ctx, url, params)
}

// ---------- 环境变量导入预览方法 ----------

// PreviewPublicEnvVars 预览公共环境变量导入。
func (c *SvcBasedClient) PreviewPublicEnvVars(
	ctx context.Context, workspaceID, filePath string,
) (*EnvVarImportPreview, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/scoped-env-vars/public-vars/preview", workspaceID)
	return c.uploadEnvFileForPreview(ctx, url, filePath)
}

// PreviewEnvScopedEnvVars 预览单环境环境变量导入。
func (c *SvcBasedClient) PreviewEnvScopedEnvVars(
	ctx context.Context, envID, filePath string,
) (*EnvVarImportPreview, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/scoped-env-vars/preview/%s", envID)
	return c.uploadEnvFileForPreview(ctx, url, filePath)
}

// PreviewAppEnvVars 预览应用环境变量导入。
func (c *SvcBasedClient) PreviewAppEnvVars(
	ctx context.Context, appID, filePath string,
) (*EnvVarImportPreview, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/env-vars/preview", appID)
	return c.uploadEnvFileForPreview(ctx, url, filePath)
}

// ---------- 环境变量 CRUD 方法 ----------

// ListPublicEnvVars 获取公共环境变量列表。
func (c *SvcBasedClient) ListPublicEnvVars(ctx context.Context, workspaceID string) ([]ScopedEnvVar, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/scoped-env-vars/public-vars", workspaceID)

	var respData listPublicEnvVarsResp
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "list public env vars")
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list public env vars failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}
	return respData.Data, nil
}

// CreateScopedEnvVar 创建公共作用域环境变量。
func (c *SvcBasedClient) CreateScopedEnvVar(
	ctx context.Context, workspaceID string, opts CreateScopedEnvVarOptions,
) (*ScopedEnvVar, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/scoped-env-vars", workspaceID)

	var respData createScopedEnvVarResp
	resp, err := c.cli.R().SetContext(ctx).SetBody(opts).SetResult(&respData).Post(url)
	if err != nil {
		return nil, errors.Wrap(err, "create scoped env var")
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("create scoped env var failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}
	return respData.Data, nil
}

// UpdateScopedEnvVar 更新公共作用域环境变量。
func (c *SvcBasedClient) UpdateScopedEnvVar(
	ctx context.Context, workspaceID, scopedEnvVarID string, opts UpdateScopedEnvVarOptions,
) (*ScopedEnvVar, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/scoped-env-vars/%s", workspaceID, scopedEnvVarID)

	var respData updateScopedEnvVarResp
	resp, err := c.cli.R().SetContext(ctx).SetBody(opts).SetResult(&respData).Put(url)
	if err != nil {
		return nil, errors.Wrap(err, "update scoped env var")
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("update scoped env var failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}
	return respData.Data, nil
}

// DeleteScopedEnvVar 删除公共作用域环境变量。
func (c *SvcBasedClient) DeleteScopedEnvVar(ctx context.Context, workspaceID, scopedEnvVarID string) error {
	url := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/scoped-env-vars/%s", workspaceID, scopedEnvVarID)

	resp, err := c.cli.R().SetContext(ctx).Delete(url)
	if err != nil {
		return errors.Wrap(err, "delete scoped env var")
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("delete scoped env var failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}
	return nil
}

// ListAppDefinedEnvVars 获取应用直接定义的环境变量列表。
func (c *SvcBasedClient) ListAppDefinedEnvVars(ctx context.Context, appID string) ([]AppDefinedEnvVar, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/env-vars", appID)

	var respData listAppDefinedEnvVarsResp
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "list app defined env vars")
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list app defined env vars failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}
	return respData.Data, nil
}

// CreateAppDefinedEnvVar 创建应用直接定义的环境变量。
func (c *SvcBasedClient) CreateAppDefinedEnvVar(
	ctx context.Context, appID string, opts CreateAppDefinedEnvVarOptions,
) (*AppDefinedEnvVar, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/env-vars", appID)

	var respData createAppDefinedEnvVarResp
	resp, err := c.cli.R().SetContext(ctx).SetBody(opts).SetResult(&respData).Post(url)
	if err != nil {
		return nil, errors.Wrap(err, "create app defined env var")
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("create app defined env var failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}
	return respData.Data, nil
}

// UpdateAppDefinedEnvVar 更新应用直接定义的环境变量。
func (c *SvcBasedClient) UpdateAppDefinedEnvVar(
	ctx context.Context, appID, key string, opts UpdateAppDefinedEnvVarOptions,
) (*AppDefinedEnvVar, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/env-vars/%s", appID, key)

	var respData updateAppDefinedEnvVarResp
	resp, err := c.cli.R().SetContext(ctx).SetBody(opts).SetResult(&respData).Put(url)
	if err != nil {
		return nil, errors.Wrap(err, "update app defined env var")
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("update app defined env var failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}
	return respData.Data, nil
}

// DeleteAppDefinedEnvVar 删除应用直接定义的环境变量。
func (c *SvcBasedClient) DeleteAppDefinedEnvVar(ctx context.Context, appID, key string) error {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/env-vars/%s", appID, key)

	resp, err := c.cli.R().SetContext(ctx).Delete(url)
	if err != nil {
		return errors.Wrap(err, "delete app defined env var")
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("delete app defined env var failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}
	return nil
}

// ListEnvScopedEnvVars 获取指定环境下的环境变量详情列表（含冲突信息）。
func (c *SvcBasedClient) ListEnvScopedEnvVars(ctx context.Context, envID string) ([]ScopedEnvVarDetailed, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/scoped-env-vars/detailed-list/%s", envID)

	var respData listEnvScopedEnvVarsResp
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "list env scoped env vars")
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list env scoped env vars failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}
	return respData.Data, nil
}

// ListAppEnvVars 获取应用在某环境下最终生效的全部环境变量。
func (c *SvcBasedClient) ListAppEnvVars(ctx context.Context, appID, envName string) ([]AppEnvVar, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/env-variables", appID, envName)

	var respData listAppEnvVarsResp
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "list app env vars")
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list app env vars failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}
	return respData.Data, nil
}

// ---------- 内部辅助方法 ----------

// uploadEnvFile 以 multipart/form-data 格式上传 .env 文件到指定 URL。
func (c *SvcBasedClient) uploadEnvFile(ctx context.Context, url, filePath string) (*EnvVarImportResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "open env file")
	}
	defer file.Close()

	var resp envVarImportResp
	r, err := c.cli.R().
		SetContext(ctx).
		SetFileReader("file", filepath.Base(filePath), file).
		SetResult(&resp).
		Post(url)
	if err != nil {
		return nil, errors.Wrap(err, "upload env file")
	}
	if r.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("import env vars failed: [%d] -> %s", r.StatusCode(), r.Body())
	}
	if resp.Data == nil {
		return nil, errors.New("unexpected empty response from import API")
	}
	return resp.Data, nil
}

// uploadEnvFileForPreview 以 multipart/form-data 格式上传 .env 文件到预览 URL。
func (c *SvcBasedClient) uploadEnvFileForPreview(
	ctx context.Context, url, filePath string,
) (*EnvVarImportPreview, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "open env file")
	}
	defer file.Close()

	var resp envVarPreviewResp
	r, err := c.cli.R().
		SetContext(ctx).
		SetFileReader("file", filepath.Base(filePath), file).
		SetResult(&resp).
		Post(url)
	if err != nil {
		return nil, errors.Wrap(err, "preview env file")
	}
	if r.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("preview env vars failed: [%d] -> %s", r.StatusCode(), r.Body())
	}
	if resp.Data == nil {
		return nil, errors.New("unexpected empty response from preview API")
	}
	return resp.Data, nil
}

// downloadEnvFile 从指定 URL 下载 .env 文件内容。
func (c *SvcBasedClient) downloadEnvFile(ctx context.Context, url string, params map[string]string) (string, error) {
	req := c.cli.R().SetContext(ctx)
	if params != nil {
		req.SetQueryParams(params)
	}
	r, err := req.Get(url)
	if err != nil {
		return "", errors.Wrap(err, "download env file")
	}
	if r.StatusCode() != http.StatusOK {
		return "", errors.Errorf("export env vars failed: [%d] -> %s", r.StatusCode(), r.Body())
	}
	return r.String(), nil
}
