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

import { onMounted, onUnmounted, ref } from 'vue';

import { debounce } from 'lodash-es';

export default function useDynamicsHeight(offsetHeight: number, influenceClassNames: string[] = []) {
  const maxHeight = ref<number>(0);
  const observersMap = new Map<Element, ResizeObserver>();

  // 根据影响高度的DOM元素计算最大高度
  const calcHeight = debounce(() => {
    const influenceDomsHeight = influenceClassNames?.reduce((total, className) => {
      const el = document.querySelector(className);
      return (el?.clientHeight ?? 0) + total;
    }, 0);
    maxHeight.value = window.innerHeight - offsetHeight - influenceDomsHeight;
  }, 100);

  // 初始化观察器
  const initObserversMap = () => {
    // 先清除所有现有观察器
    observersMap.forEach(observer => observer.disconnect());
    observersMap.clear();

    // 观察窗口大小变化
    createNewObserver(document.body);

    // 观察影响高度的DOM元素
    influenceClassNames?.forEach(className => {
      const elements = document.querySelectorAll(className);
      elements.forEach(el => createNewObserver(el));
    });
  };

  // 创建新的观察器
  const createNewObserver = (element: Element) => {
    const observer = new ResizeObserver(calcHeight);
    observer.observe(element);
    observersMap.set(element, observer);
  };

  // 刷新高度计算（供外部调用）
  const refresh = () => {
    calcHeight();
    initObserversMap();
  };

  onMounted(() => {
    refresh();
  });

  onUnmounted(() => {
    observersMap.forEach(observer => observer.disconnect());
    observersMap.clear();
  });

  return {
    maxHeight,
    refresh,
  };
}
