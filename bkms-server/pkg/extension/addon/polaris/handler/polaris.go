// Package handler contains Gin handlers for polaris-config APIs.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

// Handler handles Gin polaris-config API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

func (h *Handler) polarisConfigService() *polaris.PolarisConfigService {
	return polaris.NewPolarisConfigService(
		h.registry.PolarisConfigStore,
		polaris.NewPolarisPlatformManager(
			h.registry.DepSvcStore,
			h.registry.DepSvcInstStore,
			h.registry.PolarisConfigStore,
		),
		polaris.NewPolarisEnvStateManager(h.registry.PolarisConfigStore),
		h.registry.EnvStore,
		h.registry.AppModelStore,
		envvars.NewUnifiedEnvVarsReader(
			h.registry.ScopedEnvVarStore,
			h.registry.AppDepsVarReader,
			h.registry.PolarisVarReader,
		),
	)
}

// ListAppPolarisConfigs 获取应用的北极星配置列表。
//
//	@ID			ListAppPolarisConfigs
//	@Summary	获取应用的北极星配置列表
//	@Tags		polaris-config
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	serializer.ListAppPolarisConfigsOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/deps/polaris-configs [get]
func (h *Handler) ListAppPolarisConfigs(c *gin.Context) {
	var uriInput serializer.AppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	configs, err := h.registry.PolarisConfigStore.ListByApp(ctx, app.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list polaris configs"))
		return
	}

	outputList := lo.Map(configs, func(config *polaris.PolarisConfig, _ int) *serializer.PolarisConfigOutputObj {
		warnings := polaris.CollectConfigWarnings(
			ctx,
			h.registry.AppModelStore,
			h.registry.AppConfigFileStore,
			config,
		)
		return new(serializer.PolarisConfigOutputObj).FromModel(*config, warnings)
	})

	ginutils.OK(c, serializer.ListAppPolarisConfigsOutput{Data: outputList})
}

// CreateAppPolarisConfig 创建北极星配置。
//
//	@ID			CreateAppPolarisConfig
//	@Summary	创建北极星配置
//	@Tags		polaris-config
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string								true	"应用 ID"
//	@Param		body	body		serializer.CreateAppPolarisConfigInput	true	"请求体"
//	@Success	200		{object}	serializer.CreateAppPolarisConfigOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/deps/polaris-configs [post]
func (h *Handler) CreateAppPolarisConfig(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var jsonInput serializer.CreateAppPolarisConfigInput

	if err := ginutils.BindURIJSON(c, &uriInput, &jsonInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	polarisToken := lo.FromPtr(jsonInput.PolarisToken)
	if !jsonInput.CreateNewService && polarisToken == "" {
		bkerrs.AbortWithErr(c, bkerrs.New(
			bkerrs.ErrCodeInvalidRequest,
			"polarisToken is required when importing existing polaris service",
		))
		return
	}

	config := &polaris.PolarisConfig{
		AppID: app.ID,
		Properties: polaris.Properties{
			InstanceKey:       jsonInput.InstanceKey,
			PolarisName:       jsonInput.PolarisName,
			PolarisNamespace:  jsonInput.PolarisNamespace,
			PolarisToken:      polarisToken,
			ServicePort:       jsonInput.ServicePort,
			Direct:            lo.FromPtrOr(jsonInput.Direct, true),
			KeepNotReadyPod:   lo.FromPtrOr(jsonInput.KeepNotReadyPod, true),
			EnableHealthCheck: lo.FromPtrOr(jsonInput.EnableHealthCheck, false),
			ServiceLabels:     jsonInput.ServiceLabels,
			Operator:          lo.FromPtrOr(jsonInput.Operator, ""),
		},
		ScopeEnvNames: jsonInput.ScopeEnvNames,
	}

	if err := h.polarisConfigService().Create(ctx, app, config, jsonInput.CreateNewService); err != nil {
		if errors.Is(err, polaris.ErrConfigNameExists) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeInvalidRequest,
				"polaris config name already exists in app(%s)",
				uriInput.AppID,
			))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create polaris config"))
		return
	}

	go audit.AddOperationRecordAsync(
		c.Request.Context(),
		audit.OperationTypeCreate,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributePolaris),
		audit.WithDataAfter(config),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.CreateAppPolarisConfigOutput{
		Data: serializer.PolarisNameOutputObj{Name: config.Name},
	})
}

