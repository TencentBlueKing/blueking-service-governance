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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
)

func collectBKCIRepositoriesForCreateApp(
	appType string,
	helmSpec *bkmsapp.HelmSpec,
	buildConfig *build.Config,
) []bkci.RepositoryInitSpec {
	var repos []bkci.RepositoryInitSpec

	if buildConfig != nil && buildConfig.SourceType == build.SourceTypeCodeRepository && buildConfig.CodeRepo != nil {
		repos = append(repos, bkci.RepositoryInitSpec{
			URL:      buildConfig.CodeRepo.RepoURL,
			Alias:    buildConfig.CodeRepo.RepoAlias,
			Type:     string(buildConfig.CodeRepo.Type),
			AuthType: buildConfig.CodeRepo.AuthType,
			Username: buildConfig.CodeRepo.Username,
			Password: buildConfig.CodeRepo.Password,
		})
	}

	if appruntime.IsHelmBasedType(appType) && helmSpec != nil {
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
		URL:      helmSource.GitRepoConfig.RepoURL,
		Alias:    helmSource.GitRepoConfig.RepoAlias,
		Type:     string(helmSource.GitRepoConfig.Type),
		AuthType: helmSource.GitRepoConfig.AuthType,
		Username: helmSource.GitRepoConfig.Username,
		Password: helmSource.GitRepoConfig.Password,
	}}
}

// resolveHelmGitRepo 确定 Helm Git 来源的鉴权方式与代码库别名，见 bkci.ResolveCodeRepo
func resolveHelmGitRepo(helmSource *bkmsapp.HelmSource) error {
	if helmSource == nil || helmSource.RepoType != bkmsapp.HelmSourceRepoTypeGit || helmSource.GitRepoConfig == nil {
		return nil
	}
	gitRepo := helmSource.GitRepoConfig
	authType, alias, err := bkci.ResolveCodeRepo(
		string(gitRepo.Type),
		gitRepo.AuthType,
		gitRepo.RepoURL,
		gitRepo.RepoAlias,
	)
	if err != nil {
		return bkerrs.Wrapf(err, bkerrs.ErrCodeInvalidArgument, "resolve code repository %s", gitRepo.RepoURL)
	}
	gitRepo.AuthType, gitRepo.RepoAlias = authType, alias
	return nil
}
