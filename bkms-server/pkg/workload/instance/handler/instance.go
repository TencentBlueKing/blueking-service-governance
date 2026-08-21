/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

// Package handler contains Gin handlers for app instance APIs.
package handler

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
	instanceroute "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

var _ instanceroute.Handler = (*Handler)(nil)

// Handler handles Gin app instance API requests.
type Handler struct {
	registry *storereg.Registry
}

// New 创建应用实例 Gin Handler。
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// ListAppInstances 按最新部署记录的 LabelSelector 列出匹配 Pod 并投影为实例
// 本函数只做编排：绑定参数、鉴权取部署记录、拉 Pod、投影、合并北极星
//
// 查询模式互斥：all=true 且不带 page/pageSize 为全量；未传 all 时 page/pageSize 必填走分页
// 全量时单个 Pod 投影失败则跳过并写入 skipped，count 为成功投影数（不含跳过项）
// 分页时当前页解析失败整次 500（与改造前一致，供 CLI）；count 为 LabelSelector 匹配 Pod 总数
// 北极星拉取失败不阻塞 Pod 输出，降级为空 polarisInfos，与未注册北极星同形
// 成功响应带 resourceVersion，供 Watch 从该位点续传
//
//	@ID			ListAppInstances
//	@Summary	获取应用实例列表
//	@Description	北极星拉取失败不阻塞 Pod 输出：polarisInfos 为空数组，与未注册北极星同形，其余字段照常返回。
//	@Tags		instance
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID			path		string	true	"应用 ID"
//	@Param		envName			path		string	true	"部署环境名称"
//	@Param		trafficLaneName	query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Param		all				query		bool	false	"为 true 时一次返回全量实例；禁止同时带 page 或 pageSize"
//	@Param		page			query		int		false	"页码，取值 1-10000；分页模式必填，all=true 时禁止出现"
//	@Param		pageSize		query		int		false	"每页数量；分页模式必填，all=true 时禁止出现"
//	@Success	200				{object}	serializer.ListAppInstancesOutput
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/instances [get]
func (h *Handler) ListAppInstances(c *gin.Context) {
	// 绑定路径与查询参数；全量与分页互斥，分页时 page/pageSize 必填
	uriInput, queryInput, err := h.bindListAppInstancesQuery(c)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	// 校验 App 查看权限与 AppModel 类型，并取该环境/泳道最新部署记录
	app, record, err := h.validateAppAndGetDeployRecord(ctx, uriInput, queryInput.TrafficLaneName)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 按部署记录的命名空间 + LabelSelector 拉取匹配 Pod
	items, resourceVersion, err := h.listMatchingAppInstancePods(ctx, record)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 将窗口内 Pod 投影为实例；全量跳过失败项，分页解析失败则整次 500
	results, skipped, count, err := h.projectListedAppInstances(queryInput, items, record.ID.Hex())
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 合并该应用环境的北极星实例；拉取失败只降级这一列，不影响 Pod 列表返回
	h.attachPolarisToListedAppInstances(ctx, app.ID, uriInput.EnvName, results)

	// 成功响应带上首次 List 的 resourceVersion，Watch 必须原样带回
	ginutils.OK(c, serializer.ListAppInstancesOutput{
		Data: &serializer.PaginatedAppInstancesOutputObj{
			Count:           count,
			Results:         results,
			SkippedCount:    int64(len(skipped)),
			Skipped:         skipped,
			ResourceVersion: resourceVersion,
		},
	})
}

