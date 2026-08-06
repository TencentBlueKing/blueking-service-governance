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

// Package repo 提供 Helm 仓库访问能力：读取 Chart 文件/文件树、列出版本、解析仓库配置等。
package repo

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"unicode/utf8"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/file"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
)

// chartFileMaxSize 单个文本文件最大读取字节数（超过则置 Content 为空，仅返回元信息）
const chartFileMaxSize = 1 << 20 // 1 MiB

// ErrFileNotFound 文件不存在
var ErrFileNotFound = errors.New("file not found")

// Version 表示 Helm 仓库中的一个版本
type Version struct {
	// Name 表示版本名. 如果 Type 是 chart, 则为 Helm Chart 版本值; 如果 Type 是 branch/tag, 则为 git 仓库的分支名/标签名
	Name string
	// CreatedAt 表示版本创建时间
	CreatedAt string
	// UpdatedAt 表示版本更新时间
	UpdatedAt string
	// Digest 表示版本摘要，如果是分支/Tag，则可以为 CommitID
	Digest string
}

// FileNode 表示 Helm Chart 包内的文件树节点
type FileNode struct {
	// Name 节点名称（不含父级路径）
	Name string
	// Path 相对路径（相对于 chart 根目录，使用正斜杠）
	Path string
	// IsDir 是否为目录
	IsDir bool
	// Size 文件大小（字节）
	Size int64
	// IsBinary 是否为二进制文件（true 时不返回 Content）
	IsBinary bool
	// Content 文本内容（仅文件且大小未超限时填充）
	Content []byte
	// Children 子节点（仅目录有效，已按字母序排序，目录在前）
	Children []*FileNode
}

// Reader 负责 Helm 仓库的读取能力（读文件、读文件树、列版本）
type Reader struct {
	config *bkmsapp.HelmRepoConfig
}

// NewReader 创建一个新的 Reader
func NewReader(config *bkmsapp.HelmRepoConfig) *Reader {
	return &Reader{config: config}
}

// withExtractedChart 下载并解压指定版本的 Chart 到临时目录，回调中可访问 chartDir，函数返回时自动清理
func (r *Reader) withExtractedChart(version Version, fn func(chartDir string) error) error {
	root, err := os.MkdirTemp("", "repo-src-helm-*")
	if err != nil {
		return errors.Wrap(err, "creating temp dir")
	}
	defer os.RemoveAll(root)

	// 通过 Helm SDK 查找 Chart 在仓库中的下载地址（带认证，支持跳过 TLS 校验）
	chartURL, err := repo.FindChartInAuthAndTLSRepoURL(
		r.config.RepoURL, r.config.Username, r.config.Password,
		r.config.ChartName, version.Name, "", "", "", true,
		getter.All(&cli.EnvSettings{}),
	)
	if err != nil {
		return errors.Wrap(err, "finding chart in repo")
	}

	// 使用 Helm SDK 的 Pull action 下载并解压 Chart
	pull := action.NewPullWithOpts(action.WithConfig(&action.Configuration{}))
	pull.Settings = &cli.EnvSettings{}
	pull.DestDir = root
	pull.Version = version.Name
	pull.Username = r.config.Username
	pull.Password = r.config.Password
	pull.Untar = true
	pull.UntarDir = root

	if _, err = pull.Run(chartURL); err != nil {
		return errors.Wrap(err, "pulling chart")
	}

	chartDir := filepath.Join(root, r.config.ChartName)
	return fn(chartDir)
}

// ReadFile reads the file from the helm chart repository
// - version: the version of the repository
// - name: the name of the file to read
func (r *Reader) ReadFile(version Version, name string) ([]byte, error) {
	var content []byte
	err := r.withExtractedChart(version, func(chartDir string) error {
		// Chart 文件已下载并解压到临时目录，安全读取目标文件（防止路径穿越）
		c, rerr := file.SafeReadFile(chartDir, name)
		if rerr != nil {
			if os.IsNotExist(rerr) {
				return errors.Wrap(ErrFileNotFound, "reading file")
			}
			return errors.Wrap(rerr, "reading file")
		}
		content = c
		return nil
	})
	if err != nil {
		return nil, err
	}
	return content, nil
}

// ReadChartTree 下载并解压指定 Chart 版本，返回该 Chart 根目录及其全部子文件构成的树形结构
// 文本文件且大小未超过 chartFileMaxSize 时会一并填充 Content；二进制或超大文件仅返回元信息
func (r *Reader) ReadChartTree(version Version) (*FileNode, error) {
	var root *FileNode
	err := r.withExtractedChart(version, func(chartDir string) error {
		node, berr := buildChartFileNode(chartDir, "", filepath.Base(chartDir))
		if berr != nil {
			return berr
		}
		root = node
		return nil
	})
	if err != nil {
		return nil, err
	}
	return root, nil
}

// buildChartFileNode 递归遍历 dir，构建以 name 命名的节点；relPath 表示节点相对 chart 根目录的相对路径
func buildChartFileNode(dir, relPath, name string) (*FileNode, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "stat %s", dir)
	}

	node := &FileNode{
		Name:  name,
		Path:  relPath,
		IsDir: info.IsDir(),
		Size:  info.Size(),
	}

	if !info.IsDir() {
		if info.Size() <= chartFileMaxSize {
			content, rerr := os.ReadFile(dir) // nolint: gosec  // dir 来自 helm 解压目录，可信
			if rerr != nil {
				return nil, errors.Wrapf(rerr, "read file %s", dir)
			}
			if isBinary(content) {
				node.IsBinary = true
			} else {
				node.Content = content
			}
		} else {
			// 大文件不读取内容，前端按文件名/大小展示
			node.IsBinary = true
		}
		return node, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "read dir %s", dir)
	}
	// 排序：目录在前，按字母序
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	for _, e := range entries {
		childPath := filepath.Join(dir, e.Name())
		childRel := e.Name()
		if relPath != "" {
			childRel = filepath.Join(relPath, e.Name())
		}
		child, cerr := buildChartFileNode(childPath, childRel, e.Name())
		if cerr != nil {
			return nil, cerr
		}
		node.Children = append(node.Children, child)
	}
	return node, nil
}

// isBinary 简单启发式判定：包含 NUL 字节或非合法 UTF-8 即视为二进制
func isBinary(data []byte) bool {
	if bytes.IndexByte(data, 0) >= 0 {
		return true
	}
	return !utf8.Valid(data)
}

// ListVersions 列出仓库内指定 Chart 的所有版本。
//
// 内部委托给 FetchIndex + Index.ListChartEntries，
// 再将 []ChartEntry 映射为 []Version，以便与 ListPaginatedChartEntries 共用同一套 index.yaml 下载+解析实现。
func (r *Reader) ListVersions() ([]Version, error) {
	repoIndex, err := FetchIndex(r.config)
	if err != nil {
		return nil, errors.Wrap(err, "fetching helm repo index")
	}

	entries := repoIndex.ListChartEntries(r.config.ChartName)
	versions := make([]Version, 0, len(entries))
	for _, e := range entries {
		versions = append(versions, Version{
			Name:      e.Version,
			CreatedAt: e.Created.String(),
			Digest:    e.Digest,
		})
	}
	return versions, nil
}
