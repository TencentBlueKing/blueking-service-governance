// Package handler contains Gin handlers for workspace application defaults.
package handler

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/serializer"
)

// Handler handles workspace application-default APIs.
type Handler struct {
	registry *storereg.Registry
}

// New creates an application-default API handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

type listRulesOutput[T any] struct {
	Data []T `json:"data"`
}

type ruleOutput[T any] struct {
	Data T `json:"data"`
}

func listRules[Output any](
	h *Handler,
	c *gin.Context,
	configType appdefaults.ConfigType,
	fromModel func(*Output, appdefaults.Rule) *Output,
) {
	var uri serializer.WorkspaceURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	workspace, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	rules, err := h.registry.AppDefaultRuleStore.ListByConfigType(ctx, workspace.ID, configType)
	if err != nil {
		bkerrs.AbortWithErr(c, apiError(err, "list "+configType.String()+" application default rules"))
		return
	}
	data := make([]*Output, 0, len(rules))
	for _, rule := range rules {
		data = append(data, fromModel(new(Output), rule))
	}
	ginutils.OK(c, listRulesOutput[*Output]{Data: data})
}

func createRule[Input, Output any](
	h *Handler,
	c *gin.Context,
	configType appdefaults.ConfigType,
	toModel func(Input) appdefaults.RuleDefinition,
	fromModel func(*Output, appdefaults.Rule) *Output,
) {
	input := new(Input)
	var uri serializer.WorkspaceURIInput
	if err := ginutils.BindURIJSON(c, &uri, input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	workspace, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	definition := toModel(*input)
	created := &appdefaults.Rule{
		WorkspaceID: workspace.ID,
		ConfigType:  configType,
		EnvType:     definition.EnvType,
		Spec:        definition.Spec,
	}
	if err = appdefaults.ValidateRule(created); err != nil {
		bkerrs.AbortWithErr(c, apiError(err, "create "+configType.String()+" application default rule"))
		return
	}
	if err = h.registry.AppDefaultRuleStore.Create(ctx, created); err != nil {
		bkerrs.AbortWithErr(c, apiError(err, "create "+configType.String()+" application default rule"))
		return
	}
	recordAudit(ctx, workspace.ID, audit.OperationTypeCreate, nil, created)
	ginutils.OK(c, ruleOutput[*Output]{Data: fromModel(new(Output), *created)})
}

func updateRule[Input, Output any](
	h *Handler,
	c *gin.Context,
	configType appdefaults.ConfigType,
	toModel func(Input) appdefaults.RuleDefinition,
	fromModel func(*Output, appdefaults.Rule) *Output,
) {
	input := new(Input)
	var uri serializer.RuleURIInput
	if err := ginutils.BindURIJSON(c, &uri, input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	workspace, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ruleID, err := bson.ObjectIDFromHex(uri.RuleID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "invalid rule ID"))
		return
	}

	// Workspace and config type stay fixed by the route; updating another
	// section with the same rule ID therefore resolves as not found.
	before, err := h.registry.AppDefaultRuleStore.Get(ctx, workspace.ID, configType, ruleID)
	if err != nil {
		bkerrs.AbortWithErr(c, apiError(err, "update "+configType.String()+" application default rule"))
		return
	}
	definition := toModel(*input)
	updated := *before
	updated.EnvType = definition.EnvType
	updated.Spec = definition.Spec
	if err = appdefaults.ValidateRule(&updated); err != nil {
		bkerrs.AbortWithErr(c, apiError(err, "update "+configType.String()+" application default rule"))
		return
	}
	if err = h.registry.AppDefaultRuleStore.Update(ctx, workspace.ID, configType, &updated); err != nil {
		bkerrs.AbortWithErr(c, apiError(err, "update "+configType.String()+" application default rule"))
		return
	}
	recordAudit(ctx, workspace.ID, audit.OperationTypeUpdate, before, &updated)
	ginutils.OK(c, ruleOutput[*Output]{Data: fromModel(new(Output), updated)})
}

func deleteRule(
	h *Handler,
	c *gin.Context,
	configType appdefaults.ConfigType,
) {
	var uri serializer.RuleURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	workspace, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ruleID, err := bson.ObjectIDFromHex(uri.RuleID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "invalid rule ID"))
		return
	}

	deleted, err := h.registry.AppDefaultRuleStore.Delete(ctx, workspace.ID, configType, ruleID)
	if err != nil {
		bkerrs.AbortWithErr(c, apiError(err, "delete "+configType.String()+" application default rule"))
		return
	}
	recordAudit(ctx, workspace.ID, audit.OperationTypeDelete, deleted, nil)
	ginutils.OK(c, serializer.EmptyOutput{})
}

func apiError(err error, operation string) error {
	switch {
	case errors.Is(err, appdefaults.ErrRuleNotFound):
		return bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, operation)
	case errors.Is(err, appdefaults.ErrInvalidRule),
		errors.Is(err, appdefaults.ErrRuleConflict):
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, operation)
	default:
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, operation)
	}
}

func recordAudit(
	ctx context.Context,
	workspaceID string,
	operation audit.OperationType,
	before, after *appdefaults.Rule,
) {
	options := []audit.Option{
		audit.WithAttribute(audit.AttributeWorkspaceAppSpecDefaults),
		audit.WithWorkspaceID(workspaceID),
	}
	// Create and delete have only one side of the audit snapshot; update has
	// both.
	if before != nil {
		options = append(options, audit.WithDataBefore(before))
	}
	if after != nil {
		options = append(options, audit.WithDataAfter(after))
	}
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		operation,
		audit.ResourceTypeWorkspace,
		workspaceID,
		options...,
	)
}
