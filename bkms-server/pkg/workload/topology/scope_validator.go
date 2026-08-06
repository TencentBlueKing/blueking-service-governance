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

package topology

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
)

// maxOwnerRefDepth 动态资源 ownerReference 递归追溯的最大深度
const maxOwnerRefDepth = 2

// ScopeValidator 节点 ID 范围校验器
// 验证请求的节点 ID 是否属于当前资源范围，防止越权访问
type ScopeValidator struct {
	staticSet  map[string]bool
	clusterCfg *cluster.Config
}

// NewScopeValidator 创建 ScopeValidator 实例
func NewScopeValidator(scope *ResourceSnapshot, clusterCfg *cluster.Config) *ScopeValidator {
	return &ScopeValidator{
		staticSet:  buildStaticResourceSet(scope.Resources),
		clusterCfg: clusterCfg,
	}
}

// Validate 校验指定的 kind/namespace/name 是否在资源范围内
// 静态资源直接匹配 scope.Resources；动态资源递归追溯 ownerReference 链
func (v *ScopeValidator) Validate(ctx context.Context, kind, namespace, name string) error {
	// 静态资源直接匹配
	key := ResourceKey(kind, namespace, name)
	if v.staticSet[key] {
		return nil
	}

	// 非动态 kind 且不在静态集合中，直接拒绝
	if !potentiallyDynamicKinds[kind] {
		return ErrNodeNotInSnapshot
	}

	// 动态资源：递归追溯 ownerReference 链
	if err := v.traceOwnerRefChain(ctx, kind, namespace, name, 0); err != nil {
		return err
	}
	return nil
}

// traceOwnerRefChain 递归追溯 ownerReference 链，直到找到 scope 内的静态资源
// depth 为当前递归深度，超过 maxOwnerRefDepth 则拒绝
func (v *ScopeValidator) traceOwnerRefChain(ctx context.Context, kind, namespace, name string, depth int) error {
	if depth >= maxOwnerRefDepth {
		return ErrNodeNotInSnapshot
	}

	// 从集群获取资源对象
	resGVR, err := discovery.GetGroupVersionResource(v.clusterCfg, kind, "")
	if err != nil {
		return errors.Wrapf(ErrNodeNotInSnapshot, "resolve GVR for %s: %v", kind, err)
	}

	cli := k8sclient.NewWithGVR(v.clusterCfg, *resGVR)
	obj, err := cli.Get(ctx, namespace, name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(ErrNodeNotInSnapshot, "get %s/%s/%s from cluster: %v", kind, namespace, name, err)
	}

	// 遍历 ownerReferences
	for _, ref := range obj.GetOwnerReferences() {
		ownerKey := ResourceKey(ref.Kind, namespace, ref.Name)

		// 如果 owner 在静态资源集合中，校验通过
		if v.staticSet[ownerKey] {
			return nil
		}

		// 如果 owner 也是动态 kind，继续递归追溯
		if potentiallyDynamicKinds[ref.Kind] {
			if err := v.traceOwnerRefChain(ctx, ref.Kind, namespace, ref.Name, depth+1); err == nil {
				return nil
			}
		}
	}

	return ErrNodeNotInSnapshot
}
