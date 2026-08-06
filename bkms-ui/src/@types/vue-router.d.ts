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

import 'vue-router';

import type { MenuId } from '~/composables/use-menu';

export {};

declare module 'vue-router' {
  import type { RouteLocationRaw } from 'vue-router';

  interface RouteMeta {
    layout?: 'content' | 'default' | 'empty' | 'main';
    menuId?: 'COMPONENT' | MenuId;
    title?: string;
  }

  interface Router {
    /**
     * 智能返回导航（覆写原始 back）
     * - 有浏览历史时使用 history.back()，体验更自然
     * - 无历史时优先使用 fallback > 自动推导 parent > history.back()
     * @param fallback 兜底路由，当无历史记录时跳转到此路径
     */
    back(fallback?: RouteLocationRaw): void;

    /**
     * 原始 back()，跳过智能返回逻辑，直接调用浏览器 history.back()
     * 仅在需要绕过智能返回的极少数场景下使用
     */
    originalBack(): void;
  }
}
