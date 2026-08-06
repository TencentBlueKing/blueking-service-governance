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

import { useClipboard } from '@vueuse/core';
import { Message } from 'bkui-vue';
import { useI18n } from 'vue-i18n';

export function useCopy() {
  const { t } = useI18n();
  const { copy } = useClipboard({
    legacy: true, // 使用 execCommand 作为后备处理副本
  });
  async function copyText(value: string, successMessage?: string) {
    try {
      await copy(value);
      Message({
        message: successMessage || t('复制成功'),
        theme: 'success',
        delay: 1500,
      });
    } catch (_err) {
      Message({
        message: t('复制失败'),
        theme: 'error',
        delay: 1500,
      });
    }
  }

  return {
    copyText,
  };
}
