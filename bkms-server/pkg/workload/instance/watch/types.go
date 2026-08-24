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

// Package watch 定义应用实例投影 Watch 的领域类型与 SSE 推送
package watch

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8swatch "k8s.io/apimachinery/pkg/watch"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
)

// ErrResourceVersionGone 续传位点已过期（apiserver 410 Gone / Expired）
// 尚未成流，handler 映射为 409，前端必须重新 List 再 Watch
var ErrResourceVersionGone = errors.New("resourceVersion expired")

// EventType 实例 Watch 事件类型（平台投影事件，非原生 Pod Watch）
type EventType string

const (
	// EventAdded 实例新增
	EventAdded EventType = "ADDED"
	// EventModified 实例变更（含投影字段更新）
	EventModified EventType = "MODIFIED"
	// EventDeleted 实例删除
	EventDeleted EventType = "DELETED"
	// EventEnded 当前 Watch 流结束（非实例投影，供前端重连）
	EventEnded EventType = "ENDED"
)

// PodWatcher 按命名空间订阅 Pod 变更；实现方负责把 ResourceVersion 交给 apiserver
type PodWatcher interface {
	Watch(ctx context.Context, namespace string, opts metav1.ListOptions) (k8swatch.Interface, error)
}

// PolarisLister 拉取当前应用环境的北极星实例；失败不得拆 Watch，由 Manager 降级为空数组
type PolarisLister func(ctx context.Context) ([]*polaris.PolarisServiceInstances, error)

// RunParams 一条 Watch 连接的订阅范围与续传位点，均取自部署记录与 List 响应
type RunParams struct {
	// Namespace / LabelSelector 与 List 相同，不从请求参数推断
	Namespace     string
	LabelSelector string
	// ResourceVersion List 首次响应带回的续传位点
	ResourceVersion string
	// DeployID 写入投影的部署记录 ID
	DeployID string
}
