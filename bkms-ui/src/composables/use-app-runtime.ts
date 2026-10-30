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

import type { PublicView } from '~/@types/v1/app';

/** 从未知 props 中读取已消费的框架文件格式。 */
export function getFrameworkFileFormat(runtime: null | PublicView | undefined): 'taf' | 'yaml' | undefined {
  if (!isRuntimeModSupported(runtime, 'frameworkFile')) return undefined;
  const props = runtime?.mods?.frameworkFile?.props;
  if (!props || typeof props !== 'object' || !('format' in props)) return undefined;
  return props.format === 'taf' || props.format === 'yaml' ? props.format : undefined;
}

/** 只接受后端明确支持，不根据应用类型或语言推断能力。 */
export function isRuntimeModSupported(runtime: null | PublicView | undefined, key: string): boolean {
  return runtime?.mods?.[key]?.supported === true;
}
