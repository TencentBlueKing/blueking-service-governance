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

// Package handler 提供自动触发策略的 Gin 接口
// 本期已落地策略 CRUD / 启停 / 冲突预检；触发记录查询与构建回调仍为空实现
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/trigger"
	triggerserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/trigger/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

var _ trigger.Handler = (*Handler)(nil)

// Handler handles Gin build trigger API requests.
type Handler struct {
	registry      *storereg.Registry
	policyManager *trigger.PolicyManager
}

// New 创建 Handler；pipelineOps 传 nil，走默认蓝盾 TriggerPipelineManager
func New(registry *storereg.Registry) *Handler {
	return &Handler{
		registry: registry,
		policyManager: trigger.NewPolicyManager(
			registry.BuildTriggerPolicyStore, registry.BuildConfigStore, nil,
		),
	}
}

// policyOutput 将策略实体包成单条接口响应，Health 在 FromModel 中固定为 unknown
func policyOutput(p *trigger.Policy) triggerserializer.PolicyOutput {
	return triggerserializer.PolicyOutput{
		Data: new(triggerserializer.PolicyOutputObj).FromModel(*p),
	}
}

// conflictCheckOutput 组装预检响应：无命中为 none 且两个列表为空数组，有命中一律 error
// 预检接口始终 HTTP 200，用 level 区分，本期不写 warn
func conflictCheckOutput(hits []trigger.ConflictHit) *triggerserializer.ConflictCheckOutputObj {
	names := make([]string, 0, len(hits))
	reasons := make([]triggerserializer.ConflictReasonObj, 0, len(hits))
	for _, hit := range hits {
		names = append(names, hit.PolicyName)
		reasons = append(reasons, triggerserializer.ConflictReasonObj{
			PolicyName:  hit.PolicyName,
			OverlapType: string(hit.OverlapType),
			Message:     hit.Message,
		})
	}
	level := string(triggerserializer.ConflictLevelNone)
	if len(hits) > 0 {
		level = string(triggerserializer.ConflictLevelError)
	}
	return &triggerserializer.ConflictCheckOutputObj{
		Level:               level,
		ConflictPolicyNames: names,
		ConflictReasons:     reasons,
	}
}

// mapPolicyErr 将策略领域错误映射为 HTTP ErrCode
// 硬冲突 / 重名 / 数量上限 / 准入失败等业务规则为 INVALID_ARGUMENT；未找到为 NOT_FOUND；其余为 500
// 重名单独给中文文案，避免把英文哨兵透出
func mapPolicyErr(err error) error {
	var conflictErr *trigger.PolicyConflictError
	if errors.As(err, &conflictErr) {
		return bkerrs.New(bkerrs.ErrCodeInvalidArgument, conflictErr.Error())
	}
	switch {
	case errors.Is(err, trigger.ErrPolicyNotFound):
		return bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, err.Error())
	case errors.Is(err, trigger.ErrPolicyNameDuplicated):
		return bkerrs.New(bkerrs.ErrCodeInvalidArgument, "同应用下已存在同名触发策略")
	case errors.Is(err, trigger.ErrTooManyPolicies),
		errors.Is(err, trigger.ErrAutoGenerateTagDisabled),
		errors.Is(err, trigger.ErrUnsupportedAppType),
		errors.Is(err, trigger.ErrInvalidBranchMatch),
		errors.Is(err, trigger.ErrBuildConfigLocked):
		return bkerrs.New(bkerrs.ErrCodeInvalidArgument, err.Error())
	default:
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, err.Error())
	}
}

// ListBuildTriggerPolicies 获取应用的触发策略列表，需查看权限
// 单应用策略上限为 MaxPoliciesPerApp，一次返回全部，不支持分页
//
//	@ID				ListBuildTriggerPolicies
//	@Summary		获取应用的触发策略列表
//	@Tags			build-trigger-policies
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Success		200		{object}	triggerserializer.ListPoliciesOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		404		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/build-trigger-policies [get]
func (h *Handler) ListBuildTriggerPolicies(c *gin.Context) {
	var uriInput triggerserializer.AppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	policies, err := h.policyManager.List(ctx, uriInput.AppID)
	if err != nil {
		bkerrs.AbortWithErr(c, mapPolicyErr(err))
		return
	}
	// 用空切片而非 nil，避免 JSON 把 results 编成 null
	results := make([]*triggerserializer.PolicyOutputObj, 0, len(policies))
	for i := range policies {
		results = append(results, new(triggerserializer.PolicyOutputObj).FromModel(policies[i]))
	}
	ginutils.OK(c, triggerserializer.ListPoliciesOutput{
		Data: &triggerserializer.PolicyListOutputObjs{
			Count:   int64(len(results)),
			Results: results,
		},
	})
}

// CreateBuildTriggerPolicy 新增触发策略，需构建权限；首条会 Ensure 触发专用流水线
//
//	@ID				CreateBuildTriggerPolicy
//	@Summary		新增触发策略
//	@Tags			build-trigger-policies
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string									true	"应用 ID"
//	@Param			body	body		triggerserializer.PolicyFormInput		true	"触发策略表单"
//	@Success		200		{object}	triggerserializer.PolicyOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		404		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/build-trigger-policies [post]
func (h *Handler) CreateBuildTriggerPolicy(c *gin.Context) {
	var uriInput triggerserializer.AppURIInput
	var input triggerserializer.PolicyFormInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	policy, err := h.policyManager.Create(ctx, app, auth.MustGetUser(ctx).ID, input.ToModel())
	if err != nil {
		bkerrs.AbortWithErr(c, mapPolicyErr(err))
		return
	}
	ginutils.OK(c, policyOutput(policy))
}

