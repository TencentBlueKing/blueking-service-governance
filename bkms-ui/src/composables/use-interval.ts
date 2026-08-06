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

import { type Ref, getCurrentInstance, onBeforeUnmount, onDeactivated, onUnmounted, ref } from 'vue';

export type Fn = () => void;

export interface ITimeoutFnResult {
  isPending: Ref<boolean>;
  start: Fn;
  stop: Fn;
  timer: Ref<null | number>;
}

/**
 * 轮询
 * @param cb 回调
 * @param interval 轮询周期（支持数字或 Ref，动态更新时外部只需改 ref 值后重新 start）
 * @param immediate 立即执行
 */
export default function useIntervalFn(
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  cb: (...args: any[]) => Promise<void>,
  interval: number | Ref<number> = 5000,
  immediate = false,
): ITimeoutFnResult {
  const isPending = ref(false);
  const flag = ref(false);
  const timer = ref<null | number>(null);
  /** 内部 ref：传数字也转为 ref，保证 start() 始终读取最新值 */
  const intervalRef = typeof interval === 'number' ? ref(interval) : interval;

  const instance = getCurrentInstance();

  // 清空轮询
  function clear() {
    if (timer.value) {
      clearTimeout(timer.value);
      timer.value = null;
    }
  }
  // 停止轮询
  function stop() {
    isPending.value = false;
    flag.value = false;
    clear();
  }
  // 开始轮询
  function start(...args: unknown[]) {
    // 若此时组件已卸载，不开启轮询(异步调用场景)
    if (instance?.isUnmounted) return;
    clear();
    const ms = intervalRef.value;
    if (!ms) return;

    flag.value = true;
    async function timerFn() {
      // 上一个接口未执行完，不执行本次轮询
      if (isPending.value || !flag.value) return;

      isPending.value = true;
      await cb(...args);
      isPending.value = false;
      if (flag.value) {
        // eslint-disable-next-line @typescript-eslint/no-misused-promises
        timer.value = setTimeout(timerFn, ms) as unknown as number;
      }
    }
    // eslint-disable-next-line @typescript-eslint/no-misused-promises
    timer.value = setTimeout(() => timerFn(), immediate ? 0 : ms) as unknown as number;
  }

  if (getCurrentInstance()) {
    onBeforeUnmount(stop);
    onUnmounted(stop);
    onDeactivated(stop);
  }

  return {
    isPending,
    timer,
    start,
    stop,
  };
}
