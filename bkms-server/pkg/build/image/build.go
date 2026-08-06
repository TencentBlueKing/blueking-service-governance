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

// Package build 提供应用构建相关功能
package build

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelineparam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

const (
	// bkciDockerfileSourceRepository 表示蓝盾 Dockerfile 模板使用仓库内 Dockerfile
	bkciDockerfileSourceRepository = "repository"
	// bkciDockerfileSourceGenerated 表示蓝盾 Dockerfile 模板使用平台生成的 Dockerfile
	bkciDockerfileSourceGenerated = "bkms_generated"
	// generatedDockerfileDir 平台生成 Dockerfile 在构建工作目录下的隐藏目录
	generatedDockerfileDir = ".bkms"
	// generatedDockerfileName 平台生成 Dockerfile 文件名，避免与用户仓库中的 Dockerfile 冲突
	generatedDockerfileName = "Dockerfile.generated"
)

// ExecuteBKCIPipelineBuild 执行蓝盾流水线构建
// 返回值：流水线构建状态，流水线构建启动参数，错误
func ExecuteBKCIPipelineBuild(
	ctx context.Context,
	app *bkmsapp.Application,
	cfg *Config,
	branch, imageTag string,
) (*bkciapi.PipelineBuildState, map[string]string, error) {
	appInfo := fmt.Sprintf("<workspace: %s, appID: %s>", app.WorkspaceID, app.ID)

	var params map[string]string
	// 预先资源检查 & 生成流水线构建参数
	switch cfg.SourceType {
	case SourceTypeCodeRepository:
		if cfg.CodeRepo == nil {
			return nil, nil, errors.Errorf("code repo config not found")
		}
		// 构建来源是代码库时，需要在蓝盾上初始化代码仓库
		_, err := bkci.NewRepositoryManager(app.WorkspaceID).Initialize(
			ctx, cfg.CodeRepo.RepoURL, cfg.CodeRepo.RepoAlias,
		)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "ensure bkci repository")
		}
		// 虽然工作空间创建时候就会初始化 dockerfile 通用构建流水线，但是这里还是需要重新初始化，因为可能会有模板版本更新
		_, err = bkci.NewPipelineManager(app.WorkspaceID).Initialize(ctx, string(bkci.PipelineTypeDockerfile))
		if err != nil {
			return nil, nil, errors.Wrap(err, "init builtin dockerfile pipeline")
		}
		// 根据配置生成流水线参数
		params, err = genPipelineBuildParams(ctx, app, cfg, branch, imageTag)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "generate pipeline build params for %s", appInfo)
		}
	case SourceTypePipeline:
		if cfg.Pipeline == nil {
			return nil, nil, errors.Errorf("pipeline config not found")
		}
		// 构建来源为蓝盾流水线时，需要初始化以确保流水线存在
		// 在实际触发构建的时候才初始化流水线，避免前期反复添加不同流水线造成多余数据
		_, err := bkci.NewPipelineManager(app.WorkspaceID).Initialize(ctx, cfg.Pipeline.PipelineID)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "init pipeline")
		}
		// 根据配置生成流水线参数（镜像相关）
		params, err = genPipelineBuildRepoAndImageParams(ctx, app, branch, imageTag)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "generate pipeline build image params for %s", appInfo)
		}
		// 从配置中提取用户填写的流水线参数信息，合并到默认参数中
		for k, v := range cfg.Pipeline.Params {
			// 如果参数不存在，则使用用户填写的参数 --> 系统内置的优先级更高
			if _, ok := params[k]; !ok {
				params[k] = v
			}
		}
	default:
		return nil, nil, errors.Errorf("unsupported source type to execute bkci pipeline build: %s", cfg.SourceType)
	}

	// 获取流水线
	store, err := bkci.NewPipelineStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "create pipeline store")
	}
	pipeline, err := store.GetByWorkspaceAndType(ctx, app.WorkspaceID, cfg.PipelineType)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get workspace %s type %s pipeline", app.WorkspaceID, cfg.PipelineType)
	}

	client, err := bkciapi.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "init bkci client")
	}
	// 触发蓝盾流水线构建
	buildRef, err := client.CreatePipelineBuild(ctx, pipeline.ProjectCode, pipeline.ID, params)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "create pipeline build for %s", appInfo)
	}
	// 获取流水线构建状态
	buildState, err := client.GetPipelineBuildState(ctx, pipeline.ProjectCode, pipeline.ID, buildRef.ID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get pipeline build %s", buildRef.ID)
	}
	return buildState, params, nil
}

