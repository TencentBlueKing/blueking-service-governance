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

import componentConfig from '~/pages/plugin/component-config/index.vue';

import { i18n } from '../../modules/i18n';

import type { NavigationItem } from './types';

/**
 * 基本信息导航配置
 */
export const PLUGIN_NAVIGATION: NavigationItem[] = [
  {
    key: 'component',
    name: i18n.global.t('组件配置'),
    icon: 'single-column',
    component: componentConfig,
    meta: {
      // 无需默认 Header
      layout: 'empty',
    },
  },
];
