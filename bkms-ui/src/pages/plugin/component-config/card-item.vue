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
    class="card-item flex p-[16px] flex-col justify-center items-start gap-[16px] flex-[1_0_0] rounded-[4px] bg-[var(--Neutral-11,#FFF)] shadow-[0_2px_4px_0_#0000000d]"
  >
    <!-- 卡片头部：名称 + Tag -->
    <div class="flex flex-col gap-[6px]">
      <div class="flex items-center">
        <span class="text-[14px] font-bold mr-[8px]">{{ item.name }}</span>
        <Tag
          size="small"
          :theme="item.scopeType === 'global' ? 'success' : ''"
        >
          {{ item.scopeType === 'global' ? $t('所有空间可见') : $t('仅本空间可见') }}
        </Tag>
      </div>
      <div class="flex gap-[24px]">
        <DetailItem
          class="!w-auto !h-auto"
          :label="$t('实例数量')"
          :label-width="0"
          :value="item.appCompInstanceCount != null ? String(item.appCompInstanceCount) : '--'"
        />
        <DetailItem
          class="!w-auto !h-auto"
          :label="$t('更新人')"
          :label-width="0"
          :value="item.updater"
        />
      </div>
    </div>
    <!-- 描述 -->
    <div class="text-[#4D4F56] text-warp break-all max-h-[200px] overflow-y-auto">
      {{ item.description || '--' }}
    </div>
    <Divider
      class="m-[0px]"
      type="solid"
    />
    <!-- 卡片底部：根据类型差异化渲染 -->
    <div
      v-if="footerType === 'editable'"
      class="flex items-center gap-[8px]"
    >
      <Button
        size="small"
        @click="emit('edit', item)"
      >
        {{ $t('编辑') }}
      </Button>
      <div
        v-bk-tooltips="{
          content: $t('组件已被应用使用，不支持删除'),
          disabled: !((item.appCompInstanceCount ?? 0) > 0),
        }"
        class="inline-block"
      >
        <Button
          :disabled="(item.appCompInstanceCount ?? 0) > 0"
          size="small"
          @click="emit('delete', item)"
        >
          {{ $t('删除') }}
        </Button>
      </div>
    </div>
    <div
      v-else-if="footerType === 'builtin'"
      class="text-[#C4C6CC]"
    >
      {{ t('内置组件，不可编辑') }}
    </div>
    <div
      v-else-if="footerType === 'shared'"
      class="text-[#C4C6CC]"
    >
      {{ t('来自「{space}」', { space: item.managedByWorkspaceIDs?.join('、') ?? '' }) }}
    </div>
  </div>
</template>
<script setup lang="ts">
  import { Button, Divider, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { type ComponentDefOutputObj } from '~/@types/v1/component-defs';

  export type CardFooterType = 'builtin' | 'editable' | 'shared';

  interface IProps {
    footerType: CardFooterType;
    item: ComponentDefOutputObj;
  }

  defineProps<IProps>();
  const emit = defineEmits<{
    delete: [item: ComponentDefOutputObj];
    edit: [item: ComponentDefOutputObj];
  }>();

  const { t } = useI18n();
</script>
<style scoped>
  .card-item:hover {
    box-shadow: 0 6px 16px 0 rgba(0, 0, 0, 0.12);
  }
</style>
