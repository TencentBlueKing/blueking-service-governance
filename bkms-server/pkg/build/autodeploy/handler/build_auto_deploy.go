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

// Package handler contains Gin handlers for build auto-deploy APIs.
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/build"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask/buildpoll"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/customruntime"
	workloadruntime "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/runtime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

var _ autodeploy.Handler = (*Handler)(nil)

// Handler handles Gin build auto deploy requests.
type Handler struct {
	registry           *storereg.Registry
	customImageChecker *customruntime.ExistenceChecker
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	snapshotService := snapshot.NewService(
		registry.SnapshotStore,
		registry.BuildConfigStore,
		registry.AppStore,
	)
	return &Handler{
		registry:           registry,
		customImageChecker: customruntime.NewExistenceChecker(snapshotService),
	}
}

// newBuildService 装配构建服务，并注入平台构建镜像引用校验器
func (h *Handler) newBuildService() (*build.Service, error) {
	imageReferenceValidator := workloadruntime.NewImageReferenceValidator(
		h.registry.RuntimeImageStore,
		h.registry.SnapshotStore,
	)
	return build.NewService(
		h.registry.BuildConfigStore,
		h.registry.BuildRecordStore,
		imageReferenceValidator,
		h.customImageChecker,
	)
}

// CreateTrpcBuildDeploy 触发 TRPC 应用构建，并在构建成功后自动部署到指定环境。
//
//	@ID				CreateTrpcBuildDeploy
//	@Summary		触发 TRPC 应用构建并自动部署
//	@Tags			build-autodeploy
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string														true	"应用 ID"
//	@Param			envName	path		string														true	"环境名称"
//	@Param			body	body		serializer.CreateAppModelBuildDeployInput	true	"构建自动部署请求"
//	@Success		200		{object}	serializer.CreateBuildOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		404		{object}	bkerrs.GinErrorOutput
//	@Failure		500		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/envs/{envName}/trpc-build-deploys [post]
func (h *Handler) CreateTrpcBuildDeploy(c *gin.Context) {
	h.createAppModelBuildDeploy(c, bkmsapp.AppTypeTRPC)
}

// CreateTafBuildDeploy 触发 TAF 应用构建，并在构建成功后自动部署到指定环境。
//
//	@ID				CreateTafBuildDeploy
//	@Summary		触发 TAF 应用构建并自动部署
//	@Tags			build-autodeploy
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string														true	"应用 ID"
//	@Param			envName	path		string														true	"环境名称"
//	@Param			body	body		serializer.CreateAppModelBuildDeployInput	true	"构建自动部署请求"
//	@Success		200		{object}	serializer.CreateBuildOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		404		{object}	bkerrs.GinErrorOutput
//	@Failure		500		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/envs/{envName}/taf-build-deploys [post]
func (h *Handler) CreateTafBuildDeploy(c *gin.Context) {
	h.createAppModelBuildDeploy(c, bkmsapp.AppTypeTAF)
}

func (h *Handler) createAppModelBuildDeploy(c *gin.Context, expectedAppType string) {
	var uriInput serializer.AppEnvURIInput
	var input serializer.CreateAppModelBuildDeployInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := ginperm.ValidateAppByID(ctx, h.registry, uriInput.AppID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if app.Type != expectedAppType {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeInvalidArgument, "invalid app type: %s", app.Type))
		return
	}

	if err = perm.NewManager().HasDeployEnvPerm(ctx, app.WorkspaceID, uriInput.EnvName); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.WrapIAMNoPermission(err, app.WorkspaceID, "check env deploy perm"))
		return
	}

	env, err := h.registry.EnvStore.GetByName(ctx, app.WorkspaceID, app.ID, uriInput.EnvName)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "get env"))
		return
	}
	if !supportsBuildAutoDeployEnvType(env.Type) {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Errorf(
				bkerrs.ErrCodeInvalidArgument,
				"build auto deploy only supports non-production env, please promote and deploy separately",
			),
		)
		return
	}

	buildService, err := h.newBuildService()
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "init build service"))
		return
	}

	buildRecord, err := build.StartAndScheduleBuild(
		ctx,
		buildService,
		app,
		input.Branch,
		input.ImageTag,
		build.StartOptions{
			AutoDeploy: &buildpoll.AutoDeployArgs{
				EnvName:         uriInput.EnvName,
				TrafficLaneName: input.TrafficLaneName,
				Replicas:        input.Replicas,
			},
			EnvStore:              h.registry.EnvStore,
			AutoDeployRecordStore: h.registry.BuildAutoDeployRecordStore,
		},
	)
	if err != nil {
		// StartAndScheduleBuild 内部会跑构建前置校验，镜像引用填错或缺凭证属参数问题，
		// 与保存构建配置时的错误码保持一致；其余失败仍按内部错误上报
		errCode := bkerrs.ErrCodeInternalServerError
		if build.IsImageReferenceInvalid(err) || errors.Is(err, build.ErrWorkspaceImageCredentialMissing) {
			errCode = bkerrs.ErrCodeInvalidArgument
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, errCode, "start build auto deploy"))
		return
	}

	ginutils.OK(c, serializer.CreateBuildOutput{
		Data: new(serializer.BuildRecordOutputObj).FromModel(*buildRecord),
	})
}

func supportsBuildAutoDeployEnvType(envType string) bool {
	return bkmsenv.IsValidEnvType(envType) && !bkmsenv.IsProductionType(bkmsenv.Type(envType))
}
