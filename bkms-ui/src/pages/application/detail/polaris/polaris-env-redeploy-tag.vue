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
  <Popover
    v-if="isRedeployRequired"
    :popover-delay="[100, 100]"
    theme="popover-dark-translucent"
    trigger="hover"
  >
    <Tag class="mr-[4px] cursor-pointer">
      {{ displayName }}
      <span class="text-[#F59500]">（{{ $t('待部署生效') }}）</span>
    </Tag>
    <template #content>
      <div class="max-w-[520px] text-[12px] leading-[20px] text-[#fff]">
        <div class="mb-[6px]">
          {{ $t('您修改了以下内容，需要重新部署应用后方可生效：') }}
        </div>
        <div
          v-for="change in redeployChanges"
          :key="change.key"
          class="pl-[12px]"
        >
          <span>•&nbsp;{{ change.label }}：</span>
          <template v-if="change.oldValue === undefined">
            <span class="font-bold">{{ formatValue(change.newValue) }}</span>
          </template>
          <template v-else>
            <span>{{ formatValue(change.oldValue) }}</span>
            <span class="mx-[6px]">→</span>
            <span class="font-bold">{{ formatValue(change.newValue) }}</span>
          </template>
        </div>
        <Button
          class="mt-[6px] !px-0 text-[#3A84FF]"
          text
          theme="primary"
          @click.stop="emit('go-deploy', envName)"
        >
          <Share class="mr-[4px]" />
          {{ $t('去部署') }}
        </Button>
      </div>
    </template>
  </Popover>
  <Tag
    v-else
    class="mr-[4px]"
  >
    {{ displayName }}
  </Tag>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Button, Popover, Tag } from 'bkui-vue';
  import { Share } from 'bkui-vue/lib/icon';
  import { PolarisConfigOutputObj } from '~/@types/v1/polaris-config';

  import { formatPolarisRedeployValue, getPolarisRedeployChanges } from './redeploy-utils';

  const props = defineProps<{
    config: PolarisConfigOutputObj;
    displayName: string;
    envName: string;
  }>();

  const emit = defineEmits<{
    (e: 'go-deploy', envName: string): void;
  }>();

  // 根据环境已生效字段与当前配置的差异，判断该环境是否需要重新部署。
  const redeployChanges = computed(() => getPolarisRedeployChanges(props.config, props.envName));

  const isRedeployRequired = computed(() => redeployChanges.value.length > 0);

  function formatValue(value?: number | string) {
    return formatPolarisRedeployValue(value);
  }
</script>
