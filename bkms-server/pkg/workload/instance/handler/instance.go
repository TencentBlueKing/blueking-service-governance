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

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
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

// ListAppInstances 获取应用实例列表。
//
//	@ID			ListAppInstances
//	@Summary	获取应用实例列表
//	@Tags		instance
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID			path		string	true	"应用 ID"
//	@Param		envName			path		string	true	"部署环境名称"
//	@Param		trafficLaneName	query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Param		page			query		int		true	"页码，从 1 开始"
//	@Param		pageSize		query		int		true	"每页数量"
//	@Success	200				{object}	serializer.ListAppInstancesOutput
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/instances [get]
func (h *Handler) ListAppInstances(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var queryInput serializer.ListAppInstancesQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	// 校验 App 查看权限
	app, err := ginperm.ValidateAppByID(ctx, h.registry, uriInput.AppID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	// 目前只支持查看 AppModel 类型应用实例
	if !bkmsapp.IsAppModelType(app.Type) {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeInvalidArgument, "invalid app type: %s", app.Type))
		return
	}

	// 获取应用部署记录
	record, err := h.registry.AppModelDeployRecordStore.GetLatest(
		ctx, app.ID, uriInput.EnvName, queryInput.TrafficLaneName,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "deploy record not found"))
		return
	}

	// 获取应用实例（Pod）列表
	client := k8sclient.NewPodClient(cluster.NewConfig(record.ClusterID))
	// 根据命名空间 + 标签选择器获取 Pod 列表
	labelSelector := labels.SelectorFromSet(record.LabelSelector).String()
	pods, err := client.List(ctx, record.Namespace, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError,
			"list namespace %s labelsSelector [%s] pods", record.Namespace, labelSelector,
		))
		return
	}

	// 计算总数
	total := int64(len(pods.Items))

	// 计算分页参数（使用 max 确保最小值）
	page := max(queryInput.Page, int64(1))
	pageSize := max(queryInput.PageSize, int64(1))

	// 计算起始和结束索引（使用 min 确保不越界）
	startIdx := min((page-1)*pageSize, total)
	endIdx := min(startIdx+pageSize, total)

	// 只解析当前页的数据
	// TODO 未来支持按状态 / IP 等过滤，应该还是需要全量数据处理？
	appInstances := make([]*serializer.AppInstanceOutputObj, 0, endIdx-startIdx)
	for i := startIdx; i < endIdx; i++ {
		p := pods.Items[i]
		instance, pErr := new(serializer.AppInstanceOutputObj).FromPodManifest(p.Object, record.ID.Hex())
		if pErr != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrapf(
				pErr, bkerrs.ErrCodeInternalServerError, "parse pod %s", mapx.GetStr(p.Object, "metadata.name"),
			))
			return
		}
		appInstances = append(appInstances, instance)
	}

	mgr := polaris.NewPolarisPlatformManager(
		h.registry.DepSvcStore,
		h.registry.DepSvcInstStore,
		h.registry.PolarisConfigStore,
	)
	svcInstances, err := mgr.ListPolarisServiceInstances(ctx, app.ID, uriInput.EnvName)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list polaris service instances"))
		return
	}
	serializer.MergePolarisInfoToAppInstances(appInstances, svcInstances)

	ginutils.OK(c, serializer.ListAppInstancesOutput{
		Data: &serializer.PaginatedAppInstancesOutputObj{
			Count:   total,
			Results: appInstances,
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
	if err = deployer.BatchDeleteInstances(ctx, uriInput.EnvName, input.TrafficLaneName, input.InstanceIDs); err != nil {
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