// PatchAppPolarisConfig 更新北极星配置。
//
//	@ID			PatchAppPolarisConfig
//	@Summary	更新北极星配置
//	@Tags		polaris-config
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string								true	"应用 ID"
//	@Param		configName	path		string								true	"配置名称"
//	@Param		body		body		serializer.PatchAppPolarisConfigInput	true	"请求体"
//	@Success	200			{object}	serializer.PatchAppPolarisConfigOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/deps/polaris-configs/{configName} [patch]
func (h *Handler) PatchAppPolarisConfig(c *gin.Context) {
	var uriInput serializer.AppConfigNameURIInput
	var jsonInput serializer.PatchAppPolarisConfigInput
	if err := ginutils.BindURIJSON(c, &uriInput, &jsonInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	service := h.polarisConfigService()
	existingConfig, err := h.registry.PolarisConfigStore.Get(ctx, app.ID, uriInput.ConfigName)
	if err != nil {
		if errors.Is(err, polaris.ErrConfigNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeNotFound,
				"polaris config(%s) not found in app(%s)",
				uriInput.ConfigName, uriInput.AppID,
			))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get polaris config"))
		return
	}

	updateData := &polaris.ConfigUpdateData{
		InstanceKey:       jsonInput.InstanceKey,
		ServicePort:       jsonInput.ServicePort,
		Direct:            jsonInput.Direct,
		KeepNotReadyPod:   jsonInput.KeepNotReadyPod,
		EnableHealthCheck: jsonInput.EnableHealthCheck,
		ServiceLabels:     jsonInput.ServiceLabels,
		ScopeEnvNames:     jsonInput.ScopeEnvNames,
		PolarisToken:      jsonInput.PolarisToken,
	}

	updatedConfig, updateErr := service.Update(ctx, app, existingConfig, updateData)
	if updateErr != nil {
		if errors.Is(updateErr, polaris.ErrConfigNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeNotFound,
				"polaris config(%s) not found in app(%s)",
				uriInput.ConfigName, uriInput.AppID,
			))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(updateErr, bkerrs.ErrCodeInternalServerError, "update polaris config"))
		return
	}

	go audit.AddOperationRecordAsync(
		c.Request.Context(),
		audit.OperationTypeUpdate,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributePolaris),
		audit.WithDataBefore(existingConfig),
		audit.WithDataAfter(updatedConfig),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)

	ginutils.OK(c, new(serializer.PatchAppPolarisConfigOutput).FromModel(updatedConfig))
}

// DeleteAppPolarisConfig 删除北极星配置。
//
//	@ID			DeleteAppPolarisConfig
//	@Summary	删除北极星配置
//	@Tags		polaris-config
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		configName	path		string	true	"配置名称"
//	@Success	200			{object}	nil
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/deps/polaris-configs/{configName} [delete]
func (h *Handler) DeleteAppPolarisConfig(c *gin.Context) {
	var uriInput serializer.AppConfigNameURIInput
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

	service := h.polarisConfigService()
	existingConfig, err := h.registry.PolarisConfigStore.Get(ctx, app.ID, uriInput.ConfigName)
	if err != nil {
		if errors.Is(err, polaris.ErrConfigNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeNotFound,
				"polaris config(%s) not found in app(%s)",
				uriInput.ConfigName, uriInput.AppID,
			))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get polaris config"))
		return
	}

	if err := service.Delete(ctx, app, existingConfig); err != nil {
		if strings.Contains(err.Error(), "some instances existed in service") {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeAlreadyExists,
				"some instances existed in service, please delete the instances first",
			))
			return
		}
		if errors.Is(err, polaris.ErrConfigNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeNotFound,
				"polaris config(%s) not found in app(%s)",
				uriInput.ConfigName, uriInput.AppID,
			))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete polaris config"))
		return
	}

	go audit.AddOperationRecordAsync(
		c.Request.Context(),
		audit.OperationTypeDelete,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributePolaris),
		audit.WithDataBefore(existingConfig),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// ListAppPolarisConfigVars 获取北极星配置变量列表。
