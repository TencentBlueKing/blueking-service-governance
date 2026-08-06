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

// Package config 定义 bkms-dockerfile-generator 使用的环境变量约定及配置解析逻辑
package config

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

const (
	EnvDockerfileSourceType        = "BKMS_DOCKERFILE_SOURCE_TYPE"
	EnvDockerfileLanguage          = "BKMS_DOCKERFILE_LANGUAGE"
	EnvDockerfilePath              = "BKMS_DOCKERFILE_PATH"
	EnvDockerBuildDir              = "BKMS_DOCKER_BUILD_DIR"
	EnvDockerfileBuilderImage      = "BKMS_DOCKERFILE_BUILDER_IMAGE"
	EnvDockerfileRunnerImage       = "BKMS_DOCKERFILE_RUNNER_IMAGE"
	EnvDockerfilePreBuildCommands  = "BKMS_DOCKERFILE_PRE_BUILD_COMMANDS"
	EnvDockerfileBuildCommands     = "BKMS_DOCKERFILE_BUILD_COMMANDS"
	EnvDockerfileRuntimeEnvCommand = "BKMS_DOCKERFILE_RUNTIME_ENV_COMMANDS"
	EnvDockerfileStartCommand      = "BKMS_DOCKERFILE_START_COMMAND"
	EnvDockerBuildArgNames         = "BKMS_DOCKER_BUILD_ARG_NAMES"
	EnvImageName                   = "BKMS_IMAGE_NAME"

	SourceTypeBKMSGenerated = "bkms_generated"
	SourceTypeRepository    = "repository"
)

var safeImageNameRegexp = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// Config 保存 Dockerfile Generator 从流水线环境变量读取到的输入契约
type Config struct {
	SourceType          string
	Language            string
	DockerfilePath      string
	DockerBuildDir      string
	BuilderImage        string
	RunnerImage         string
	PreBuildCommands    string
	BuildCommands       string
	RuntimeEnvCommands  string
	StartCommand        string
	DockerBuildArgNames string
	ImageName           string
}

// LoadFromEnviron 从 os.Environ 格式的数据中读取并校验 Dockerfile 生成配置
func LoadFromEnviron(environ []string) (Config, error) {
	envs := parseEnviron(environ)
	cfg := Config{
		SourceType:          strings.TrimSpace(envs[EnvDockerfileSourceType]),
		Language:            strings.TrimSpace(envs[EnvDockerfileLanguage]),
		DockerfilePath:      strings.TrimSpace(envs[EnvDockerfilePath]),
		DockerBuildDir:      strings.TrimSpace(envs[EnvDockerBuildDir]),
		BuilderImage:        strings.TrimSpace(envs[EnvDockerfileBuilderImage]),
		RunnerImage:         strings.TrimSpace(envs[EnvDockerfileRunnerImage]),
		PreBuildCommands:    envs[EnvDockerfilePreBuildCommands],
		BuildCommands:       envs[EnvDockerfileBuildCommands],
		RuntimeEnvCommands:  envs[EnvDockerfileRuntimeEnvCommand],
		StartCommand:        strings.TrimSpace(envs[EnvDockerfileStartCommand]),
		DockerBuildArgNames: envs[EnvDockerBuildArgNames],
		ImageName:           strings.TrimSpace(envs[EnvImageName]),
	}

	if cfg.SourceType == "" {
		return Config{}, errors.Errorf("missing required environment variable %s", EnvDockerfileSourceType)
	}

	switch cfg.SourceType {
	case SourceTypeBKMSGenerated:
		if err := validateBKMSGeneratedConfig(cfg); err != nil {
			return Config{}, err
		}
		return cfg, nil
	case SourceTypeRepository:
		return cfg, nil
	default:
		return Config{}, errors.Errorf("unsupported %s value %q", EnvDockerfileSourceType, cfg.SourceType)
	}
}

func validateBKMSGeneratedConfig(cfg Config) error {
	requiredValues := map[string]string{
		EnvDockerfileLanguage:     cfg.Language,
		EnvDockerfilePath:         cfg.DockerfilePath,
		EnvDockerfileBuilderImage: cfg.BuilderImage,
		EnvDockerfileRunnerImage:  cfg.RunnerImage,
		EnvImageName:              cfg.ImageName,
	}
	for name, value := range requiredValues {
		if strings.TrimSpace(value) == "" {
			return errors.Errorf("missing required environment variable %s", name)
		}
	}
	if err := validateImageName(cfg.ImageName); err != nil {
		return err
	}
	return nil
}

func validateImageName(imageName string) error {
	if imageName == "." || imageName == ".." || !safeImageNameRegexp.MatchString(imageName) {
		return errors.Errorf(
			"invalid image name %q: only letters, digits, dot, underscore and hyphen are allowed", imageName,
		)
	}
	return nil
}

func parseEnviron(environ []string) map[string]string {
	envs := make(map[string]string, len(environ))
	for _, item := range environ {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		envs[key] = value
	}
	return envs
}