// genPipelineBuildParams 生成蓝盾流水线构建参数
func genPipelineBuildParams(
	ctx context.Context, app *bkmsapp.Application, cfg *Config, branch, imageTag string,
) (map[string]string, error) {
	// 如果没有代码仓库配置，则无法生成蓝盾流水线参数
	if cfg.CodeRepo == nil {
		return nil, errors.Errorf("code repo config not found")
	}

	// 构建目录默认值为项目根目录，如果用户有设置，需要确保没有多余的后缀
	buildDir := lo.Ternary(cfg.CodeRepo.SourceDir == "", ".", strings.TrimSuffix(cfg.CodeRepo.SourceDir, "/"))
	// 蓝盾要求的 dockerfile path 是相对于工作空间的，这里需要进行拼接
	dockerfilePath := genDockerfilePath(cfg.CodeRepo, buildDir)
	// 将 dockerBuildArgs map 转换成字符串，并按照字母序排序
	buildArgs := lo.MapToSlice(cfg.CodeRepo.DockerBuildArgs, func(k, v string) string { return k + "=" + v })
	sort.Strings(buildArgs)
	// 将 dockerBuildArgs map 中的 key 进行提取，按照字母序排序后转成 json 数组字符串
	buildArgNames := lo.Keys(cfg.CodeRepo.DockerBuildArgs)
	sort.Strings(buildArgNames)
	encodedBuildArgNames, err := encodeDockerBuildArgNames(buildArgNames)
	if err != nil {
		return nil, errors.Wrapf(err, "encode Docker build arg names for %s", app.ID)
	}

	// 平台通用构建相关参数需要先生成，避免 platform 模式配置错误被后续外部依赖错误遮蔽
	platformBuildParams, err := genPlatformBuildParams(cfg.CodeRepo, app)
	if err != nil {
		return nil, errors.Wrapf(err, "generate platform build params for %s", app.ID)
	}

	// 镜像相关参数
	params, err := genPipelineBuildRepoAndImageParams(ctx, app, branch, imageTag)
	if err != nil {
		return nil, errors.Wrapf(err, "generate pipeline build image params for %s", app.ID)
	}
	// 代码库相关参数（代码库信息部分）
	params[pipelineparam.RepoURL] = cfg.CodeRepo.RepoURL
	params[pipelineparam.RepoAlias] = cfg.CodeRepo.RepoAlias
	// Docker 构建相关参数：当前蓝盾模板仍以 Dockerfile 变量承接镜像构建。
	params[pipelineparam.DockerBuildDir] = buildDir
	// repositoryDockerfile 模式下是仓库内 Dockerfile 路径；platform 模式下是蓝盾侧中间 Dockerfile 的写入目标路径。
	params[pipelineparam.DockerfilePath] = dockerfilePath
	params[pipelineparam.DockerBuildArgs] = strings.Join(buildArgs, "\n")
	params[pipelineparam.DockerBuildArgNames] = encodedBuildArgNames

	// repositoryDockerfile 模式下也会写入固定参数集，保证蓝盾模板行为可预测
	for k, v := range platformBuildParams {
		params[k] = v
	}

	return params, nil
}

// genDockerfilePath 生成蓝盾 DockerBuildAndPushImage 使用的 Dockerfile 路径。
//
// 该路径始终相对于流水线工作空间：repositoryDockerfile 模式使用用户仓库中的 Dockerfile；
// platform 模式使用构建目录下的 .bkms/Dockerfile.generated，避免覆盖用户已有 Dockerfile。
func genDockerfilePath(cfg *RepositoryConfig, buildDir string) string {
	if cfg.EffectiveImageBuildMode() == ImageBuildModePlatform {
		return filepath.Join(buildDir, generatedDockerfileDir, generatedDockerfileName)
	}
	// dockerfile 文件名默认为 Dockerfile
	dockerfile := lo.Ternary(cfg.Dockerfile == "", "Dockerfile", cfg.Dockerfile)
	return filepath.Join(buildDir, dockerfile)
}

