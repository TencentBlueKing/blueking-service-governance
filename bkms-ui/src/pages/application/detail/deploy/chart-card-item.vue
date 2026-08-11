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
  <div class="chart-card-item flex flex-col bg-[#fff] rounded-[4px] h-[180px]">
    <!-- 头部 -->
    <div class="flex items-center justify-between px-[16px] py-[12px]">
      <slot name="title">
        <span class="text-[#313238] text-[14px] font-medium leading-[22px]">{{ title }}</span>
      </slot>
      <!-- <i
        class="bkms-icon bkms-icon-filliscreen-line text-[#979BA5] text-[14px] cursor-pointer hover:text-[#3A84FF]"
        @click="handleFullscreen"
      ></i> -->
    </div>

    <!-- 主体：左侧图表 + 右侧图例表格；整体高度由右侧图例表格内容决定，左侧图表随之一致 -->
    <div class="flex flex-shrink-0 min-h-0">
      <!-- 图表区：由包内 ExploreChart 负责渲染（内部走全局 $api 拉数据）；内置图例已隐藏。
           高度随右侧图例表格自适应（flex stretch 同高），并设最小高度避免图例过少时图表塌陷 -->
      <div
        ref="chartContainerRef"
        class="min-w-0 w-[50%] flex flex-col min-h-[130px]"
      >
        <ExploreChart
          ref="exploreChartRef"
          class="min-h-0 flex-1"
          :custom-options="customOptions"
          :panel="panel"
          :show-title="false"
          @duration-change="onDurationChange"
        />
      </div>

      <!-- 右侧图例表格：展示各实例的 Min/Max/Avg 统计，支持表头排序 -->
      <Table
        border="none"
        class="w-[50%]"
        :column-config="{
          resizable: false,
        }"
        :data="legendRows"
        :height="130"
        :loading="tableLoading"
        :max-height="legendMaxHeight"
        :row-config="{
          isHover: false,
        }"
        :sort-config="sortConfig"
      >
        <template #empty>
          <span class="text-[#979BA5] text-[12px]">{{ $t('暂无数据') }}</span>
        </template>
        <TableColumn
          field="name"
          :min-width="150"
        >
          <template #default="{ row }">
            <div
              class="flex items-center cursor-pointer"
              @click="onLegendItemClick(row, $event)"
            >
              <span
                class="legend-color-dot mr-[8px]"
                :style="{ backgroundColor: row.show ? row.color : '#C4C6CC' }"
              ></span>
              <span :style="{ color: row.show ? '#63656E' : '#C4C6CC' }">{{ row.alias || row.name }}</span>
            </div>
          </template>
        </TableColumn>
        <TableColumn
          field="min"
          label="Min"
          :min-width="80"
          sortable
          :width="100"
        />
        <TableColumn
          field="max"
          label="Max"
          :min-width="80"
          sortable
          :width="100"
        />
        <TableColumn
          field="avg"
          label="Avg"
          :min-width="80"
          sortable
          :width="100"
        />
      </Table>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';

  import { ExploreChart } from '@blueking/monitor-vue3-components';
  import { Table, TableColumn } from '@blueking/table';
  import * as echarts from 'echarts/core';
  import { formatOffsetLabel, getCachedMetric, getColorByOffset, isPercentUnit } from '~/pages/application/detail/deploy/monitor-chart-bridge';
  import { useAppDetail } from '~/stores/app-detail';

  // 将所有 ExploreChart 加入同一 echarts group，实现跨图 tooltip 联动
  const CHART_GROUP = 'monitor-sideslider';
  echarts.connect(CHART_GROUP);

  import type { MetricTimeSeries, TimeSeriesItem } from '~/@types/v1/bkintegrations-bkmonitor';

  interface IProps {
    /** 是否为面积图（true=堆叠面积，false=普通折线无填充），默认 true */
    area?: boolean;
    /** 图表高度 */
    chartHeight?: number | string;
    /** 时间对比偏移量列表（秒） */
    compareOffsets?: number[];
    /** 环境名称（作为接口入参，单环境模式） */
    envName?: string;
    /** 实例 ID → 环境名称映射（多环境模式） */
    instanceEnvMap?: Record<string, string>;
    /** 实例名称列表 */
    instances: string[];
    /** 汇聚周期（秒） */
    interval?: number;
    /** 指标类型（同时作为接口入参与图表区分） */
    metricKey: string;
    /** 标题文字（无 title 插槽时生效） */
    title?: string;
  }

  const props = withDefaults(defineProps<IProps>(), {
    chartHeight: 320,
    interval: 60,
    title: '',
    area: true,
  });

  const emits = defineEmits<{
    // (e: 'fullscreen'): void;
    (e: 'metric-data', payload: { data: MetricTimeSeries | undefined; metricKey: string }): void;
  }>();

  defineSlots<{
    default?(): unknown;
    title?(): unknown;
  }>();

  /** 图表容器 DOM ref，用于 ResizeObserver 获取实际宽度以计算绘图区宽度 */
  const chartContainerRef = ref<HTMLElement | null>(null);
  /** 图表绘图区实际宽度（px），已排除纵坐标标签区域（~55px）。
   *  通过 ResizeObserver 实时追踪，供父组件用于 auto 汇聚周期计算。 */
  const plotWidth = ref(0);

  let resizeObserver: ResizeObserver | undefined;
  onMounted(() => {
    if (chartContainerRef.value) {
      resizeObserver = new ResizeObserver(([entry]) => {
        // 图表容器宽度 − 纵坐标标签区域（~55px）≈ 绘图区宽度
        plotWidth.value = Math.max(entry.contentRect.width - 55, 0);
      });
      resizeObserver.observe(chartContainerRef.value);
    }
  });
  onBeforeUnmount(() => {
    resizeObserver?.disconnect();
  });

  defineExpose({ plotWidth });

  const appDetailStore = useAppDetail();

  const exploreChartRef = ref<InstanceType<typeof ExploreChart>>();

  // ExploreChart 由内部 useEcharts 通过全局 $api 拉数据；targets.data 携带自建接口所需参数，
  // useEcharts 会把时间范围（来自父组件 provide 的 timeRange）并入 resultParams 后传入桥接。
  const panel = computed(() => ({
    dashboardId: CHART_GROUP,
    title: props.title,
    subTitle: props.metricKey,
    targets: [
      {
        apiModule: 'bkmonitor',
        apiFunc: 'getInstanceTimeSeries',
        // 不在 target 上设置 alias：包内 useEcharts 会回退到 item.alias（即桥接里设置的实例名 s.instance），
        // 这样每条折线名称都能对应到具体实例。
        data: {
          appID: appDetailStore.appID,
          compareOffsets: props.compareOffsets,
          envName: props.envName,
          instanceEnvMap: props.instanceEnvMap,
          instances: props.instances,
          metricKey: props.metricKey,
          interval: props.interval,
        },
      },
    ],
    options: { time_series: { type: 'line' } },
  }));

  // 复现原堆叠面积样式（area=false 时退化为普通折线图：不填充、不堆叠）
  interface EChartOptionLike {
    [key: string]: unknown;
    series?: Array<Record<string, unknown>>;
  }

  const customOptions = {
    options: (opts: EChartOptionLike): EChartOptionLike => {
      if (Array.isArray(opts.series)) {
        opts.series = opts.series.map((s: Record<string, unknown>) => {
          // 桥接层不再用 __CMP__ 前缀标记 series，对比项通过 isCompare 字段显式识别
          const isCompare = s.isCompare === true;
          const base = props.area ? { ...s, areaStyle: {} } : { ...s, stack: undefined };
          if (isCompare) {
            return { ...base, lineStyle: { type: 'dashed', opacity: 0.5, width: 1.5 }, itemStyle: { opacity: 0.5 } };
          }
          return base;
        });
      }
      return opts;
    },
  };

  /** ExploreChart 实例上通过 setup 暴露的图例相关成员，用于点击表格行联动图表显隐 series */
  interface ExploreChartInstance {
    legendData: LegendSourceItem[];
    handleSelectLegend: (payload: {
      actionType: 'click' | 'shift-click';
      item: { disabled?: boolean; hidden?: boolean; name: string; show?: boolean };
    }) => void;
  }

  // 直接复用 ExploreChart 内部同一份 legendData（含 min/max/avg 与 show 状态），
  // 点击表格行时调用其 handleSelectLegend，即可驱动同一份 options，实现图表与图例联动。
  const legendRows = ref<LegendRow[]>([]);
  /** 表格 loading：接口请求期间显示加载状态 */
  const tableLoading = ref(true);


  function syncLegendRows() {
    if (!props.instances?.length) {
      legendRows.value = [];
      return;
    }
    // 优先从桥接缓存读取 API 返回的 stat（min/max/avg），作为 legendRows 的数据源，
    // 同时引用图表图例中的 color/show 属性保持联动
    const chartLegend = (exploreChartRef.value as unknown as ExploreChartInstance | undefined)?.legendData;
    const metric = getCachedMetric(props.metricKey);
    // 以 displayName（去 __CMP__ 前缀）为 key 反查原 series；
    // bridge 的 alias/name 都是 displayName，因此图例 item.name / item.alias 可以直接命中
    const seriesByName = new Map<string, TimeSeriesItem>();
    if (metric?.series) {
      for (const s of metric.series) {
        const inst = s.instance ?? '';
        const displayName = inst.startsWith('__CMP__') ? inst.slice('__CMP__'.length) : inst;
        if (displayName) seriesByName.set(displayName, s);
      }
    }
    const percentScale = isPercentUnit(metric?.unit) ? 100 : 1;

    /** 根据 metricKey 与 API 原始单位推导图例表格显示单位及数值缩放（非网络指标） */
    function resolveLegendUnit(key: string, rawUnit: string | undefined): { scale: number; suffix: string } {
      if (!rawUnit) return { scale: 1, suffix: '' };
      const low = rawUnit.toLowerCase();
      // 使用量指标（cpu_usage / memory_usage）：不追加单位后缀，但保留数值缩放
      if (key === 'cpu_usage') {
        const mem = low === 'byte' || low.includes('byte') ? 1 / (1024 * 1024) : 1;
        return { scale: mem, suffix: '' };
      }
      // 使用率：percentScale 已处理 ×100，仅追加 %
      if (low === 'percent' || low === '%' || low.includes('percent') || low.includes('%')) {
        return { scale: 1, suffix: '%' };
      }
      // 内存：byte → MiB
      if (low === 'byte' || low.includes('byte')) {
        return { scale: 1 / (1024 * 1024), suffix: ' MiB' };
      }
      // 其他（如 CPU cores）保留原始单位
      return { scale: 1, suffix: ` ${rawUnit}` };
    }

    // 网络带宽：根据所有实例中的最大值自动选择 KB/MB/GB 级别
    let valueScale: number;
    let unitSuffix: string;
    const isNetwork = /^network_(receive|transmit|in|out|read|write)/.test(props.metricKey);
    if (isNetwork && metric?.series) {
      let maxAbs = 0;
      for (const s of metric.series) {
        if (s.stat?.max?.[1] != null) maxAbs = Math.max(maxAbs, Math.abs((s.stat.max[1] as number) * percentScale));
        if (s.stat?.avg?.[1] != null) maxAbs = Math.max(maxAbs, Math.abs((s.stat.avg[1] as number) * percentScale));
      }
      const units = [' B/s', ' KB/s', ' MB/s', ' GB/s'];
      let unitIdx = 0;
      while (unitIdx < units.length - 1 && maxAbs >= 1024) {
        maxAbs /= 1024;
        unitIdx++;
      }
      valueScale = 1 / 1024 ** unitIdx;
      unitSuffix = units[unitIdx];
    } else {
      const result = resolveLegendUnit(props.metricKey, metric?.unit);
      valueScale = result.scale;
      unitSuffix = result.suffix;
    }
    /** 截断至两位小数（不四舍五入），不足补零。
     *  用 toFixed(20) 获取完整小数表示避免精度丢失，再 split/slice 截断。 */
    function truncFixed2(n: number): string {
      const s = n.toFixed(20);
      const dot = s.indexOf('.');
      if (dot === -1) return `${s}.00`;
      return `${s.slice(0, dot)}.${s.slice(dot + 1, dot + 3)}`;
    }
    const rows: LegendRow[] = (chartLegend ?? []).map(item => {
      // 优先读 ExploreChart 透传的 isCompare 字段，否则回退到反查 seriesItem 的原 instance 前缀
      const seriesItem = seriesByName.get(item.name) ?? seriesByName.get(item.alias ?? '');
      const isCompare = (item as LegendSourceItem).isCompare === true
        || (!!seriesItem && (seriesItem.instance ?? '').startsWith('__CMP__'));
      const stat = seriesItem?.stat;
      const fmt = (v: number | undefined) => (v != null ? truncFixed2(v * percentScale * valueScale) + unitSuffix : '');
      return {
        name: item.name,
        alias: item.alias ?? item.name,
        color: item.color,
        show: item.show,
        min: fmt(stat?.min?.[1] as number | undefined),
        max: fmt(stat?.max?.[1] as number | undefined),
        avg: fmt(stat?.avg?.[1] as number | undefined),
        isCompare,
      } as LegendRow;
    });

    // 时间对比：对缺少数据的 offset 合成占位行
    if (props.compareOffsets?.length) {
      // 用 formatOffsetLabel(offset) 做 label 匹配去重，避免 instance 名中 '-' 干扰
      const existingLabels = new Set<string>();
      for (const row of rows) {
        if (!row.isCompare) continue;
        for (const offset of props.compareOffsets) {
          const label = formatOffsetLabel(offset);
          if ((row.alias as string).startsWith(label + '-')) {
            existingLabels.add(label);
            break;
          }
        }
      }
      const synthesized: LegendRow[] = [];
      props.compareOffsets.forEach(offset => {
        const label = formatOffsetLabel(offset);
        if (existingLabels.has(label)) return;
        const color = getColorByOffset(offset);
        props.instances!.forEach(instance => {
          synthesized.push({
            alias: `${label}-${instance}`,
            name: `${label}-${instance}`,
            color,
            show: true,
            min: '--',
            max: '--',
            avg: '--',
            isCompare: true,
            synthesized: true,
          } as LegendRow);
        });
      });
      legendRows.value = synthesized.length ? [...rows, ...synthesized] : rows;
    } else {
      legendRows.value = rows;
    }
    tableLoading.value = false;
  }

  /** 监听 ExploreChart 内部 legendData 变化，同步到表格并关闭 loading */
  watch(
    () => (exploreChartRef.value as unknown as ExploreChartInstance | undefined)?.legendData,
    () => {
      syncLegendRows();
    },
    { immediate: true },
  );
  /** 实例或汇聚周期变化时（即将发起新请求），先清空图例表格并显示 loading */
  watch(
    () => [props.instances, props.interval],
    () => {
      if (!props.instances?.length) {
        tableLoading.value = false;
        return;
      }
      tableLoading.value = true;
      legendRows.value = [];
    },
    { immediate: true },
  );

  /** 图例表格行数据 */
  interface LegendRow {
    alias: string;
    avg: string;
    color: string;
    isCompare?: boolean;
    max: string;
    min: string;
    name: string;
    show: boolean;
    synthesized?: boolean;
  }

  /** ExploreChart 内部 useChartLegend 返回的图例项结构 */
  interface LegendSourceItem {
    alias?: string;
    avg?: number | string;
    color: string;
    /** 是否为时间对比系列（由桥接层显式标记，UI 不再依赖字符串前缀判断） */
    isCompare?: boolean;
    max?: number | string;
    min?: number | string;
    name: string;
    show: boolean;
  }

  const legendMaxHeight = computed(() =>
    typeof props.chartHeight === 'number' ? props.chartHeight : parseInt(String(props.chartHeight), 10) || 320,
  );

  /** Min/Max/Avg 列点击表头排序：数值解析后按大小排序（值含单位后缀如 "KB/s"、"%"） */
  const sortConfig = ref({
    multiple: false,
    trigger: 'cell',
    sortMethod({ data, sortList }: { data: LegendRow[]; sortList: { field: string; order: 'asc' | 'desc' }[] }) {
      const { field, order } = sortList[0];
      return data.sort((a: LegendRow, b: LegendRow) => {
        const va = parseFloat(String(a[field as keyof LegendRow] ?? '')) || 0;
        const vb = parseFloat(String(b[field as keyof LegendRow] ?? '')) || 0;
        return order === 'asc' ? va - vb : vb - va;
      });
    },
  });

  // function handleFullscreen() {
  //   cardRef.value?.requestFullscreen?.().catch(() => {});
  //   emits('fullscreen');
  // }

  /** ExploreChart 数据就绪（duration-change）后，从桥接缓存读取本指标时序回传父组件，以驱动表格聚合 */
  function onDurationChange() {
    syncLegendRows();
    emits('metric-data', { metricKey: props.metricKey, data: getCachedMetric(props.metricKey) });
  }

  /** 点击图例项（颜色圆点+名称），对齐 CommonLegend 的 selectLegend 事件：
   *  普通点击 = 单选/仅显示当前；Shift+点击 = 多选切换。两者均联动 ExploreChart 同一份 options。 */
  function onLegendItemClick(row: LegendRow, event?: MouseEvent) {
    const inst = exploreChartRef.value as unknown as ExploreChartInstance | undefined;
    if (!inst) return;
    const actionType = event?.shiftKey ? 'shift-click' : 'click';
    inst.handleSelectLegend({ actionType, item: { name: row.name } });
    syncLegendRows();
  }
</script>

<style scoped>
  .chart-card-item {
    box-shadow: 0 2px 4px 0 #1912290d;
  }

  /* 隐藏 ExploreChart 内部 CommonLegend（图例已迁移到右侧 Table） */
  .chart-card-item :deep(.explore-chart .common-legend) {
    display: none;
  }

  /* 图例表格中的颜色标识（与包内 CommonLegend 圆点风格一致） */
  .legend-color-dot {
    display: inline-block;
    width: 12px;
    height: 4px;
    border-radius: 2px;
    flex-shrink: 0;
  }
</style>
