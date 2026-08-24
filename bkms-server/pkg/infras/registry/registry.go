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

// Package registry provides utils for container registry.
package registry

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// ImageDetail 包含指定的 TAG 镜像的详细信息
type ImageDetail struct {
	Tag       string
	Digest    string
	MediaType string
	BuiltAt   time.Time
	Size      int64
}

// HumanizeSize 格式化大小为人类可读的格式
func (d *ImageDetail) HumanizeSize() string {
	const unit = 1024
	if d.Size < unit {
		return fmt.Sprintf("%d B", d.Size)
	}
	div, exp := int64(unit), 0
	for n := d.Size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(d.Size)/float64(div), "KMGTPE"[exp])
}

const (
	// defaultTimeout 默认的远程请求超时时间
	defaultTimeout = 30 * time.Second
	// defaultDialTimeout 建立 TCP 连接的超时时间，避免仓库地址不可达时请求长期挂起
	defaultDialTimeout = 10 * time.Second
	// defaultTLSHandshakeTimeout TLS 握手超时时间
	defaultTLSHandshakeTimeout = 10 * time.Second
)

// IsTagNotFound 判断错误链中是否包含“镜像 tag 不存在”错误
func IsTagNotFound(err error) bool {
	var transportErr *transport.Error
	return errors.As(err, &transportErr) && transportErr.StatusCode == http.StatusNotFound
}

// IsAuthRequired 判断错误链中是否包含“镜像仓库鉴权失败”错误。
func IsAuthRequired(err error) bool {
	var transportErr *transport.Error
	if errors.As(err, &transportErr) {
		return transportErr.StatusCode == http.StatusUnauthorized || transportErr.StatusCode == http.StatusForbidden
	}

	// fallback: 某些错误经过包装后不再能还原为 transport.Error，
	// 此时退化为基于错误文案的宽松判断，用于识别 registry 鉴权失败场景。
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "authentication required") || strings.Contains(errMsg, "unauthorized")
}

// Client 客户端
type Client struct {
	nameOpts   []name.Option
	remoteOpts []remote.Option
}

// New ...
func New(username, password string, insecure bool) *Client {
	client := &Client{}

	// 设置用户认证
	if username != "" && password != "" {
		auth := &authn.Basic{Username: username, Password: password}
		client.remoteOpts = append(client.remoteOpts, remote.WithAuth(auth))
	}
	// 构建 Transport，强制设置超时时间
	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: defaultDialTimeout}).DialContext,
		TLSHandshakeTimeout:   defaultTLSHandshakeTimeout,
		ResponseHeaderTimeout: defaultTimeout,
	}

	// 允许跳过 TLS 验证
	if insecure {
		client.nameOpts = append(client.nameOpts, name.Insecure)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // G402
	}

	client.remoteOpts = append(client.remoteOpts, remote.WithTransport(transport))

	return client
}

// ListAllTags 获取仓库的全量 TAG 列表（按名称降序排列）
// repoName: 仓库名称
// - 格式：[registry/][namespace/]repository
// - 示例：mirrors.tencent.com/bkpaas/bkpaas-app-operator、library/ubuntu、nginx 等
func (c *Client) ListAllTags(ctx context.Context, repoName string) ([]string, error) {
	repo, err := name.NewRepository(repoName, c.nameOpts...)
	if err != nil {
		return nil, errors.Wrapf(err, "parse repo name %s", repoName)
	}

	tags, err := remote.List(repo, c.withContext(ctx)...)
	if err != nil {
		return nil, errors.Wrapf(err, "list tags of %s", repoName)
	}

	// 按 Tag 名称降序排列
	slices.SortFunc(tags, func(a, b string) int { return strings.Compare(b, a) })

	return tags, nil
}

// ListTags 获取仓库的所有 TAG（带关键字过滤和分页）
// repoName: 仓库名称
// - 格式：[registry/][namespace/]repository
// - 示例：mirrors.tencent.com/bkpaas/bkpaas-app-operator、library/ubuntu、nginx 等
// keyword: 搜索关键字（如果有需要的话）
// page: 页码，从 1 开始
// pageSize: 每页大小，默认为 100
// 返回值：tags: 指定页的 TAG 列表，total: 总数量，err: 错误信息
func (c *Client) ListTags(
	ctx context.Context, repoName, keyword string, page, pageSize int,
) ([]string, int, error) {
	tags, err := c.ListAllTags(ctx, repoName)
	if err != nil {
		return nil, 0, err
	}

	// 过滤关键字（如果有需要的话）
	if keyword != "" {
		tags = lo.Filter(tags, func(item string, _ int) bool {
			return strings.Contains(item, keyword)
		})
	}

	// 过滤后的总数量
	total := len(tags)

	// 对 TAGs 分页
	if total > 0 {
		start := (page - 1) * pageSize
		end := page * pageSize

		// 检查边界，防止索引越界
		if start >= total {
			tags = []string{}
		} else {
			if end > total {
				end = total
			}
			tags = tags[start:end]
		}
	}

	return tags, total, nil
}

