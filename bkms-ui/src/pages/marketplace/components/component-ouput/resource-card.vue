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
  <CollapseCard>
    <template #header-left>
      <span class="text-[14px] font-bold">{{ kind }}</span>
      <Tag
        class="ml-[8px]"
        theme="success"
      >
        {{ $t('附加资源') }}
      </Tag>
    </template>
    <template #header-right>
      <div class="flex items-center gap-[4px] text-[12px] text-[#4D4F56]">
        <span
          class="inline-block bg-[#699DF4] w-[12px] h-[12px] border-[#3A84FF] border-[1px] border-solid rounded-[4px] mr-[6px]"
        ></span>
        <span>{{ $t('附加资源') }}</span>
      </div>
    </template>
    <MsEditor
      class="!h-[300px] w-full"
      :model-value="manifestYaml"
      :readonly="true"
    />
  </CollapseCard>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { Tag } from 'bkui-vue';
  import { PreviewResourceOutput } from '~/@types/v1/component-defs';
  import { convertToYaml } from '~/common/util';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';

  const props = defineProps<Required<Pick<PreviewResourceOutput, 'kind' | 'manifest'>>>();

  const manifestYaml = computed(() => convertToYaml(props.manifest));
</script>
