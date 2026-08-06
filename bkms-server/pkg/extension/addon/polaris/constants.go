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

package polaris

const (
	// 在 Pod 上打上如下注解时，集群内 Bcs-Polaris-Operator 会根据注解操作北极星实例
	// AnnotationKeyWeight 北极星权重注解, 设置时会对应修改北极星实例的权重
	AnnotationKeyWeight = "weight.tencent.bkbcs.polaris"
	// AnnotationKeyIsolate 北极星隔离注解，设置为 true 时会隔离北极星实例
	AnnotationKeyIsolate = "isolate.tencent.bkbcs.polaris"
)
