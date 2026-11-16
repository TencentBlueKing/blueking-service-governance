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

// Package handler contains Gin handlers for platform workspace image registry APIs.
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/workspace/imageregistry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/workspace/imageregistry/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

var _ imageregistry.Handler = (*Handler)(nil)

// Handler handles Gin platform workspace image registry API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// GetPlatWorkspaceImageRegistry 查询空间当前生效的镜像仓库
//
//	@ID			GetPlatWorkspaceImageRegistry
//	@Summary	查询空间当前生效的镜像仓库
//	@Tags		platmgt
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.ImageRegistryResponse
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/plat-mgt/workspaces/{workspaceID}/image-registry [get]
func (h *Handler) GetPlatWorkspaceImageRegistry(c *gin.Context) {
	var path serializer.WorkspacePath
	if err := ginutils.BindURI(c, &path); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	reg, err := h.manager().Get(c.Request.Context(), path.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, toAPIError(err, "get platform workspace image registry"))
		return
	}

	ginutils.OK(c, serializer.ImageRegistryResponse{Data: serializer.NewImageRegistryOutput(reg)})
}

// UpdatePlatWorkspaceImageRegistry 切换空间的镜像仓库，或修改外部镜像仓库
//
//	@ID			UpdatePlatWorkspaceImageRegistry
//	@Summary	切换空间的镜像仓库，或修改外部镜像仓库
//	@Tags		platmgt
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string								true	"工作空间 ID"
//	@Param		body		body		serializer.UpdateImageRegistryInput	true	"镜像仓库信息"
//	@Success	200			{object}	serializer.ImageRegistryResponse
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/plat-mgt/workspaces/{workspaceID}/image-registry [put]
func (h *Handler) UpdatePlatWorkspaceImageRegistry(c *gin.Context) {
	var path serializer.WorkspacePath
	var input serializer.UpdateImageRegistryInput
	if err := ginutils.BindURIJSON(c, &path, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	reg, err := h.manager().Update(c.Request.Context(), path.WorkspaceID, imageregistry.UpdateParams{
		Type:     bkmsreg.ImageRegistryType(input.Type),
		Registry: input.Registry,
		Username: input.Username,
		Password: input.Password,
	})
	if err != nil {
		bkerrs.AbortWithErr(c, toAPIError(err, "update platform workspace image registry"))
		return
	}

	ginutils.OK(c, serializer.ImageRegistryResponse{Data: serializer.NewImageRegistryOutput(reg)})
}

func (h *Handler) manager() *imageregistry.Manager {
	return imageregistry.NewManager(h.registry.DB)
}

var (
	errInvalidRegistryAddress = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"PLATMGT_IMAGE_REGISTRY_INVALID_ADDRESS",
		"invalid image registry address, expected host/namespace without a protocol prefix",
	)
	errRegistryPasswordRequired = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"PLATMGT_IMAGE_REGISTRY_PASSWORD_REQUIRED",
		"image registry password is required",
	)
	errBuiltinRegistryNotFound = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"PLATMGT_BUILTIN_IMAGE_REGISTRY_NOT_FOUND",
		"builtin image registry not found",
	)
	errRegistryAuthFailed = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"PLATMGT_IMAGE_REGISTRY_AUTH_FAILED",
		"image registry authentication failed or push permission denied",
	)
	errRegistryNamespaceInaccessible = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"PLATMGT_IMAGE_REGISTRY_NAMESPACE_INACCESSIBLE",
		"image registry namespace does not exist or is inaccessible",
	)
	errRegistryUnreachable = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"PLATMGT_IMAGE_REGISTRY_UNREACHABLE",
		"image registry is unreachable",
	)
	errBkCICredentialCreateFailed = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"PLATMGT_IMAGE_REGISTRY_CREDENTIAL_CREATE_FAILED",
		"failed to create bkci image credential",
	)
)

// toAPIError 把 Manager 的错误映射为接口错误。
func toAPIError(err error, internalMessage string) error {
	switch {
	case errors.Is(err, bkmsworkspace.ErrWorkspaceNotFound):
		return bkmsworkspace.ErrWorkspaceNotFound.WithCause(err)
	case errors.Is(err, imageregistry.ErrInvalidRegistryAddress):
		return errInvalidRegistryAddress.WithCause(err)
	case errors.Is(err, imageregistry.ErrPasswordRequired):
		return errRegistryPasswordRequired.WithCause(err)
	case errors.Is(err, imageregistry.ErrBuiltinRegistryNotFound):
		return errBuiltinRegistryNotFound.WithCause(err)
	case errors.Is(err, imageregistry.ErrRegistryAuthFailed):
		return errRegistryAuthFailed.WithCause(err)
	case errors.Is(err, imageregistry.ErrRegistryNamespaceInaccessible):
		return errRegistryNamespaceInaccessible.WithCause(err)
	case errors.Is(err, imageregistry.ErrRegistryUnreachable):
		return errRegistryUnreachable.WithCause(err)
	case errors.Is(err, imageregistry.ErrBkCICredentialCreateFailed):
		return errBkCICredentialCreateFailed.WithCause(err)
	default:
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, internalMessage)
	}
}