// genPlatformBuildParams 生成镜像构建方式相关的流水线参数。
//
// - repositoryDockerfile 模式：sourceType=repository，其余平台生成 Dockerfile 参数为空字符串。
// - platform 模式：sourceType=bkms_generated，写入 language / builderImage / runnerImage / 各命令数组 / start。
//
// 命令数组以 JSON 字符串数组传递，避免流水线环境变量注入吞掉真实换行
func genPlatformBuildParams(cfg *RepositoryConfig, app *bkmsapp.Application) (map[string]string, error) {
	// 注意：ImageBuildToolchainBaseURL 在蓝盾流水线模板中被标记为 valueNotEmpty（必填），
	// 即使 repositoryDockerfile 模式下脚本会跳过使用它，蓝盾参数校验仍要求非空，因此始终传入有效值。
	params := map[string]string{
		pipelineparam.DockerfileSourceType:         bkciDockerfileSourceRepository,
		pipelineparam.ImageBuildToolchainBaseURL:   config.G.ImageBuild.ToolchainBaseURL,
		pipelineparam.DockerfileLanguage:           "",
		pipelineparam.DockerfileBuilderImage:       "",
		pipelineparam.DockerfileRunnerImage:        "",
		pipelineparam.DockerfilePreBuildCommands:   "",
		pipelineparam.DockerfileBuildCommands:      "",
		pipelineparam.DockerfileRuntimeEnvCommands: "",
		pipelineparam.DockerfileStartCommand:       "",
	}
	if cfg.EffectiveImageBuildMode() != ImageBuildModePlatform || cfg.PlatformBuildConfig == nil {
		return params, nil
	}

	language, err := getPlatformDockerfileLanguage(app)
	if err != nil {
		return nil, err
	}

	platBuildCfg := cfg.PlatformBuildConfig
	params[pipelineparam.DockerfileSourceType] = bkciDockerfileSourceGenerated
	params[pipelineparam.ImageBuildToolchainBaseURL] = config.G.ImageBuild.ToolchainBaseURL
	params[pipelineparam.DockerfileLanguage] = language
	params[pipelineparam.DockerfileBuilderImage] = platBuildCfg.BuilderImage
	params[pipelineparam.DockerfileRunnerImage] = platBuildCfg.RunnerImage

	if cmds := platBuildCfg.Commands; cmds != nil {
		preBuildCommands, err := encodeDockerfileCommands(cmds.PreBuild)
		if err != nil {
			return nil, errors.Wrap(err, "encode Dockerfile pre-build commands")
		}
		buildCommands, err := encodeDockerfileCommands(cmds.Build)
		if err != nil {
			return nil, errors.Wrap(err, "encode Dockerfile build commands")
		}
		runtimeEnvCommands, err := encodeDockerfileCommands(cmds.RuntimeEnv)
		if err != nil {
			return nil, errors.Wrap(err, "encode Dockerfile runtime env commands")
		}
		params[pipelineparam.DockerfilePreBuildCommands] = preBuildCommands
		params[pipelineparam.DockerfileBuildCommands] = buildCommands
		params[pipelineparam.DockerfileRuntimeEnvCommands] = runtimeEnvCommands
		params[pipelineparam.DockerfileStartCommand] = cmds.Start
	}
	return params, nil
}

func encodeDockerfileCommands(commands []string) (string, error) {
	if len(commands) == 0 {
		return "", nil
	}

	data, err := json.Marshal(commands)
	if err != nil {
		return "", errors.Wrap(err, "marshal Dockerfile commands")
	}
	return string(data), nil
}

func encodeDockerBuildArgNames(names []string) (string, error) {
	if len(names) == 0 {
		return "", nil
	}

	data, err := json.Marshal(names)
	if err != nil {
		return "", errors.Wrap(err, "marshal Docker build arg names")
	}
	return string(data), nil
}

func getPlatformDockerfileLanguage(app *bkmsapp.Application) (string, error) {
	if app == nil || app.TrpcSpec == nil {
		return "", errors.Errorf("missing TrpcSpec for platform image build")
	}
	language := strings.TrimSpace(app.TrpcSpec.Language)
	if language == "" {
		return "", errors.Errorf("missing TrpcSpec language for platform image build")
	}
	return language, nil
}

// genPipelineBuildRepoAndImageParams 生成蓝盾流水线构建参数（镜像和仓库版本信息相关）
func genPipelineBuildRepoAndImageParams(
	ctx context.Context, app *bkmsapp.Application, branch, imageTag string,
) (map[string]string, error) {
	registry, err := workspace.GetWorkspaceImageRegistry(ctx, app.WorkspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "get workspace image registry")
	}

	return map[string]string{
		// 代码库相关参数（仅版本部分）TODO 目前仅支持 git 分支构建
		pipelineparam.RepoCheckoutBy: "BRANCH",
		pipelineparam.RepoRevision:   branch,
		// 镜像仓库相关参数
		pipelineparam.ImageRegistry: registry.Registry,
		pipelineparam.ImageName:     app.Name,
		pipelineparam.ImageTag:      imageTag,
		// 已经预先添加到蓝盾上的镜像仓库凭证 ID
		pipelineparam.ImageCredential: registry.BkCICredentialID,
	}, nil
}
