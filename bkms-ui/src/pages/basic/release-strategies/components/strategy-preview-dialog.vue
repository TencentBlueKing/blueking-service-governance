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
  <Dialog
    v-model:is-show="visible"
    :quick-close="false"
    :width="960"
  >
    <template #header>
      <div class="flex items-center gap-[8px]">
        <span class="text-[20px] leading-[28px] text-[#313238]">{{ t('预览策略') }}</span>
        <template v-if="strategyName">
          <span class="h-[12px] w-px shrink-0 bg-[#dcdee5]"></span>
          <span class="text-[14px] leading-[22px] text-[#979ba5]">{{ strategyName }}</span>
        </template>
      </div>
    </template>

    <div class="flex flex-col gap-[16px]">
      <p class="text-[12px] leading-[20px] text-[#979ba5]">
        {{ t('按配置比例示意（方格只表示比例）；实例数在部署时按应用实际实例数换算') }}
      </p>

      <div class="flex flex-col gap-[16px] overflow-hidden rounded-[2px] bg-[#f5f7fa] px-[20px] py-[16px]">
        <!-- 摘要栏：白底卡片，指标间以 1px 竖线分隔 -->
        <div class="flex w-full items-center rounded-[2px] border border-solid border-[#f0f1f5] bg-white py-[10px]">
          <template
            v-for="(item, index) in summaryItems"
            :key="item.label"
          >
            <span
              v-if="index > 0"
              class="h-[50px] w-px shrink-0 bg-[#eaebf0]"
            ></span>
            <div class="flex min-w-0 flex-1 flex-col gap-[2px] px-[16px]">
              <span class="text-[12px] leading-[20px] text-[#979ba5]">{{ item.label }}</span>
              <span class="flex items-baseline gap-[4px] overflow-hidden">
                <span class="text-[20px] font-700 leading-[28px] text-[#313238]">{{ item.value }}</span>
                <span class="text-[12px] leading-[20px] text-[#979ba5]">{{ item.unit }}</span>
              </span>
            </div>
          </template>
        </div>

        <!-- 批次时间线 -->
        <div class="relative flex w-full flex-col">
          <span class="pointer-events-none absolute bottom-[24px] left-[11px] top-[8px] w-[2px] bg-[#dcdee5]"></span>
          <template
            v-for="(row, index) in rows"
            :key="row.key"
          >
            <div
              class="relative flex w-full items-start gap-[12px] overflow-hidden py-[4px]"
              data-testid="preview-row"
            >
              <span
                class="relative z-10 flex size-[24px] shrink-0 items-center justify-center rounded-full bg-[#3a84ff] text-[12px] font-700 leading-[20px] text-white"
              >
                {{ index + 1 }}
              </span>
              <div class="flex min-w-0 flex-1 flex-col items-start gap-[8px] overflow-hidden pt-[2px]">
                <div class="flex w-full items-center justify-between gap-[8px]">
                  <div class="flex min-w-0 items-center gap-[8px] overflow-hidden">
                    <div class="flex shrink-0 items-center gap-[8px] overflow-hidden">
                      <span class="whitespace-nowrap text-[12px] font-700 leading-[20px] text-[#313238]">
                        {{ row.label }}
                      </span>
                      <span
                        v-if="row.isFull"
                        class="flex h-[16px] shrink-0 items-center rounded-[2px] bg-[#e1ecff] px-[6px] text-[10px] leading-[16px] text-[#1768ef]"
                        data-testid="preview-row-full"
                      >
                        {{ t('全量') }}
                      </span>
                    </div>
                    <span
                      v-if="row.hasPause"
                      class="flex h-[16px] shrink-0 items-center rounded-[2px] bg-[#f0f1f5] px-[6px] text-[10px] leading-[16px] text-[#4d4f56]"
                    >
                      {{ row.pauseText }}
                    </span>
                  </div>
                  <div class="flex shrink-0 items-center gap-[4px] text-[12px] leading-[20px]">
                    <span class="font-700 text-[#313238]">{{ row.countValue }}</span>
                    <span class="text-[#979ba5]">{{ row.countLabel }}</span>
                  </div>
                </div>

                <div class="flex w-full items-start gap-[4px] overflow-hidden">
                  <span
                    v-for="(square, squareIndex) in buildPreviewSquares(row)"
                    :key="squareIndex"
                    class="h-[12px] min-w-px flex-1 rounded-[2px]"
                    :class="squareClassMap[square]"
                  ></span>
                </div>
              </div>
            </div>

            <!-- 暂停标签：位于时间线轴上，紧跟所属批次 -->
            <div
              v-if="row.hasPause"
              class="flex w-full items-center overflow-hidden py-[6px] pl-[32px]"
              data-testid="preview-row-pause"
            >
              <span
                class="flex h-[18px] items-center rounded-[2px] px-[8px] text-[12px] leading-[20px]"
                :class="pauseTagClassMap[row.pauseTheme]"
              >
                {{ formatPauseTagText(row, t) }}
              </span>
            </div>
          </template>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex items-center justify-end gap-[8px]">
        <Button
          v-if="showSubmit"
          data-testid="preview-submit"
          theme="primary"
          @click="handleSubmit"
        >
          {{ t('提交') }}
        </Button>
        <Button @click="visible = false">
          {{ t('关闭') }}
        </Button>
      </div>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Button, Dialog } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';

  import { buildPreviewSquares, buildStrategyPreviewRows, buildStrategyPreviewSummary } from '../utils';

  import type { StrategyBatchForm, StrategyPreviewRow } from '../utils';

  interface Props {
    /** 已通过表单校验的批次（弹窗内不做校验） */
    batches: StrategyBatchForm[];
    /** 策略名称：作为标题副标题展示，便于确认预览对象 */
    name?: string;
    /** 是否展示「提交」按钮（查看态预览不需要提交） */
    showSubmit?: boolean;
  }

  const props = withDefaults(defineProps<Props>(), {
    name: '',
    showSubmit: true,
  });

  const emit = defineEmits<{ submit: [] }>();

  const visible = defineModel<boolean>('isShow', { default: false });

  const { t } = useI18n();

  const strategyName = computed(() => props.name.trim());

  const rows = computed(() => buildStrategyPreviewRows(props.batches, t));

  const summaryItems = computed(() => {
    const summary = buildStrategyPreviewSummary(props.batches);
    return [
      { label: t('批次数'), value: summary.batchCount, unit: t('批') },
      { label: t('暂停次数'), value: summary.pauseCount, unit: t('次') },
      { label: t('首批数量'), value: summary.firstCountValue, unit: summary.firstCountUnit },
      { label: t('流量批次'), value: summary.trafficBatchCount, unit: t('个') },
    ];
  });

  /** 暂停标签文案：时间线标签与设计稿一致用「·」分隔 */
  function formatPauseTagText(row: StrategyPreviewRow, tf: (key: string) => string): string {
    return row.pauseTheme === 'warn' ? tf('永久暂停 · 手动继续') : row.pauseText;
  }

  /** 弹窗内提交：关闭预览并交给外层执行保存 */
  function handleSubmit() {
    visible.value = false;
    emit('submit');
  }

  // 实例方块状态色：已更新 / 本批新增 / 待更新
  const squareClassMap: Record<string, string> = {
    prev: 'bg-[#3a84ff]',
    new: 'bg-[#a3c5fd]',
    pending: 'bg-[#dcdee5]',
  };

  // 暂停标签配色：永久暂停用告警橙，定时暂停用主蓝
  const pauseTagClassMap: Record<StrategyPreviewRow['pauseTheme'], string> = {
    warn: 'bg-[#fdeed8] text-[#e38b02]',
    brand: 'bg-[#e1ecff] text-[#1768ef]',
  };
</script>