// UpdateAppInstances 更新应用实例（支持单/多/全量实例更新）。
//
//	@ID			UpdateAppInstances
//	@Summary	更新应用实例（支持单/多/全量实例更新）
//	@Tags		instance
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string								true	"应用 ID"
//	@Param		envName	path		string								true	"部署环境名称"
//	@Param		body	body		serializer.UpdateAppInstancesInput	true	"更新实例请求"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/instances [put]
func (h *Handler) UpdateAppInstances(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var input serializer.UpdateAppInstancesInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := h.validateEditableAppModel(ctx, uriInput.AppID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	// 操作实例（Pod）都需要有部署当前 Env 的权限
	if err = perm.NewManager().HasDeployEnvPerm(ctx, app.WorkspaceID, uriInput.EnvName); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNoPermission, "check deploy env perm"))
		return
	}

	// 获取应用部署器，执行灰度更新操作
	deployer := h.newDeployer(app)
	if err = deployer.UpdateInstances(
		ctx,
		uriInput.EnvName,
		input.TrafficLaneName,
		input.ImageTag,
		input.UpdateStrategy,
		input.InstanceIDs,
	); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "update app %s instances", app.ID))
		return
	}

	// 实例更新操作记录
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		audit.OperationTypeGray,
		audit.ResourceTypeInstance,
		strings.Join(input.InstanceIDs, ","),
		audit.WithAttribute(audit.AttributeInstance),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
		audit.WithEnvName(uriInput.EnvName),
		audit.WithDataAfter(map[string]any{"imageTag": input.ImageTag}),
	)
	ginutils.OK(c, serializer.EmptyOutput{})
}

// ScaleAppInstances 扩缩容应用实例数量。
//
//	@ID			ScaleAppInstances
//	@Summary	扩缩容应用实例数量
//	@Tags		instance
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string								true	"应用 ID"
//	@Param		envName	path		string								true	"部署环境名称"
//	@Param		body	body		serializer.ScaleAppInstancesInput	true	"扩缩容请求"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/instances/operations/scale [put]
func (h *Handler) ScaleAppInstances(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var input serializer.ScaleAppInstancesInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := h.validateEditableAppModel(ctx, uriInput.AppID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	// 操作实例（Pod）都需要有部署当前 Env 的权限
	if err = perm.NewManager().HasDeployEnvPerm(ctx, app.WorkspaceID, uriInput.EnvName); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNoPermission, "check deploy env perm"))
		return
	}

	// 获取应用当前环境生效的部署规格
	spec, err := appspec.GetEnvEffective(
		ctx,
		h.registry.AppSpecStore,
		h.registry.AppModelStore,
		app.ID,
		uriInput.EnvName,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get env effective app spec"))
		return
	}
	oldReplicas := *spec.Resources.Replicas

	// 获取应用部署器，执行实例扩缩容操作
	deployer := h.newDeployer(app)
	if err = deployer.Scale(ctx, uriInput.EnvName, input.TrafficLaneName, input.TargetReplicas); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "scale app %s instances", app.ID))
		return
	}

	// 扩缩容操作记录
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		audit.OperationTypeScale,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributeReplicas),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
		audit.WithEnvName(uriInput.EnvName),
		audit.WithDataBefore(oldReplicas),
		audit.WithDataAfter(input.TargetReplicas),
	)
	metrics.InstanceScaled(oldReplicas, input.TargetReplicas)
	ginutils.OK(c, serializer.EmptyOutput{})
}

// BatchDeleteAppInstances 批量删除指定的应用实例，同时缩容副本数量。
//
//	@ID			BatchDeleteAppInstances
//	@Summary	批量删除指定的应用实例，同时缩容副本数量
//	@Tags		instance
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string										true	"应用 ID"
//	@Param		envName	path		string										true	"部署环境名称"
//	@Param		body	body		serializer.BatchDeleteAppInstancesInput	true	"批量删除实例请求"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/instances/operations/batch_delete [post]
func (h *Handler) BatchDeleteAppInstances(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var input serializer.BatchDeleteAppInstancesInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := h.validateEditableAppModel(ctx, uriInput.AppID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	// 操作实例（Pod）都需要有部署当前 Env 的权限
	if err = perm.NewManager().HasDeployEnvPerm(ctx, app.WorkspaceID, uriInput.EnvName); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNoPermission, "check deploy env perm"))
		return
	}

	// 获取应用部署器，执行删除实例操作
	deployer := h.newDeployer(app)
	err = deployer.BatchDeleteInstances(ctx, uriInput.EnvName, input.TrafficLaneName, input.InstanceIDs)
	if err != nil {
		bkerrs.AbortWithErr(
			c, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "batch delete app %s instances", app.ID),
		)
		return
	}

	// 删除实例操作记录
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		audit.OperationTypeDelete,
		audit.ResourceTypeInstance,
		strings.Join(input.InstanceIDs, ","),
		audit.WithAttribute(audit.AttributeInstance),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
		audit.WithEnvName(uriInput.EnvName),
		audit.WithDataBefore(input.InstanceIDs),
	)
	ginutils.OK(c, serializer.EmptyOutput{})
}

