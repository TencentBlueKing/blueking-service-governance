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

import { computed, nextTick } from 'vue';

import useLeaveConfirm from '~/composables/use-leave-confirm';

import type { StrategyFormData } from './utils';

/**
 * 策略抽屉表单的「离开确认」增强。
 *
 * useLeaveConfirm 的入参类型为 Ref<T> | T，且内部用 `'value' in formModel` 判断是否为 Ref，
 * 因此：直接传 reactive 对象是安全的（无 value 属性 → 走 T 分支，deep watch 正常生效）；
 * 但传对象字面量 getter（{ get value() {} }）既不是 Ref 也不满足 T 的结构，
 * 会造成类型报错，且对 reactive 包装反而丢失响应式。这里用 computed 生成真正的 Ref，
 * 既满足类型约束，也让 hook 内部统一走 ref 分支。
 */
export default function useStrategyFormDirty(formModel: StrategyFormData) {
  const { confirmBox, forceCleanDirtyTag } = useLeaveConfirm<StrategyFormData>(computed(() => formModel));

  /**
   * 清空 dirty 标记：等待表单回填（fillForm）产生的深层变更全部触发后再清除。
   * forceCleanDirtyTag 只等一个 tick，而深层 watch 会按「最长路径」多等几个 tick，
   * 回填批次（嵌套数组）时单个 tick 不足以覆盖，因此这里用双 tick 兜底。
   */
  function resetDirty() {
    nextTick(() => {
      nextTick(() => forceCleanDirtyTag());
    });
  }

  return { confirmBox, resetDirty };
}
