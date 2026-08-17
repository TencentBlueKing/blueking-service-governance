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

// Package handler contains Gin handlers for dependency service APIs.
package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

const redisPlanName = "default"

// Handler handles Gin dependency service API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

func (h *Handler) serviceManager() *depservice.ServiceManager {
	return depservice.New(
		h.registry.DepSvcStore,
		h.registry.DepSvcInstStore,
		h.registry.DepSvcBindingStore,
		h.registry.EnvStore,
	)
}

// CreateRedisInstance 申请创建 Redis 依赖服务实例。
//
//	@ID			CreateRedisInstance
//	@Summary	创建 Redis 依赖服务实例
//	@Tags		depservice-redis
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string								true	"工作空间 ID"
//	@Param		body		body		serializer.CreateRedisInstanceInput	true	"请求体"
//	@Success	201			{object}	serializer.CreateRedisInstanceOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/deps/redis [post]
func (h *Handler) CreateRedisInstance(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var jsonInput serializer.CreateRedisInstanceInput
	if err := ginutils.BindURIJSON(c, &uriInput, &jsonInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	createParams := jsonInput.ToCreateParams()
	if err := createParams.Validate(); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "validate redis create params"))
		return
	}

	instID, err := h.serviceManager().CreateServiceInstance(ctx, &depservice.CreateServiceInstanceParams{
		Name:        jsonInput.Name,
		ServiceName: provider.ServiceNameRedis,
		PlanName:    redisPlanName,
		ScopeType:   model.ScopeType(jsonInput.ScopeType),
		ScopeValue:  jsonInput.ScopeValue,
		WorkspaceID: uriInput.WorkspaceID,
		Description: jsonInput.Description,
		Operator:    auth.MustGetUser(ctx).ID,
		Params:      createParams,
	})
	if err != nil {
		bkerrs.AbortWithErr(c, mapManagerErr(err, "create redis instance"))
		return
	}

	ginutils.Created(c, serializer.CreateRedisInstanceOutput{
		Data: serializer.CreateRedisInstanceOutputObj{
			ID:     instID.Hex(),
			Status: string(model.ProvisioningStatus),
		},
	})
}

// ListRedisInstances 查询 workspace 下 Redis 实例列表。
//
//	@ID			ListRedisInstances
//	@Summary	查询 Redis 依赖服务实例列表
//	@Tags		depservice-redis
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		status		query		string	false	"实例状态"
//	@Param		scopeType	query		string	false	"作用域类型"
//	@Success	200			{object}	serializer.ListRedisInstancesOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/deps/redis [get]
func (h *Handler) ListRedisInstances(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var queryInput serializer.ListRedisInstancesQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	mgr := h.serviceManager()
	insts, err := mgr.ListServiceInstances(ctx, &depservice.ListServiceInstancesParams{
		WorkspaceID: uriInput.WorkspaceID,
		ServiceName: provider.ServiceNameRedis,
		Status:      model.InstanceStatus(queryInput.Status),
		ScopeType:   model.ScopeType(queryInput.ScopeType),
	})
	if err != nil {
		bkerrs.AbortWithErr(c, mapManagerErr(err, "list redis instances"))
		return
	}

	usedApps, err := h.usedAppIDsByInstance(ctx, &model.BindingQueryOptions{
		WorkspaceID: uriInput.WorkspaceID,
		ServiceName: provider.ServiceNameRedis,
	})
	if err != nil {
		bkerrs.AbortWithErr(c, mapManagerErr(err, "list redis bindings"))
		return
	}

	ginutils.OK(c, serializer.ListRedisInstancesOutput{
		Data: lo.Map(insts, func(inst *model.ServiceInstance, _ int) *serializer.RedisInstanceOutputObj {
			return new(serializer.RedisInstanceOutputObj).FromModel(inst).WithUsedAppIDs(usedApps[inst.ID])
		}),
	})
}

// GetRedisInstance 查询 Redis 实例详情。
//
//	@ID			GetRedisInstance
//	@Summary	查询 Redis 依赖服务实例详情
//	@Tags		depservice-redis
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		instanceID	path		string	true	"实例 ID"
//	@Success	200			{object}	serializer.GetRedisInstanceOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/deps/redis/{instanceID} [get]
func (h *Handler) GetRedisInstance(c *gin.Context) {
	var uriInput serializer.WorkspaceInstanceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	inst, err := h.loadRedisInstance(ctx, uriInput.WorkspaceID, uriInput.InstanceID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	usedApps, err := h.usedAppIDsByInstance(ctx, &model.BindingQueryOptions{InstanceID: inst.ID})
	if err != nil {
		bkerrs.AbortWithErr(c, mapManagerErr(err, "list redis bindings"))
		return
	}

	ginutils.OK(c, serializer.GetRedisInstanceOutput{
		Data: new(serializer.RedisInstanceOutputObj).FromModel(inst).WithUsedAppIDs(usedApps[inst.ID]),
	})
}

// DeleteRedisInstance 删除 Redis 实例。
//
//	@ID			DeleteRedisInstance
//	@Summary	删除 Redis 依赖服务实例
//	@Tags		depservice-redis
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		instanceID	path		string	true	"实例 ID"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/deps/redis/{instanceID} [delete]
func (h *Handler) DeleteRedisInstance(c *gin.Context) {
	var uriInput serializer.WorkspaceInstanceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	inst, err := h.loadRedisInstance(ctx, uriInput.WorkspaceID, uriInput.InstanceID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if err = h.serviceManager().DeleteServiceInstance(ctx, inst.ID); err != nil {
		bkerrs.AbortWithErr(c, mapManagerErr(err, "delete redis instance"))
		return
	}

	ginutils.OK(c, serializer.EmptyOutput{})
}

func (h *Handler) loadRedisInstance(
	ctx context.Context,
	workspaceID, instanceID string,
) (*model.ServiceInstance, error) {
	objID, err := bson.ObjectIDFromHex(instanceID)
	if err != nil {
		return nil, bkerrs.New(bkerrs.ErrCodeInvalidRequest, "invalid instanceID")
	}

	inst, err := h.serviceManager().GetServiceInstance(ctx, objID)
	if err != nil {
		return nil, mapManagerErr(err, "get redis instance")
	}
	if inst.WorkspaceID != workspaceID || inst.ServiceName != provider.ServiceNameRedis {
		return nil, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "redis instance %s not found", instanceID)
	}
	return inst, nil
}

func (h *Handler) usedAppIDsByInstance(
	ctx context.Context,
	opts *model.BindingQueryOptions,
) (map[bson.ObjectID][]string, error) {
	bindings, err := h.registry.DepSvcBindingStore.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make(map[bson.ObjectID][]string)
	for _, binding := range bindings {
		for _, instID := range binding.InstanceIDs {
			if !lo.Contains(result[instID], binding.AppID) {
				result[instID] = append(result[instID], binding.AppID)
			}
		}
	}
	return result, nil
}

func mapManagerErr(err error, message string) error {
	if err == nil {
		return nil
	}
	if model.AsNotFoundError(err) {
		return bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, message)
	}
	if errors.Is(err, model.ErrBindingNameExists) || errors.Is(err, model.ErrInstanceNameExists) {
		return bkerrs.New(bkerrs.ErrCodeAlreadyExists, err.Error())
	}
	if errors.Is(err, depservice.ErrInvalidArgument) {
		return bkerrs.New(bkerrs.ErrCodeInvalidRequest, err.Error())
	}
	return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, message)
}
