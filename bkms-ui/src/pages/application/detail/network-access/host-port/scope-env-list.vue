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
  <div class="flex flex-wrap gap-x-[24px] gap-y-[8px]">
    <div
      v-for="env in scopeEnvItems"
      :key="env.name"
      class="inline-flex items-center h-[22px] text-[12px] text-[#4D4F56]"
    >
      <span class="max-w-[160px] truncate">{{ env.displayName }}</span>
      <Tag
        v-if="env.type && envTypeMap[env.type]"
        :class="['ml-[4px] shrink-0', envTypeTagClassMap[env.type]]"
        size="small"
      >
        {{ envTypeMap[env.type].name }}
      </Tag>
      <Popover
        v-if="env.pendingChanges"
        :popover-delay="[100, 100]"
        theme="popover-dark-translucent"
        trigger="hover"
      >
        <span class="ml-[4px] cursor-pointer text-[#F59500]">（{{ $t('待部署生效') }}）</span>
        <template #content>
          <div class="max-w-[520px] text-[12px] leading-[20px] text-[#fff]">
            <div class="mb-[6px]">{{ $t('您修改了以下内容，需要重新部署应用后方可生效：') }}</div>
            <div
              v-for="change in env.pendingChanges"
              :key="change"
              class="pl-[12px]"
            >
              <span>•&nbsp;{{ change }}</span>
            </div>
            <Button
              v-if="showGoDeploy"
              class="mt-[6px] !px-0 text-[#3A84FF]"
              text
              theme="primary"
              @click.stop="$emit('goDeploy', env.name)"
            >
              <Share class="mr-[4px]" />
              {{ $t('去部署') }}
            </Button>
          </div>
        </template>
      </Popover>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { Button, Popover, Tag } from 'bkui-vue';
  import { Share } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';

  import { hasEnvPendingChange } from './host-port-utils';

  import type { HostPortEnvStates, ScopeEnv } from './host-port-utils';
  import type { HostPortEnvStateOutput } from '~/@types/v1/hostport';

  /** 带预计算待部署文案的生效范围项 */
  interface ScopeEnvItem extends ScopeEnv {
    /** 有待部署时为文案列表；否则为 null（不展示气泡） */
    pendingChanges: null | string[];
  }

  const props = withDefaults(
    defineProps<{
      /** 环境列表 */
      envs: ScopeEnv[];
      /** 后端返回的联邦环境待部署状态；key 即为生效范围环境名 */
      envStates: HostPortEnvStates;
      showGoDeploy?: boolean;
      /** 查看态展示待部署标记；编辑态只列生效环境 */
      showPending?: boolean;
    }>(),
    { showGoDeploy: true, showPending: true },
  );

  defineEmits<{
    goDeploy: [envName: string];
  }>();

  const { t } = useI18n();

  /** 预计算 pendingChanges，模板只遍历一次，避免 has + get */
  const scopeEnvItems = computed<ScopeEnvItem[]>(() =>
    props.envs.map(env => {
      const state = props.envStates[env.name];
      const pendingChanges =
        props.showPending && state && hasEnvPendingChange(state) ? formatPendingChanges(state) : null;
      return { ...env, pendingChanges };
    }),
  );

  /** 将 pendingAdd / pendingRemove 转成气泡内展示文案（二者独立，不拼成「修改」） */
  function formatPendingChanges(state: HostPortEnvStateOutput) {
    const added = state.pendingAddPorts || [];
    const removed = state.pendingRemovePorts || [];
    const changes: string[] = [];
    if (added.length > 0) changes.push(t('新增端口：{ports}', { ports: added.join('、') }));
    if (removed.length > 0) changes.push(t('删除端口：{ports}', { ports: removed.join('、') }));
    return changes;
  }
</script>
