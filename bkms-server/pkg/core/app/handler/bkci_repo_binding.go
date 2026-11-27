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
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/coderepo"
)

// collectRepositoriesForCreateApp 创建应用时要保存的代码库
func collectRepositoriesForCreateApp(
	workspaceID string,
	appType string,
	helmSpec *bkmsapp.HelmSpec,
	buildConfig *build.Config,
) []coderepo.SaveSpec {
	var repos []coderepo.SaveSpec

	if buildConfig != nil && buildConfig.SourceType == build.SourceTypeCodeRepository && buildConfig.CodeRepo != nil {
		repos = append(repos, coderepo.SaveSpec{
			WorkspaceID: workspaceID,
			Type:        string(buildConfig.CodeRepo.Type),
			URL:         buildConfig.CodeRepo.RepoURL,
			Alias:       buildConfig.CodeRepo.RepoAlias,
			Username:    buildConfig.CodeRepo.Username,
			Password:    buildConfig.CodeRepo.Password,
		})
	}

	if appruntime.IsHelmBasedType(appType) && helmSpec != nil {
		repos = append(repos, helmSourceRepositoriesToSave(workspaceID, helmSpec.HelmSource)...)
	}

	return repos
}

// helmSourceRepositoriesToSave Helm Git 来源要保存的代码库
func helmSourceRepositoriesToSave(workspaceID string, helmSource *bkmsapp.HelmSource) []coderepo.SaveSpec {
	if helmSource == nil ||
		helmSource.RepoType != bkmsapp.HelmSourceRepoTypeGit ||
		helmSource.GitRepoConfig == nil {
		return nil
	}
	return []coderepo.SaveSpec{{
		WorkspaceID: workspaceID,
		Type:        string(helmSource.GitRepoConfig.Type),
		URL:         helmSource.GitRepoConfig.RepoURL,
		Alias:       helmSource.GitRepoConfig.RepoAlias,
		Username:    helmSource.GitRepoConfig.Username,
		Password:    helmSource.GitRepoConfig.Password,
	}}
}

// resolveHelmGitRepo 确定 Helm Git 来源的接入方式与别名
func resolveHelmGitRepo(helmSource *bkmsapp.HelmSource) error {
	if helmSource == nil || helmSource.RepoType != bkmsapp.HelmSourceRepoTypeGit || helmSource.GitRepoConfig == nil {
		return nil
	}
	gitRepo := helmSource.GitRepoConfig
	sourceType, err := coderepo.ResolveSourceType(string(gitRepo.Type))
	if err != nil {
		return coderepo.ToBkErr(err, "resolve source type of helm git repository")
	}
	alias, err := coderepo.ResolveAlias(sourceType, gitRepo.RepoURL, gitRepo.RepoAlias)
	if err != nil {
		return coderepo.ToBkErr(err, "resolve alias of helm git repository")
	}
	gitRepo.Type, gitRepo.RepoAlias = bkmsapp.GitRepoType(sourceType.Name), alias
	return nil
}

// toRepositoryBkErr 保存代码库失败的接口错误
func toRepositoryBkErr(err error) error {
	return coderepo.ToBkErr(err, "save workspace repositories")
}
