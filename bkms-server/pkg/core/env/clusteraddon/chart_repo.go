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

package clusteraddon

import (
	"os"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

// GetConfigHelmRepoInfo 从全局配置获取 Helm 仓库信息
func GetConfigHelmRepoInfo() (url, username, password string) {
	return config.G.ClusterAddons.HelmRepoURL,
		config.G.ClusterAddons.HelmRepoUsername,
		config.G.ClusterAddons.HelmRepoPassword
}

// RepoIndex 仓库 Index 缓存，下载一次后可多次按 chartName 查询版本列表。
// 仅用于单次请求内的临时缓存，非跨请求缓存。
type RepoIndex struct {
	index *repo.IndexFile
}

// NewRepoIndex 用给定的 IndexFile 创建 RepoIndex 实例
func NewRepoIndex(index *repo.IndexFile) *RepoIndex {
	return &RepoIndex{index: index}
}

// FetchRepoIndex 从全局配置的 Helm 仓库下载 index.yaml 并返回 RepoIndex
// NOTE: 后续考虑是否需要增加一个内存缓存， 避免重复获取 helm 仓库信息
func FetchRepoIndex() (*RepoIndex, error) {
	repoURL, username, password := GetConfigHelmRepoInfo()

	entry := &repo.Entry{
		URL:      repoURL,
		Username: username,
		Password: password,
	}

	chartRepo, err := repo.NewChartRepository(entry, getter.All(&cli.EnvSettings{}))
	if err != nil {
		return nil, errors.Wrap(err, "creating chart repository")
	}

	indexFilePath, err := chartRepo.DownloadIndexFile()
	if err != nil {
		return nil, errors.Wrap(err, "downloading helm repo index")
	}
	defer os.Remove(indexFilePath)

	indexFile, err := repo.LoadIndexFile(indexFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "loading helm repo index file")
	}

	return &RepoIndex{index: indexFile}, nil
}

// ListChartNames 返回仓库中所有 chart 的名称列表
func (r *RepoIndex) ListChartNames() []string {
	names := make([]string, 0, len(r.index.Entries))
	for name := range r.index.Entries {
		names = append(names, name)
	}
	return names
}

// ListChartVersions 从已加载的 Index 中查询指定 chart 的所有版本列表（最新版在前）
func (r *RepoIndex) ListChartVersions(chartName string) []string {
	entries, ok := r.index.Entries[chartName]
	if !ok {
		return nil
	}

	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		versions = append(versions, entry.Version)
	}
	return versions
}
