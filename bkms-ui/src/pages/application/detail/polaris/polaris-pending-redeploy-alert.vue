<!--
 - TencentBlueKing is pleased to support the open source community by making
 - 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 - Copyright (C) Tencent. All rights reserved.
 - Licensed under the MIT License (the "License"); you may not use this file except
 - in compliance with the License. You may obtain a copy of the License at
 -
 -  http://opensource.org/licenses/MIT
 -
 - Unless required by applicable law or agreed to in writing, software distributed under
 - the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 - either express or implied. See the License for the specific language governing permissions and
 - limitations under the License.
 -
 - We undertake not to change the open source license (MIT license) applicable
 - to the current version of the project delivered to anyone in the future.
-->

<template>
  <div
    v-if="pendingRedeployGroups.length"
    class="pending-redeploy-panel mb-[16px] overflow-hidden rounded-[2px] border border-[#F9D090]"
  >
    <div
      class="flex cursor-pointer items-center justify-between bg-[#FDF4E8] px-[16px] py-[8px]"
      @click="isExpanded = !isExpanded"
    >
      <div class="flex min-w-0 items-center">
        <i class="bkms-icon bkms-icon-triangle-warning shrink-0 text-[14px] text-[#F59500]"></i>
        <span class="ml-[8px] text-[12px] font-bold text-[#F59500]">
          {{ $t('北极星配置待重新部署生效') }}
        </span>
        <span
          class="ml-[8px] shrink-0 rounded-[2px] bg-[#FCE5C0] px-[8px] py-[1px] text-[12px] leading-[18px] text-[#E38B02]"
        >
          <i18n-t keypath="涉及 {0} 个环境">
            <span>{{ pendingRedeployEnvCount }}</span>
          </i18n-t>
        </span>
      </div>
      <div class="ml-[16px] flex shrink-0 items-center">
        <Button
          :loading="refreshing"
          text
          theme="primary"
          @click.stop="handleRefresh"
          @mousedown.stop
        >
          <i class="bkms-icon bkms-icon-refresh mr-[2px]"></i>
          {{ $t('刷新') }}
        </Button>
        <AngleDown
          :class="[
            'ml-[8px] text-[22px] text-[#979BA5] transition-transform duration-200',
            { 'rotate-180': isExpanded },
          ]"
        />
      </div>
    </div>

    <div
      v-if="isExpanded"
      class="bg-[#fff] px-[16px] pb-[4px]"
    >
      <div class="pending-redeploy-grid border-b border-[#F0E3C8] py-[8px] text-[12px] text-[#979BA5]">
        <span>{{ $t('待部署环境') }}</span>
        <span>{{ $t('北极星服务名') }}</span>
        <span>{{ $t('修改内容') }}</span>
        <span>{{ $t('操作') }}</span>
      </div>
      <div
        v-for="group in pendingRedeployGroups"
        :key="group.envName"
        class="pending-redeploy-grid border-b border-[#F5F7FA] last:border-b-0"
      >
        <div class="flex min-w-0 items-center">
          <OverflowTitle
            class="min-w-0 text-[12px] text-[#313238]"
            :content="group.envDisplayName"
            placement="top-start"
            resizeable
            type="tips"
          />
          <Tag
            v-if="group.envType && envTypeMap[group.envType]"
            :class="['ml-[6px] shrink-0', envTypeTagClassMap[group.envType]]"
            size="small"
          >
            {{ envTypeMap[group.envType].name }}
          </Tag>
        </div>
        <div class="pending-redeploy-items min-w-0">
          <div
            v-for="item in group.items"
            :key="item.configName"
            class="pending-redeploy-item-grid border-b border-[#F5F7FA] py-[10px] last:border-b-0"
          >
            <div class="w-full min-w-0 overflow-hidden">
              <OverflowTitle
                :key="item.polarisName || '--'"
                class="w-full min-w-0 text-[12px] text-[#313238]"
                :content="item.polarisName || '--'"
                placement="top-start"
                resizeable
                type="tips"
              />
            </div>
            <div class="min-w-0 text-[12px] leading-[20px] text-[#979BA5]">
              <template v-if="item.changes.length">
                <div
                  v-for="change in item.changes"
                  :key="change.key"
                >
                  {{ change.label }}
                  <template v-if="change.oldValue !== undefined">
                    {{ formatRedeployValue(change.oldValue) }}
                    <span class="mx-[4px] text-[#C4C6CC]">→</span>
                  </template>
                  <span class="font-medium text-[#313238]">{{ formatRedeployValue(change.newValue) }}</span>
                </div>
              </template>
              <span v-else>
                {{
                  isPendingDeleteStatus(item.status)
                    ? $t('解除关联（重新部署后实例将从北极星摘除）')
                    : $t('北极星配置待重新部署生效')
                }}
              </span>
            </div>
          </div>
        </div>
        <Button
          class="pending-redeploy-action !px-0 justify-self-start"
          :disabled="isEnvDeploying(group.envName)"
          text
          theme="primary"
          @click="emit('go-deploy', group.envName)"
        >
          {{ isEnvDeploying(group.envName) ? $t('部署中') : $t('去部署') }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref } from 'vue';

  import { Button, OverflowTitle, Tag } from 'bkui-vue';
  import { AngleDown } from 'bkui-vue/lib/icon';
  import { PolarisConfigOutputObj } from '~/@types/v1/polaris-config';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';

  import { formatPolarisRedeployValue, getPolarisRedeployChanges, isImmediateRegister } from './redeploy-utils';

  import type { PolarisRedeployChange } from './redeploy-utils';
  import type { EnvOutput } from '~/@types/v1/env';

  interface PendingRedeployGroup {
    envDisplayName: string;
    envName: string;
    envType?: string;
    items: PendingRedeployItem[];
  }

  interface PendingRedeployItem {
    changes: PolarisRedeployChange[];
    configName: string;
    polarisName?: string;
    status: string;
  }

  type PolarisEnvStateWithStatus = NonNullable<PolarisConfigOutputObj['envStates']>[string] & {
    status?: string;
  };

  const props = defineProps<{
    configs: PolarisConfigOutputObj[];
    deployingEnvNames: string[];
    envList: EnvOutput[];
    refreshing: boolean;
  }>();

  const emit = defineEmits<{
    (e: 'go-deploy', envName: string): void;
    (e: 'refresh'): void;
  }>();

  const isExpanded = ref(true);
  const deployingEnvNameSet = computed(() => new Set(props.deployingEnvNames));

  /** 顶部待部署提示数据：基于完整配置列表按环境分组，避免被搜索筛选隐藏。 */
  const pendingRedeployGroups = computed<PendingRedeployGroup[]>(() => {
    const groups = new Map<string, PendingRedeployGroup>();
    props.configs.forEach((config, configIndex) => {
      if (isImmediateRegister(config)) return;

      Object.entries(config.envStates || {}).forEach(([envName, state]) => {
        const status = getPolarisEnvStatus((state as PolarisEnvStateWithStatus).status);
        if (!isPendingRedeployStatus(status)) return;

        const envInfo = getEnvInfo(envName);
        const group = groups.get(envName) || {
          envDisplayName: envInfo.displayName,
          envName,
          envType: envInfo.envType,
          items: [],
        };
        group.items.push({
          changes: getPolarisRedeployChanges(config, envName),
          configName: config.name || String(configIndex),
          polarisName: config.polarisName,
          status,
        });
        groups.set(envName, group);
      });
    });

    return Array.from(groups.values());
  });

  /** 顶部提示数量按环境统计，而不是按北极星配置数量统计。 */
  const pendingRedeployEnvCount = computed(() => pendingRedeployGroups.value.length);

  /** 统一复用北极星待部署变更值格式化逻辑，供顶部提示表格渲染。 */
  function formatRedeployValue(value?: number | string) {
    return formatPolarisRedeployValue(value);
  }

  /** 根据环境名补齐环境展示名和类型，接口无环境详情时回退展示原环境名。 */
  function getEnvInfo(envName: string) {
    const env = props.envList.find(item => item.name === envName);
    return {
      displayName: env?.displayName || envName,
      envType: env?.type,
    };
  }

  /** 兼容 envStates.status 的大小写和空格差异。 */
  function getPolarisEnvStatus(status?: string) {
    return status?.trim().toLowerCase() || '';
  }

  /** 手动刷新时阻止重复请求。 */
  function handleRefresh() {
    if (props.refreshing) return;
    emit('refresh');
  }

  /** 同一环境存在任意部署中记录时，禁用该环境的重复部署入口。 */
  function isEnvDeploying(envName: string) {
    return deployingEnvNameSet.value.has(envName);
  }

  /** pendingDelete 表示 scope 已置空，重新部署后会从北极星摘除实例。 */
  function isPendingDeleteStatus(status?: string) {
    return getPolarisEnvStatus(status) === 'pendingdelete';
  }

  /** 非空状态且不是 deployed，就认为需要重新部署后生效。 */
  function isPendingRedeployStatus(status?: string) {
    const normalizedStatus = getPolarisEnvStatus(status);
    return !!normalizedStatus && normalizedStatus !== 'deployed';
  }
</script>

<style lang="postcss" scoped>
  .pending-redeploy-grid {
    display: grid;
    grid-template-columns: 200px minmax(220px, 1fr) minmax(280px, 1.2fr) 64px;
    column-gap: 16px;
    align-items: center;
  }

  .pending-redeploy-items {
    grid-column: 2 / 4;
  }

  .pending-redeploy-action {
    grid-column: 4;
  }

  .pending-redeploy-item-grid {
    display: grid;
    grid-template-columns: minmax(220px, 1fr) minmax(280px, 1.2fr);
    column-gap: 16px;
    align-items: center;
  }
</style>
