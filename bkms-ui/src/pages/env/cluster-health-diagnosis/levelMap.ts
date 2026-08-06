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

import { TagThemeEnum } from 'bkui-vue/lib/shared';
import { type CheckItemOutput } from '~/@types/v1/bkintegrations-kubeinsight';
/** 状态类型 */
export type LevelType = 'INFO' | 'RECOVERED' | 'RISK' | 'WARN' | Required<CheckItemOutput>['level'];

/** 状态类型对应的值 */
export const LEVEL_VALUE = {
  RISK: 'RISK',
  WARN: 'WARN',
  INFO: 'INFO',
  RECOVERED: 'RECOVERED',
} as const satisfies Record<LevelType, string>;
/** 状态类型用于ui展示的数据 */
export const LEVEL_FOR_UI: Record<
  LevelType,
  {
    // 用于状态列 是否已恢复
    isRecovered: boolean;
    // 告警资源列 左侧告警图标背景色
    resourceKeyColumnBg?: string;
    // Tag组件theme属性
    tagTheme: TagThemeEnum;
  }
> = {
  RISK: {
    resourceKeyColumnBg: '#EA3636',
    isRecovered: false,
    tagTheme: TagThemeEnum.DANGER,
  },
  WARN: {
    resourceKeyColumnBg: '#F59500',
    isRecovered: false,
    tagTheme: TagThemeEnum.DANGER,
  },
  INFO: {
    isRecovered: false,
    tagTheme: TagThemeEnum.DANGER,
  },
  RECOVERED: {
    isRecovered: true,
    tagTheme: TagThemeEnum.SUCCESS,
  },
};
