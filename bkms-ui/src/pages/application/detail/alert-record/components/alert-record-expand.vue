<template>
  <div class="grid grid-cols-2 py-[24px] pl-[140px] pr-[24px]">
    <FieldItem
      :container-height="32"
      :field-value="t('级别')"
      :field-width="100"
      value-color="#313238"
    >
      <template #value>
        <SeverityLabel :severity="row.severity" />
      </template>
    </FieldItem>
    <FieldItem
      :container-height="32"
      :field-value="t('处理状态')"
      :field-width="100"
      :value="statusLabel"
      value-color="#313238"
    />
    <FieldItem
      :container-height="32"
      :field-value="t('开始时间')"
      :field-width="100"
      :value="row.beginTime ? formatDateString(row.beginTime * 1000) : '--'"
      value-color="#313238"
    />
    <FieldItem
      :container-height="32"
      :field-value="t('负责人')"
      :field-width="100"
      :value="ownerText"
      value-color="#313238"
    />
    <FieldItem
      :container-height="32"
      :field-value="t('持续时间')"
      :field-width="100"
      :value="row.duration || '--'"
      value-color="#313238"
    />
    <FieldItem
      class="!min-h-[32px] !h-auto !items-start"
      value-color="#313238"
    >
      <template #field>
        <div class="w-[100px] min-h-[32px] leading-[32px] text-align-end text-[#979BA5]">{{ t('告警内容') }}：</div>
      </template>
      <template #value>
        <div
          class="pt-[8px] pb-[4px] text-[12px] text-[#313238] break-all whitespace-pre-wrap"
          style="max-height: 120px; overflow: auto"
        >
          {{ row.description || '--' }}
        </div>
      </template>
    </FieldItem>
  </div>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { useI18n } from 'vue-i18n';
  import FieldItem from '~/components/field-item.vue';
  import useTime from '~/composables/use-time';

  import SeverityLabel from './severity-label.vue';

  import type { AlertEventOutput } from '~/@types/v1/bkintegrations-bkmonitor';

  interface IProps {
    row: AlertEventOutput;
  }

  const props = defineProps<IProps>();

  const { t } = useI18n();
  const { formatDateString } = useTime();

  /** 状态文案映射（与主列表状态筛选 RECOVERED / ABNORMAL / CLOSED 保持一致） */
  const STATUS_LABEL_MAP: Record<string, string> = {
    RECOVERED: t('已恢复'),
    ABNORMAL: t('未恢复'),
    CLOSED: t('已失效'),
  };

  const statusLabel = computed(() => STATUS_LABEL_MAP[props.row.status || ''] || '--');

  /** 负责人列表：row.assignee 为字符串数组，逗号隔开展示 */
  const ownerText = computed(() => {
    const assignees = props.row.assignee?.filter(Boolean);
    return assignees?.length ? assignees.join(', ') : '--';
  });
</script>
