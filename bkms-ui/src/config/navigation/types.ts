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

import type { Component } from 'vue';

/**
 * 基础导航菜单项
 */
export interface BaseNavigationItem {
  /** 菜单对应的组件 */
  component?: Component;
  /** 是否禁用 */
  disabled?: boolean;
  /** 菜单图标 */
  icon?: string;
  /** 菜单唯一标识 */
  key: string;
  /** 菜单名称 */
  name: string;
  /** 额外的元数据 */
  meta?: {
    /** 自定义 class */
    class?: string;
    /** 布局类型：default（带默认 header 布局） | empty（无 header 布局），不配置默认为 default */
    layout?: 'default' | 'empty';
  };
}

/**
 * 导航配置项（联合类型）
 */
export type NavigationItem = BaseNavigationItem | NavigationGroup | NavigationSub;

/**
 * 分组导航菜单项
 */
interface NavigationGroup {
  /** 子菜单项 */
  children: BaseNavigationItem[];
  /** 折叠时显示的名称 */
  foldName: string;
  /** 分组唯一标识 */
  key: string;
  /** 分组名称 */
  name: string;
}

/**
 * 子导航菜单项
 */
interface NavigationSub {
  /** 子菜单项 */
  children: BaseNavigationItem[];
  /** 子导航唯一标识 */
  key: string;
  /** 子导航标题 */
  title: string;
}
