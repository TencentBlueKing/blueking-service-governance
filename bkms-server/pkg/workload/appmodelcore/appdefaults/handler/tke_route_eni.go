package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// ListTkeRouteEniRules lists workspace TKE Route ENI initialization rules.
//
//	@ID			ListWorkspaceAppSpecTkeRouteEniRules
//	@Summary	查询工作空间 TKE Route ENI 默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.ListTkeRouteEniRulesOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/tke-route-eni [get]
func (h *Handler) ListTkeRouteEniRules(c *gin.Context) {
	listRules(
		h,
		c,
		appspec.AppSpecSectionTkeRouteEni,
		(*serializer.TkeRouteEniRuleOutputObj).FromModel,
	)
}

// CreateTkeRouteEniRule creates a workspace TKE Route ENI initialization rule.
//
//	@ID			CreateWorkspaceAppSpecTkeRouteEniRule
//	@Summary	新增工作空间 TKE Route ENI 默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string						   true	"工作空间 ID"
//	@Param		body		body		serializer.TkeRouteEniRuleInput true	"TKE Route ENI 默认配置规则"
//	@Success	200			{object}	serializer.TkeRouteEniRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/tke-route-eni [post]
func (h *Handler) CreateTkeRouteEniRule(c *gin.Context) {
	createRule(
		h,
		c,
		appspec.AppSpecSectionTkeRouteEni,
		serializer.TkeRouteEniRuleInput.ToModel,
		(*serializer.TkeRouteEniRuleOutputObj).FromModel,
	)
}

// UpdateTkeRouteEniRule replaces a workspace TKE Route ENI initialization rule.
//
//	@ID			UpdateWorkspaceAppSpecTkeRouteEniRule
//	@Summary	编辑工作空间 TKE Route ENI 默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string						   true	"工作空间 ID"
//	@Param		ruleID		path		string						   true	"规则 ID"
//	@Param		body		body		serializer.TkeRouteEniRuleInput true	"TKE Route ENI 默认配置规则"
//	@Success	200			{object}	serializer.TkeRouteEniRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/tke-route-eni/{ruleID} [put]
func (h *Handler) UpdateTkeRouteEniRule(c *gin.Context) {
	updateRule(
		h,
		c,
		appspec.AppSpecSectionTkeRouteEni,
		serializer.TkeRouteEniRuleInput.ToModel,
		(*serializer.TkeRouteEniRuleOutputObj).FromModel,
	)
}

// DeleteTkeRouteEniRule deletes a workspace TKE Route ENI initialization rule.
//
//	@ID			DeleteWorkspaceAppSpecTkeRouteEniRule
//	@Summary	删除工作空间 TKE Route ENI 默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		ruleID		path		string	true	"规则 ID"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/tke-route-eni/{ruleID} [delete]
func (h *Handler) DeleteTkeRouteEniRule(c *gin.Context) {
	deleteRule(h, c, appspec.AppSpecSectionTkeRouteEni)
}
