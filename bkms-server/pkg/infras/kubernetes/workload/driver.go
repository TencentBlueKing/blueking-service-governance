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

// Package workload 按工作负载 Kind 提供统一的处理入口（GVR、状态解析、副本视图、支持的操作），
// 使新增工作负载类型只需实现并注册一个 Driver，无需在各调用方追加 Kind 分支判断。
//
// Driver 均为纯函数实现，不访问集群：读取资源仍由调用方完成，Driver 只负责解释 manifest。
package workload

import (
	"sync"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

// ErrUnsupportedKind 工作负载 Kind 没有对应的 Driver 实现
var ErrUnsupportedKind = errors.New("unsupported workload kind")

// Capabilities 工作负载支持的部署操作
type Capabilities struct {
	// InplaceUpdate 是否支持原地升级（不重建 Pod 更新镜像，用于灰度 / 全量更新）
	InplaceUpdate bool
	// SelectedPodDeletion 是否支持通过工作负载删除指定的 Pod
	SelectedPodDeletion bool
}

// View 工作负载的通用视图，屏蔽各类型 Spec 结构的差异
type View struct {
	// Replicas 期望副本数，字段缺失时为 nil
	Replicas *int32
	// Containers Pod 模板中的容器列表
	Containers []corev1.Container
}

// ParseOptions 状态解析的环境相关参数
type ParseOptions struct {
	// Federation 资源是否位于联邦集群（联邦网关返回的字段与直连集群不同）
	Federation bool
}

// Driver 单一工作负载 Kind 的处理入口
type Driver interface {
	// Kind 工作负载类型
	Kind() string
	// GVR 该类型在集群中的 GroupVersionResource
	GVR() schema.GroupVersionResource
	// ParseStatus 解析 manifest 得到综合状态
	ParseStatus(manifest map[string]any, opts ParseOptions) (*k8sstatus.Result, error)
	// View 从 manifest 提取副本数与容器列表
	View(manifest map[string]any) (*View, error)
}

// MainWorkloadDriver 可作为部署主工作负载的 Driver
type MainWorkloadDriver interface {
	Driver
	// Capabilities 该类型支持的部署操作
	Capabilities() Capabilities
}

var (
	mu sync.RWMutex
	// drivers kind -> Driver
	drivers = map[string]Driver{}
	// mainKinds 可作为主工作负载的 Kind，按注册顺序排列，决定部署记录推断主工作负载时的优先级
	mainKinds []string
)

// 新增工作负载类型时，实现 Driver 并在此登记；登记顺序即主工作负载的推断优先级
func init() {
	Register(deploymentDriver{})
	Register(gameDeploymentDriver{})
}

// Register 注册 Driver，重复注册同一 Kind 时后者覆盖前者
func Register(d Driver) {
	mu.Lock()
	defer mu.Unlock()

	kind := d.Kind()
	if _, ok := drivers[kind]; !ok {
		if _, isMain := d.(MainWorkloadDriver); isMain {
			mainKinds = append(mainKinds, kind)
		}
	}
	drivers[kind] = d
}

// Get 获取指定 Kind 的 Driver，未注册时返回 ErrUnsupportedKind
func Get(kind string) (Driver, error) {
	mu.RLock()
	defer mu.RUnlock()

	d, ok := drivers[kind]
	if !ok {
		return nil, errors.Wrapf(ErrUnsupportedKind, "kind %s", kind)
	}
	return d, nil
}

// GetMain 获取指定 Kind 的主工作负载 Driver，未注册或不能作为主工作负载时返回 ErrUnsupportedKind
func GetMain(kind string) (MainWorkloadDriver, error) {
	d, err := Get(kind)
	if err != nil {
		return nil, err
	}
	main, ok := d.(MainWorkloadDriver)
	if !ok {
		return nil, errors.Wrapf(ErrUnsupportedKind, "kind %s can not be a main workload", kind)
	}
	return main, nil
}

// MainDrivers 返回可作为主工作负载的 Driver，按优先级由高到低排列
func MainDrivers() []MainWorkloadDriver {
	mu.RLock()
	defer mu.RUnlock()

	out := make([]MainWorkloadDriver, 0, len(mainKinds))
	for _, kind := range mainKinds {
		if d, ok := drivers[kind].(MainWorkloadDriver); ok {
			out = append(out, d)
		}
	}
	return out
}

// IsMainKind 判断给定 Kind 是否可作为主工作负载
func IsMainKind(kind string) bool {
	mu.RLock()
	defer mu.RUnlock()

	_, ok := drivers[kind].(MainWorkloadDriver)
	return ok
}
