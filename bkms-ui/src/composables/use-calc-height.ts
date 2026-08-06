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

import type { Ref } from 'vue';
import { isRef, onBeforeUnmount, onMounted, ref } from 'vue';

import { throttle } from 'lodash-es';

type ElementType = HTMLElement | Ref<any> | string; // 节点

interface IConfig {
  calc?: ElementType | ElementType[]; // 要设置高度的元素
  el?: ElementType | ElementType[]; // 要设置高度的元素
  id?: number | string;
  offset?: number;
  prop?: 'height' | 'max-height';
}

/**
 * 动态计算元素高度
 * eg: useContentHeight({ el: { id: 'id1' }, els: { classes: ['class1', 'class2'], els: testRef } })
 * 上面案列表示: id元素高度 = 100vh - class1元素高度 - class2元素高度 - testRef元素高度
 * @param config
 * @returns
 */
export default function useContentHeight(config: IConfig | IConfig[]) {
  const style = ref<Record<string, any>>({});
  // 统一数据结构为数组
  const parseToArr = <T>(data?: T | T[]) => {
    if (!data) return [];

    return Array.isArray(data) ? data : [data];
  };

  // 统一dom数据格式
  const parseDomData = (el?: ElementType | ElementType[]): HTMLElement[] => {
    const data: ElementType[] = parseToArr<ElementType>(el);
    return data
      .map(el => {
        if (typeof el === 'string') {
          return document.querySelector(el);
        }
        if (isRef(el)) {
          return el.value instanceof HTMLElement ? el.value : (el.value as any)?.$el;
        }
        return el;
      })
      .filter(el => !!el);
  };

  // 设置内容高度
  const setContentHeight = (config: IConfig) => {
    const { prop } = config || {};

    const calcEleData = parseDomData(config.calc);
    if (!calcEleData.length) return;

    // 需要减去的元素高度
    const offset = calcEleData.reduce((pre, dom) => {
      pre += dom?.getBoundingClientRect()?.height || 0;
      return pre;
    }, config.offset || 0);

    const sty = {
      [prop || 'max-height']: `calc(100vh - ${offset}px)`,
    };

    if (config.id) {
      style.value[config.id] = sty;
    } else {
      style.value = sty;
    }
    // 设置元素高度
    const elData = parseDomData(config.el);
    elData.forEach(el => {
      Object.keys(sty).forEach(key => {
        el.style[key as any] = sty[key];
      });
    });
  };

  // 重新计算高度
  const init = () => {
    const configList = parseToArr(config);
    configList.forEach(item => setContentHeight(item));
  };

  onMounted(() => {
    const observer = new MutationObserver(
      throttle(
        () => {
          init();
        },
        300,
        {
          leading: false,
          trailing: true,
        },
      ),
    );

    observer.observe(document.body, {
      childList: true,
      attributes: true,
    });

    onBeforeUnmount(() => {
      observer.takeRecords();
      observer.disconnect();
    });
  });

  return {
    style,
    init,
  };
}
