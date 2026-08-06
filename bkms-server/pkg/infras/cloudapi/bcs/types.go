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

package bcs

// Project BCS 项目信息
type Project struct {
	// ID 项目 id, 如: 01234560123456e3acc1024d6bcs44b5
	ID string
	// Code 项目 code, 如: paasv3
	Code string
	// Name 项目名称, 如: paas-v3
	Name string
	// Description 项目描述, 如: 蓝鲸开发者中心
	Description string
	// Kind 项目类型, 如: k8s/mesos
	// k8s 表示为 容器项目，如果为空则为蓝盾项目
	Kind string

	// BizID 项目关联的 cc 业务 id, 如: 398
	BizID string

	// IsOffline 项目是否启用容器服务, 和 bcs 接口参数保持命名一致
	// true: 未启用, false: 启用
	IsOffline bool
}

// Cluster BCS 集群信息
type Cluster struct {
	// ID 集群 id, 如: BCS-K8S-40001
	ID string
	// Name 集群名称, 如: 专用集群
	Name string
	// Type 集群类型, 如: virtual(虚拟), single(独立)
	Type string
	// Environment 集群环境, 如: prod
	Environment string

	// IsShared 是否共享
	IsShared bool

	// Description 集群描述, 如: 用于开发测试的集群, 勿动
	Description string

	// Status 集群状态, 如: RUNNING
	Status string
}

// Namespace BCS 集群的命名空间信息
type Namespace struct {
	Name string
	// Status 命名空间状态, 如: Active
	Status string
}
