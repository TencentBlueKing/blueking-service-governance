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

package handler

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
)

func collectBKCIRepositoriesForCreateApp(
	appType string,
	helmSpec *bkmsapp.HelmSpec,
	buildConfig *build.Config,
) []bkci.RepositoryInitSpec {
	var repos []bkci.RepositoryInitSpec

	if buildConfig != nil && buildConfig.SourceType == build.SourceTypeCodeRepository && buildConfig.CodeRepo != nil {
		repos = append(repos, bkci.RepositoryInitSpec{
			URL:   buildConfig.CodeRepo.RepoURL,
			Alias: buildConfig.CodeRepo.RepoAlias,
		})
	}

	if bkmsapp.IsHelmBasedType(appType) && helmSpec != nil {
		repos = append(repos, helmSourceRepositoriesToBind(helmSpec.HelmSource)...)
	}

	return repos
}

func helmSourceRepositoriesToBind(helmSource *bkmsapp.HelmSource) []bkci.RepositoryInitSpec {
	if helmSource == nil ||
		helmSource.RepoType != bkmsapp.HelmSourceRepoTypeGit ||
		helmSource.GitRepoConfig == nil {
		return nil
	}
	return []bkci.RepositoryInitSpec{{
		URL:   helmSource.GitRepoConfig.RepoURL,
		Alias: helmSource.GitRepoConfig.RepoAlias,
	}}
}
