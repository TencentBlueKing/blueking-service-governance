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

// createAppRequest 创建应用后端 API 请求体
type createAppRequest struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	BuildConfig  *BuildConfigSpec  `json:"buildConfig"`
	AppModelSpec *AppModelSpecSpec `json:"appModelSpec,omitempty"`
	HelmSpec     *HelmSpecSpec     `json:"helmSpec,omitempty"`
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

// buildCreateAppRequest 构建后端 API 创建应用请求体
func buildCreateAppRequest(appID string, spec *AppCreateSpec) (*createAppRequest, error) {
	req := &createAppRequest{
		ID:          appID,
		Name:        spec.Name,
		Type:        spec.Type,
		BuildConfig: spec.BuildConfig,
	}

	// 根据应用类型设置对应的 spec
	switch spec.Type {
	case constant.AppTypeTrpc, constant.AppTypeTaf:
		req.AppModelSpec = spec.AppModelSpec
	case constant.AppTypeHelm, constant.AppTypeAgones:
		if spec.HelmSpec != nil {
			// 填充 valueFiles 默认值
			fillHelmValueFilesDefault(spec.HelmSpec)
		}
		req.HelmSpec = spec.HelmSpec
	default:
		return nil, errors.Errorf("unsupported app type: %s", spec.Type)
	}

	return req, nil
}

// fillHelmValueFilesDefault 填充 helmSpec.helmSource.valueFiles 默认值
func fillHelmValueFilesDefault(helmSpec *HelmSpecSpec) {
	if helmSpec.HelmSource != nil && len(helmSpec.HelmSource.ValueFiles) == 0 {
		helmSpec.HelmSource.ValueFiles = []string{constant.DefaultHelmValueFile}
	}
}
