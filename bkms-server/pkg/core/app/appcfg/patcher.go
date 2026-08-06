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

package appcfg

import "context"

// ConfigPatcher 定义了对配置文件内容进行补丁的接口。
// 每个实现负责检查配置中是否缺少特定配置块，如果缺少则注入。
type ConfigPatcher interface {
	// Patch 对给定的配置内容进行补丁操作，返回补丁后的内容。
	// 实现应当遵循"不覆盖"原则：如果目标配置路径已存在，应直接返回原始内容。
	Patch(ctx context.Context, appID, envName, content string) (string, error)
}
