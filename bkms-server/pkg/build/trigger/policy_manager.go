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

package trigger

import (
	"context"
	"strings"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
)

const (
	policyIDRandLen = 16
	// lockedBuildConfigHint 存在策略时拦截构建配置修改的提示
	lockedBuildConfigHint = "存在自动触发策略时不可修改，请先删除全部策略"
)

var (
	// ErrTooManyPolicies 单应用策略数量已达上限
	ErrTooManyPolicies = errors.New("单个应用最多配置 5 条触发策略")
	// ErrAutoGenerateTagDisabled 应用未开启自动生成 tag
	ErrAutoGenerateTagDisabled = errors.New("未开启自动生成镜像 tag，无法保存触发策略")
	// ErrUnsupportedAppType 应用构建来源或代码库类型不支持触发策略
	ErrUnsupportedAppType = errors.New("仅源码仓库且代码库类型为工蜂的应用可配置触发策略")
	// ErrBuildConfigLocked 存在策略时禁止修改被锁定的构建配置字段
	ErrBuildConfigLocked = errors.New(lockedBuildConfigHint)
	// ErrInvalidBranchMatch 分支匹配值不满足业务规则
	ErrInvalidBranchMatch = errors.New("分支匹配值不合法")
)

// sentinelMessageError 对外只暴露 msg，Unwrap 仍指向哨兵以便 errors.Is
type sentinelMessageError struct {
	sentinel error
	msg      string
}

func (e *sentinelMessageError) Error() string { return e.msg }
func (e *sentinelMessageError) Unwrap() error { return e.sentinel }

// withSentinelMessage 对外返回 msg，Unwrap 仍指向 sentinel，便于 errors.Is 同时给前端可读原因
func withSentinelMessage(sentinel error, msg string) error {
	return &sentinelMessageError{sentinel: sentinel, msg: msg}
}

// PolicyForm 策略表单，供创建 / 更新 / 冲突预检共用
type PolicyForm struct {
	Name             string
	Event            Event
	BranchMatchMode  BranchMatchMode
	BranchMatchValue string
	PathFilter       string
}

// PipelineOps 隔离触发专用流水线 Ensure / Cleanup，便于单测注入
type PipelineOps interface {
	Ensure(ctx context.Context, workspaceID, appID string) (*bkci.Pipeline, error)
	Cleanup(ctx context.Context, workspaceID, appID string) error
}

type defaultPipelineOps struct{}

func (defaultPipelineOps) Ensure(
	ctx context.Context,
	workspaceID, appID string,
) (*bkci.Pipeline, error) {
	return bkci.NewTriggerPipelineManager(workspaceID).Ensure(ctx, appID)
}

func (defaultPipelineOps) Cleanup(ctx context.Context, workspaceID, appID string) error {
	return bkci.NewTriggerPipelineManager(workspaceID).Cleanup(ctx, appID)
}

// PolicyManager 触发策略后端管理：CRUD、冲突检测、流水线生命周期调用
type PolicyManager struct {
	policies    PolicyStore
	buildConfig imagebuild.ConfigStore
	pipelines   PipelineOps
}

// NewPolicyManager 创建策略管理器，pipelineOps 为空时使用默认蓝盾实现
func NewPolicyManager(
	policies PolicyStore,
	buildConfig imagebuild.ConfigStore,
	pipelineOps PipelineOps,
) *PolicyManager {
	if pipelineOps == nil {
		pipelineOps = defaultPipelineOps{}
	}
	return &PolicyManager{
		policies:    policies,
		buildConfig: buildConfig,
		pipelines:   pipelineOps,
	}
}

// List 返回应用下全部策略，按创建时间升序
func (m *PolicyManager) List(ctx context.Context, appID string) ([]Policy, error) {
	policies, err := m.policies.List(ctx, appID)
	if err != nil {
		return nil, errors.Wrapf(err, "list trigger policies of app %s", appID)
	}
	return policies, nil
}

// Create 创建触发策略；首条创建前 Ensure 流水线，失败不落库
// 本期不同步工蜂触发器，triggerID 留空
func (m *PolicyManager) Create(
	ctx context.Context,
	app *bkmsapp.Application,
	creator string,
	form PolicyForm,
) (*Policy, error) {
	normalized, err := m.validateForm(form)
	if err != nil {
		return nil, err
	}
	if err = m.ensureAppEligible(ctx, app.ID); err != nil {
		return nil, err
	}

	existing, err := m.policies.List(ctx, app.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "list trigger policies of app %s", app.ID)
	}
	if len(existing) >= MaxPoliciesPerApp {
		return nil, ErrTooManyPolicies
	}
	if err = m.ensureNoConflict(normalized, existing, ""); err != nil {
		return nil, err
	}

	pipelineID, err := m.pipelineIDForCreate(ctx, app.WorkspaceID, app.ID, existing)
	if err != nil {
		return nil, err
	}

	policy := &Policy{
		ID:               PolicyIDPrefix + stringx.Random(policyIDRandLen),
		AppID:            app.ID,
		Name:             normalized.Name,
		Event:            normalized.Event,
		BranchMatchMode:  normalized.BranchMatchMode,
		BranchMatchValue: normalized.BranchMatchValue,
		PathFilter:       normalized.PathFilter,
		Status:           StatusEnabled,
		PipelineID:       pipelineID,
		TriggerID:        "",
		Creator:          creator,
	}
	if err = m.policies.Create(ctx, policy); err != nil {
		return nil, err
	}
	// FIXME: 同步工蜂触发器
	return policy, nil
}

