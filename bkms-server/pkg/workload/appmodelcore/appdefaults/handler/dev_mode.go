package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// ListDevModeRules lists workspace dev-mode initialization rules.
//
//	@ID			ListWorkspaceAppSpecDevModeRules
//	@Summary	查询工作空间开发模式默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.ListDevModeRulesOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/dev-mode [get]
func (h *Handler) ListDevModeRules(c *gin.Context) {
	listRules(h, c, appspec.AppSpecSectionDevMode, (*serializer.DevModeRuleOutputObj).FromModel)
}

// CreateDevModeRule creates a workspace dev-mode initialization rule.
//
//	@ID			CreateWorkspaceAppSpecDevModeRule
//	@Summary	新增工作空间开发模式默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string						true	"工作空间 ID"
//	@Param		body		body		serializer.DevModeRuleInput	true	"开发模式默认配置规则"
//	@Success	200			{object}	serializer.DevModeRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/dev-mode [post]
func (h *Handler) CreateDevModeRule(c *gin.Context) {
	createRule(
		h,
		c,
		appspec.AppSpecSectionDevMode,
		serializer.DevModeRuleInput.ToModel,
		(*serializer.DevModeRuleOutputObj).FromModel,
	)
}

// UpdateDevModeRule replaces a workspace dev-mode initialization rule.
//
//	@ID			UpdateWorkspaceAppSpecDevModeRule
//	@Summary	编辑工作空间开发模式默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string						true	"工作空间 ID"
//	@Param		ruleID		path		string						true	"规则 ID"
//	@Param		body		body		serializer.DevModeRuleInput	true	"开发模式默认配置规则"
//	@Success	200			{object}	serializer.DevModeRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/dev-mode/{ruleID} [put]
func (h *Handler) UpdateDevModeRule(c *gin.Context) {
	updateRule(
		h,
		c,
		appspec.AppSpecSectionDevMode,
		serializer.DevModeRuleInput.ToModel,
		(*serializer.DevModeRuleOutputObj).FromModel,
	)
}

// DeleteDevModeRule deletes a workspace dev-mode initialization rule.
//
//	@ID			DeleteWorkspaceAppSpecDevModeRule
//	@Summary	删除工作空间开发模式默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		ruleID		path		string	true	"规则 ID"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/dev-mode/{ruleID} [delete]
func (h *Handler) DeleteDevModeRule(c *gin.Context) {
	deleteRule(h, c, appspec.AppSpecSectionDevMode)
}
