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

import { appNavigationConfig } from '~/config/navigation/app';
import { BASIC_NAVIGATION } from '~/config/navigation/basic';
import { PLATFORM_NAVIGATION } from '~/config/navigation/platform';
import { PLUGIN_NAVIGATION } from '~/config/navigation/plugin';
import { useSpaceStore } from '~/stores/space';

import type { AppNavigationType } from '~/config/navigation/app';
import type { NavigationItem } from '~/config/navigation/types';

// 根据 menuId 获取对应的菜单列表
export type MenuIdType = 'APP' | 'BASIC' | 'PLATFORM' | 'PLUGIN';

// 获取空间设置导航菜单
export function getBasicMenuList(): NavigationItem[] {
  const spaceStore = useSpaceStore();

  if (!spaceStore.currentSpace) return [];
  return BASIC_NAVIGATION;
}

// 获取应用管理导航菜单（指定type）
export function getMenuList(type: AppNavigationType): NavigationItem[] {
  const spaceStore = useSpaceStore();

  if (!spaceStore.currentSpace || !type) return [];
  return appNavigationConfig[type] || [];
}

// 获取平台管理导航菜单
export function getPlatformMenuList(): NavigationItem[] {
  return PLATFORM_NAVIGATION;
}

// 获取组件管理导航菜单
export function getPluginMenuList(): NavigationItem[] {
  const spaceStore = useSpaceStore();

  if (!spaceStore.currentSpace) return [];
  return PLUGIN_NAVIGATION;
}

const menuGetterMap: Record<MenuIdType, (type?: AppNavigationType) => NavigationItem[]> = {
  APP: (type?: AppNavigationType) => (type ? getMenuList(type) : []),
  BASIC: () => getBasicMenuList(),
  PLATFORM: () => getPlatformMenuList(),
  PLUGIN: () => getPluginMenuList(),
};

export function getMenuListByMenuId(menuId: MenuIdType, type?: AppNavigationType): NavigationItem[] {
  const getter = menuGetterMap[menuId];
  return getter ? getter(type) : [];
}
