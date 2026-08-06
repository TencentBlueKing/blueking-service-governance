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

// Package handler contains Gin handlers for port-pool APIs.
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/portpool"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/portpool/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// Handler handles Gin port-pool API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// ListPortPools 获取端口池列表。
//
//	@ID			ListPortPools
//	@Summary	获取端口池列表
//	@Tags		port-pool
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		envID	path		string	true	"环境 ID"
//	@Success	200		{object}	serializer.ListPortPoolsOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/envs/{envID}/port-pools [get]
func (h *Handler) ListPortPools(c *gin.Context) {
	var uriInput serializer.EnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	env, err := perm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	configs, err := portpool.NewPortPoolService().ListByEnv(ctx, env)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list portpool configs"))
		return
	}

	outputList := lo.Map(configs, func(config *portpool.PortPoolConfig, _ int) *serializer.PortPoolConfigOutputObj {
		return new(serializer.PortPoolConfigOutputObj).FromModel(*config)
	})

	ginutils.OK(c, serializer.ListPortPoolsOutput{Data: outputList})
}

// CreatePortPool 创建端口池。
//
//	@ID			CreatePortPool
//	@Summary	创建端口池
//	@Tags		port-pool
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		envID	path		string						true	"环境 ID"
//	@Param		body	body		serializer.CreatePortPoolInput	true	"请求体"
//	@Success	200		{object}	nil
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/envs/{envID}/port-pools [post]
func (h *Handler) CreatePortPool(c *gin.Context) {
	var uriInput serializer.EnvURIInput
	var jsonInput serializer.CreatePortPoolInput
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

	poolItems := lo.Map(jsonInput.PoolItems, func(item serializer.PortPoolItemInput, _ int) portpool.PoolItem {
		return item.ToModel()
	})

	config := &portpool.PortPoolConfig{
		Name:        jsonInput.Name,
		PoolItems:   poolItems,
		EnvID:       env.ID.Hex(),
		WorkspaceID: env.WorkspaceID,
		EnvName:     env.Name,
	}

	if err = portpool.NewPortPoolService().Create(ctx, config, env); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create portpool config"))
		return
	}

	go audit.AddOperationRecordAsync(
		c.Request.Context(),
		audit.OperationTypeCreate,
		audit.ResourceTypePortPool,
		jsonInput.Name,
		audit.WithDataAfter(config),
		audit.WithWorkspaceID(env.WorkspaceID),
		audit.WithEnvName(env.Name),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// UpdatePortPool 更新端口池。
//
//	@ID			UpdatePortPool
//	@Summary	更新端口池
//	@Tags		port-pool
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		envID	path		string						true	"环境 ID"
//	@Param		name	path		string						true	"端口池名称"
//	@Param		body	body		serializer.UpdatePortPoolInput	true	"请求体"
//	@Success	200		{object}	nil
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/envs/{envID}/port-pools/{name} [put]
func (h *Handler) UpdatePortPool(c *gin.Context) {
	var uriInput serializer.EnvNameURIInput
	var jsonInput serializer.UpdatePortPoolInput
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

	poolItems := lo.Map(jsonInput.PoolItems, func(item serializer.PortPoolItemInput, _ int) portpool.PoolItem {
		return item.ToModel()
	})

	result, err := portpool.NewPortPoolService().Update(ctx, env, uriInput.Name, poolItems)
	if err != nil {
		if errors.Is(err, portpool.ErrConfigNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeNotFound,
				"portpool config(%s) not found in env(%s)",
				uriInput.Name, uriInput.EnvID,
			))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update portpool config"))
		return
	}

	go audit.AddOperationRecordAsync(
		c.Request.Context(),
		audit.OperationTypeUpdate,
		audit.ResourceTypePortPool,
		uriInput.Name,
		audit.WithDataBefore(result.Before),
		audit.WithDataAfter(result.After),
		audit.WithWorkspaceID(env.WorkspaceID),
		audit.WithEnvName(env.Name),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// DeletePortPool 删除端口池。
//
//	@ID			DeletePortPool
//	@Summary	删除端口池
//	@Tags		port-pool
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		envID	path		string	true	"环境 ID"
//	@Param		name	path		string	true	"端口池名称"
//	@Success	200		{object}	nil
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/envs/{envID}/port-pools/{name} [delete]
func (h *Handler) DeletePortPool(c *gin.Context) {
	var uriInput serializer.EnvNameURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	env, err := perm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ppSvc := portpool.NewPortPoolService()
	existingConfig, err := ppSvc.Get(ctx, env, uriInput.Name)
	if err != nil {
		if errors.Is(err, portpool.ErrConfigNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeNotFound,
				"portpool config(%s) not found in env(%s)",
				uriInput.Name, uriInput.EnvID,
			))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get portpool config"))
		return
	}

	if err = ppSvc.Delete(ctx, env, uriInput.Name); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete portpool config"))
		return
	}

	go audit.AddOperationRecordAsync(
		c.Request.Context(),
		audit.OperationTypeDelete,
		audit.ResourceTypePortPool,
		uriInput.Name,
		audit.WithDataBefore(existingConfig),
		audit.WithWorkspaceID(env.WorkspaceID),
		audit.WithEnvName(env.Name),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}
