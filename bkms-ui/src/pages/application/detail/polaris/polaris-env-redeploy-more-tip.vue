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
  <div class="max-w-[620px] text-[12px] leading-[20px] text-[#fff]">
    <div
      v-for="env in displayEnvs"
      :key="env.name"
      class="mb-[8px]"
    >
      <div class="flex items-center gap-[8px] text-[14px] leading-[24px]">
        <span class="font-bold">•&nbsp;{{ env.displayName }}</span>
        <span
          v-if="env.changes.length"
          class="font-bold text-[#F59500]"
        >
          ({{ $t('待部署生效') }})
        </span>
        <Button
          v-if="env.changes.length"
          class="!px-0 text-[#3A84FF] ml-[8px] !text-[12px]"
          text
          theme="primary"
          @click.stop="emit('go-deploy', env.name)"
        >
          <Share class="mr-[4px]" />
          {{ $t('去部署') }}
        </Button>
      </div>
      <div
        v-for="change in env.changes"
        :key="change.key"
        class="pl-[26px]"
      >
        <span>{{ change.label }}：</span>
        <template v-if="change.oldValue === undefined">
          <span class="font-bold">{{ formatValue(change.newValue) }}</span>
        </template>
        <template v-else>
          <span>{{ formatValue(change.oldValue) }}</span>
          <span class="mx-[6px]">→</span>
          <span class="font-bold">{{ formatValue(change.newValue) }}</span>
        </template>
      </div>
    </div>
    <div
      v-if="redeployCount"
      class="mt-[4px] text-[12px]"
    >
      {{ $t('您修改了北极星配置信息，以上环境需要重新部署应用后方可生效') }}
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Button } from 'bkui-vue';
  import { Share } from 'bkui-vue/lib/icon';
  import { PolarisConfigOutputObj } from '~/@types/v1/polaris-config';

  import { formatPolarisRedeployValue, getPolarisRedeployChanges, PolarisRedeployChange } from './redeploy-utils';

  import type { EnvOutput } from '~/@types/v1/env';

  interface DisplayEnv {
    changes: PolarisRedeployChange[];
    displayName: string;
    name: string;
  }

  const props = defineProps<{
    config: PolarisConfigOutputObj;
    envList: EnvOutput[];
    envNames: string[];
  }>();

  const emit = defineEmits<{
    (e: 'go-deploy', envName: string): void;
  }>();

  const envInfoMap = computed(() => {
    const map = new Map<string, EnvOutput>();
    props.envList.forEach(env => {
      if (env.name) {
        map.set(env.name, env);
      }
    });
    return map;
  });

  // “更多”弹层展示当前作用域内环境的待生效字段差异。
  const displayEnvs = computed(() =>
    props.envNames.map<DisplayEnv>(envName => ({
      name: envName,
      displayName: getEnvDisplayName(envName),
      changes: getPolarisRedeployChanges(props.config, envName),
    })),
  );

  // 只统计存在待部署差异的环境，用于决定是否展示整体重新部署提示文案。
  const redeployCount = computed(
    () =>
      props.envNames.filter(envName => {
        const changes = getPolarisRedeployChanges(props.config, envName);
        return changes.length > 0;
      }).length,
  );

  function formatValue(value?: number | string) {
    return formatPolarisRedeployValue(value);
  }

  function getEnvDisplayName(envName: string) {
    const envInfo = envInfoMap.value.get(envName);
    return envInfo?.displayName || envName;
  }
</script>
