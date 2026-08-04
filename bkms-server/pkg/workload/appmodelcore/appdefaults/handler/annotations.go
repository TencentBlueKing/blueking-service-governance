package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// ListAnnotationsRules lists workspace annotations initialization rules.
//
//	@ID			ListWorkspaceAppSpecAnnotationsRules
//	@Summary	查询工作空间注解默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.ListAnnotationsRulesOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/annotations [get]
func (h *Handler) ListAnnotationsRules(c *gin.Context) {
	listRules(h, c, appspec.AppSpecSectionAnnotations, (*serializer.AnnotationsRuleOutputObj).FromModel)
}

// CreateAnnotationsRule creates a workspace annotations initialization rule.
//
//	@ID			CreateWorkspaceAppSpecAnnotationsRule
//	@Summary	新增工作空间注解默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string							true	"工作空间 ID"
//	@Param		body		body		serializer.AnnotationsRuleInput	true	"注解默认配置规则"
//	@Success	200			{object}	serializer.AnnotationsRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/annotations [post]
func (h *Handler) CreateAnnotationsRule(c *gin.Context) {
	createRule(
		h,
		c,
		appspec.AppSpecSectionAnnotations,
		serializer.AnnotationsRuleInput.ToModel,
		(*serializer.AnnotationsRuleOutputObj).FromModel,
	)
}

// UpdateAnnotationsRule replaces a workspace annotations initialization rule.
//
//	@ID			UpdateWorkspaceAppSpecAnnotationsRule
//	@Summary	编辑工作空间注解默认配置规则
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string							true	"工作空间 ID"
//	@Param		ruleID		path		string							true	"规则 ID"
//	@Param		body		body		serializer.AnnotationsRuleInput	true	"注解默认配置规则"
//	@Success	200			{object}	serializer.AnnotationsRuleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/annotations/{ruleID} [put]
func (h *Handler) UpdateAnnotationsRule(c *gin.Context) {
	updateRule(
		h,
		c,
		appspec.AppSpecSectionAnnotations,
		serializer.AnnotationsRuleInput.ToModel,
		(*serializer.AnnotationsRuleOutputObj).FromModel,
	)
}

// DeleteAnnotationsRule deletes a workspace annotations initialization rule.
//
//	@ID			DeleteWorkspaceAppSpecAnnotationsRule
//	@Summary	删除工作空间注解默认配置规则
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		ruleID		path		string	true	"规则 ID"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/app-spec/annotations/{ruleID} [delete]
func (h *Handler) DeleteAnnotationsRule(c *gin.Context) {
	deleteRule(h, c, appspec.AppSpecSectionAnnotations)
}
