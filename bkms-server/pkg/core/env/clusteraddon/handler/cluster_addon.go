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

// Package handler contains Gin handlers for cluster-addon APIs.
package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	clusteraddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// Handler handles Gin cluster-addon API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

func (h *Handler) addonDefs() *clusteraddon.ClusterAddonDefStore {
	return store.Of(h.registry.DB, clusteraddon.ClusterAddonDefMeta)
}

// ListClusterAddons 查询可安装的集群插件列表。
//
//	@ID			ListClusterAddons
//	@Summary	查询可安装的集群插件列表
//	@Tags		cluster-addon
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		envID		path		string	true	"环境 ID"
//	@Param		namespace	query		string	false	"命名空间，默认为插件定义中的 defaultNamespace"
//	@Success	200			{object}	serializer.ListClusterAddonsOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/envs/{envID}/cluster-addons [get]
func (h *Handler) ListClusterAddons(c *gin.Context) {
	var uriInput serializer.EnvURIInput
	var queryInput serializer.ListClusterAddonsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	env, err := perm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if env.Cluster.ClusterID == "" {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeInvalidArgument,
			"environment %s has no cluster configured", uriInput.EnvID))
		return
	}

	h.respondAddonList(c, clusteraddon.TargetOfEnv(env), queryInput.Namespace)
}

// respondAddonList 查询目标集群上的插件列表并写入响应，环境维度与集群维度的列表接口共用
func (h *Handler) respondAddonList(c *gin.Context, target clusteraddon.ClusterTarget, namespace string) {
	ctx := c.Request.Context()

	addonDefs, err := clusteraddon.ListDefs(ctx, h.addonDefs())
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list cluster addon defs"))
		return
	}

	repoIndex, repoErr := clusteraddon.FetchRepoIndex()
	if repoErr != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(repoErr, bkerrs.ErrCodeInternalServerError, "fetch helm repo index"))
		return
	}

	addons := clusteraddon.BuildAddonInfoList(ctx, addonDefs, target, namespace, repoIndex)

	ginutils.OK(c, &serializer.ListClusterAddonsOutput{
		Addons: lo.Map(addons, func(addon *clusteraddon.ClusterAddonInfo, _ int) *serializer.ClusterAddonInfoOutput {
			return new(serializer.ClusterAddonInfoOutput).FromModel(*addon)
		}),
	})
}

// UpsertClusterAddon 部署/更新集群插件。
//
//	@ID			UpsertClusterAddon
//	@Summary	部署/更新集群插件
//	@Tags		cluster-addon
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		envID		path		string								true	"环境 ID"
//	@Param		addonName	path		string								true	"插件名称"
//	@Param		body		body		serializer.UpsertClusterAddonInput	true	"请求体"
//	@Success	201			{object}	nil
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/envs/{envID}/cluster-addons/{addonName} [post]
func (h *Handler) UpsertClusterAddon(c *gin.Context) {
	var uriInput serializer.EnvAddonURIInput
	var jsonInput serializer.UpsertClusterAddonInput
	if err := ginutils.BindURIJSON(c, &uriInput, &jsonInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	env, err := perm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if env.Cluster.ClusterID == "" {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeInvalidArgument,
			"environment %s has no cluster configured", uriInput.EnvID))
		return
	}

	if err = h.upsertAddon(ctx, clusteraddon.TargetOfEnv(env), uriInput.AddonName, jsonInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	go audit.AddOperationRecordAsync(
		c.Request.Context(),
		audit.OperationTypeDeploy,
		audit.ResourceTypeClusterAddon,
		uriInput.AddonName,
		audit.WithDataAfter(map[string]any{
			"chartVersion": jsonInput.ChartVersion,
			"values":       jsonInput.Values,
		}),
		audit.WithWorkspaceID(env.WorkspaceID),
		audit.WithEnvName(env.Name),
	)

	ginutils.JSON(c, 201, nil)
}

// upsertAddon 在目标集群上加锁安装或更新插件，环境维度与集群维度的安装接口共用，返回的错误已包装为 API 错误
func (h *Handler) upsertAddon(
	ctx context.Context,
	target clusteraddon.ClusterTarget,
	addonName string,
	input serializer.UpsertClusterAddonInput,
) error {
	addonDef, err := h.addonDefs().Get(ctx, clusteraddon.ByName(addonName))
	if err != nil {
		if errors.Is(err, clusteraddon.ErrClusterAddonDefNotFound) {
			return bkerrs.Errorf(bkerrs.ErrCodeNotFound, "cluster addon %s not found", addonName)
		}
		return bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "get cluster addon def %s", addonName)
	}

	namespace := addonDef.GetNamespace(input.Namespace)

	installLock := clusteraddon.NewInstallLock(target.ClusterID, namespace, addonName)
	if ok := installLock.Acquire(ctx); !ok {
		return bkerrs.New(bkerrs.ErrCodeAborted, "concurrent install/update conflict occurred")
	}
	defer installLock.Release(ctx)

	if err = clusteraddon.InstallOrUpgradeClusterAddon(
		ctx, addonDef, target, namespace, input.ChartVersion, input.Values,
	); err != nil {
		if errors.Is(err, clusteraddon.ErrAddonNotApplicable) {
			return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "deploy cluster addon")
		}
		return bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "deploy cluster addon %s", addonName)
	}
	return nil
}

// DeleteClusterAddon 卸载集群插件。
//
//	@ID			DeleteClusterAddon
//	@Summary	卸载集群插件
//	@Tags		cluster-addon
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		envID		path		string	true	"环境 ID"
//	@Param		addonName	path		string	true	"插件名称"
//	@Param		namespace	query		string	false	"命名空间，默认为插件定义中的 defaultNamespace"
//	@Success	200			{object}	serializer.DeleteClusterAddonOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/envs/{envID}/cluster-addons/{addonName} [delete]
func (h *Handler) DeleteClusterAddon(c *gin.Context) {
	var uriInput serializer.EnvAddonURIInput
	var queryInput serializer.DeleteClusterAddonQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	env, err := perm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	clusterID := env.Cluster.ClusterID
	if clusterID == "" {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeInvalidArgument,
			"environment %s has no cluster configured", uriInput.EnvID))
		return
	}

	addonDef, err := h.addonDefs().Get(
		ctx, clusteraddon.ByName(uriInput.AddonName),
	)
	if err != nil {
		if errors.Is(err, clusteraddon.ErrClusterAddonDefNotFound) {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Errorf(bkerrs.ErrCodeNotFound, "cluster addon %s not found", uriInput.AddonName),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError,
			"get cluster addon def %s", uriInput.AddonName))
		return
	}

	namespace := addonDef.GetNamespace(queryInput.Namespace)

	installLock := clusteraddon.NewInstallLock(clusterID, namespace, uriInput.AddonName)
	if ok := installLock.Acquire(ctx); !ok {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeAborted, "concurrent uninstall conflict occurred"))
		return
	}
	defer installLock.Release(ctx)

	if err = clusteraddon.UninstallClusterAddon(ctx, addonDef, clusterID, namespace); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError,
			"uninstall cluster addon %s", uriInput.AddonName))
		return
	}

	go audit.AddOperationRecordAsync(
		c.Request.Context(),
		audit.OperationTypeUninstall,
		audit.ResourceTypeClusterAddon,
		uriInput.AddonName,
		audit.WithWorkspaceID(env.WorkspaceID),
		audit.WithEnvName(env.Name),
	)

	ginutils.OK(c, serializer.DeleteClusterAddonOutput{
		Status:  string(helm.StatusUninstalled),
		Message: "addon uninstalled successfully",
	})
}
