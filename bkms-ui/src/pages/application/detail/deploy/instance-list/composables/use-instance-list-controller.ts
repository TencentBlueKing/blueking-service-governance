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

import { type ComputedRef, type Ref, computed } from 'vue';

import useInterval from '~/composables/use-interval';

import { canInstanceGrayDeploy } from '../instance-utils';
import { type UseInstanceActionsOptions, useInstanceActions } from './use-instance-actions';

import type InstanceActionsHost from '../components/instance-actions-host.vue';
import type { InstanceRowActionPayload } from '../types';

interface UseInstanceListControllerOptions
  extends Omit<UseInstanceActionsOptions, 'isAllInstancesSelected' | 'selectedCount' | 'timer'> {
  actionsHostRef: Ref<InstanceType<typeof InstanceActionsHost> | null>;
  isAllInstancesSelected: ComputedRef<boolean>;
  pollInterval: number;
  selectedCount: ComputedRef<number> | Ref<number>;
  beforeRowAction?: (payload: InstanceRowActionPayload) => void;
}

// 统一收口实例列表的批量操作、轮询和行操作处理。
export function useInstanceListController(options: UseInstanceListControllerOptions) {
  const { actionsHostRef, beforeRowAction, isAllInstancesSelected, pollInterval, selectedCount, ...actionOptions } =
    options;

  const canGrayDeploy = computed(() => {
    const selections = actionOptions.getSelectedInstances();
    return selections.length > 0 && selections.every(instance => canInstanceGrayDeploy(instance));
  });

  const { start, stop, timer } = useInterval(async () => {
    await Promise.resolve(actionOptions.refreshData());
  }, pollInterval);

  const instanceActions = useInstanceActions(
    {
      ...actionOptions,
      isAllInstancesSelected,
      selectedCount,
      timer: { start, stop },
    },
    actionsHostRef,
  );

  // 在统一入口里执行行操作前置逻辑并分发动作。
  function handleRowAction(payload: InstanceRowActionPayload) {
    beforeRowAction?.(payload);
    instanceActions.handleRowAction(payload);
  }

  return {
    canGrayDeploy,
    handleRowAction,
    instanceActions,
    timer,
    startPolling: start,
    stopPolling: stop,
  };
}
