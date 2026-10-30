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

// Package app 提供应用创建相关的处理逻辑
package app

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
)

// createAppRequest 创建 Helm / Agones 应用的后端 API 请求体
type createAppRequest struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Type        string           `json:"type"`
	BuildConfig *BuildConfigSpec `json:"buildConfig"`
	HelmSpec    *HelmSpecSpec    `json:"helmSpec,omitempty"`
}

// createBKMSAppRequest 创建平台托管应用（trpc / taf / blank）的后端 API 请求体
type createBKMSAppRequest struct {
	ID                  string                   `json:"id"`
	Name                string                   `json:"name"`
	Framework           string                   `json:"framework"`
	Language            string                   `json:"language"`
	BuildConfig         *BuildConfigSpec         `json:"buildConfig"`
	AppModelSpec        *bkmsAppModelSpec        `json:"appModelSpec"`
	FrameworkConfigFile *FrameworkConfigFileSpec `json:"frameworkConfigFile,omitempty"`
}

// bkmsAppModelSpec 平台托管应用的启动配置
type bkmsAppModelSpec struct {
	Command   []string       `json:"command,omitempty"`
	Args      []string       `json:"args,omitempty"`
	ShCommand string         `json:"shCommand,omitempty"`
	EnvVars   []VariableSpec `json:"envVars,omitempty"`
}

// buildAppID 根据应用名称与 API 返回的后缀拼接应用 ID
// 如果拼接后超过最大长度限制则返回错误，提示用户缩短名称或直接指定 id
func buildAppID(name, suffix string) (string, error) {
	id := name + suffix
	if len(id) > maxIDLength {
		return "", errors.Errorf(
			"generated app ID '%s' exceeds %d characters, please use a shorter name or specify 'id' field directly",
			id, maxIDLength)
	}
	return id, nil
}

// buildCreateAppRequest 构建后端 API 创建应用请求体。
// Helm / Agones 返回 legacy，平台托管应用（trpc / taf / blank）返回 bkms。
// 调用前需先执行 spec.Validate()，其中已完成旧写法的归一化。
func buildCreateAppRequest(appID string, spec *AppCreateSpec) (*createAppRequest, *createBKMSAppRequest, error) {
	if spec.Type == constant.AppTypeHelm || spec.Type == constant.AppTypeAgones {
		if spec.HelmSpec != nil {
			// 填充 valueFiles 默认值
			fillHelmValueFilesDefault(spec.HelmSpec)
		}
		return &createAppRequest{
			ID:          appID,
			Name:        spec.Name,
			Type:        spec.Type,
			BuildConfig: spec.BuildConfig,
			HelmSpec:    spec.HelmSpec,
		}, nil, nil
	}

	if spec.Framework == "" {
		return nil, nil, errors.Errorf("unsupported app type: %s", spec.Type)
	}

	modelSpec := &bkmsAppModelSpec{}
	if spec.AppModelSpec != nil {
		modelSpec.Command = spec.AppModelSpec.Command
		modelSpec.Args = spec.AppModelSpec.Args
		modelSpec.ShCommand = spec.AppModelSpec.ShCommand
		modelSpec.EnvVars = spec.AppModelSpec.EnvVars
	}
	return nil, &createBKMSAppRequest{
		ID:                  appID,
		Name:                spec.Name,
		Framework:           spec.Framework,
		Language:            spec.Language,
		BuildConfig:         spec.BuildConfig,
		AppModelSpec:        modelSpec,
		FrameworkConfigFile: spec.FrameworkConfigFile,
	}, nil
}

// fillHelmValueFilesDefault 填充 helmSpec.helmSource.valueFiles 默认值
func fillHelmValueFilesDefault(helmSpec *HelmSpecSpec) {
	if helmSpec.HelmSource != nil && len(helmSpec.HelmSource.ValueFiles) == 0 {
		helmSpec.HelmSource.ValueFiles = []string{constant.DefaultHelmValueFile}
	}
}
