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
  <div class="flex h-full flex-col bg-[#fff]">
    <div class="flex-1 overflow-auto p-[24px]">
      <div class="flex justify-center mb-[24px]">
        <Success
          fill="#2CAF5E"
          height="72px"
          width="72px"
        />
      </div>
      <h3 class="mb-[16px] text-center text-[24px] font-normal leading-[32px] text-[#313238]">
        {{ $t('配置已保存，需重新部署后生效') }}
      </h3>
      <p class="mb-[26px] text-center text-[14px] leading-[22px] text-[#63656E]">
        {{ $t('你修改了以下配置，这些变更需要') }}
        <span class="font-bold text-[#F59500]">{{ $t('重新部署应用') }}</span>
        {{ $t('后才会真正生效：') }}
      </p>
      <Loading :loading="loading">
        <div class="min-h-[240px] bg-[#F5F7FA] p-[16px]">
          <div class="mb-[12px] text-[14px] font-bold leading-[24px] text-[#313238]">
            {{ $t('待部署的环境（{0}）', [redeployRequiredEnvs.length]) }}
          </div>
          <p class="mb-[8px] text-[14px] leading-[22px] text-[#63656E]">
            {{ $t('请点击以下部署链接进行部署，或前往') }}
            <Button
              class="align-baseline !px-0"
              text
              theme="primary"
              @click="emit('go-deploy', redeployRequiredEnvs[0]?.name)"
            >
              {{ $t('「部署管理」') }}
            </Button>
            {{ $t('选择对应环境执行部署；') }}
          </p>
          <p class="mb-[16px] text-[14px] leading-[22px] text-[#63656E]">
            {{ $t('本次操作会记录在部署历史中，可追溯、可回滚；') }}
          </p>
          <div class="flex flex-col gap-[10px]">
            <div
              v-for="env in redeployRequiredEnvs"
              :key="env.name"
              class="bg-[#fff]"
            >
              <div class="flex justify-between h-[32px] items-center border-b border-[#F0F1F5] px-[10px]">
                <div class="h-full flex items-center">
                  <span class="mr-[10px] text-[16px] text-[#4D4F56]">•</span>
                  <span
                    v-bk-tooltips="env.displayName"
                    class="truncate text-[12px] font-bold"
                  >
                    {{ env.displayName }}
                  </span>
                  <Tag
                    v-if="env.envType && envTypeMap[env.envType]"
                    :class="['ml-[8px] shrink-0', envTypeTagClassMap[env.envType]]"
                    size="small"
                  >
                    {{ envTypeMap[env.envType].name }}
                  </Tag>
                </div>
                <Button
                  class="ml-[16px] shrink-0"
                  text
                  theme="primary"
                  @click="emit('go-deploy', env.name)"
                >
                  {{ $t('去部署') }}
                  <Share class="ml-[4px]" />
                </Button>
              </div>
              <div class="px-[30px] py-[8px] text-[12px] text-[#63656E] leading-[24px]">
                <div
                  v-for="change in env.changes"
                  :key="change.key"
                >
                  <span>{{ change.label }}：</span>
                  <template v-if="change.oldValue === undefined">
                    <span class="font-bold text-[#313238]">{{ formatRedeployValue(change.newValue) }}</span>
                  </template>
                  <template v-else>
                    <span>{{ formatRedeployValue(change.oldValue) }}</span>
                    <span class="mx-[8px]">→</span>
                    <span class="font-bold text-[#313238]">{{ formatRedeployValue(change.newValue) }}</span>
                  </template>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Loading>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Button, Loading, Tag } from 'bkui-vue';
  import { Share, Success } from 'bkui-vue/lib/icon';
  import { PolarisConfigOutputObj } from '~/@types/v1/polaris-config';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';

  import { formatPolarisRedeployValue, getPolarisRedeployChanges } from './redeploy-utils';

  import type { PolarisRedeployChange } from './redeploy-utils';
  import type { EnvOutput } from '~/@types/v1/env';

  interface RedeployRequiredEnv {
    changes: PolarisRedeployChange[];
    displayName: string;
    envType?: string;
    name: string;
  }

  const props = withDefaults(
    defineProps<{
      config?: PolarisConfigOutputObj;
      envList: EnvOutput[];
      loading?: boolean;
    }>(),
    {
      loading: false,
    },
  );

  const emit = defineEmits<{
    (e: 'close'): void;
    (e: 'go-deploy', envName?: string): void;
    (e: 'no-redeploy'): void;
  }>();

  const notifiedNoRedeployConfigName = ref('');

  const envInfoMap = computed(() => {
    const map = new Map<string, EnvOutput>();
    props.envList.forEach(env => {
      if (env.name) {
        map.set(env.name, env);
      }
    });
    return map;
  });

  // 保存后展示的待部署环境列表，只包含当前作用域内环境。
  const redeployRequiredEnvs = computed(() => {
    if (!props.config) return [];

    return (props.config.scopeEnvNames || []).reduce<RedeployRequiredEnv[]>((result, envName) => {
      const changes = getPolarisRedeployChanges(props.config!, envName);
      if (changes.length === 0) return result;

      const envInfo = envInfoMap.value.get(envName);
      result.push({
        name: envName,
        displayName: envInfo?.displayName || envName,
        envType: envInfo?.type,
        changes,
      });
      return result;
    }, []);
  });

  function formatRedeployValue(value?: number | string) {
    return formatPolarisRedeployValue(value);
  }

  watch(
    () => [props.loading, props.config?.name, redeployRequiredEnvs.value.length] as const,
    ([loading, configName, redeployCount]) => {
      if (loading || !configName || redeployCount > 0 || notifiedNoRedeployConfigName.value === configName) return;

      notifiedNoRedeployConfigName.value = configName;
      emit('no-redeploy');
    },
    { immediate: true },
  );
</script>
