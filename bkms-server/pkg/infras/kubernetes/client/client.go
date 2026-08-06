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

// Package client 提供 k8s dynamic client 实现，用于操作 k8s 集群资源
package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/ptr"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
)

const (
	// listApiLimit 列表查询单次条数限制
	listApiLimit = 1000

	// defaultFieldManager SSA 模式下的默认 FieldManager 标识
	defaultFieldManager = "bkms-server"
)

var (
	// ErrResourceNotFound k8s 资源在集群中不存在
	ErrResourceNotFound = errors.New("k8s resource not found")

	// ErrResourceAlreadyExists k8s 资源已存在
	ErrResourceAlreadyExists = errors.New("k8s resource already exists")
)

// Client k8s 资源客户端
type Client struct {
	cli dynamic.Interface
	cfg *cluster.Config
	gvr schema.GroupVersionResource
}

// New 新建 k8s 资源客户端
func New(cfg *cluster.Config, kind string) (*Client, error) {
	gvr, err := discovery.GetGroupVersionResource(cfg, kind, "")
	if err != nil {
		return nil, errors.Wrapf(err, "get GroupResourceVersion of %s in cluster %s", kind, cfg.ClusterID)
	}
	return &Client{cli: dynamic.NewForConfigOrDie(cfg.Rest), cfg: cfg, gvr: *gvr}, nil
}

// NewWithGVR 新建 k8s 资源客户端（指定 GroupVersionResource）
func NewWithGVR(cfg *cluster.Config, gvr schema.GroupVersionResource) *Client {
	return &Client{cli: dynamic.NewForConfigOrDie(cfg.Rest), cfg: cfg, gvr: gvr}
}

// PaginateList 分页获取资源列表
//
// 注意：由于 Kubernetes API 的 continue token 机制是顺序的，为了获取第 N 页数据，
// 需要从第 1 页开始依次遍历。因此，当 page 较大时，性能会受到影响。
func (c *Client) PaginateList(
	ctx context.Context, namespace string, page, pageSize int64, opts metav1.ListOptions,
) (*unstructured.UnstructuredList, error) {
	// 限制页码：1 <= page
	page = max(page, 1)
	// 限制分页大小：1 <= pageSize <= 1000
	pageSize = max(min(pageSize, listApiLimit), 1)

	// 设置分页参数
	opts.Limit = pageSize
	opts.Continue = ""

	var result *unstructured.UnstructuredList
	var err error

	// 循环直到达到目标页码
	// 注意：必须从第 1 页开始遍历，因为 continue token 是顺序的
	for curPage := int64(1); curPage <= page; curPage++ {
		result, err = c.cli.Resource(c.gvr).Namespace(namespace).List(ctx, opts)
		if err != nil {
			action := fmt.Sprintf("paginate list (page: %d, page size: %d)", page, pageSize)
			return nil, errors.Wrap(err, c.genResActionDesc(action, c.gvr, namespace, ""))
		}

		// 如果已经是目标页，直接返回
		if curPage == page {
			return result, nil
		}

		// 检查是否还有下一页
		continueToken := result.GetContinue()
		if continueToken == "" {
			// 没有更多数据，说明请求的页码超出了实际数据范围，此时返回空列表，但保留元数据
			return &unstructured.UnstructuredList{
				Object: result.Object,
				Items:  []unstructured.Unstructured{},
			}, nil
		}

		// 设置下一页的 continueToken
		opts.Continue = continueToken
	}

	return nil, errors.Errorf("page %d not found", page)
}

// List 获取资源列表（全量）
func (c *Client) List(
	ctx context.Context, namespace string, opts metav1.ListOptions,
) (*unstructured.UnstructuredList, error) {
	var obj map[string]any
	var items []unstructured.Unstructured

	opts.Limit = listApiLimit
	opts.Continue = ""
	// 循环获取全量数据
	for {
		ret, err := c.cli.Resource(c.gvr).Namespace(namespace).List(ctx, opts)
		if err != nil {
			return nil, errors.Wrap(err, c.genResActionDesc("list", c.gvr, namespace, ""))
		}
		obj = ret.Object
		items = append(items, ret.Items...)
		if ret.GetContinue() == "" {
			break
		}
		opts.Continue = ret.GetContinue()
	}

	return &unstructured.UnstructuredList{Object: obj, Items: items}, nil
}

// Get 获取资源
func (c *Client) Get(
	ctx context.Context, namespace, name string, opts metav1.GetOptions,
) (*unstructured.Unstructured, error) {
	ret, err := c.cli.Resource(c.gvr).Namespace(namespace).Get(ctx, name, opts)
	if err != nil {
		// 资源不存在时抛出指定异常
		if k8serrors.IsNotFound(err) {
			return nil, ErrResourceNotFound
		}
		return nil, errors.Wrap(err, c.genResActionDesc("get", c.gvr, namespace, name))
	}
	return ret, nil
}

// Create 创建资源
func (c *Client) Create(
	ctx context.Context, namespace string, manifest map[string]any, opts metav1.CreateOptions,
) (*unstructured.Unstructured, error) {
	// 检查 manifest 的合法性
	if err := c.validateManifest(manifest); err != nil {
		return nil, errors.Wrap(err, "validate manifest")
	}
	obj := &unstructured.Unstructured{Object: manifest}
	ret, err := c.cli.Resource(c.gvr).Namespace(namespace).Create(ctx, obj, opts)
	if err != nil {
		// 资源已经存在时，抛出指定异常
		if k8serrors.IsAlreadyExists(err) {
			return nil, ErrResourceAlreadyExists
		}
		resName := mapx.GetStr(manifest, "metadata.name")
		return nil, errors.Wrap(err, c.genResActionDesc("create", c.gvr, namespace, resName))
	}
	return ret, nil
}

