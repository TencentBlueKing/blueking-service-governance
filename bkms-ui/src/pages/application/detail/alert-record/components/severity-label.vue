<template>
  <div class="flex items-center gap-[8px] text-[12px]">
    <span
      class="inline-block w-[4px] h-[14px] rounded-[1px]"
      :style="{ backgroundColor: color }"
    ></span>
    <span>{{ label }}</span>
  </div>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { useI18n } from 'vue-i18n';

  interface IProps {
    /** 告警级别：1=致命 2=预警 3=提醒 */
    severity?: number;
  }

  const props = defineProps<IProps>();

  const { t } = useI18n();

  const SEVERITY_COLOR_MAP: Record<number, string> = {
    1: '#E71818',
    2: '#FF9C01',
    3: '#3A84FF',
  };

  const SEVERITY_LABEL_MAP: Record<number, string> = {
    1: t('致命'),
    2: t('预警'),
    3: t('提醒'),
  };

  const color = computed(() => SEVERITY_COLOR_MAP[props.severity || 0] || '#979BA5');
  const label = computed(() => SEVERITY_LABEL_MAP[props.severity || 0] || '--');
</script>
