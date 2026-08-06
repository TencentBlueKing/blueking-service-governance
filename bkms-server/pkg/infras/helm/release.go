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

// Package helm release.go 提供基于 Helm SDK 的 Release 原子化查询能力
package helm

import (
	"strconv"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
)

// GetReleaseStatus 获取 Release 详细状态
func GetReleaseStatus(cfg *action.Configuration, releaseName string) (*Release, error) {
	statusAction := action.NewStatus(cfg)
	release, err := statusAction.Run(releaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "get release %s status", releaseName)
	}

	result := &Release{
		Name:      release.Name,
		Namespace: release.Namespace,
		Version:   strconv.Itoa(release.Version),
		DeployResult: DeployResult{
			Status:      release.Info.Status,
			Description: release.Info.Description,
			CreatedAt:   release.Info.LastDeployed.String(),
		},
	}
	if release.Chart != nil && release.Chart.Metadata != nil {
		result.Chart = Chart{
			Name:        release.Chart.Metadata.Name,
			Version:     release.Chart.Metadata.Version,
			AppVersion:  release.Chart.Metadata.AppVersion,
			Description: release.Chart.Metadata.Description,
		}
	}
	return result, nil
}

// GetReleaseValues 获取 Release 当前使用的 Values
// revision 为 0 时获取最新版本的 Values
func GetReleaseValues(cfg *action.Configuration, releaseName string, revision int) (map[string]any, error) {
	getValues := action.NewGetValues(cfg)
	if revision > 0 {
		getValues.Version = revision
	}
	values, err := getValues.Run(releaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "get release %s values (revision=%d)", releaseName, revision)
	}
	return values, nil
}

// GetReleaseManifest 获取 Release 的 Manifest（包含所有已部署资源的 YAML）
func GetReleaseManifest(cfg *action.Configuration, releaseName string) (string, error) {
	getAction := action.NewGet(cfg)
	release, err := getAction.Run(releaseName)
	if err != nil {
		return "", errors.Wrapf(err, "get release %s manifest", releaseName)
	}
	return release.Manifest, nil
}
