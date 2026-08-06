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

import { type ComputedRef, type Ref } from 'vue';

import { InfoBox, Message } from 'bkui-vue';
import { useI18n } from 'vue-i18n';
import { type AppInstanceOutputObj } from '~/@types/v1/instance';
import { InstanceService } from '~/api/modules/v1';
import { useTabManager } from '~/composables/use-tab-manager';
import { useAppDetail } from '~/stores/app-detail';

import { renderDeleteInfoBoxContent } from '../instance-utils';
import { provideInstanceActionContext } from './use-instance-action-context';

import type { default as InstanceActionsHost } from '../components/instance-actions-host.vue';
import type { InstanceActionContext, InstanceRowAction } from '../types';

export interface UseInstanceActionsOptions extends InstanceActionContext {
  /** 是否选中了所有实例 */
  isAllInstancesSelected: ComputedRef<boolean>;
  /** 跨页全选支持（单环境特有） */
  isCrossPageSelection?: Ref<boolean>;
  /** 选中实例数量 */
  selectedCount: ComputedRef<number> | Ref<number>;
  /** 管理命令侧栏环境展示名 */
  commandEnvDisplayName?: () => string;
  /** 灰度弹窗环境展示名（单环境） */
  grayEnvDisplayName?: () => string;
  /** 计算灰度提交的 instanceIds（单环境跨页全选） */
  resolveGrayInstanceIds?: () => string[] | undefined;
}

export function useInstanceActions(
  options: UseInstanceActionsOptions,
  actionsHostRef: Ref<InstanceType<typeof InstanceActionsHost> | null>,
) {
  const { login } = useLoginInstance();
  const { showDeleteDialog } = useDeleteInstances(options);

  provideInstanceActionContext({
    getEnvName: options.getEnvName,
    getSelectedInstances: options.getSelectedInstances,
    clearSelections: options.clearSelections,
    refreshData: options.refreshData,
    timer: options.timer,
  });

  function openGray(instance?: AppInstanceOutputObj, envName?: string) {
    const resolvedEnvName = envName || options.getEnvName({ instance });
    const instances = instance ? [instance] : options.getSelectedInstances();
    const instanceIds = instance ? undefined : options.resolveGrayInstanceIds?.();

    actionsHostRef.value?.openGray({
      envName: resolvedEnvName,
      envDisplayName: options.grayEnvDisplayName?.(),
      instances,
      instanceIds,
    });
  }

  function openWeight(instance: AppInstanceOutputObj, envName?: string) {
    const resolvedEnvName = envName || options.getEnvName({ instance });
    actionsHostRef.value?.openWeight(instance, resolvedEnvName);
  }

  function openLog(instance: AppInstanceOutputObj, envName?: string) {
    const resolvedEnvName = envName || options.getEnvName({ instance });
    actionsHostRef.value?.openLog(instance, resolvedEnvName);
  }

  function openAdminCommand() {
    const instanceIds = options.getSelectedInstances().map(item => item.id) as string[];
    actionsHostRef.value?.openCommand(instanceIds, options.commandEnvDisplayName?.());
  }

  function openMonitor(initialInstances?: string[], envName?: string, envNames?: string[]) {
    const resolvedEnvName = envName || options.getEnvName();
    const instanceIds = initialInstances
      ? initialInstances
      : options
          .getSelectedInstances()
          .map(item => item.id)
          .filter((id): id is string => !!id);
    const selection = instanceIds.length > 0 ? { [resolvedEnvName]: instanceIds } : undefined;
    actionsHostRef.value?.openMonitor(selection, resolvedEnvName, envNames);
  }

  function openDelete() {
    showDeleteDialog(options.getSelectedInstances, () => options.getEnvName());
  }

  async function handleLogin(instance: AppInstanceOutputObj, envName?: string) {
    const resolvedEnvName = envName || options.getEnvName({ instance });
    await login(instance, resolvedEnvName);
  }

  /** 统一行操作分发 */
  function handleRowAction(payload: {
    action: InstanceRowAction | string;
    envName: string;
    instance: AppInstanceOutputObj;
  }) {
    const { action, envName, instance } = payload;
    if (!(['gray', 'log', 'login', 'monitor', 'weight'] as InstanceRowAction[]).includes(action as InstanceRowAction)) {
      return;
    }

    switch (action as InstanceRowAction) {
      case 'gray':
        openGray(instance, envName);
        break;
      case 'log':
        openLog(instance, envName);
        break;
      case 'login':
        handleLogin(instance, envName);
        break;
      case 'monitor':
        openMonitor(instance.id ? [instance.id] : undefined, envName);
        break;
      case 'weight':
        openWeight(instance, envName);
        break;
    }
  }

  return {
    handleLogin,
    handleRowAction,
    openAdminCommand,
    openDelete,
    openGray,
    openLog,
    openMonitor,
    openWeight,
  };
}

/** 弹出删除确认 InfoBox 并执行批量删除 */
function useDeleteInstances(context: InstanceActionContext) {
  const { t } = useI18n();
  const appDetailStore = useAppDetail();

  function showDeleteDialog(getSelectedInstances: () => AppInstanceOutputObj[], getEnvName: () => string) {
    const selectedInstances = getSelectedInstances();
    const envName = getEnvName();
    if (!envName) return;

    InfoBox({
      title: `${t('确认删除该实例')}?`,
      content: renderDeleteInfoBoxContent(selectedInstances),
      headerAlign: 'center',
      footerAlign: 'center',
      contentAlign: 'left',
      confirmButtonTheme: 'danger',
      confirmText: t('删除'),
      cancelText: t('取消'),
      async onConfirm() {
        const instanceIds = selectedInstances.map(instance => instance.id) as string[];
        const result = await InstanceService.batchDeleteAppInstances(
          {
            appID: appDetailStore.appID,
            envName,
            instanceIDs: instanceIds,
          },
          { isBodyParam: true },
        )
          .then(() => true)
          .catch(() => false);

        if (result) {
          Message({
            theme: 'success',
            message: t('操作成功'),
          });
          context.timer?.stop();
          context.clearSelections();
          await context.refreshData();
          context.timer?.start();
        }
      },
    });
  }

  return { showDeleteDialog };
}

/** 登录实例 */
function useLoginInstance() {
  const appDetailStore = useAppDetail();
  const { openTab, isTabOpen } = useTabManager();

  async function login(instance: AppInstanceOutputObj, envName: string) {
    const instanceKey = `${instance.ip}-${instance.id}`;
    if (isTabOpen(instanceKey)) {
      await openTab(instanceKey);
      return;
    }
    try {
      const res = await InstanceService.createAppInstanceWebConsole({
        appID: appDetailStore.appID,
        envName,
        instanceID: instance?.id ?? '',
      });
      if (!res?.url) {
        return;
      }
      await openTab(res.url, instanceKey);
    } catch (error) {
      console.error(error);
    }
  }

  return { login };
}