// Update 更新策略表单字段，排除自身后复检冲突；不调用 Ensure / Cleanup
func (m *PolicyManager) Update(
	ctx context.Context,
	appID, policyID string,
	form PolicyForm,
) (*Policy, error) {
	normalized, err := m.validateForm(form)
	if err != nil {
		return nil, err
	}
	if err = m.ensureAutoTagEnabled(ctx, appID); err != nil {
		return nil, err
	}

	current, err := m.policies.Get(ctx, appID, policyID)
	if err != nil {
		return nil, err
	}
	existing, err := m.policies.List(ctx, appID)
	if err != nil {
		return nil, errors.Wrapf(err, "list trigger policies of app %s", appID)
	}
	if err = m.ensureNoConflict(normalized, existing, policyID); err != nil {
		return nil, err
	}

	current.Name = normalized.Name
	current.Event = normalized.Event
	current.BranchMatchMode = normalized.BranchMatchMode
	current.BranchMatchValue = normalized.BranchMatchValue
	current.PathFilter = normalized.PathFilter
	if err = m.policies.Update(ctx, current); err != nil {
		return nil, err
	}
	// FIXME: 同步工蜂触发器
	return m.policies.Get(ctx, appID, policyID)
}

// PatchStatus 只更新启停状态，不重做冲突检测，不调用 Ensure / Cleanup
func (m *PolicyManager) PatchStatus(
	ctx context.Context,
	appID, policyID string,
	status Status,
) (*Policy, error) {
	if err := m.policies.UpdateStatus(ctx, appID, policyID, status); err != nil {
		return nil, err
	}
	// FIXME: 同步工蜂触发器
	return m.policies.Get(ctx, appID, policyID)
}

// Delete 删除策略；末条须先 Cleanup，失败则策略仍在
func (m *PolicyManager) Delete(ctx context.Context, workspaceID, appID, policyID string) error {
	if _, err := m.policies.Get(ctx, appID, policyID); err != nil {
		return err
	}
	existing, err := m.policies.List(ctx, appID)
	if err != nil {
		return errors.Wrapf(err, "list trigger policies of app %s", appID)
	}
	if len(existing) == 1 {
		if err = m.pipelines.Cleanup(ctx, workspaceID, appID); err != nil {
			return errors.Wrap(err, "cleanup trigger pipeline")
		}
	}
	if err = m.policies.Delete(ctx, appID, policyID); err != nil {
		return err
	}
	// FIXME: 同步工蜂触发器
	return nil
}

// CheckConflict 预检分支匹配重叠且不落库
// excludePolicyID 为要排除的策略 ID，编辑时跳过自身；对应请求字段 excludeTriggerID
func (m *PolicyManager) CheckConflict(
	ctx context.Context,
	appID, excludePolicyID string,
	form PolicyForm,
) ([]ConflictHit, error) {
	normalized, err := m.validateForm(form)
	if err != nil {
		return nil, err
	}
	existing, err := m.policies.List(ctx, appID)
	if err != nil {
		return nil, errors.Wrapf(err, "list trigger policies of app %s", appID)
	}
	hits := m.collectConflicts(normalized, existing, excludePolicyID)
	return hits, nil
}

// GuardBuildConfigUpdate 存在任意策略时拦截改 repoAlias、sourceType、关闭自动生成 tag
func (m *PolicyManager) GuardBuildConfigUpdate(
	ctx context.Context,
	appID string,
	before, after *imagebuild.Config,
) error {
	existing, err := m.policies.List(ctx, appID)
	if err != nil {
		return errors.Wrapf(err, "list trigger policies of app %s", appID)
	}
	if len(existing) == 0 {
		return nil
	}
	if before.SourceType != after.SourceType {
		return ErrBuildConfigLocked
	}
	if repoAliasOf(before) != repoAliasOf(after) {
		return ErrBuildConfigLocked
	}
	if before.TagConfig.IsAutoGenerateEnabled() && !after.TagConfig.IsAutoGenerateEnabled() {
		return ErrBuildConfigLocked
	}
	return nil
}

// repoAliasOf 取出构建配置上的仓库别名，未配置代码库时视为空串以便与更新后对比
func repoAliasOf(cfg *imagebuild.Config) string {
	if cfg == nil || cfg.CodeRepo == nil {
		return ""
	}
	return cfg.CodeRepo.RepoAlias
}

