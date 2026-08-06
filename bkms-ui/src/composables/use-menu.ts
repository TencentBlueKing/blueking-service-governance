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

import { computed } from 'vue';

import { useSpaceStore } from '~/stores/space';

import { i18n } from '../modules/i18n';

export type MenuId = 'APP' | 'BASIC' | 'ENV' | 'PLATFORM' | 'PLUGIN';
export type MenuItem = (typeof navListArr)[number];
export interface NavItem {
  id: MenuId;
  name: string;
  params?: Record<string, string>;
  title: string;
}

const navListArr = [
  {
    id: 'APP',
    title: i18n.global.t('应用管理'),
    name: 'app',
  },
  {
    id: 'ENV',
    title: i18n.global.t('环境管理'),
    name: 'env',
  },
  // {
  //   id: 'COMPONENT',
  //   title: i18n.global.t('组件市场'),
  //   name: 'component',
  // },
  {
    id: 'PLUGIN',
    title: i18n.global.t('组件管理'),
    name: 'plugin',
  },
  {
    id: 'BASIC',
    title: i18n.global.t('空间设置'),
    name: 'basic',
  },
] as const;

export const getNavList = () => {
  const spaceStore = useSpaceStore();

  const navList = computed(() => {
    if (!spaceStore.currentSpace) return [];
    return navListArr.reduce<Array<NavItem>>((acc, item) => {
      acc.push({
        ...item,
        params: {
          space: spaceStore.currentSpace,
        },
      });
      return acc;
    }, []);
  });

  return navList;
};
