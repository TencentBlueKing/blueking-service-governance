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

package deploytypes

// ImageTagEnvPair 镜像标签与环境名称的去重组合（聚合查询结果）
type ImageTagEnvPair struct {
	// ImageTag 镜像标签
	ImageTag string `bson:"imageTag"`
	// EnvName 环境名称
	EnvName string `bson:"envName"`
}

// ChartVersionEnvPair Helm Chart 版本号与环境名称的去重组合（聚合查询结果）
type ChartVersionEnvPair struct {
	// ChartVersion Chart 版本号
	ChartVersion string `bson:"chartVersion"`
	// EnvName 环境名称
	EnvName string `bson:"envName"`
}
