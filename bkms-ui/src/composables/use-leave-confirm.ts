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

import { nextTick, ref, watch } from 'vue';
import type { Ref } from 'vue';

import { InfoBox } from 'bkui-vue';
import { useI18n } from 'vue-i18n';
interface ConfirmBoxOptions {
  // 自定义验证函数，返回 true 表示需要弹窗确认
  validates?: Array<() => boolean>;
}

/**
 * 离开确认 Hook - 监听表单是否被用户触发过
 * @param formModel 表单数据（支持响应式数据）
 * @returns confirmBox 方法、isDirty 标记、withPausedWatch 回调方法、forceCleanDirtyTag 方法
 *
 * @example 场景1：组件内初始化（推荐使用 withPausedWatch）
 * const { confirmBox, withPausedWatch } = useLeaveConfirm(formModel);
 *
 * // 编辑态初始化时暂停监听，避免触发 watch
 * withPausedWatch(() => {
 *   formModel.value = { ...apiData };
 * });
 *
 * // 关闭前检查
 * await confirmBox();
 *
 * @example 场景2：跨组件数据传递（推荐使用 forceCleanDirtyTag）
 * const { confirmBox, forceCleanDirtyTag } = useLeaveConfirm(formModel);
 *
 * // 侧栏打开时，父组件通过 v-model 传入数据
 * watch(isShow, val => {
 *   if (val) {
 *     init(); // 初始化可能触发 watch
 *     forceCleanDirtyTag(); // 统一清除初始化产生的 dirty 标记
 *   }
 * });
 */
export default function useLeaveConfirm<T extends Record<string, any> = any>(formModel?: Ref<T> | T) {
  const { t } = useI18n();

  // 标记表单是否被用户触发过
  const isDirty = ref(false);

  // 标记是否正在初始化（用于跳过初始化时的 watch 触发）
  const initializing = ref(false);

  if (formModel) {
    // 获取表单值
    const getFormValue = (): T => ('value' in formModel ? formModel.value : formModel);

    watch(
      () => getFormValue(),
      () => {
        // 初始化期间不设置 dirty
        if (initializing.value) {
          return;
        }
        isDirty.value = true;
      },
      { deep: true },
    );
  }

  /**
   * 在暂停 watch 的保护下执行回调
   * @param callback 回调函数
   * @description 推荐场景：组件内初始化，需要精确控制哪些赋值不触发 watch
   */
  function withPausedWatch(callback: () => void) {
    initializing.value = true;
    callback();
    nextTick(() => {
      initializing.value = false;
    });
  }

  /**
   * 强制清除 dirty 标记（在 nextTick 清除）
   * @description 推荐场景：侧栏/弹窗打开时初始化，允许 watch 触发后再清除标记，保存表单后离开
   */
  function forceCleanDirtyTag(callback?: () => void) {
    nextTick(() => {
      isDirty.value = false;
      callback?.();
    });
  }

  /**
   * 离开确认弹窗
   * @param options 配置项
   * @returns Promise<boolean> true 表示确认离开，false 表示取消
   */
  function confirmBox(useDirtyTag = true, options?: ConfirmBoxOptions): Promise<boolean> {
    return new Promise<boolean>(resolve => {
      // 判断是否需要弹窗
      let needConfirm = false;

      if (options?.validates && options.validates.length > 0) {
        // 使用 validates 数组验证
        const allValidatesPassed = options.validates.every(validate => !validate());
        if (useDirtyTag) {
          // useDirtyTag 为 true：需要满足 isDirty && validates 全部通过
          needConfirm = isDirty.value || allValidatesPassed;
        } else {
          // useDirtyTag 为 false：只看 validates 是否全部通过
          needConfirm = allValidatesPassed;
        }
      } else {
        // 没有 validates 时，使用默认 isDirty 判断
        needConfirm = useDirtyTag && isDirty.value;
      }

      // 如果不需要确认，直接返回 true
      if (!needConfirm) {
        resolve(true);
        return;
      }

      // 弹出确认框
      InfoBox({
        title: `${t('确认离开当前页')}？`,
        extCls: 'leave-confirm-box-index',
        content: t('离开将会导致未保存信息丢失'),
        confirmText: t('离开'),
        cancelText: t('取消'),
        onCancel: () => resolve(false),
        onConfirm: () => {
          isDirty.value = false;
          resolve(true);
        },
      });
    });
  }

  return {
    confirmBox,
    withPausedWatch,
    forceCleanDirtyTag,
  };
}
