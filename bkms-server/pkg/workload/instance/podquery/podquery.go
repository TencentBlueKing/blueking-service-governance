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

// Package podquery 按部署位置查询该次部署的 Pod 及相关集群事件，供实例列表、事件列表、部署诊断等入口复用
package podquery

import (
	"context"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/clusterresources"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

// Location 一次部署的 Pod 所在位置，通常取自部署记录
type Location struct {
	// ClusterID 部署所在集群
	ClusterID string
	// Namespace 部署所在命名空间
	Namespace string
	// LabelSelector 圈定该次部署 Pod 的标签
	LabelSelector map[string]string
}

// PodLister 按部署位置列出该次部署的 Pod
type PodLister struct {
	client *k8sclient.PodClient
	loc    Location
}

// NewPodLister 创建 PodLister，集群与命名空间以部署位置为准，不从请求参数推断
func NewPodLister(loc Location) *PodLister {
	return &PodLister{
		client: k8sclient.NewPodClient(cluster.NewConfig(loc.ClusterID)),
		loc:    loc,
	}
}

// List 列出 LabelSelector 匹配的 Pod，返回结果保留集群 List 的 resourceVersion，供 Watch 从该位点续传
// 选择器为空时不做拦截，由调用方按自身语义决定是否先行校验
func (l *PodLister) List(ctx context.Context) (*unstructured.UnstructuredList, error) {
	selector := labels.SelectorFromSet(l.loc.LabelSelector).String()
	pods, err := l.client.List(ctx, l.loc.Namespace, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, errors.Wrapf(err, "list namespace %s labelSelector [%s] pods", l.loc.Namespace, selector)
	}
	return pods, nil
}

// ResourceRef 事件关联的一个集群资源
type ResourceRef struct {
	// Kind 资源类型，如 Deployment、Pod
	Kind string
	// Name 资源名称
	Name string
}

// BuildWorkloadRefs 把部署记录上的工作负载转成事件查询用的资源引用
func BuildWorkloadRefs(keys []appmodeldeploy.ResourceKey) []ResourceRef {
	refs := make([]ResourceRef, 0, len(keys))
	for _, rk := range keys {
		refs = append(refs, ResourceRef{Kind: rk.Kind, Name: rk.Name})
	}
	return refs
}

// BuildPodRefs 把 Pod 清单转成事件查询用的资源引用，跳过没有名称的条目
func BuildPodRefs(pods []unstructured.Unstructured) []ResourceRef {
	refs := make([]ResourceRef, 0, len(pods))
	for _, pod := range pods {
		if name := mapx.GetStr(pod.Object, "metadata.name"); name != "" {
			refs = append(refs, ResourceRef{Kind: k8skind.Po, Name: name})
		}
	}
	return refs
}

// EventFilter 事件查询的过滤与分页条件
type EventFilter struct {
	// Level 事件级别，空表示不过滤
	Level string
	// StartedAt 起始时间戳，0 表示不限
	StartedAt int64
	// EndedAt 结束时间戳，0 表示不限
	EndedAt int64
	// Page 页码，从 1 开始
	Page int
	// PageSize 每页条数
	PageSize int
}

// EventLister 查询环境所在集群中与指定资源相关的事件
type EventLister struct {
	client      *clusterresources.ApiClient
	projectCode string
	clusterID   string
}

// NewEventLister 以当前请求用户的身份创建事件查询客户端，集群取自环境配置
func NewEventLister(ctx context.Context, env *envmodel.Environment) (*EventLister, error) {
	user, err := auth.GetUser(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get request user")
	}
	client, err := clusterresources.New(user)
	if err != nil {
		return nil, errors.Wrap(err, "create cluster resources client")
	}
	return &EventLister{
		client:      client,
		projectCode: env.Cluster.ProjectCode,
		clusterID:   env.Cluster.ClusterID,
	}, nil
}

// List 查询命名空间内与 refs 相关的事件
func (l *EventLister) List(
	ctx context.Context,
	namespace string,
	refs []ResourceRef,
	filter EventFilter,
) (*clusterresources.PaginatedEvents, error) {
	kinds := make([]string, 0, len(refs))
	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		kinds = append(kinds, ref.Kind)
		names = append(names, ref.Name)
	}
	page, err := l.client.ListEvents(ctx, l.projectCode, l.clusterID, clusterresources.ListEventParams{
		Namespace:     namespace,
		ResourceKinds: kinds,
		ResourceNames: names,
		Level:         filter.Level,
		StartedAt:     filter.StartedAt,
		EndedAt:       filter.EndedAt,
		Page:          filter.Page,
		PageSize:      filter.PageSize,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "list events in cluster %s namespace %s", l.clusterID, namespace)
	}
	return page, nil
}
