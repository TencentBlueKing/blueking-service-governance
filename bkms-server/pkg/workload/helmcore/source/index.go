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

package repo

import (
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
)

// ChartEntry 表示 Helm Repo index.yaml 中某个 chart 的一条版本记录，
// 仅保留与制品列表展示/去重相关的字段。
type ChartEntry struct {
	// Version Chart 版本号（semver）
	Version string
	// Created 产物推入仓库的时间
	Created time.Time
	// Digest Chart 产物摘要，可用于校验产物唯一性
	Digest string
}

// Index 对已加载的 Helm 仓库 index.yaml 的轻量封装。
// 下载并解析一次后，可多次按 chartName 查询版本 entries。
type Index struct {
	index *repo.IndexFile
}

// NewIndex 基于已构造好的 IndexFile 创建 Index，便于测试时注入 in-memory fixture
func NewIndex(index *repo.IndexFile) *Index {
	return &Index{index: index}
}

// FetchIndex 根据 HelmRepoConfig 拉取并加载远端仓库的 index.yaml。
//
// 实现方式：使用 Helm SDK 的 repo.NewChartRepository + ChartRepository.DownloadIndexFile
// 下载 index.yaml 到临时文件，再通过 repo.LoadIndexFile 解析。
// 该方式天然支持带 basic auth 的仓库（如 bkrepo 内建 Helm 仓库），
// 与 Reader.ReadFile / Reader.ReadChartTree 的下载方式保持一致。
func FetchIndex(cfg *bkmsapp.HelmRepoConfig) (*Index, error) {
	if cfg == nil {
		return nil, errors.New("helm repo config is nil")
	}

	entry := &repo.Entry{
		URL:      cfg.RepoURL,
		Username: cfg.Username,
		Password: cfg.Password,
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

	return &Index{index: indexFile}, nil
}

// ListChartEntries 返回 index 中指定 chart 的全部版本 entries。
//
// - 保持 Helm SDK 返回的默认顺序（语义上为版本降序，最新在前）；
// - 若 chart 不存在于 index 中，返回空切片（长度为 0，非 nil）；
// - 返回的 ChartEntry 仅携带制品列表场景所需字段（version/created/digest）。
func (r *Index) ListChartEntries(chartName string) []ChartEntry {
	if r == nil || r.index == nil {
		return []ChartEntry{}
	}
	versions, ok := r.index.Entries[chartName]
	if !ok {
		return []ChartEntry{}
	}

	entries := make([]ChartEntry, 0, len(versions))
	for _, v := range versions {
		if v == nil || v.Metadata == nil {
			continue
		}
		entries = append(entries, ChartEntry{
			Version: v.Version,
			Created: v.Created,
			Digest:  v.Digest,
		})
	}
	return entries
}

// PaginatedChartEntries 表示分页查询 Chart 版本列表的结果。
type PaginatedChartEntries struct {
	// Entries 当前页的版本记录
	Entries []ChartEntry
	// TotalCount 过滤后的总条数（用于前端分页计算）
	TotalCount int64
}

// ListPaginatedChartEntries 查询指定 chart 的版本列表，支持 keyword 过滤和内存分页。
//
// 流程：ListChartEntries → 按 version 关键字过滤 → 按 version 降序分页。
// keyword 为空时不过滤；page/pageSize 越界时 Entries 为空切片、TotalCount 仍为过滤后总数。
func (r *Index) ListPaginatedChartEntries(chartName, keyword string, page, pageSize int64) PaginatedChartEntries {
	allEntries := r.ListChartEntries(chartName)

	// keyword 模糊过滤（按 version 做大小写不敏感子串匹配）
	filtered := allEntries
	if keyword != "" {
		lowered := strings.ToLower(keyword)
		filtered = make([]ChartEntry, 0, len(allEntries))
		for _, e := range allEntries {
			if strings.Contains(strings.ToLower(e.Version), lowered) {
				filtered = append(filtered, e)
			}
		}
	}

	totalCount := int64(len(filtered))

	// 内存分页（page 越界返回空切片）
	if pageSize <= 0 || page <= 0 {
		return PaginatedChartEntries{Entries: []ChartEntry{}, TotalCount: totalCount}
	}
	start := (page - 1) * pageSize
	if start >= totalCount {
		return PaginatedChartEntries{Entries: []ChartEntry{}, TotalCount: totalCount}
	}
	end := start + pageSize
	if end > totalCount {
		end = totalCount
	}

	return PaginatedChartEntries{
		Entries:    filtered[start:end],
		TotalCount: totalCount,
	}
}
