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
      <span class="text-[14px] font-bold">{{ targetKind }}</span>
      <Tag
        class="ml-[8px]"
        theme="info"
      >
        {{ $t('工作负载 Patch') }}
      </Tag>
    </template>
    <template #header-right>
      <div class="flex items-center gap-[4px] text-[12px] text-[#4D4F56]">
        <span
          class="inline-block bg-[#65C389] w-[12px] h-[12px] border-[#2CAF5E] border-[1px] border-solid rounded-[4px] mr-[6px]"
        ></span>
        <span>{{ $t('工作负载 Patch 内容') }}</span>
      </div>
    </template>
    <MsEditor
      class="!h-[300px] w-full"
      :is-diff="true"
      :model-value="patchedYaml"
      :original="baseYaml"
      :readonly="true"
    />
  </CollapseCard>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { Tag } from 'bkui-vue';
  import { PreviewPatchOutput } from '~/@types/v1/component-defs';
  import { convertToYaml } from '~/common/util';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';

  const props = defineProps<Required<PreviewPatchOutput>>();

  const baseYaml = computed(() => convertToYaml(props.baseManifest));
  const patchedYaml = computed(() => convertToYaml(props.patchedManifest));
</script>

<style lang="postcss" scoped>
  :deep(.monaco-diff-editor) .diffOverview {
    background-color: #1e1e1e;
  }
</style>
