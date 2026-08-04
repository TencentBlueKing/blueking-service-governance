package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// ListResourcesRules lists workspace resources initialization rules.
//
//	@ID			ListWorkspaceAppSpecResourcesRules
//	@Summary	查询工作空间资源规格默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.ListResourcesRulesOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/resources [get]
func (h *Handler) ListResourcesRules(c *gin.Context) {
	listRules(h, c, appspec.AppSpecSectionResources, (*serializer.ResourcesRuleOutputObj).FromModel)
}

// CreateResourcesRule creates a workspace resources initialization rule.
//
//	@ID			CreateWorkspaceAppSpecResourcesRule
//	@Summary	新增工作空间资源规格默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string							true	"工作空间 ID"
//	@Param		body		body		serializer.ResourcesRuleInput	true	"资源规格默认配置规则"
//	@Success	200			{object}	serializer.ResourcesRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/resources [post]
func (h *Handler) CreateResourcesRule(c *gin.Context) {
	createRule(
		h,
		c,
		appspec.AppSpecSectionResources,
		serializer.ResourcesRuleInput.ToModel,
		(*serializer.ResourcesRuleOutputObj).FromModel,
	)
}

// UpdateResourcesRule replaces a workspace resources initialization rule.
//
//	@ID			UpdateWorkspaceAppSpecResourcesRule
//	@Summary	编辑工作空间资源规格默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string							true	"工作空间 ID"
//	@Param		ruleID		path		string							true	"规则 ID"
//	@Param		body		body		serializer.ResourcesRuleInput	true	"资源规格默认配置规则"
//	@Success	200			{object}	serializer.ResourcesRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/resources/{ruleID} [put]
func (h *Handler) UpdateResourcesRule(c *gin.Context) {
	updateRule(
		h,
		c,
		appspec.AppSpecSectionResources,
		serializer.ResourcesRuleInput.ToModel,
		(*serializer.ResourcesRuleOutputObj).FromModel,
	)
}

// DeleteResourcesRule deletes a workspace resources initialization rule.
//
//	@ID			DeleteWorkspaceAppSpecResourcesRule
//	@Summary	删除工作空间资源规格默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		ruleID		path		string	true	"规则 ID"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/resources/{ruleID} [delete]
func (h *Handler) DeleteResourcesRule(c *gin.Context) {
	deleteRule(h, c, appspec.AppSpecSectionResources)
}
