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
	"maps"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/ptr"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
)

const (
	// listApiLimit 列表查询单次条数限制
	listApiLimit = 1000

	// defaultFieldManager SSA / Merge Patch 写入资源时的默认 FieldManager 标识
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
// namespace 传 metav1.NamespaceAll 时，会跨所有命名空间 List
func (c *Client) List(
	ctx context.Context, namespace string, opts metav1.ListOptions,
) (*unstructured.UnstructuredList, error) {
	var obj map[string]any
	var items []unstructured.Unstructured
	var resourceVersion string

	opts.Limit = listApiLimit
	opts.Continue = ""
	// 循环获取全量数据
	for {
		ret, err := c.cli.Resource(c.gvr).Namespace(namespace).List(ctx, opts)
		if err != nil {
			return nil, errors.Wrap(err, c.genResActionDesc("list", c.gvr, namespace, ""))
		}

		// 续传 Watch 必须用首次 List 的 resourceVersion，避免 continue 翻页覆盖后丢空窗事件
		if resourceVersion == "" {
			resourceVersion = ret.GetResourceVersion()
		}

		obj = ret.Object
		items = append(items, ret.Items...)
		if ret.GetContinue() == "" {
			break
		}
		opts.Continue = ret.GetContinue()
	}

	list := &unstructured.UnstructuredList{Object: obj, Items: items}
	if resourceVersion != "" {
		list.SetResourceVersion(resourceVersion)
	}

	return list, nil
}

// Watch 从 opts.ResourceVersion 起订阅资源变更；调用方负责 Stop
// 连接时长由调用方通过 opts.TimeoutSeconds 约束，本方法不写死上限
// 这里不校验 ResourceVersion：留空是 apiserver 允许的用法（从当前状态起推），
// 是否必须带续传位点属于业务语义，由调用方在上层校验
// BOOKMARK 原样返回，是否转发由调用方过滤
func (c *Client) Watch(
	ctx context.Context, namespace string, opts metav1.ListOptions,
) (watch.Interface, error) {
	w, err := c.cli.Resource(c.gvr).Namespace(namespace).Watch(ctx, opts)
	if err != nil {
		return nil, errors.Wrap(err, c.genResActionDesc("watch", c.gvr, namespace, ""))
	}

	return w, nil
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
	// 联邦环境网关要求对象体中有 namespace 字段
	obj := &unstructured.Unstructured{Object: withMeta(manifest, namespace)}
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

// Upsert 创建或更新资源。
//
// 非联邦集群使用 Server-Side Apply：字段由 FieldManager 管理，省略字段会被清掉。
// 联邦集群的 BCS 网关不支持 application/apply-patch+yaml，改走 JSON Merge Patch。
// 为贴近 SSA「省略即删除」，联邦路径用 last-applied 注解做三路合并：上次写过、本次省略的字段会置 null；
// 从未写入的字段（如 Service clusterIP、status）仍由 apiserver 保留。
func (c *Client) Upsert(
	ctx context.Context, namespace string, manifest map[string]any, opts metav1.PatchOptions,
) (*unstructured.Unstructured, error) {
	if c.cfg.IsFederation() {
		return c.upsertMergePatch(ctx, namespace, manifest, opts)
	}
	return c.upsertSSA(ctx, namespace, manifest, opts)
}

// upsertSSA 使用 Server-Side Apply 创建或更新资源。
func (c *Client) upsertSSA(
	ctx context.Context, namespace string, manifest map[string]any, opts metav1.PatchOptions,
) (*unstructured.Unstructured, error) {
	if err := c.validateManifest(manifest); err != nil {
		return nil, errors.Wrap(err, "validate manifest")
	}

	// 清理 nested metadata 的零值 creationTimestamp，避免 SSA 严格校验失败
	sanitizeManifestForSSA(manifest)

	data, err := json.Marshal(manifest)
	if err != nil {
		return nil, errors.Wrap(err, "marshal manifest to JSON")
	}

	if opts.FieldManager == "" {
		opts.FieldManager = defaultFieldManager
	}
	if opts.Force == nil {
		opts.Force = ptr.To(true)
	}

	resName := mapx.GetStr(manifest, "metadata.name")
	return c.Patch(ctx, namespace, resName, types.ApplyPatchType, data, opts)
}

// upsertMergePatch 使用三路 JSON Merge Patch 创建或更新资源，供联邦集群使用。
//
// 与 upsertSSA 的不一致（无法用 Merge Patch 完全复现 SSA）：
//   - 删除依据是 last-applied 注解，不是 FieldManager：只清「上次本路径写过、本次省略」的字段；
//     SSA 清的是本 FieldManager 拥有且本次未再 apply 的字段。opts.Force 在此无效，不会抢其他 manager 的字段。
//   - 资源上还没有 last-applied 时（非本路径创建的存量对象），第一次 upsert 不会删除实况里的多余字段。
//   - JSON Merge Patch 把数组当成整体替换；SSA 按 schema 可能按 name 合并列表项。
//   - 会在对象上写入 bkms.tencent.com/last-applied-configuration，SSA 不会。
func (c *Client) upsertMergePatch(
	ctx context.Context, namespace string, manifest map[string]any, opts metav1.PatchOptions,
) (*unstructured.Unstructured, error) {
	if err := c.validateManifest(manifest); err != nil {
		return nil, errors.Wrap(err, "validate manifest")
	}

	desired, err := prepareFederationDesired(manifest)
	if err != nil {
		return nil, err
	}

	if opts.FieldManager == "" {
		opts.FieldManager = defaultFieldManager
	}
	resName := mapx.GetStr(manifest, "metadata.name")

	live, err := c.Get(ctx, namespace, resName, metav1.GetOptions{})
	if errors.Is(err, ErrResourceNotFound) {
		return c.Create(ctx, namespace, desired, metav1.CreateOptions{FieldManager: opts.FieldManager})
	}
	if err != nil {
		return nil, err
	}
	return c.applyFederationThreeWay(ctx, namespace, resName, desired, live, opts)
}

func (c *Client) applyFederationThreeWay(
	ctx context.Context,
	namespace, name string,
	desired map[string]any,
	live *unstructured.Unstructured,
	opts metav1.PatchOptions,
) (*unstructured.Unstructured, error) {
	current, err := json.Marshal(live.Object)
	if err != nil {
		return nil, errors.Wrap(err, "marshal live object")
	}
	modified, err := json.Marshal(desired)
	if err != nil {
		return nil, errors.Wrap(err, "marshal desired object")
	}
	patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(lastAppliedFromLive(live), modified, current)
	if err != nil {
		return nil, errors.Wrap(err, "create three-way merge patch")
	}
	return c.Patch(ctx, namespace, name, types.MergePatchType, patch, opts)
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
	// namespace 不能在调用方的 manifest 中指定，必须在 func 参数中指定；Create 会再写入对象体。
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

// withMeta 返回写入 namespace 后的 manifest，namespace 为空时原样返回。
//
// 只浅拷贝顶层与 metadata 两层，既不修改调用方传入的 map，也不像 unstructured.DeepCopy
// 那样要求 manifest 里全是 JSON 兼容类型（调用方常直接写 int 等 Go 字面量）。
func withMeta(manifest map[string]any, namespace string) map[string]any {
	if namespace == "" {
		return manifest
	}
	obj := maps.Clone(manifest)
	if obj == nil {
		obj = map[string]any{}
	}
	metadata, _ := obj["metadata"].(map[string]any)
	metadata = maps.Clone(metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["namespace"] = namespace
	obj["metadata"] = metadata
	return obj
}

// sanitizeManifestForSSA 递归清理嵌套 metadata 中的 creationTimestamp。
// typed struct（如 GameDeployment）转 unstructured 后，spec.template.metadata 可能带上零值时间戳。
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
