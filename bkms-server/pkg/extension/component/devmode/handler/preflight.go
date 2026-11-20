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

package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/bcstoken"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// Handler DevMode handler
type Handler struct {
	registry *storereg.Registry
}

// New 创建 DevMode handler
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// DevModePublishPreflight 开发模式 Publish 预检接口。
// 校验应用、环境、devmode 配置，并返回 CLI publish 所需的全部上下文信息。
//
//	@ID			DevModePublishPreflight
//	@Summary	开发模式 Publish 预检
//	@Tags		devmode
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string					true	"应用 ID"
//	@Param		envName	path		string					true	"环境名称"
//	@Param		body	body		slz.PreflightBodyInput	true	"预检请求体"
//	@Success	200		{object}	slz.PreflightOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/devmode/{appID}/envs/{envName}/preflight [post]
func (h *Handler) DevModePublishPreflight(c *gin.Context) {
	var uriInput slz.PreflightURIInput
	var bodyInput slz.PreflightBodyInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err := ginutils.BindJSON(c, &bodyInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()

	// 校验环境配置
	app, env, err := h.validateEnv(ctx, uriInput.AppID, uriInput.EnvName)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 获取 devmode 生效配置
	devModeSpec, err := h.getEffectiveDevMode(ctx, app, env.Name)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 解析实例列表
	instanceIDs, err := h.resolveInstanceIDs(
		ctx, uriInput.AppID, uriInput.EnvName, bodyInput.InstanceIDs, bodyInput.PublishAll,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 获取 BCS 项目 Code，用于无 Token 时引导用户创建
	bcsProjectCode, err := h.getBCSProjectCode(ctx, env.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 获取用户 Token
	token, err := h.fetchActiveToken(ctx, bcsProjectCode)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 组装响应
	ginutils.OK(c, buildPreflightOutput(env, devModeSpec, token, instanceIDs))
}

// validateEnv 校验基础设施配置、应用、环境和集群信息。
// 返回应用和环境信息，校验失败时返回 bkerrs 错误。
func (h *Handler) validateEnv(
	ctx context.Context, appID, envName string,
) (*bkmsapp.Application, *envmodel.Environment, error) {
	// 检查集群 API 地址是否配置
	if err := checkClusterAPIConfigured(); err != nil {
		return nil, nil, err
	}

	// 校验 app 和 env（包含权限检查）
	app, env, err := perm.ValidateAppEnvByName(ctx, h.registry, appID, envName, perm.TypeView)
	if err != nil {
		return nil, nil, err
	}

	if err = checkEnvSupported(env); err != nil {
		return nil, nil, err
	}
	return app, env, nil
}

// checkClusterAPIConfigured 检查集群 API 地址是否配置
func checkClusterAPIConfigured() error {
	if config.G.BCS.BaseUrl == "" {
		return bkerrs.New(
			bkerrs.ErrCodeInvalidRequest,
			"publish is not supported: cluster API base URL is not configured",
		)
	}
	return nil
}

// checkEnvSupported 校验环境是否支持 devmode 操作：非 production 且集群信息完整。
func checkEnvSupported(env *envmodel.Environment) error {
	// 拒绝 production 环境
	if bkmsenv.IsProductionType(bkmsenv.Type(env.Type)) {
		return bkerrs.Errorf(
			bkerrs.ErrCodeInvalidRequest,
			"environment %s is a production environment, publish is not allowed", env.Name,
		)
	}

	// 校验集群信息完整性
	if env.Cluster.ClusterID == "" || env.Cluster.Namespace == "" {
		return bkerrs.New(
			bkerrs.ErrCodeInvalidRequest,
			"publish is not supported: cluster configuration is incomplete (clusterID or namespace is empty)",
		)
	}
	return nil
}

// getEffectiveDevMode 获取并校验 devmode 生效配置。
// 返回 devmode 配置，未启用或获取失败时返回 bkerrs 错误。
func (h *Handler) getEffectiveDevMode(
	ctx context.Context, app *bkmsapp.Application, envName string,
) (*appspec.DevModeSpec, error) {
	devModeSpec, err := appspec.GetEnvEffectiveSection(
		ctx, h.registry.DB,
		app.ID, envName, appspec.DevModeSection,
	)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "failed to get devmode configuration")
	}

	if devModeSpec == nil || devModeSpec.Enabled == nil || !*devModeSpec.Enabled {
		return nil, bkerrs.Errorf(
			bkerrs.ErrCodeInvalidRequest,
			"devmode is not enabled for app %s in environment %s", app.ID, envName,
		)
	}
	props, err := devmode.RequireProps(app)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "dev mode")
	}
	devModeSpec.SetPaths(props.WorkPath, props.MountPath)

	return devModeSpec, nil
}

