<!--
 - TencentBlueKing is pleased to support the open source community by making
 - 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 - Copyright (C) Tencent. All rights reserved.
 - Licensed under the MIT License (the "License"); you may not use this file except
 - in compliance with the License. You may obtain a copy of the License at
 -
 -  http://opensource.org/licenses/MIT
 -
 - Unless required by applicable law or agreed to in writing, software distributed
 - under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 - either express or implied. See the License for the specific language governing permissions and
 - limitations under the License.
 -
 - We undertake not to change the open source license (MIT license) applicable
 - to the current version of the project delivered to anyone in the future.
-->

<template>
  <div class="rounded-[2px] bg-[#F5F7FA] p-[16px]">
    <div class="mb-[12px] flex items-baseline gap-[8px]">
      <span class="text-[14px] font-bold leading-[22px] text-[#313238]">{{ $t('发布预览') }}</span>
      <span class="text-[12px] leading-[20px] text-[#979BA5]">
        {{ $t('按所选环境的 {0} 个实例换算', [preview.summary.totalReplicas]) }}
      </span>
    </div>
    <!-- 摘要栏：label 在上，数值 + 单位在下 -->
    <div
      class="mb-[16px] flex items-stretch rounded-[2px] border border-[#DCDEE5] bg-white px-[24px] py-[14px]"
      data-testid="release-summary"
    >
      <template
        v-for="(item, index) in summaryItems"
        :key="item.label"
      >
        <div
          v-if="index > 0"
          class="mx-[24px] h-[36px] w-[1px] shrink-0 self-center bg-[#DCDEE5]"
        ></div>
        <div class="min-w-0 flex-1">
          <div class="text-[12px] leading-[16px] text-[#979BA5]">{{ item.label }}</div>
          <div class="mt-[6px] flex items-baseline">
            <span class="text-[20px] font-semibold leading-[24px] text-[#313238]">{{ item.value }}</span>
            <span class="ml-[4px] text-[12px] leading-[16px] text-[#63656E]">{{ item.unit }}</span>
          </div>
        </div>
      </template>
    </div>
    <!-- 批次时间线：编号圆节点 + 批次描述 + 进度方块 + 暂停标签 -->
    <div class="relative pl-[30px]">
      <div class="absolute bottom-[22px] left-[12px] top-[12px] w-[2px] bg-[#DCDEE5]"></div>
      <div
        v-for="row in preview.rows"
        :key="row.batchIndex"
        class="relative pb-[16px] last:pb-0"
      >
        <span
          class="absolute left-[-30px] top-0 flex h-[24px] w-[24px] items-center justify-center rounded-full bg-[#3A84FF] text-[12px] font-medium text-white"
        >
          {{ row.batchIndex }}
        </span>
        <div class="flex items-center justify-between gap-[8px]">
          <div class="flex min-w-0 items-center gap-[6px]">
            <span class="text-[13px] font-bold leading-[20px] text-[#313238]">{{ batchTitle(row) }}</span>
            <Tag
              v-if="row.isFull"
              size="small"
              theme="info"
            >
              {{ $t('全量') }}
            </Tag>
          </div>
          <div class="shrink-0 text-[13px] leading-[20px] text-[#313238]">
            <span class="font-bold">{{
              $t('新版本 {0} · 旧版本 {1}', [row.updatedCount, preview.summary.totalReplicas - row.updatedCount])
            }}</span>
          </div>
        </div>
        <div
          class="mt-[8px] flex gap-[4px]"
          data-testid="batch-squares"
        >
          <!-- 方块按实例数比例均分宽度（如 1 实例 = 100% 宽）；浅蓝=本批要发布，深蓝=此前批次已发布，灰=未发布 -->
          <span
            v-for="(square, squareIndex) in buildBatchSquares(row, preview.summary.totalReplicas)"
            :key="squareIndex"
            class="h-[10px] flex-1 rounded-[2px]"
            :class="{
              'bg-[#3A84FF]': square === 'prev',
              'bg-[#A3C5FD]': square === 'new',
              'bg-[#DCDEE5]': square === 'pending',
            }"
          ></span>
        </div>
        <Tag
          v-if="row.pause"
          class="mt-[10px]"
          :theme="row.pause.mode === 'permanent' ? 'warning' : 'info'"
        >
          {{
            row.pause.mode === 'permanent'
              ? $t('永久暂停 · 手动继续')
              : $t('暂停 {0} 秒后自动继续', [row.pause.seconds])
          }}
        </Tag>
        <!-- 无暂停配置的批次补充「不停」标签；末批之后无下一批，不展示 -->
        <Tag
          v-else-if="row.batchIndex < preview.rows.length"
          class="mt-[10px] !text-[#4D4F56]"
        >
          {{ $t('不停，直接进入下一批') }}
        </Tag>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';

  import { type ReleasePreviewResult, type ReleasePreviewRow, buildBatchSquares } from './release-preview';

  const props = defineProps<{
    /** 发布预览数据（buildReleasePreview 的返回值） */
    preview: ReleasePreviewResult;
  }>();

  const { t } = useI18n();

  const summaryItems = computed(() => [
    { label: t('批次数'), value: props.preview.summary.batchCount, unit: t('批') },
    { label: t('暂停次数'), value: props.preview.summary.pauseCount, unit: t('次') },
    { label: t('实例总数'), value: props.preview.summary.totalReplicas, unit: t('个') },
    { label: t('流量批次'), value: props.preview.summary.trafficBatchCount, unit: t('个') },
  ]);

  /** 批次标题：首批数值 target 为「N 个实例」，其余为「累计 N 个（P%）」 */
  function batchTitle(row: ReleasePreviewRow): string {
    const head = t('第 {0} 批', [row.batchIndex]);
    if (row.batchIndex === 1 && !row.isPercent) {
      return `${head} · ${t('{0} 个实例', [row.updatedCount])}`;
    }
    const percentSuffix = row.isPercent ? `（${row.targetText}）` : '';
    return `${head} · ${t('累计 {0} 个', [row.updatedCount])}${percentSuffix}`;
  }
</script>

<style lang="postcss" scoped>
  :deep(.bk-tag-info) {
    background-color: #e1ecff;
    color: #1768ef;
  }

  :deep(.bk-tag-warning) {
    background-color: #fdeed8;
    color: #e38b02;
  }
</style>
