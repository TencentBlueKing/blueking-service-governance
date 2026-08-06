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

// Package bkrepo 蓝盾制品库（Docker 镜像仓库、Helm 仓库等）相关接入实现
package bkrepo

// RepoType 制品库仓库类型
type RepoType string

const (
	// RepoTypeDocker Docker 仓库
	RepoTypeDocker RepoType = "DOCKER"
	// RepoTypeHelm Helm 仓库
	RepoTypeHelm RepoType = "HELM"
)