// getBCSProjectCode 从工作空间获取 BCS 项目 Code。
func (h *Handler) getBCSProjectCode(ctx context.Context, workspaceID string) (string, error) {
	ws, err := store.Of(h.registry.DB, workspace.Meta).Get(ctx, workspace.ByWorkspaceID(workspaceID))
	if err != nil {
		return "", bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "get workspace %s", workspaceID)
	}
	return ws.BkSystems.BkBCSProjectCode, nil
}

// fetchActiveToken 获取当前用户的有效 BCS Token
func (h *Handler) fetchActiveToken(ctx context.Context, bcsProjectCode string) (string, error) {
	token, err := bcstoken.FetchActiveUserToken(ctx, bcsProjectCode)
	if err != nil {
		if errors.Is(err, bcstoken.ErrNoActiveUserToken) {
			return "", bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "fetch active BCS user token")
		}
		return "", bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "fetch active BCS user token")
	}
	return token, nil
}

// resolveInstanceIDs 根据请求参数解析目标实例 ID 列表。
// all=true 时返回所有 Running 状态的 Pod 名称；否则校验指定的 instanceIDs 是否存在。
func (h *Handler) resolveInstanceIDs(
	ctx context.Context, appID, envName string, instanceIDs []string, all bool,
) ([]string, error) {
	// 获取最新部署记录
	record, err := appmodeldeploy.GetLatest(ctx, h.registry.DB, appID, envName, "")
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "no deploy record found for this environment")
		}
		return nil, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "get latest deploy record of app %s env %s", appID, envName,
		)
	}

	// 通过 LabelSelector 拉取匹配的 Pod 列表
	podClient := k8sclient.NewPodClient(cluster.NewConfig(record.ClusterID))
	labelSelector := labels.SelectorFromSet(record.LabelSelector).String()
	pods, err := podClient.List(ctx, record.Namespace, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError,
			"failed to list pods in namespace %s", record.Namespace,
		)
	}

	if all {
		return h.getAllRunningInstanceIDs(pods.Items)
	}
	return h.validateSpecifiedInstanceIDs(pods.Items, instanceIDs)
}

// getAllRunningInstanceIDs 从 Pod 列表中筛选所有 Running 状态的实例 ID。
func (h *Handler) getAllRunningInstanceIDs(
	items []unstructured.Unstructured,
) ([]string, error) {
	runningIDs := lo.FilterMap(items, func(pod unstructured.Unstructured, _ int) (string, bool) {
		status := mapx.GetStr(pod.Object, "status.phase")
		name := mapx.GetStr(pod.Object, "metadata.name")
		return name, status == "Running" && name != ""
	})
	if len(runningIDs) == 0 {
		return nil, bkerrs.New(bkerrs.ErrCodeNotFound, "no running instances found")
	}
	return runningIDs, nil
}

// validateSpecifiedInstanceIDs 校验指定的实例 ID 是否存在于 Pod 列表中。
func (h *Handler) validateSpecifiedInstanceIDs(
	items []unstructured.Unstructured, instanceIDs []string,
) ([]string, error) {
	if len(instanceIDs) == 0 {
		return nil, bkerrs.New(
			bkerrs.ErrCodeInvalidRequest,
			"instanceIDs is required when not targeting all instances",
		)
	}

	podNameSet := make(map[string]struct{}, len(items))
	for _, pod := range items {
		name := mapx.GetStr(pod.Object, "metadata.name")
		if name != "" {
			podNameSet[name] = struct{}{}
		}
	}

	var notFound []string
	for _, id := range instanceIDs {
		if _, ok := podNameSet[id]; !ok {
			notFound = append(notFound, id)
		}
	}
	if len(notFound) > 0 {
		return nil, bkerrs.Errorf(
			bkerrs.ErrCodeNotFound,
			"instances not found: %s", fmt.Sprintf("%v", notFound),
		)
	}

	return instanceIDs, nil
}

// buildClusterAddress 构造集群 API 完整地址，如 {baseUrl}/clusters/{clusterID}/
func buildClusterAddress(clusterID string) string {
	return fmt.Sprintf("%s/clusters/%s/", strings.TrimRight(config.G.BCS.BaseUrl, "/"), clusterID)
}

// buildPreflightOutput 组装 preflight 响应数据。
func buildPreflightOutput(
	env *envmodel.Environment,
	devModeSpec *appspec.DevModeSpec,
	token string,
	instanceIDs []string,
) *slz.PreflightOutput {
	address := buildClusterAddress(env.Cluster.ClusterID)

	var workPath, mountPath string
	if devModeSpec.WorkPath != nil {
		workPath = *devModeSpec.WorkPath
	}
	if devModeSpec.MountPath != nil {
		mountPath = *devModeSpec.MountPath
	}

	return &slz.PreflightOutput{
		Data: &slz.PreflightData{
			Token:       token,
			Address:     address,
			Namespace:   env.Cluster.Namespace,
			InstanceIDs: instanceIDs,
			DevMode: &slz.PreflightDevMode{
				WorkPath:  workPath,
				MountPath: mountPath,
			},
		},
	}
}
