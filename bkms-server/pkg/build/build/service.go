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

// Package build 提供应用构建业务逻辑编排。
package build

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelineparam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelinevar"
	imgbuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// BuildResult 构建执行结果
type BuildResult struct {
	PipelineType string
	Record       *imgbuild.Record
}

// Service 负责构建相关业务编排
type Service struct {
	buildConfigStore        imgbuild.ConfigStore
	buildRecordStore        imgbuild.RecordStore
	imageReferenceValidator ImageReferenceValidator
	customImageChecker      CustomImageChecker
}

// NewService 创建构建服务。
// 依赖参数按"先依赖后数据"顺序传入，且所有依赖均为必选：任何一个为 nil 都会直接返回错误，
// 避免运行时才发现依赖缺失。
func NewService(
	buildConfigStore imgbuild.ConfigStore,
	buildRecordStore imgbuild.RecordStore,
	imageReferenceValidator ImageReferenceValidator,
	customImageChecker CustomImageChecker,
) (*Service, error) {
	if buildConfigStore == nil || buildRecordStore == nil {
		return nil, errors.New("build service dependencies not initialized")
	}
	if imageReferenceValidator == nil {
		return nil, errors.New("imageReferenceValidator must not be nil")
	}
	if customImageChecker == nil {
		return nil, errors.New("customImageChecker must not be nil")
	}
	return &Service{
		buildConfigStore:        buildConfigStore,
		buildRecordStore:        buildRecordStore,
		imageReferenceValidator: imageReferenceValidator,
		customImageChecker:      customImageChecker,
	}, nil
}

// validateBeforeBuild 校验构建触发前置条件
func (s *Service) validateBeforeBuild(ctx context.Context, app *bkmsapp.Application, cfg *imgbuild.Config) error {
	if cfg == nil || cfg.SourceType != imgbuild.SourceTypeCodeRepository || cfg.CodeRepo == nil {
		return nil
	}
	if cfg.CodeRepo.EffectiveImageBuildMode() != imgbuild.ImageBuildModePlatform {
		return nil
	}
	if config.G.ImageBuild.ToolchainBaseURL == "" {
		return errors.New("imageBuild.toolchainBaseURL is required for platform image build")
	}
	if app == nil {
		return errors.New("application must not be nil for platform generated Dockerfile build")
	}
	// 平台生成 Dockerfile 目前仅支持 TRPC 应用；其他类型（如 TAF/Helm）直接拒绝并返回精确的类型错误
	if app.Type != bkmsapp.AppTypeTRPC {
		return errors.Errorf("platform generated Dockerfile build does not support app type %s", app.Type)
	}
	// TRPC 应用中，暂时只支持 Go 语言的自动构建
	if app.TrpcSpec == nil || app.TrpcSpec.Language != appmodel.LanguageGo {
		return errors.New("platform generated Dockerfile build only supports Go language for now")
	}
	// 确保 builder & runner 镜像可用：落在工作空间生效镜像源路径下的按自定义镜像向仓库确认，
	// 其余仍要求来自平台维护的运行时镜像清单
	if err := ValidatePlatformBuildImages(
		ctx, s.imageReferenceValidator, s.customImageChecker, cfg, app.WorkspaceID,
	); err != nil {
		return errors.Wrap(err, "validate platform generated Dockerfile build images")
	}
	return nil
}

// Build 执行构建并落库，返回构建结果
func (s *Service) Build(ctx context.Context, app *bkmsapp.Application, branch, imageTag string) (*BuildResult, error) {
	cfg, err := s.buildConfigStore.Get(ctx, app.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "get app %s build config", app.ID)
	}

	if err = s.validateBeforeBuild(ctx, app, cfg); err != nil {
		return nil, errors.Wrapf(err, "validate app %s build before start", app.ID)
	}

	buildState, params, err := imgbuild.ExecuteBKCIPipelineBuild(ctx, app, cfg, branch, imageTag)
	if err != nil {
		return nil, errors.Wrapf(err, "execute bkci pipeline build for %s", app.ID)
	}

	timeNow := time.Now()
	startTimestamp := cast.ToInt64(buildState.Variables[pipelinevar.BuildStartTime])
	buildRecord := &imgbuild.Record{
		WorkspaceID: app.WorkspaceID,
		AppID:       app.ID,
		PipelineID:  buildState.PipelineID,
		BuildID:     buildState.BuildID,
		Params:      params,
		Artifact: fmt.Sprintf(
			"%s/%s:%s",
			params[pipelineparam.ImageRegistry],
			params[pipelineparam.ImageName],
			params[pipelineparam.ImageTag],
		),
		Status:      imgbuild.StatusRunning,
		Operator:    auth.MustGetUser(ctx).ID,
		TriggerType: imgbuild.TriggerTypeManual,
		StartedAt:   lo.Ternary(startTimestamp != 0, time.UnixMilli(startTimestamp), timeNow),
		CreatedAt:   timeNow,
		UpdatedAt:   timeNow,
	}
	if err = s.buildRecordStore.Create(ctx, buildRecord); err != nil {
		return nil, errors.Wrapf(err, "create build record for %s", app.ID)
	}

	return &BuildResult{
		PipelineType: cfg.PipelineType,
		Record:       buildRecord,
	}, nil
}
