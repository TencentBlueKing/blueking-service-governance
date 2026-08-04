package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// ListLifecycleRules lists workspace lifecycle initialization rules.
//
//	@ID			ListWorkspaceAppSpecLifecycleRules
//	@Summary	查询工作空间生命周期默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.ListLifecycleRulesOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/lifecycle [get]
func (h *Handler) ListLifecycleRules(c *gin.Context) {
	listRules(h, c, appspec.AppSpecSectionLifecycle, (*serializer.LifecycleRuleOutputObj).FromModel)
}

// CreateLifecycleRule creates a workspace lifecycle initialization rule.
//
//	@ID			CreateWorkspaceAppSpecLifecycleRule
//	@Summary	新增工作空间生命周期默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string						  true	"工作空间 ID"
//	@Param		body		body		serializer.LifecycleRuleInput true	"生命周期默认配置规则"
//	@Success	200			{object}	serializer.LifecycleRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/lifecycle [post]
func (h *Handler) CreateLifecycleRule(c *gin.Context) {
	createRule(
		h,
		c,
		appspec.AppSpecSectionLifecycle,
		serializer.LifecycleRuleInput.ToModel,
		(*serializer.LifecycleRuleOutputObj).FromModel,
	)
}

// UpdateLifecycleRule replaces a workspace lifecycle initialization rule.
//
//	@ID			UpdateWorkspaceAppSpecLifecycleRule
//	@Summary	编辑工作空间生命周期默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string						  true	"工作空间 ID"
//	@Param		ruleID		path		string						  true	"规则 ID"
//	@Param		body		body		serializer.LifecycleRuleInput true	"生命周期默认配置规则"
//	@Success	200			{object}	serializer.LifecycleRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/lifecycle/{ruleID} [put]
func (h *Handler) UpdateLifecycleRule(c *gin.Context) {
	updateRule(
		h,
		c,
		appspec.AppSpecSectionLifecycle,
		serializer.LifecycleRuleInput.ToModel,
		(*serializer.LifecycleRuleOutputObj).FromModel,
	)
}

// DeleteLifecycleRule deletes a workspace lifecycle initialization rule.
//
//	@ID			DeleteWorkspaceAppSpecLifecycleRule
//	@Summary	删除工作空间生命周期默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		ruleID		path		string	true	"规则 ID"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/lifecycle/{ruleID} [delete]
func (h *Handler) DeleteLifecycleRule(c *gin.Context) {
	deleteRule(h, c, appspec.AppSpecSectionLifecycle)
}