// Update 更新资源
func (c *Client) Update(
	ctx context.Context, namespace, name string, manifest map[string]any, opts metav1.UpdateOptions,
) (*unstructured.Unstructured, error) {
	// 检查 manifest 的合法性
	if err := c.validateManifest(manifest); err != nil {
		return nil, errors.Wrap(err, "validate manifest")
	}

	obj := &unstructured.Unstructured{Object: manifest}
	ret, err := c.cli.Resource(c.gvr).Namespace(namespace).Update(ctx, obj, opts)
	if err != nil {
		return nil, errors.Wrap(err, c.genResActionDesc("update", c.gvr, namespace, name))
	}
	return ret, nil
}

// Upsert 创建或更新资源（如果资源存在则更新，否则创建）
// 基于 Server-Side Apply (SSA) 实现，具备原生 upsert 语义，
// 服务端自动处理字段合并和不可变字段保留，无需关心 resourceVersion、clusterIP 等字段
func (c *Client) Upsert(
	ctx context.Context, namespace string, manifest map[string]any, opts metav1.PatchOptions,
) (*unstructured.Unstructured, error) {
	// 检查 manifest 的合法性
	if err := c.validateManifest(manifest); err != nil {
		return nil, errors.Wrap(err, "validate manifest")
	}

	// 清理 manifest 中嵌套 metadata 的零值 creationTimestamp，
	// 避免 SSA 严格校验时因 CRD schema 未声明该字段而报错
	// （例如 spec.template.metadata.creationTimestamp 由 typed struct 零值序列化产生）
	sanitizeManifestForSSA(manifest)

	// 将 manifest 序列化为 JSON，作为 SSA Patch 的数据
	data, err := json.Marshal(manifest)
	if err != nil {
		return nil, errors.Wrap(err, "marshal manifest to JSON")
	}

	// 设置默认 FieldManager
	if opts.FieldManager == "" {
		opts.FieldManager = defaultFieldManager
	}
	// 强制接管字段所有权，避免与其他管理器冲突
	if opts.Force == nil {
		opts.Force = ptr.To(true)
	}

	resName := mapx.GetStr(manifest, "metadata.name")
	return c.Patch(ctx, namespace, resName, types.ApplyPatchType, data, opts)
}

// Patch 更新资源
func (c *Client) Patch(
	ctx context.Context, namespace, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions,
) (*unstructured.Unstructured, error) {
	ret, err := c.cli.Resource(c.gvr).Namespace(namespace).Patch(ctx, name, pt, data, opts)
	if err != nil {
		return nil, errors.Wrap(err, c.genResActionDesc("patch", c.gvr, namespace, name))
	}
	return ret, nil
}

// Delete 删除资源
func (c *Client) Delete(
	ctx context.Context, namespace, name string, opts metav1.DeleteOptions,
) error {
	err := c.cli.Resource(c.gvr).Namespace(namespace).Delete(ctx, name, opts)
	if err != nil {
		// 资源不存在时无需处理（允许重复删除）
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, c.genResActionDesc("delete", c.gvr, namespace, name))
	}
	return nil
}

// validateManifest 校验传入的 manifest 的合法性
func (c *Client) validateManifest(manifest map[string]any) error {
	// namespace 不能在 manifest 中指定，必须在 func 参数中指定
	if mapx.GetStr(manifest, "metadata.namespace") != "" {
		return errors.Errorf("namespace must provided as func parameter")
	}
	// manifest 中必须指定 metadata.name
	resName := mapx.GetStr(manifest, "metadata.name")
	if resName == "" {
		return errors.Errorf("metadata.name not found")
	}
	return nil
}

// sanitizeManifestForSSA 清理 manifest 中不应出现在 SSA Patch 中的字段
//
// 当 typed struct（如 GameDeployment）通过 runtime.DefaultUnstructuredConverter.ToUnstructured 转换后，
// 嵌套的 ObjectMeta（如 spec.template.metadata）中的零值 creationTimestamp 会被序列化出来，
// 但某些 CRD 的 OpenAPI schema 并未声明该字段，导致 SSA 严格校验失败。
// 此函数递归清理所有嵌套 metadata 中的 creationTimestamp 字段，使 SSA Patch 只包含有效字段。
func sanitizeManifestForSSA(obj map[string]any) {
	for key, val := range obj {
		child, ok := val.(map[string]any)
		if !ok {
			continue
		}
		// 如果当前 key 是 "metadata" 且不在顶层则删除 creationTimestamp 字段
		//（顶层 metadata.creationTimestamp 由 apiserver 管理，无需清理）
		if key == "metadata" {
			delete(child, "creationTimestamp")
		}
		// 递归处理子对象
		sanitizeManifestForSSA(child)
	}
}

// genResActionDesc 生成资源操作描述（Wrap 错误时使用）
func (c *Client) genResActionDesc(action string, gvr schema.GroupVersionResource, namespace, name string) string {
	// 例如：list Deployment
	desc := fmt.Sprintf("%s %s", action, gvr.String())
	// 如果有指定命名空间，加上
	if namespace != "" {
		desc += fmt.Sprintf(" in namespace '%s'", namespace)
	}
	// 如果有指定资源名称，加上
	if name != "" {
		desc += fmt.Sprintf(" with name '%s'", name)
	}
	return desc
}
