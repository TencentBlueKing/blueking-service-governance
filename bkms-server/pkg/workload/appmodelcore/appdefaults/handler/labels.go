package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// ListLabelsRules lists workspace labels initialization rules.
//
//	@ID			ListWorkspaceAppSpecLabelsRules
//	@Summary	查询工作空间标签默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.ListLabelsRulesOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/labels [get]
func (h *Handler) ListLabelsRules(c *gin.Context) {
	listRules(h, c, appspec.AppSpecSectionLabels, (*serializer.LabelsRuleOutputObj).FromModel)
}

// CreateLabelsRule creates a workspace labels initialization rule.
//
//	@ID			CreateWorkspaceAppSpecLabelsRule
//	@Summary	新增工作空间标签默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string					   true	"工作空间 ID"
//	@Param		body		body		serializer.LabelsRuleInput true	"标签默认配置规则"
//	@Success	200			{object}	serializer.LabelsRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/labels [post]
func (h *Handler) CreateLabelsRule(c *gin.Context) {
	createRule(
		h,
		c,
		appspec.AppSpecSectionLabels,
		serializer.LabelsRuleInput.ToModel,
		(*serializer.LabelsRuleOutputObj).FromModel,
	)
}

// UpdateLabelsRule replaces a workspace labels initialization rule.
//
//	@ID			UpdateWorkspaceAppSpecLabelsRule
//	@Summary	编辑工作空间标签默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string					   true	"工作空间 ID"
//	@Param		ruleID		path		string					   true	"规则 ID"
//	@Param		body		body		serializer.LabelsRuleInput true	"标签默认配置规则"
//	@Success	200			{object}	serializer.LabelsRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/labels/{ruleID} [put]
func (h *Handler) UpdateLabelsRule(c *gin.Context) {
	updateRule(
		h,
		c,
		appspec.AppSpecSectionLabels,
		serializer.LabelsRuleInput.ToModel,
		(*serializer.LabelsRuleOutputObj).FromModel,
	)
}

// DeleteLabelsRule deletes a workspace labels initialization rule.
//
//	@ID			DeleteWorkspaceAppSpecLabelsRule
//	@Summary	删除工作空间标签默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		ruleID		path		string	true	"规则 ID"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/labels/{ruleID} [delete]
func (h *Handler) DeleteLabelsRule(c *gin.Context) {
	deleteRule(h, c, appspec.AppSpecSectionLabels)
}
