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

// Package helm deploy validator.go contains common validator functions for deploy.
package helm

import (
	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

// validateApp 校验应用信息
func validateApp(app *bkmsapp.Application) error {
	if app == nil {
		return errors.New("app is nil")
	}
	if app.HelmSpec == nil {
		return errors.New("app helm spec is nil")
	}
	if app.HelmSpec.HelmSource == nil {
		return errors.New("app helm source is nil")
	}

	helmSource := app.HelmSpec.HelmSource
	switch helmSource.RepoType {
	case bkmsapp.HelmSourceRepoTypeHelm:
		if helmSource.HelmRepoConfig == nil {
			return errors.New("app helm repo config is nil")
		}
		if helmSource.HelmRepoConfig.RepoURL == "" {
			return errors.New("app helm repo config repo url is empty")
		}
		if helmSource.HelmRepoConfig.ChartName == "" {
			return errors.New("app helm repo config chart name is empty")
		}
		return nil
	case bkmsapp.HelmSourceRepoTypeGit:
		if helmSource.GitRepoConfig == nil {
			return errors.New("app helm source git repo config is nil")
		}
		return nil
	case bkmsapp.HelmSourceRepoTypeBCS:
		return errors.New("app helm source repo type BCSRepo is not supported")
	default:
		return errors.Errorf("app helm source repo type %s is invalid", helmSource.RepoType)
	}
}

// validateEnv 校验环境信息
func validateEnv(env *bkmsenv.Environment) error {
	if env == nil {
		return errors.New("env is nil")
	}
	if env.Cluster.ProjectCode == "" {
		return errors.New("env cluster project code is empty")
	}
	if env.Cluster.ClusterID == "" {
		return errors.New("env cluster id is empty")
	}
	if env.Cluster.Namespace == "" {
		return errors.New("env cluster namespace is empty")
	}
	return nil
}
