package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// ListProbeRules lists workspace probe initialization rules.
//
//	@ID			ListWorkspaceAppSpecProbeRules
//	@Summary	查询工作空间探针默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.ListProbeRulesOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/probe [get]
func (h *Handler) ListProbeRules(c *gin.Context) {
	listRules(h, c, appspec.AppSpecSectionProbe, (*serializer.ProbeRuleOutputObj).FromModel)
}

// CreateProbeRule creates a workspace probe initialization rule.
//
//	@ID			CreateWorkspaceAppSpecProbeRule
//	@Summary	新增工作空间探针默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string					  true	"工作空间 ID"
//	@Param		body		body		serializer.ProbeRuleInput true	"探针默认配置规则"
//	@Success	200			{object}	serializer.ProbeRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/probe [post]
func (h *Handler) CreateProbeRule(c *gin.Context) {
	createRule(
		h,
		c,
		appspec.AppSpecSectionProbe,
		serializer.ProbeRuleInput.ToModel,
		(*serializer.ProbeRuleOutputObj).FromModel,
	)
}

// UpdateProbeRule replaces a workspace probe initialization rule.
//
//	@ID			UpdateWorkspaceAppSpecProbeRule
//	@Summary	编辑工作空间探针默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string					  true	"工作空间 ID"
//	@Param		ruleID		path		string					  true	"规则 ID"
//	@Param		body		body		serializer.ProbeRuleInput true	"探针默认配置规则"
//	@Success	200			{object}	serializer.ProbeRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/probe/{ruleID} [put]
func (h *Handler) UpdateProbeRule(c *gin.Context) {
	updateRule(
		h,
		c,
		appspec.AppSpecSectionProbe,
		serializer.ProbeRuleInput.ToModel,
		(*serializer.ProbeRuleOutputObj).FromModel,
	)
}

// DeleteProbeRule deletes a workspace probe initialization rule.
//
//	@ID			DeleteWorkspaceAppSpecProbeRule
//	@Summary	删除工作空间探针默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		ruleID		path		string	true	"规则 ID"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/probe/{ruleID} [delete]
func (h *Handler) DeleteProbeRule(c *gin.Context) {
	deleteRule(h, c, appspec.AppSpecSectionProbe)
}
