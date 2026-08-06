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

import { h } from 'vue';

import { type AppInstanceOutputObj } from '~/@types/v1/instance';
import { i18n } from '~/modules/i18n';

const { t } = i18n.global;

/** 判断实例是否可以灰度（只有 Running 和 Pending 状态可以灰度） */
export function canInstanceGrayDeploy(instance: AppInstanceOutputObj): boolean {
  return instance.status === 'Running' || instance.status === 'Pending';
}

/** 登录：仅 Running 状态可登录 */
export function canLogin(instance: AppInstanceOutputObj): boolean {
  return instance.status === 'Running';
}

/** 判断实例的北极星状态是否全部健康 */
export function isPolarisHealthy(instance: AppInstanceOutputObj): boolean {
  return instance.polarisInfos?.every(p => p.isHealthy) ?? false;
}

/** 日志：Running、CrashLoopBackOff、Error、Completed、Succeeded 状态可查看日志 */
const LOG_ALLOWED_STATUSES = new Set(['Running', 'CrashLoopBackOff', 'Error', 'Completed', 'Succeeded']);
export function canViewLog(instance: AppInstanceOutputObj): boolean {
  return LOG_ALLOWED_STATUSES.has(instance.status!);
}

/** 渲染删除确认 InfoBox 内容 */
export function renderDeleteInfoBoxContent(selectedInstances: AppInstanceOutputObj[]) {
  if (selectedInstances.length === 1) {
    return h('div', [
      h('div', { class: 'py-[12px]' }, [t('实例: {0}', [`${selectedInstances[0]?.id} (${selectedInstances[0]?.ip})`])]),
      h('div', { class: 'bg-[#F5F7FA] py-[12px] px-[16px]' }, [t('此操作将删除该实例，并调整实例数')]),
    ]);
  }

  const instances = selectedInstances.map((item, i) =>
    h('div', { class: ['px-[16px] leading-[32px]', { 'bg-[#FAFBFD]': i % 2 > 0 }] }, [`${item.id} (${item.ip})`]),
  );

  return h('div', [
    h('div', { class: 'bg-[#F5F7FA] py-[12px] px-[16px]' }, [t('此操作将删除以下实例，并调整实例数')]),
    h(
      'div',
      {
        class:
          'bg-[#F5F7FA] leading-[32px] px-[16px] mt-[16px] border-1 border-[#EAEBF0] rounded-[2px] border-b-transparent',
      },
      [h('span', [t('已选择以下 {0} 个实例', [selectedInstances.length])])],
    ),
    h(
      'div',
      {
        class: 'max-h-[200px] overflow-auto border-1 border-t-transparent border-[#EAEBF0] rounded-[2px] p-[12px]',
      },
      [...instances],
    ),
  ]);
}