// UpdateAppInstancePolaris 更新应用实例的北极星注解（权重 / 隔离）。
//
//	@ID			UpdateAppInstancePolaris
//	@Summary	更新应用实例的北极星注解（权重 / 隔离）
//	@Tags		instance
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string										true	"应用 ID"
//	@Param		envName	path		string										true	"部署环境名称"
//	@Param		body	body		serializer.UpdateAppInstancePolarisInput	true	"更新北极星注解请求"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/instances/operations/polaris [put]
func (h *Handler) UpdateAppInstancePolaris(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var input serializer.UpdateAppInstancePolarisInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := h.validateEditableAppModel(ctx, uriInput.AppID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	// 操作实例（Pod）都需要有部署当前 Env 的权限
	if err = perm.NewManager().HasDeployEnvPerm(ctx, app.WorkspaceID, uriInput.EnvName); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNoPermission, "check deploy env perm"))
		return
	}
	// weight 和 isolate 至少要设置一个
	if input.Weight == nil && input.Isolate == nil {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeInvalidArgument, "weight and isolate cannot both be empty"))
		return
	}

	deployer := h.newDeployer(app)
	if err = deployer.UpdateInstancePolaris(
		ctx,
		uriInput.EnvName,
		input.TrafficLaneName,
		input.InstanceIDs,
		input.Weight,
		input.Isolate,
	); err != nil {
		bkerrs.AbortWithErr(
			c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update instance polaris annotations"),
		)
		return
	}

	auditData := map[string]any{}
	if input.Weight != nil {
		auditData["weight"] = *input.Weight
	}
	if input.Isolate != nil {
		auditData["isolate"] = *input.Isolate
	}
	// 操作记录
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		audit.OperationTypeUpdate,
		audit.ResourceTypeInstance,
		strings.Join(input.InstanceIDs, ","),
		audit.WithAttribute(audit.AttributeInstance),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
		audit.WithEnvName(uriInput.EnvName),
		audit.WithDataAfter(auditData),
	)
	ginutils.OK(c, serializer.EmptyOutput{})
}

// validateEditableAppModel 校验 App 编辑权限并确认应用类型为 AppModel。
func (h *Handler) validateEditableAppModel(ctx context.Context, appID string) (*bkmsapp.Application, error) {
	// 校验 App 编辑权限
	app, err := ginperm.ValidateAppByID(ctx, h.registry, appID, ginperm.TypeEdit)
	if err != nil {
		return nil, err
	}
	// 目前只支持操作 AppModel 类型应用实例
	if !bkmsapp.IsAppModelType(app.Type) {
		return nil, bkerrs.Errorf(bkerrs.ErrCodeInvalidArgument, "invalid app type: %s", app.Type)
	}
	return app, nil
}

// newDeployer 创建 AppModel 应用部署器。
func (h *Handler) newDeployer(app *bkmsapp.Application) *appmodeldeploy.Deployer {
	return appmodeldeploy.NewDeployer(
		h.registry.AppModelDeployRecordStore,
		h.registry.AppModelDeployResourceSnapshotStore,
		h.registry.BuildAutoDeployRecordStore,
		h.registry.AppModelStore,
		workload.NewBuilderService(
			h.registry.ScopedEnvVarStore,
			h.registry.AppDepsVarReader,
			h.registry.PolarisVarReader,
			h.registry.WorkspaceCompsStore,
			h.registry.PolarisConfigStore,
			h.registry.BscpCfgStore,
			h.registry.AppModelStore,
			h.registry.AppSpecStore,
			h.registry.BuildConfigStore,
		),
		h.registry.AppSpecStore,
		h.registry.BuildConfigStore,
		h.registry.AppConfigFileStore,
		polaris.NewPolarisEnvStateManager(h.registry.PolarisConfigStore),
		app,
	)
}
