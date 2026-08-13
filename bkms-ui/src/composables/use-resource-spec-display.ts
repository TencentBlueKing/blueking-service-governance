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

import { useI18n } from 'vue-i18n';

/** 资源规格展示所需字段，部署概览行与默认规则 spec 均可传入 */
export interface ResourceSpecDisplay {
  cpuLimits?: string;
  cpuRequests?: string;
  memoryLimits?: string;
  memoryRequests?: string;
}

/** 资源规格列展示：简要文案 + hover 完整 Requests / Limits */
export function useResourceSpecDisplay() {
  const { t } = useI18n();

  /** 生成资源规格列的简要文案；CPU 和内存均缺失时返回空字符串以展示“--”。 */
  function getResourceText(spec?: null | ResourceSpecDisplay) {
    if (!spec?.cpuLimits && !spec?.memoryLimits) return '';
    return t('{0} 核 / {1}', [spec?.cpuLimits || '--', spec?.memoryLimits || '--']);
  }

  /** 生成资源规格悬浮提示，完整展示 Requests 与 Limits。 */
  function getResourceTips(spec?: null | ResourceSpecDisplay) {
    return [
      t('CPU：Requests {0} 核 / Limits {1} 核', [spec?.cpuRequests || '--', spec?.cpuLimits || '--']),
      t('内存：Requests {0} / Limits {1}', [spec?.memoryRequests || '--', spec?.memoryLimits || '--']),
    ];
  }

  return {
    getResourceText,
    getResourceTips,
  };
}