// GetTagDetail 获取单个镜像 TAG 的详细信息
func (c *Client) GetTagDetail(ctx context.Context, repoName, tagName string) (ImageDetail, error) {
	var detail ImageDetail
	detail.Tag = tagName

	// 创建标签的完整引用
	tagRef, err := name.ParseReference(fmt.Sprintf("%s:%s", repoName, tagName), c.nameOpts...)
	if err != nil {
		return detail, errors.Wrapf(err, "parse tag reference %s:%s", repoName, tagName)
	}

	// 获取 Manifest 描述符
	desc, err := remote.Get(tagRef, c.withContext(ctx)...)
	if err != nil {
		return detail, errors.Wrapf(err, "get manifest of %s:%s", repoName, tagName)
	}

	detail.Digest = desc.Digest.String()
	detail.MediaType = string(desc.MediaType)

	// 获取完整镜像信息以计算大小
	img, err := desc.Image()
	if err != nil {
		return detail, errors.Wrap(err, "get image from descriptor")
	}

	// 计算总大小（所有层未压缩大小的总和）
	layers, err := img.Layers()
	if err != nil {
		return detail, errors.Wrap(err, "get layers")
	}

	var totalSize int64
	for _, layer := range layers {
		size, sErr := layer.Size()
		if sErr != nil {
			return detail, errors.Wrap(sErr, "failed to get layer size")
		}
		totalSize += size
	}
	detail.Size = totalSize

	// 获取镜像的构建时间
	cfgFile, err := img.ConfigFile()
	if err != nil {
		return detail, errors.Wrap(err, "get image config file")
	}
	detail.BuiltAt = cfgFile.Created.Time

	return detail, nil
}

// HeadManifest 用 HEAD 确认远端仓库中指定 tag 的 manifest 是否存在，不拉取层数据
func (c *Client) HeadManifest(ctx context.Context, repoName, tagName string) error {
	_, err := c.headDescriptor(ctx, repoName, tagName)
	return err
}

// DeleteTag 删除远端仓库中的指定镜像标签
// 利用开源库 go-containerregistry 删除镜像，只能删除底层镜像，无法删除对应仓库如 bkrepo、mirrors 仓库自己额外的元数据
// 删除后在仓库相应页面还有可能看到镜像，但使用命令拉取/同步仓库无法拉到对应镜像，在 bkms 侧不受影响
func (c *Client) DeleteTag(ctx context.Context, repoName, tagName string) error {
	// 获取镜像 desc 信息
	desc, err := c.headDescriptor(ctx, repoName, tagName)
	if err != nil {
		return err
	}

	// 构建 digest 引用：repoName@sha256:xxxxx
	digestRef, err := name.ParseReference(fmt.Sprintf("%s@%s", repoName, desc.Digest.String()), c.nameOpts...)
	if err != nil {
		return errors.Wrap(err, "parse digest reference")
	}

	// 按 digest 删除
	if err = remote.Delete(digestRef, c.withContext(ctx)...); err != nil {
		return errors.Wrap(err, "delete tag")
	}
	return nil
}

// ListTagsWithDetail 获取仓库所有标签的详细信息（使用并发）
//
// 适用场景：需要获取镜像标签的完整信息（包括大小、摘要、媒体类型等）时使用，
// 相比 ListTags 只返回标签名称，此方法会返回每个标签的详细元数据信息
//
// keyword: 搜索关键字（如果有需要的话）
// page: 页码，从 1 开始
// pageSize: 每页大小，默认为 100
// 返回值：details: 指定页的镜像详细信息列表，total: 总数量，err: 错误信息
func (c *Client) ListTagsWithDetail(
	ctx context.Context, repoName, keyword string, page, pageSize int,
) ([]ImageDetail, int, error) {
	// 获取所有 TAG
	tags, total, err := c.ListTags(ctx, repoName, keyword, page, pageSize)
	if err != nil {
		return nil, 0, errors.Wrap(err, "list tags")
	}

	// 控制并发数量
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)
	results := make(chan ImageDetail, len(tags))

	// 为每个 TAG 启动一个协程
	for _, tag := range tags {
		wg.Go(func() {
			// 获取信号量许可，使用 defer 延迟释放
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 获取标签详情
			detail, gErr := c.GetTagDetail(ctx, repoName, tag)
			if gErr != nil {
				// 单个失败不影响其他请求，设置为空数据即可
				log.ErrorNoContextf("failed to get details for repo %s tag %s: %v", repoName, tag, gErr)
				detail = ImageDetail{Tag: tag}
			}

			results <- detail
		})
	}

	// 等待所有请求完成
	wg.Wait()
	close(results)

	details := lo.ChannelToSlice(results)
	slices.SortFunc(details, func(a, b ImageDetail) int {
		return strings.Compare(b.Tag, a.Tag)
	})
	return details, total, nil
}

// headDescriptor 按 repoName:tag 做 HEAD，返回 digest 描述符供删除等后续步骤使用
func (c *Client) headDescriptor(ctx context.Context, repoName, tagName string) (*v1.Descriptor, error) {
	tagRef, err := name.ParseReference(fmt.Sprintf("%s:%s", repoName, tagName), c.nameOpts...)
	if err != nil {
		return nil, errors.Wrapf(err, "parse tag reference %s:%s", repoName, tagName)
	}
	desc, err := remote.Head(tagRef, c.withContext(ctx)...)
	if err != nil {
		return nil, errors.Wrapf(err, "head manifest %s:%s", repoName, tagName)
	}
	return desc, nil
}

// withContext 在基础 remote 选项后追加 context，使请求可随调用方取消或超时中断。
//
// 必须复制切片而非直接 append 到 c.remoteOpts：底层数组共享会让并发调用互相覆盖 context
func (c *Client) withContext(ctx context.Context) []remote.Option {
	opts := make([]remote.Option, 0, len(c.remoteOpts)+1)
	opts = append(opts, c.remoteOpts...)
	return append(opts, remote.WithContext(ctx))
}
