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

package installtask

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrNotFound 安装任务不存在
	ErrNotFound = errors.New("cluster addon install task not found")
	// ErrRunning 集群命名空间内已有该 Addon 运行中的安装任务
	ErrRunning = errors.New("cluster addon install task is already running")
)

// Store 安装任务存储
type Store = store.GenericStore[InstallTask]

// Meta 安装任务的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - clusterID + namespace + addonName + createdAt（降序）：查询 Addon 在集群上最近的任务
// - 唯一（仅 status 为 running 的文档）：clusterID + namespace + addonName，约束同时至多一个运行中的任务
//
// 唯一索引与 clusteraddon.NewInstallLock 一致不带租户：集群是物理资源，不同租户对同一 Addon 的安装也必须互斥。
var Meta = store.Meta[InstallTask]{
	Collection:   "cluster_addon_install_tasks",
	TenantScoped: true,
	NotFound:     ErrNotFound,
	Duplicate:    ErrRunning,
}

// ByTarget 匹配集群命名空间内指定 Addon 的任务
func ByTarget(clusterID, namespace, addonName string) store.Opt {
	return store.Combine(
		store.Where("clusterID", clusterID),
		store.Where("namespace", namespace),
		store.Where("addonName", addonName),
	)
}

// LatestFirst 按创建时间降序
func LatestFirst() store.Opt {
	return store.Desc("createdAt")
}