// validateForm 规范化表单并做跨字段校验：all 时匹配值必须为空，eq/prefix 时匹配值必填
// 失败仍可用 errors.Is(..., ErrInvalidBranchMatch) 识别
func (m *PolicyManager) validateForm(form PolicyForm) (PolicyForm, error) {
	form.Name = strings.TrimSpace(form.Name)
	form.PathFilter = strings.TrimSpace(form.PathFilter)
	form.Event = Event(strings.TrimSpace(string(form.Event)))
	form.BranchMatchMode = BranchMatchMode(strings.TrimSpace(string(form.BranchMatchMode)))

	switch form.BranchMatchMode {
	case BranchMatchModeAll:
		if strings.TrimSpace(form.BranchMatchValue) != "" {
			return PolicyForm{}, withSentinelMessage(ErrInvalidBranchMatch, "匹配方式为全部时匹配值必须为空")
		}
		form.BranchMatchValue = ""
	case BranchMatchModeEq, BranchMatchModePrefix:
		values := parseMatchValues(form.BranchMatchMode, form.BranchMatchValue)
		if len(values) == 0 {
			return PolicyForm{}, withSentinelMessage(ErrInvalidBranchMatch, "匹配方式为等于或前缀时匹配值必填")
		}
		form.BranchMatchValue = joinMatchValues(values)
	default:
		return PolicyForm{}, withSentinelMessage(
			ErrInvalidBranchMatch,
			"未知分支匹配方式 "+string(form.BranchMatchMode),
		)
	}
	return form, nil
}

// ensureAppEligible 创建准入：仅源码仓库 + 工蜂，且已开启自动生成 tag
func (m *PolicyManager) ensureAppEligible(ctx context.Context, appID string) error {
	cfg, err := m.loadBuildConfig(ctx, appID)
	if err != nil {
		return err
	}
	if cfg.SourceType != imagebuild.SourceTypeCodeRepository ||
		cfg.CodeRepo == nil ||
		cfg.CodeRepo.Type != imagebuild.RepositoryTypeTGit {
		return ErrUnsupportedAppType
	}
	if !cfg.TagConfig.IsAutoGenerateEnabled() {
		return ErrAutoGenerateTagDisabled
	}
	return nil
}

// ensureAutoTagEnabled 更新准入：只复核自动生成 tag，仓库类型改动由 GuardBuildConfigUpdate 拦住
func (m *PolicyManager) ensureAutoTagEnabled(ctx context.Context, appID string) error {
	cfg, err := m.loadBuildConfig(ctx, appID)
	if err != nil {
		return err
	}
	if !cfg.TagConfig.IsAutoGenerateEnabled() {
		return ErrAutoGenerateTagDisabled
	}
	return nil
}

// loadBuildConfig 读取应用构建配置，未命中或存储失败原样 Wrap 给上层
func (m *PolicyManager) loadBuildConfig(ctx context.Context, appID string) (*imagebuild.Config, error) {
	cfg, err := m.buildConfig.Get(ctx, appID)
	if err != nil {
		return nil, errors.Wrapf(err, "get build config of app %s", appID)
	}
	return cfg, nil
}

// ensureNoConflict 提交路径复检冲突，命中则返回 PolicyConflictError，由 handler 映射为 400
func (m *PolicyManager) ensureNoConflict(form PolicyForm, existing []Policy, excludePolicyID string) error {
	hits := m.collectConflicts(form, existing, excludePolicyID)
	if len(hits) == 0 {
		return nil
	}
	return &PolicyConflictError{Hits: hits}
}

// collectConflicts 收集与已有策略的硬冲突，已停用策略仍占冲突空间
// excludePolicyID 非空时跳过该条，供编辑自身排除
func (m *PolicyManager) collectConflicts(
	form PolicyForm,
	existing []Policy,
	excludePolicyID string,
) []ConflictHit {
	hits := make([]ConflictHit, 0)
	for _, policy := range existing {
		if excludePolicyID != "" && policy.ID == excludePolicyID {
			continue
		}
		if hit := m.detectOverlap(form, policy); hit != nil {
			hits = append(hits, *hit)
		}
	}
	return hits
}

// pipelineIDForCreate 首条策略创建前 Ensure 流水线并取其 ID；后续复用已有策略的 pipelineID，不再 Ensure
func (m *PolicyManager) pipelineIDForCreate(
	ctx context.Context,
	workspaceID, appID string,
	existing []Policy,
) (string, error) {
	if len(existing) == 0 {
		pipeline, err := m.pipelines.Ensure(ctx, workspaceID, appID)
		if err != nil {
			return "", errors.Wrap(err, "ensure trigger pipeline")
		}
		return pipeline.ID, nil
	}
	for _, policy := range existing {
		if policy.PipelineID != "" {
			return policy.PipelineID, nil
		}
	}
	return "", errors.New("existing trigger policies missing pipelineID")
}
