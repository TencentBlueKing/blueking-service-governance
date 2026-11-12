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

import type { OperationGroupOutputObj } from '~/@types/v1/operation-audit';

/**
 * 操作记录的环境列展示名：优先使用后端返回的环境展示名，
 * 环境已删除或未返回展示名时回退环境内部名，都没有时展示占位符。
 */
export function getOperationEnvDisplayName(group?: OperationGroupOutputObj) {
  return group?.envDisplayName || group?.envName || '--';
}
