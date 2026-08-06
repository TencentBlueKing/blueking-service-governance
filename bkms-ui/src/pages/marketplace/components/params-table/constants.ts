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

import type { ParamType } from '~/components/type-param-select.vue';

/** 面板切换后通知 ParamsTable 刷新宽度的 provide/inject key */
export const REFRESH_TABLE_SIGNAL = Symbol('refreshTableSignal');

/** 内置变量 provide/inject key */
export const BUILTIN_VARS_SYMBOL = Symbol('builtinVars');

/** 后端类型 -> 前端类型映射 */
export const TYPE_MAP: Record<string, ParamType> = {
  STRING: 'String',
  INT: 'Number',
  TEXT: 'Text',
  SELECT: 'Select',
  BOOL: 'Boolean',
  MAP: 'Map',
};