// UpdateBuildTriggerPolicy 更新触发策略。
//
//	@ID				UpdateBuildTriggerPolicy
//	@Summary		更新触发策略
//	@Tags			build-trigger-policies
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID		path		string								true	"应用 ID"
//	@Param			policyID	path		string								true	"触发策略 ID"
//	@Param			body		body		triggerserializer.PolicyFormInput	true	"触发策略表单"
//	@Success		200			{object}	triggerserializer.PolicyOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Failure		404			{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/build-trigger-policies/{policyID} [put]
func (h *Handler) UpdateBuildTriggerPolicy(c *gin.Context) {
	var uriInput triggerserializer.PolicyURIInput
	var input triggerserializer.PolicyFormInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	policy, err := h.policyManager.Update(ctx, uriInput.AppID, uriInput.PolicyID, input.ToModel())
	if err != nil {
		bkerrs.AbortWithErr(c, mapPolicyErr(err))
		return
	}
	ginutils.OK(c, policyOutput(policy))
}

// PatchBuildTriggerPolicyStatus 启用或停用触发策略。
//
//	@ID				PatchBuildTriggerPolicyStatus
//	@Summary		启用或停用触发策略
//	@Tags			build-trigger-policies
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID		path		string											true	"应用 ID"
//	@Param			policyID	path		string											true	"触发策略 ID"
//	@Param			body		body		triggerserializer.PatchPolicyStatusInput		true	"启停参数"
//	@Success		200			{object}	triggerserializer.PolicyOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Failure		404			{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/build-trigger-policies/{policyID}/status [patch]
func (h *Handler) PatchBuildTriggerPolicyStatus(c *gin.Context) {
	var uriInput triggerserializer.PolicyURIInput
	var input triggerserializer.PatchPolicyStatusInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	policy, err := h.policyManager.PatchStatus(ctx, uriInput.AppID, uriInput.PolicyID, input.Status())
	if err != nil {
		bkerrs.AbortWithErr(c, mapPolicyErr(err))
		return
	}
	ginutils.OK(c, policyOutput(policy))
}

// DeleteBuildTriggerPolicy 删除触发策略，需构建权限；末条会先 Cleanup 流水线，失败则策略仍在
//
//	@ID				DeleteBuildTriggerPolicy
//	@Summary		删除触发策略
//	@Tags			build-trigger-policies
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID		path	string	true	"应用 ID"
//	@Param			policyID	path	string	true	"触发策略 ID"
//	@Success		204			{object}	nil
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Failure		404			{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/build-trigger-policies/{policyID} [delete]
func (h *Handler) DeleteBuildTriggerPolicy(c *gin.Context) {
	var uriInput triggerserializer.PolicyURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if err = h.policyManager.Delete(ctx, app.WorkspaceID, uriInput.AppID, uriInput.PolicyID); err != nil {
		bkerrs.AbortWithErr(c, mapPolicyErr(err))
		return
	}
	ginutils.NoContent(c)
}

// CheckBuildTriggerPolicyConflict 预检触发策略的重叠冲突，始终 HTTP 200，用 level 区分 none / error
//
//	@ID				CheckBuildTriggerPolicyConflict
//	@Summary		预检触发策略的重叠冲突
//	@Tags			build-trigger-policies
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string									true	"应用 ID"
//	@Param			body	body		triggerserializer.ConflictCheckInput	true	"冲突预检参数"
//	@Success		200		{object}	triggerserializer.ConflictCheckOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		404		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/build-trigger-policies/conflict-check [post]
func (h *Handler) CheckBuildTriggerPolicyConflict(c *gin.Context) {
	var uriInput triggerserializer.AppURIInput
	var input triggerserializer.ConflictCheckInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 编辑时排除自身：JSON 字段 excludeTriggerID 传的是策略 ID，不是蓝盾 triggerID
	hits, err := h.policyManager.CheckConflict(
		ctx, uriInput.AppID, input.ExcludeTriggerID, input.Policy.ToModel(),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, mapPolicyErr(err))
		return
	}
	ginutils.OK(c, triggerserializer.ConflictCheckOutput{Data: conflictCheckOutput(hits)})
}

// ListBuildTriggerPolicyRecords 获取触发策略的触发记录列表。
//
//	@ID				ListBuildTriggerPolicyRecords
//	@Summary		获取触发策略的触发记录列表
//	@Tags			build-trigger-policies
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID		path		string	true	"应用 ID"
//	@Param			policyID	path		string	true	"触发策略 ID"
//	@Param			result		query		string	false	"结果筛选：built / skipped / failed，留空表示不筛选"
//	@Param			page		query		int		true	"分页参数：页码，从 1 开始"
//	@Param			pageSize	query		int		true	"分页参数：每页数量，支持 5/10/20/50/100"
//	@Success		200			{object}	triggerserializer.ListTriggerRecordsOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Failure		404			{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/build-trigger-policies/{policyID}/records [get]
func (h *Handler) ListBuildTriggerPolicyRecords(c *gin.Context) {
	var uriInput triggerserializer.PolicyURIInput
	var queryInput triggerserializer.ListTriggerRecordsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ginutils.OK(c, triggerserializer.ListTriggerRecordsOutput{
		Data: &triggerserializer.PaginatedTriggerRecordOutputObjs{
			Count:   0,
			Results: []*triggerserializer.TriggerRecordOutputObj{},
		},
	})
}
