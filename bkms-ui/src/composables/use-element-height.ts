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

import { effectScope, nextTick, onScopeDispose, ref, toValue, watch } from 'vue';
import type { MaybeRefOrGetter } from 'vue';

import { useResizeObserver } from '@vueuse/core';

interface UseElementHeightOptions {
  /** 默认高度 */
  defaultHeight?: number;
  /** 是否立即获取初始高度，默认 true */
  immediate?: boolean;
  /** 监听的数据源，当值从 true 变为 false 时重新获取高度（常用于 loading 状态） */
  watchSource?: MaybeRefOrGetter<boolean>;
}

/**
 * 监听元素高度，并在容器尺寸变化时自动更新
 *
 * 支持在组件内和组件外使用：
 * - 组件内：随组件销毁自动清理
 * - 组件外：调用返回的 stop() 手动清理
 *
 * @param target 目标元素的 ref
 * @param options 配置项
 * @returns 元素高度的响应式引用及控制方法
 */
export function useElementHeight(
  target: MaybeRefOrGetter<HTMLElement | null | undefined>,
  options: UseElementHeightOptions = {},
) {
  const { watchSource, defaultHeight = 0, immediate = true } = options;

  const height = ref(defaultHeight);

  const scope = effectScope();

  function updateHeight() {
    const el = toValue(target);
    if (el) {
      height.value = el.offsetHeight;
    }
  }

  scope.run(() => {
    // 监听容器尺寸变化
    useResizeObserver(target, () => {
      updateHeight();
    });

    // 监听指定属性变化
    if (watchSource) {
      watch(
        () => toValue(watchSource),
        (newVal, oldVal) => {
          if (oldVal && !newVal) {
            nextTick(updateHeight);
          }
        },
      );
    }

    // 立即获取初始高度
    if (immediate) {
      nextTick(updateHeight);
    }
  });

  function stop() {
    scope.stop();
  }

  // 若在组件/外层 scope 内调用，随其销毁自动清理
  onScopeDispose(stop);

  return {
    height,
    updateHeight,
    stop,
  };
}