//
//	@ID			ListAppPolarisConfigVars
//	@Summary	获取北极星配置变量列表
//	@Tags		polaris-config
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		configName	path		string	true	"配置名称"
//	@Success	200			{object}	serializer.ListAppPolarisConfigVarsOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/deps/polaris-configs/{configName}/vars [get]
func (h *Handler) ListAppPolarisConfigVars(c *gin.Context) {
	var uriInput serializer.AppConfigNameURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	config, err := h.registry.PolarisConfigStore.Get(ctx, app.ID, uriInput.ConfigName)
	if err != nil {
		if errors.Is(err, polaris.ErrConfigNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeNotFound,
				"polaris config(%s) not found in app(%s)",
				uriInput.ConfigName, uriInput.AppID,
			))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get polaris config"))
		return
	}

	vars := lo.Map(config.GetVars(), func(v polaris.ConfigVar, _ int) serializer.PolarisConfigVarOutput {
		return *new(serializer.PolarisConfigVarOutput).FromModel(v)
	})

	ginutils.OK(c, serializer.ListAppPolarisConfigVarsOutput{Data: vars})
}

// ValidateAppPolarisConfig 校验北极星配置（创建前预校验）。
//
//	@ID			ValidateAppPolarisConfig
//	@Summary	校验北极星配置（创建前预校验）
//	@Tags		polaris-config
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string								true	"应用 ID"
//	@Param		body	body		serializer.CreateAppPolarisConfigInput	true	"请求体"
//	@Success	200		{object}	serializer.ValidateAppPolarisConfigOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/deps/polaris-configs/validate [post]
func (h *Handler) ValidateAppPolarisConfig(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var jsonInput serializer.CreateAppPolarisConfigInput
	if err := ginutils.BindURIJSON(c, &uriInput, &jsonInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	config := &polaris.PolarisConfig{
		AppID: app.ID,
		Properties: polaris.Properties{
			PolarisName:      jsonInput.PolarisName,
			PolarisNamespace: jsonInput.PolarisNamespace,
		},
		ScopeEnvNames: jsonInput.ScopeEnvNames,
	}

	warnings := polaris.CollectConfigWarnings(
		ctx,
		h.registry.AppModelStore,
		h.registry.AppConfigFileStore,
		config,
	)

	ginutils.OK(c, serializer.ValidateAppPolarisConfigOutput{Warnings: warnings})
}

// PutEnvWeight 更新指定环境的北极星实例权重。
//
//	@ID			PutEnvWeight
//	@Summary	更新指定环境的北极星实例权重
//	@Tags		polaris-config
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string						true	"应用 ID"
//	@Param		configName	path		string						true	"配置名称"
//	@Param		envName		path		string						true	"环境名称"
//	@Param		body		body		serializer.PutEnvWeightInput	true	"请求体"
//	@Success	200			{object}	serializer.PutEnvWeightOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	500			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/deps/polaris-configs/{configName}/envs/{envName}/weight [put]
func (h *Handler) PutEnvWeight(c *gin.Context) {
	var uriInput serializer.AppConfigEnvNameURIInput
	var jsonInput serializer.PutEnvWeightInput
	if err := ginutils.BindURIJSON(c, &uriInput, &jsonInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	config, err := h.registry.PolarisConfigStore.Get(ctx, app.ID, uriInput.ConfigName)
	if err != nil {
		if errors.Is(err, polaris.ErrConfigNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeNotFound,
				"polaris config(%s) not found in app(%s)",
				uriInput.ConfigName, uriInput.AppID,
			))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get polaris config"))
		return
	}

	// 权重更新仅允许在已部署的环境或配置作用域内的环境中进行
	// 其中已部署的环境将自动同步到集群，没部署的则存入配置，下发时使用
	if !config.IsAvailableInEnv(uriInput.EnvName) &&
		!config.GetEnvState(uriInput.EnvName).IsDeployed() {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(
			bkerrs.ErrCodeInvalidRequest,
			"env(%s) is not in scope of polaris config(%s) and has not been deployed",
			uriInput.EnvName, uriInput.ConfigName,
		))
		return
	}

	service := h.polarisConfigService()
	updatedConfig, err := service.UpdateEnvWeight(ctx, app, config, uriInput.EnvName, *jsonInput.Weight)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update env weight"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributePolaris),
		audit.WithDataBefore(config),
		audit.WithDataAfter(updatedConfig),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)

	ginutils.OK(c, new(serializer.PutEnvWeightOutput).FromModel(updatedConfig))
}
