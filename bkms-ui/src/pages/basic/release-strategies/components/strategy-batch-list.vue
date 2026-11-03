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
  <div class="flex w-[280px] shrink-0 flex-col gap-[16px]">
    <div class="flex max-h-[320px] flex-1 flex-col gap-[8px] overflow-y-auto">
      <div
        v-for="(batch, index) in batches"
        :key="batch.key"
        :aria-pressed="batch.key === activeKey"
        class="group relative flex h-[56px] w-full shrink-0 cursor-pointer flex-col justify-center gap-[2px] border border-solid px-[12px] py-[8px] text-left outline-none transition-colors"
        :class="cardClass(batch)"
        :data-batch-key="batch.key"
        data-testid="batch-card"
        role="button"
        tabindex="0"
        @click="emit('select', batch.key)"
        @keydown.enter.prevent="emit('select', batch.key)"
        @keydown.space.prevent="emit('select', batch.key)"
      >
        <span class="flex min-w-0 items-center gap-[6px] pr-[20px]">
          <span class="truncate text-[12px] font-700 leading-[20px] text-[#313238]">
            {{ formatBatchCardTitle(batch, index, t) }}
          </span>
        </span>
        <!-- 第二行：有校验错误时用红色错误文案替换暂停说明，未修正前不可提交 -->
        <span
          v-if="errors[batch.key]"
          class="truncate text-[12px] leading-[20px] text-[#ea3636]"
          data-testid="batch-card-error"
        >
          {{ errors[batch.key] }}
        </span>
        <span
          v-else
          class="truncate text-[12px] leading-[20px] text-[#979ba5]"
        >
          {{ formatBatchPauseText(batch, t) }}
        </span>

        <!-- 删除入口：仅 hover 时出现，默认与选中态均隐藏；至少保留一个批次 -->
        <button
          v-if="!readonly && batches.length > 1"
          :aria-label="t('删除')"
          class="pointer-events-none absolute right-[12px] top-[18px] flex h-[20px] w-[20px] items-center justify-center rounded-[2px] text-[14px] text-[#979ba5] opacity-0 transition-opacity group-hover:pointer-events-auto group-hover:opacity-100 hover:bg-[#f0f1f5] hover:text-[#ea3636]"
          data-testid="batch-remove"
          :title="t('删除')"
          type="button"
          @click.stop="emit('remove', batch.key)"
        >
          <Del />
        </button>
      </div>
    </div>

    <button
      v-if="!readonly"
      class="flex h-[32px] shrink-0 items-center justify-center rounded-[2px] border border-dashed border-[#c4c6cc] bg-white text-[14px] leading-[20px] text-[#3a84ff] transition-colors hover:border-[#3a84ff] hover:bg-[#f0f5ff]"
      type="button"
      @click="emit('add')"
    >
      <i class="bkms-icon bkms-icon-jiahao mr-[2px] text-[16px]"></i>
      {{ t('更新批次') }}
    </button>
  </div>
</template>

<script setup lang="ts">
  import { Del } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';

  import { formatBatchCardTitle, formatBatchPauseText } from '../utils';

  import type { StrategyBatchForm } from '../utils';

  interface Props {
    /** 当前选中的批次 key */
    activeKey: string;
    batches: StrategyBatchForm[];
    /** 批次行级错误，key 为批次 key */
    errors: Record<string, string>;
    /** 只读模式隐藏新增与删除入口 */
    readonly?: boolean;
  }

  const props = withDefaults(defineProps<Props>(), {
    readonly: false,
  });

  const emit = defineEmits<{
    add: [];
    remove: [key: string];
    select: [key: string];
  }>();

  const { t } = useI18n();

  /** 卡片三态样式：默认白底、悬停浅灰底、选中浅蓝底带左侧竖条；校验失败统一红框（选中态左侧竖条保持主蓝） */
  function cardClass(batch: StrategyBatchForm): string[] {
    const isActive = batch.key === props.activeKey;
    const isError = Boolean(props.errors[batch.key]);
    if (isActive) {
      return ['bg-[#f0f5ff]', isError ? 'border-[#ea3636]' : 'border-[#a3c5fd]', 'shadow-[inset_3px_0_0_0_#3a84ff]'];
    }
    return ['bg-white hover:border-[#dcdee5] hover:bg-[#f5f7fa]', isError ? 'border-[#ea3636]' : 'border-[#eaebf0]'];
  }
</script>
