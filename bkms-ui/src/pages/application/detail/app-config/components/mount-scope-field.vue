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
  <Radio.Group
    v-model="scopeModel"
    class="flex w-full flex-col"
  >
    <Radio label="all">{{ $t('全部环境') }}</Radio>
    <Radio
      class="!ml-0 mt-[10px]"
      label="envs"
      >{{ $t('指定环境') }}</Radio
    >
    <EnvGroupSelect
      v-show="scopeModel === 'envs'"
      v-model="envNamesModel"
      class="mt-[12px] w-full"
      :env-list="envList"
    />
  </Radio.Group>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Radio } from 'bkui-vue';
  import EnvGroupSelect from '~/components/env-group-selector.vue';

  import type { EnvOutput } from '~/@types/v1/env';
  import type { ConfigFileMountScope } from '~/pages/application/detail/app-config/use-config-file-defs';

  /** 挂载范围类型：全部环境 / 指定环境 */
  export type MountScope = ConfigFileMountScope;

  const props = defineProps<{
    /** 可选环境列表 */
    envList: EnvOutput[];
    /** 指定环境下已选中的环境名集合 */
    envNames: string[];
    /** 挂载范围 */
    scope: MountScope;
  }>();

  const emit = defineEmits<{
    'update:envNames': [value: string[]];
    'update:scope': [value: MountScope];
  }>();

  const scopeModel = computed({
    get: () => props.scope,
    set: (value: MountScope) => emit('update:scope', value),
  });
  const envNamesModel = computed({
    get: () => props.envNames,
    set: (value: string[]) => emit('update:envNames', value),
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-radio-label) {
    font-size: 14px;
  }
</style>
