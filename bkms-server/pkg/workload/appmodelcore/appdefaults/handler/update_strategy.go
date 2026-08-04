package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// ListUpdateStrategyRules lists workspace update-strategy initialization rules.
//
//	@ID			ListWorkspaceAppSpecUpdateStrategyRules
//	@Summary	查询工作空间更新策略默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.ListUpdateStrategyRulesOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/update-strategy [get]
func (h *Handler) ListUpdateStrategyRules(c *gin.Context) {
	listRules(
		h,
		c,
		appspec.AppSpecSectionUpdateStrategy,
		(*serializer.UpdateStrategyRuleOutputObj).FromModel,
	)
}

// CreateUpdateStrategyRule creates a workspace update-strategy initialization rule.
//
//	@ID			CreateWorkspaceAppSpecUpdateStrategyRule
//	@Summary	新增工作空间更新策略默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string								true	"工作空间 ID"
//	@Param		body		body		serializer.UpdateStrategyRuleInput	true	"更新策略默认配置规则"
//	@Success	200			{object}	serializer.UpdateStrategyRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/update-strategy [post]
func (h *Handler) CreateUpdateStrategyRule(c *gin.Context) {
	createRule(
		h,
		c,
		appspec.AppSpecSectionUpdateStrategy,
		serializer.UpdateStrategyRuleInput.ToModel,
		(*serializer.UpdateStrategyRuleOutputObj).FromModel,
	)
}

// UpdateUpdateStrategyRule replaces a workspace update-strategy initialization rule.
//
//	@ID			UpdateWorkspaceAppSpecUpdateStrategyRule
//	@Summary	编辑工作空间更新策略默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string								true	"工作空间 ID"
//	@Param		ruleID		path		string								true	"规则 ID"
//	@Param		body		body		serializer.UpdateStrategyRuleInput	true	"更新策略默认配置规则"
//	@Success	200			{object}	serializer.UpdateStrategyRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/update-strategy/{ruleID} [put]
func (h *Handler) UpdateUpdateStrategyRule(c *gin.Context) {
	updateRule(
		h,
		c,
		appspec.AppSpecSectionUpdateStrategy,
		serializer.UpdateStrategyRuleInput.ToModel,
		(*serializer.UpdateStrategyRuleOutputObj).FromModel,
	)
}

// DeleteUpdateStrategyRule deletes a workspace update-strategy initialization rule.
//
//	@ID			DeleteWorkspaceAppSpecUpdateStrategyRule
//	@Summary	删除工作空间更新策略默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		ruleID		path		string	true	"规则 ID"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/update-strategy/{ruleID} [delete]
func (h *Handler) DeleteUpdateStrategyRule(c *gin.Context) {
	deleteRule(h, c, appspec.AppSpecSectionUpdateStrategy)
}
