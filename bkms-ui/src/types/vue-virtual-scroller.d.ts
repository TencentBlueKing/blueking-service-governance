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

declare module 'vue-virtual-scroller' {
  import type { Component } from 'vue';

  type ItemKey = number | string;

  interface RecycleScrollerProps<T = unknown> {
    buffer?: number;
    class?: unknown;
    direction?: 'horizontal' | 'vertical';
    items?: T[];
    itemSecondarySize?: number;
    itemSize?: number;
    keyField?: string;
    listClass?: unknown;
    listTag?: string;
    minItemSize?: number;
    pageMode?: boolean;
    prerender?: number;
    skipHover?: boolean;
    style?: unknown;
    typeField?: string;
    viewClass?: unknown;
    viewTag?: string;
  }

  interface RecycleScrollerSlotProps<T = unknown> {
    active: boolean;
    index: number;
    item: T;
    itemKey: ItemKey;
  }

  interface RecycleScrollerInstance<T = unknown> {
    $props: RecycleScrollerProps<T>;
    $slots: {
      after?: () => unknown;
      before?: () => unknown;
      default?: (props: RecycleScrollerSlotProps<T>) => unknown;
      empty?: () => unknown;
    };
  }

  // vue-virtual-scroller@2.0.0-beta.8 does not ship type declarations.
  // The generic constructor lets Vue infer slot item types from `items` where possible.
  // eslint-disable-next-line @typescript-eslint/naming-convention
  export const RecycleScroller: new <T = unknown>() => Component & {
    $props: RecycleScrollerProps<T>;
    $slots: {
      [key: string]: (...args: unknown[]) => unknown;
      after?: () => unknown;
      before?: () => unknown;
      default: (props: RecycleScrollerSlotProps<T>) => unknown;
      empty?: () => unknown;
    };
  };
}
