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
  <Sideslider
    v-model:is-show="isShow"
    :width="effectiveSliderWidth"
    @closed="handleClose"
  >
    <template #header>
      <div class="w-full flex items-center justify-between px-[16px]">
        <span class="text-[#313238] text-[16px]">{{ $t('实例监控') }}</span>
        <div class="flex items-center">
          <DatePicker
            v-model="value"
            behavior="simplicity"
            format="YYYY-MM-DD HH:mm:ss"
            :need-timezone="false"
            :version="2"
            @update:model-value="handleValueChange"
          />
          <Divider
            class="h-[12px] min-h-[16px] mr-[12px]"
            color="#DCDEE5"
            direction="vertical"
            type="solid"
          />
          <IntervalSelector
            v-model:value="autoRefreshInterval"
            @refresh="onManualRefresh"
          />
          <i
            v-if="!isFullScreen"
            class="bkms-icon bkms-icon-filliscreen-line cursor-pointer ml-[16px]"
            @click="handleFullScreen"
          ></i>
          <i
            v-else
            class="bkms-icon bkms-icon bkms-icon-un-full-screen-2 cursor-pointer ml-[16px]"
            @click="handleFullScreen"
          ></i>
        </div>
      </div>
    </template>

    <div
      class="flex items-center border-b border-[#DCDEE5] h-[48px] bg-[#fff] px-[24px] shadow-[0_3px_5.5px_0_#00000012]"
    >
      <!-- 实例名称 -->
      <div class="flex items-center">
        <span class="bg-[#FAFBFD] h-[32px] leading-[30px] text-[#4D4F56] text-[12px] px-[6px] b b-r-0 b-[#c4c6cc]">{{
          $t('实例名称')
        }}</span>
        <Select
          ref="selectRef"
          v-model="selectedInstance"
          class="w-[360px]"
          :clearable="false"
          collapse-tags
          custom-content
          display-key="name"
          id-key="id"
          :loading="loading"
          multiple
          multiple-mode="tag"
          @tag-remove="handleRemoveTag"
          @toggle="handleSelectBlur"
        >
          <div
            class="flex items-center px-[12px] py-[6px] text-[#63656E] cursor-pointer hover:bg-[#f5f7fa]"
            @click.stop="handleToggleAll"
          >
            <Checkbox
              :before-change="handleToggleAll"
              :indeterminate="isPartiallySelected"
              :model-value="isAllSelected"
              size="small"
            />
            <span class="ml-[6px]">{{ $t('全选') }}</span>
          </div>
          <Tree
            ref="treeRef"
            children="children"
            :class="{ 'hide-tree-action': envGroups.length <= 1 }"
            :data="treeData"
            :expand-all="true"
            label="name"
            :node-content-action="['checked']"
            node-key="id"
            show-checkbox
            :show-node-type-icon="false"
            @node-checked="handleNodeChecked"
          >
            <template #node="{ name }">
              <span class="text-[12px] text-[#63656E] pr-[12px] cursor-pointer">{{ name }}</span>
            </template>
          </Tree>
        </Select>
      </div>
      <!-- 汇聚周期 -->
      <Select
        v-model="interval"
        class="w-[240px] ml-[16px]"
        :clearable="false"
      >
        <template #prefix>
          <span class="bg-[#FAFBFD] leading-[32px] text-[#4D4F56] text-[12px] px-[6px] b-r b-[#c4c6cc]">{{
            $t('汇聚周期')
          }}</span>
        </template>
        <Select.Option
          v-for="option in convergenceOptions"
          :id="option.id"
          :key="option.id"
          :name="option.name"
        />
      </Select>
      <!-- 时间对比 -->
      <div class="flex items-center ml-[16px] flex-shrink-0">
        <span class="text-[#4D4F56] text-[12px] mr-[8px]">{{ $t('时间对比') }}</span>
        <Switcher
          v-model="timeCompareEnabled"
          :disabled="isMultiEnv"
          theme="primary"
          v-bk-tooltips="{ content: $t('多环境不支持时间对比'), disabled: !isMultiEnv, placement: 'top' }"
          @change="handleTimeCompareToggle"
        />
        <template v-if="timeCompareEnabled">
          <Select
            v-model="compareOffsetSecondsList"
            behavior="simplicity"
            class="w-[180px] ml-[8px]"
            clearable
            multiple
            @clear="handleCompareOffsetClear"
          >
            <Select.Option
              v-for="option in compareOffsetOptions"
              :id="option.id"
              :key="String(option.id)"
              :name="option.name"
            />
            <template #extension>
              <div style="padding: 0 12px 6px">
                <span
                  v-if="!showCustomInput"
                  class="text-[#3A84FF] cursor-pointer text-[12px]"
                  @click="handleShowCustomInput"
                >{{ $t('自定义') }}</span>
                <template v-if="showCustomInput">
                  <Input
                    v-model="customOffsetText"
                    :placeholder="$t('按照提示输入')"
                    size="small"
                    @enter="handleCustomOffsetConfirm"
                  >
                    <template #suffix>
                      <i
                        v-bk-tooltips="{ content: $t('自定义输入格式提示'), placement: 'top', boundary: 'parent' }"
                        class="bkms-icon bkms-icon-help-document text-[#C4C6CC] flex items-center mr-[4px]"
                      />
                    </template>
                    <template #append>
                      <span class="text-[#3A84FF] cursor-pointer" @click="handleCustomOffsetConfirm">{{ $t('确定') }}</span>
                    </template>
                  </Input>
                </template>
              </div>
            </template>
          </Select>
        </template>
      </div>
    </div>
    <div class="overflow-auto px-[24px] pt-[24px] pb-[24px] bg-[#F5F7FA] h-[calc(100vh-105px)]">
      <div class="mb-[24px] bg-[#fff] h-[48px] px-[16px] py-[8px] shadow-[0_3px_5.5px_0_#00000012]">
        <Radio.Group
          v-model="metricFilter"
          type="capsule"
        >
          <Radio.Button label="all">{{ $t('全部') }}</Radio.Button>
          <Radio.Button label="cpu">{{ $t('CPU') }}</Radio.Button>
          <Radio.Button label="memory">{{ $t('内存') }}</Radio.Button>
          <Radio.Button label="network">{{ $t('网络') }}</Radio.Button>
          <!-- <Radio.Button label="disk">{{ $t('存储') }}</Radio.Button> -->
        </Radio.Group>
      </div>

      <Table
        v-show="metricFilter === 'all'"
        class="mb-[24px]"
        :data="tableData"
        :max-height="500"
      >
        <TableColumn
          field="name"
          :label="$t('实例名称')"
          :min-width="260"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            <span>{{ row.name }}</span>
          </template>
        </TableColumn>
        <TableColumn
          field="cpuLimitUsageRate"
          :label="$t('CPU 使用率')"
          :min-width="200"
        >
          <template #default="{ row }">
            <div class="flex items-center">
              <Progress
                color="#3A84FF"
                :percent="Math.min(row.cpuLimitUsageRate, 100)"
                :show-text="false"
                :stroke-width="6"
                type="line"
              />
              <span class="text-[#63656E] text-[12px] ml-[4px]">{{ formatDecimal(row.cpuLimitUsageRate) }}%</span>
            </div>
          </template>
        </TableColumn>
        <TableColumn
          field="memLimitUsageRate"
          :label="$t('内存使用率')"
          :min-width="120"
        >
          <template #default="{ row }">
            <span class="text-[#63656E] text-[12px]">{{ formatDecimal(row.memLimitUsageRate) }}%</span>
          </template>
        </TableColumn>
        <TableColumn
          field="cpuUsage"
          :label="$t('CPU 用量（cores）')"
          :min-width="120"
        >
          <template #default="{ row }">
            <span class="text-[#63656E] text-[12px]">
              {{ formatDecimal(row.cpuUsage) }}
            </span>
          </template>
        </TableColumn>
        <TableColumn
          field="memUsage"
          :label="$t('内存用量（MiB）')"
          :min-width="120"
        >
          <template #default="{ row }">
            <span class="text-[#63656E] text-[12px]">
              {{ formatBytes(row.memUsage) }}
            </span>
          </template>
        </TableColumn>
      </Table>

      <!-- 图表 -->
      <ToggleCard
        v-for="(group, index) in metricGroups"
        v-show="isMetricGroupVisible(group.key)"
        :key="group.name"
        :class="['mb-[24px]', { '!mb-0': index === metricGroups.length - 1 }]"
        content-class="grid grid-cols-1 gap-[16px] pt-[6px]"
        :name="group.name"
        normal-bg-color="#EAEBF0"
        type="normal"
      >
        <chart-card-item
          v-for="item in group.items"
          :key="item.metricKey"
          :ref="setChartCardRef"
          :area="item.area !== false"
          :compare-offsets="chartParams.compareOffsets"
          :env-name="props.envName"
          :instance-env-map="instanceEnvMap"
          :instances="chartParams.instances"
          :interval="chartParams.interval"
          :metric-key="item.metricKey"
          :title="item.title"
          @metric-data="onMetricData"
        >
          <template #title>
            <div class="flex items-center">
              <span class="text-[#313238] text-[14px] font-[500] mr-[8px]">{{ item.title }}</span>
              <Tag v-bk-tooltips="$t('数据步长')">{{ formatIntervalText(chartParams.interval || 0) }}</Tag>
            </div>
          </template>
        </chart-card-item>
      </ToggleCard>
    </div>
  </Sideslider>
</template>

<script lang="ts" setup>
  import {
    type ComponentPublicInstance,
    computed,
    defineAsyncComponent,
    nextTick,
    onBeforeUnmount,
    onMounted,
    provide,
    ref,
    watch,
  } from 'vue';

  import DatePicker from '@blueking/date-picker';
  import { Table, TableColumn } from '@blueking/table';
  import { Checkbox, Divider, Input, Message, Progress, Radio, Select, Sideslider, Switcher, Tag, Tree } from 'bkui-vue';
  import dayjs from 'dayjs';
  import { debounce } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { InstanceService } from '~/api/modules/v1';
  import useInterval from '~/composables/use-interval';
  import { isPercentUnit, setChartTimeRange } from '~/pages/application/detail/deploy/monitor-chart-bridge';
  import { useAppDetail } from '~/stores/app-detail';
  import { useDeployEnvStore } from '~/stores/deploy-env';

  import IntervalSelector from '@/components/interval-selector.vue';

  import type { DateValue } from '@blueking/date-picker';
  import type { Dayjs } from 'dayjs';
  import type {
    GetInstanceTimeSeriesRequest,
    MetricTimeSeries,
    TimeSeriesItem,
  } from '~/@types/v1/bkintegrations-bkmonitor';

  import '@blueking/date-picker/vue3/vue3.css';

  const ChartCardItem = defineAsyncComponent(() => import('@/pages/application/detail/deploy/chart-card-item.vue'));

  type ChartParams = Pick<GetInstanceTimeSeriesRequest, 'endTime' | 'instances' | 'interval' | 'startTime'> & {
    compareOffsets?: number[];
  };

  interface EnvGroup {
    envName: string;
    instances: string[];
  }

  const props = defineProps<{
    envName?: string;
    envNames?: string[];
    initialSelection?: Record<string, string[]>;
  }>();

  /** 监控实例行数据（由 9 个 GetInstanceTimeSeries 指标聚合得到） */
  interface MonitorInstanceRow {
    /** CPU Limit 使用率（%） */
    cpuLimitUsageRate: number;
    /** CPU Request 使用率（%） */
    cpuRequestUsageRate?: number;
    /** CPU 使用量（核数） */
    cpuUsage: number;
    /** 内存 Limit 使用率（%） */
    memLimitUsageRate: number;
    /** 内存 Request 使用率（%） */
    memRequestUsageRate?: number;
    /** 内存使用量（GB） */
    memUsage: number;
    name: string;
    /** 网络入带宽 */
    networkReceive?: number;
    /** 网络出带宽 */
    networkTransmit?: number;
    status: 'Failed' | 'Pending' | 'Running';
  }

  const { t } = useI18n();

  const isShow = defineModel<boolean>('isShow');

  const appDetailStore = useAppDetail();
  const envStore = useDeployEnvStore();
  const sliderWidth = ref<number | string>(1200);

  /** 监听窗口宽度变化，用于小屏适配 */
  const windowWidth = ref(window.innerWidth);
  function onWindowResize() {
    windowWidth.value = window.innerWidth;
  }
  onMounted(() => window.addEventListener('resize', onWindowResize));
  onBeforeUnmount(() => window.removeEventListener('resize', onWindowResize));

  /** 实际显示宽度：全屏时 100%，非全屏时≤1200px，小屏时不超过窗口 90% */
  const effectiveSliderWidth = computed(() => {
    if (sliderWidth.value === '100%') return '100%';
    const maxAllowed = windowWidth.value < 1366 ? Math.floor(windowWidth.value * 0.9) : 1200;
    return maxAllowed;
  });

  /** 按环境分组的实例列表（多环境模式使用） */
  const envGroups = ref<EnvGroup[]>([]);

  /** 是否为多环境模式 */
  const isMultiEnv = computed(() => envGroups.value.length > 1);

  /** 实例 ID → 环境名称映射（传给 chart-card-item 和 bridge，支持多环境查询） */
  const instanceEnvMap = computed(() => {
    const map: Record<string, string> = {};
    for (const group of envGroups.value) {
      for (const inst of group.instances) {
        map[inst] = group.envName;
      }
    }
    return map;
  });

  async function handleClose() {
    isShow.value = false;
    selectedInstance.value = [];
    committedInstances.value = [];
    instances.value = [];
    statusByName.value = {};
    metricData.value = {};
    envGroups.value = [];
    sliderWidth.value = 1200;
    metricFilter.value = 'all';
    // 重置时间对比状态
    timeCompareEnabled.value = false;
    compareOffsetSecondsList.value = [];
    showCustomInput.value = false;
    customOffsetText.value = '';
    compareOffsetOptions.value = compareOffsetOptions.value.filter(opt => typeof opt.id === 'number');
  }

  /** Pod 实例名称列表，作为 GetInstanceTimeSeries 的 instances 入参 */
  const instances = ref<string[]>([]);
  /** 实例名 → Pod 状态映射 */
  const statusByName = ref<Record<string, string>>({});
  /** 实例列表加载中 */
  const loading = ref(false);
  /** 各 metricKey 的时序数据（由子组件 chart-card-item 通过 metric-data 事件回填） */
  const metricData = ref<Partial<Record<string, MetricTimeSeries>>>({});

  // 时间范围（秒级 Unix 时间戳），默认最近 2 天
  const endTime = ref<number>(Math.floor(Date.now() / 1000));
  const startTime = ref<number>(endTime.value - 2 * 24 * 60 * 60);

  // 供 ExploreChart 内部 useEcharts 消费的全局注入：timeRange 控制图表 X 轴显示范围，
  // 必须与实际选择的时间范围同步，否则图表会按默认值裁剪显示窗口
  const timeRangeRef = ref<string[]>([
    `${startTime.value ? dayjs.unix(startTime.value).format('YYYY-MM-DD HH:mm:ss') : 'now/d'}`,
    `${endTime.value ? dayjs.unix(endTime.value).format('YYYY-MM-DD HH:mm:ss') : 'now'}`,
  ]);
  const refreshImmediateRef = ref(0);
  provide('timeRange', timeRangeRef);
  // 与 @blueking/monitor-vue3-components 中 ExploreChart 的通信契约。
  // ExploreChart 内部会 inject("refreshImmediate") 并在 watcher 中依赖它
  provide('refreshImmediate', refreshImmediateRef);
  // 预置时间范围到桥接，确保图表组件挂载首拉即使用正确时间窗
  setChartTimeRange(startTime.value, endTime.value);

  const selectedInstance = ref<string[]>([]);
  const treeRef = ref<InstanceType<typeof Tree>>();
  const selectRef = ref<InstanceType<typeof Select>>();

  /** 供 bk-tree 渲染的数据：环境名作为父节点，实例名作为子节点 */
  interface TreeNodeData {
    children?: TreeNodeData[];
    id: string;
    name: string;
  }
  const treeData = computed<TreeNodeData[]>(() => {
    if (envGroups.value.length <= 1) {
      // 单环境：平铺所有实例，无父节点分组
      const allInstances = envGroups.value.flatMap(g => g.instances);
      return allInstances.map(inst => ({ id: inst, name: inst }));
    }
    return envGroups.value.map(group => ({
      id: group.envName,
      name: group.envName,
      children: group.instances.map(inst => ({ id: inst, name: inst })),
    }));
  });

  /** 标记是否正在程序内部同步树勾选状态，防止 handleNodeChecked 重复触发 */
  const isInternalSync = ref(false);

  /** 是否已全选所有实例 */
  const isAllSelected = computed(() => {
    if (!instances.value.length) return false;
    return selectedInstance.value.length === instances.value.length;
  });

  /** 是否为部分选中状态（控制全选 checkbox 的 indeterminate） */
  const isPartiallySelected = computed(() => selectedInstance.value.length > 0 && !isAllSelected.value);

  /** 树节点勾选变化 → 同步 selectedInstance（仅叶子节点即实例） */
  function handleNodeChecked(nodes: TreeNodeData[]) {
    if (isInternalSync.value) return;
    const leafIds = nodes.filter(node => !node.children?.length).map(node => node.id);
    selectedInstance.value = leafIds;
    selectRef.value?.setSelected(nodes.filter(node => !node.children?.length));
  }

  /** 删除 Select 标签 → 取消树节点勾选 */
  function handleRemoveTag(id: string) {
    treeRef.value?.setCheckedById(id, false);
    selectedInstance.value = selectedInstance.value.filter(i => i !== id);
    selectRef.value?.setSelected(selectedInstance.value.map(i => ({ id: i, name: i })));
    handleFiltersChange();
  }

  /** Select 失焦时，仅值变化才发起请求，避免每次勾选节点都触发 */
  function handleSelectBlur() {
    const current = selectedInstance.value;
    const last = committedInstances.value;
    const isChanged = last.length !== current.length || !last.every(id => current.includes(id));
    if (isChanged) {
      handleFiltersChange();
    }
  }
  /** 全选 / 取消全选 */
  function handleToggleAll() {
    if (!treeRef.value) return;
    const checked = !isAllSelected.value;
    treeRef.value.setChecked(instances.value, checked);
    if (checked) {
      selectedInstance.value = [...instances.value];
      selectRef.value?.setSelected(instances.value.map(id => ({ id, name: id })));
    } else {
      selectedInstance.value = [];
      selectRef.value?.setSelected([]);
    }
  }

  /** 失焦确认后的实例选择（表格用），防止选择过程中表格抖动 */
  const committedInstances = ref<string[]>([]);
  /** 自动刷新间隔（毫秒），-1 表示关闭；来自 IntervalSelector，用于定时触发图表重拉 */
  const autoRefreshInterval = ref<number>(-1);
  /** 汇聚周期（秒），默认 auto（按时间范围与图表宽度自动计算） */
  const interval = ref<number>(1);
  /** 存储任意一个 chart-card-item 子组件实例，用于读取其暴露的 plotWidth */
  const chartCardRef = ref<InstanceType<typeof ChartCardItem> | null>(null);
  /** 函数 ref：v-for 中每次调用都会更新，最终持有最后一个实例（所有实例宽度相同） */
  function setChartCardRef(el: ComponentPublicInstance | Element | null) {
    if (el) chartCardRef.value = el as InstanceType<typeof ChartCardItem>;
  }
  /** 图表绘图区实际宽度（px），来自子组件 ResizeObserver 实时追踪；fallback 850 */
  const chartPlotWidth = computed(() => chartCardRef.value?.plotWidth || 850);

  /** auto 模式下以图表实际可显示像素数为基准计算数据步长，确保每个像素约对应一个数据点 */
  const autoInterval = computed((): number => {
    if (!startTime.value || !endTime.value || startTime.value >= endTime.value) return 60;
    const duration = endTime.value - startTime.value;
    const secondsPerPoint = Math.ceil(duration / chartPlotWidth.value);
    return Math.max(secondsPerPoint, 60); // 最小 60s
  });
  /** 将秒数格式化为可读步长文本（如 120 → "2m", 420 → "7m"） */
  function formatIntervalText(seconds: number): string {
    if (seconds <= 60) return `1m`;
    const minutes = Math.ceil(seconds / 60);
    if (minutes < 60) return `${minutes}m`;
    const hours = Math.ceil(minutes / 60);
    if (hours < 24) return `${hours}h`;
    const days = Math.ceil(hours / 24);
    return `${days}d`;
  }
  /** 实际传给图表的汇聚间隔：auto 模式取计算值，否则取用户手动选择值 */
  const effectiveInterval = computed(() => (interval.value > 1 ? interval.value : autoInterval.value));
  const convergenceOptions = computed(() => [
    { id: 1, name: 'auto' },
    { id: 60, name: t('{0} 秒', [60]) },
    { id: 120, name: t('{0} 分钟', [2]) },
    { id: 300, name: t('{0} 分钟', [5]) },
    { id: 1800, name: t('{0} 分钟', [30]) },
    { id: 3600, name: t('{0} 小时', [1]) },
  ]);

  /** 监控指标分组配置（metricKey + 标题），由父组件统一维护 */
  interface MetricGroup {
    /** area 面积填充 */
    items: Array<{ area?: boolean; metricKey: string; title: string }>;
    /** 分组标识，供 metricFilter 筛选时匹配 */
    key: 'cpu' | 'disk' | 'memory' | 'network';
    name: string;
  }

  /** 拉取全量 Pod 实例（分页累加），返回名称与状态映射 */
  async function fetchInstances(
    appID: string,
    envName: string,
  ): Promise<{ names: string[]; status: Record<string, string> }> {
    const names: string[] = [];
    const status: Record<string, string> = {};
    let page = 1;
    const pageSize = 100;
    let total = Number.POSITIVE_INFINITY;
    while (names.length < total) {
      const res = await InstanceService.listAppInstances({ appID, envName, page, pageSize }).catch(() => ({
        count: '0',
        results: [],
      }));
      const results = (res?.results ?? []) as Array<{ id?: string; status?: string }>;
      results.forEach(r => {
        if (r.id) {
          names.push(r.id);
          status[r.id] = r.status ?? '';
        }
      });
      total = Number(res?.count ?? results.length);
      if (results.length === 0) break;
      page += 1;
    }
    return { names, status };
  }

  /** 取指定实例在某指标时序中的最新数据点数值（优先 stat.last，回退 dataPoints 末项）。
   *  scale 用于百分比单位还原（×100），与图表侧缩放保持一致。 */
  function getLatestValue(series: TimeSeriesItem[] | undefined, instance: string, scale = 1): number {
    const item = series?.find(s => s.instance === instance);
    if (!item) return 0;
    const last = item.stat?.last;
    if (last && last.length >= 2) return (last[1] ?? 0) * scale;
    const points = item.dataPoints;
    if (points && points.length) {
      const tail = points[points.length - 1];
      return (tail?.[0] ?? 0) * scale;
    }
    return 0;
  }
  const metricGroups = computed<MetricGroup[]>(() => [
    {
      key: 'cpu',
      items: [
        { metricKey: 'cpu_usage', title: t('CPU使用量') },
        { metricKey: 'cpu_limit_usage', title: t('CPU limits 使用率') },
        { metricKey: 'cpu_request_usage', title: t('CPU requests 使用率') },
      ],
      name: t('CPU'),
    },
    {
      key: 'memory',
      items: [
        { metricKey: 'memory_usage', title: t('内存使用量（Working Set）') },
        { metricKey: 'memory_limit_usage', title: t('内存 limits 使用率') },
        { metricKey: 'memory_request_usage', title: t('内存 requests 使用率') },
      ],
      name: t('内存'),
    },
    {
      key: 'network',
      items: [
        { metricKey: 'network_receive', title: t('网络入带宽') },
        { metricKey: 'network_transmit', title: t('网络出带宽') },
      ],
      name: t('网络'),
    },
    // {
    //   key: 'disk',
    //   items: [{ metricKey: 'disk_usage', title: t('磁盘使用量'), area: false }],
    //   name: t('存储'),
    // },
  ]);

  /** 各图表卡片共用的查询参数：仅关注实例与时间范围（metricKey 由子组件独立 prop 传入） */
  const chartParams = ref<ChartParams>({
    endTime: endTime.value,
    instances: selectedInstance.value,
    interval: effectiveInterval.value,
    startTime: startTime.value,
    compareOffsets: [],
  });

  // ─── 时间对比状态 ────────────────────────────────────────────
  /** 时间对比开关 */
  const timeCompareEnabled = ref(false);
  /** 多选对比偏移列表（number=秒数，string=自定义文本如'1m'） */
  const compareOffsetSecondsList = ref<(number | string)[]>([]);
  /** 下拉框内是否显示自定义输入框 */
  const showCustomInput = ref(false);
  /** 自定义输入文本 */
  const customOffsetText = ref('');
  /** 对比偏移选项：预设项 + 自定义项 */
  const compareOffsetOptions = ref<Array<{ id: number | string; name: string }>>([
    { id: 3600, name: t('1 小时前') },
    { id: 86400, name: t('昨天') },
    { id: 7 * 86400, name: t('上周') },
    { id: 30 * 86400, name: t('一月前') },
  ]);

  /** 自定义时间对比偏移量上限：2 年（秒） */
  const MAX_COMPARE_OFFSET = 2 * 365 * 86400;

  /** 解析自定义偏移字符串为秒数。格式：/^[1-9]\d*(m|h|d|w|M|y)$/，超过上限返回 0 */
  function parseOffset(text: string): number {
    if (!text) return 0;
    const match = text.trim().match(/^[1-9]\d*(m|h|d|w|M|y)$/);
    if (!match) return 0;
    const unitMap: Record<string, number> = {
      m: 60, h: 3600, d: 86400, w: 7 * 86400, M: 30 * 86400, y: 365 * 86400,
    };
    const result = Number.parseInt(text) * unitMap[match[1]];
    if (result > MAX_COMPARE_OFFSET) return 0;
    return result;
  }
  /** 实际生效的偏移秒数列表 */
  const effectiveCompareOffsets = computed(() =>
    compareOffsetSecondsList.value.map(v => (typeof v === 'string' ? parseOffset(v) : v)),
  );

  function handleTimeCompareToggle(val: boolean) {
    if (!val) {
      showCustomInput.value = false;
      compareOffsetSecondsList.value = [];
    }
  }
  function handleCompareOffsetClear() {
    compareOffsetSecondsList.value = [];
  }
  function handleShowCustomInput() {
    showCustomInput.value = true;
    customOffsetText.value = '';
  }
  function handleCustomOffsetConfirm() {
    const text = customOffsetText.value.trim();
    const seconds = parseOffset(text);
    if (seconds <= 0) {
      const isFormatError = !text.match(/^[1-9]\d*(m|h|d|w|M|y)$/);
      Message({ theme: 'warning', message: isFormatError ? t('按照提示输入') : t('对比时间不能超过 2 年') });
      return;
    }
    const matched = compareOffsetOptions.value.find(item => item.id === seconds);
    const value = matched ? seconds : text;
    if (!matched) {
      const existing = compareOffsetOptions.value.find(item => item.id === text);
      if (!existing) compareOffsetOptions.value.push({ id: text, name: text });
    }
    if (!compareOffsetSecondsList.value.includes(value)) {
      compareOffsetSecondsList.value = [...compareOffsetSecondsList.value, value];
    }
    showCustomInput.value = false;
  }
  // ─── 时间对比 end ───────────────────────────────────

  /** 实例选择变化时同步到 chartParams，触发子组件重新请求 */
  function handleFiltersChange() {
    chartParams.value = {
      endTime: endTime.value,
      instances: selectedInstance.value,
      interval: effectiveInterval.value,
      startTime: startTime.value,
      compareOffsets: timeCompareEnabled.value ? effectiveCompareOffsets.value : [],
    };
    // 失焦确认后同步到表格用 ref，防止选择过程中表格抖动
    committedInstances.value = [...selectedInstance.value];
  }
  /** startTime / endTime / instances / 实例选择变化时同步到 chartParams，触发图表重拉 */
  watch(
    () => [startTime.value, endTime.value, instances.value, interval.value, JSON.stringify(timeCompareEnabled.value ? effectiveCompareOffsets.value : [])],
    () => {
      handleFiltersChange();
      setChartTimeRange(startTime.value, endTime.value);
      // 同步 timeRange，让 ExploreChart X 轴显示完整时间范围
      timeRangeRef.value = [
        dayjs.unix(startTime.value).format('YYYY-MM-DD HH:mm:ss'),
        dayjs.unix(endTime.value).format('YYYY-MM-DD HH:mm:ss'),
      ];
      refreshImmediateRef.value += 1;
    },
  );

  /** 拉取全量 Pod 实例（分页累加），返回名称与状态映射（父组件仅负责实例列表） */
  async function loadInstances() {
    const appID = appDetailStore.appID;
    if (!appID) return;

    // 确定需要拉取实例的环境列表
    const envsToQuery = props.envNames?.length
      ? props.envNames
      : props.envName
        ? [props.envName]
        : [envStore.currentEnv].filter(Boolean);

    if (!envsToQuery.length) return;

    loading.value = true;
    try {
      // 按环境分别请求实例列表
      const allInstances: string[] = [];
      const allStatus: Record<string, string> = {};
      const groups: EnvGroup[] = [];

      for (const envName of envsToQuery) {
        const { names, status } = await fetchInstances(appID, envName).catch(() => ({
          names: [] as string[],
          status: {} as Record<string, string>,
        }));
        groups.push({ envName, instances: names });
        allInstances.push(...names);
        Object.assign(allStatus, status);
      }

      instances.value = allInstances;
      statusByName.value = allStatus;
      envGroups.value = groups;

      // 根据 initialSelection 预选中实例
      if (props.initialSelection) {
        const selected: string[] = [];
        for (const ids of Object.values(props.initialSelection)) {
          selected.push(...ids);
        }
        selectedInstance.value = selected.filter(id => allInstances.includes(id));
      } else if (allInstances.length) {
        selectedInstance.value = allInstances;
      }
    } finally {
      loading.value = false;
    }

    // 同步树组件的初始勾选状态
    await nextTick();
    if (!treeRef.value) return;
    isInternalSync.value = true;
    treeRef.value.setChecked(selectedInstance.value, true);
    isInternalSync.value = false;
  }

  /** 将 Pod 原始状态映射为表格状态枚举 */
  function mapStatus(raw?: string): MonitorInstanceRow['status'] {
    if (!raw) return 'Running';
    if (raw.includes('Pending')) return 'Pending';
    if (raw.includes('Failed') || raw.includes('Error')) return 'Failed';
    return 'Running';
  }

  /** 立即刷新：防抖 300ms，防止连续点击触发多次重拉 */
  const onManualRefresh = debounce(async () => {
    loadInstances();
    refreshImmediateRef.value += 1;
  }, 300);

  /** 自动刷新轮询：直接将 ref 传给 useInterval，每次 start() 读取最新 interval */
  const { start: startPolling, stop: stopPolling } = useInterval(async () => {
    loadInstances();
    refreshImmediateRef.value += 1;
  }, autoRefreshInterval);

  watch(
    [autoRefreshInterval, isShow],
    ([interval, show]) => {
      stopPolling();
      if (interval > 0 && show) {
        startPolling();
      }
    },
    { immediate: true },
  );

  const isFullScreen = computed(() => sliderWidth.value === '100%');

  // 全屏
  function handleFullScreen() {
    if (sliderWidth.value === 1200) {
      sliderWidth.value = '100%';
    } else {
      sliderWidth.value = 1200;
    }
  }

  /** 子组件回传的单个指标时序数据，回填到 metricData 以驱动表格 */
  function onMetricData(payload: { data: MetricTimeSeries | undefined; metricKey: string }) {
    metricData.value = { ...metricData.value, [payload.metricKey]: payload.data };
  }

  watch(isShow, val => {
    if (val) {
      setChartTimeRange(startTime.value, endTime.value);
      loadInstances();
    }
  });

  /** 将 bytes 统一转换为 MiB（1 MiB = 1024^2 bytes），保留 2 位小数；非有限值返回 '-'。 */
  function formatBytes(value: number): string {
    if (!Number.isFinite(value) || value < 0) return '-';
    return `${formatDecimal(value / 1024 / 1024, 2)}`;
  }
  /** 回显时最多显示指定小数位（默认 5 位），截断不四舍五入；非有限值返回 '-'。
   *  注意：此处仅用于展示，聚合/派生计算仍基于真实值，避免截断累积误差。 */
  function formatDecimal(value: number, digits = 2): string {
    if (value === null || value === undefined || !Number.isFinite(value)) return '-';
    const negative = value < 0 ? '-' : '';
    const abs = Math.abs(value);
    const [intPart, decPart = ''] = abs.toString().split('.');
    const trimmed = decPart.slice(0, digits);
    return negative + (trimmed ? `${intPart}.${trimmed}` : intPart);
  }

  /** 表格数据源：由各指标最新数据点聚合为实例行（保留真实值，回显由 formatDecimal 截取） */
  const tableData = computed<MonitorInstanceRow[]>(() => {
    const rows = instances.value.map(name => {
      // 百分比指标（利用率）按 unit 放大 100 倍，与图表侧缩放保持一致
      const pctScale = (key: keyof NonNullable<typeof metricData.value>) =>
        isPercentUnit(metricData.value[key]?.unit) ? 100 : 1;
      const cpuUsage = getLatestValue(metricData.value.cpu_usage?.series, name);
      const cpuLimitUsageRate = getLatestValue(
        metricData.value.cpu_limit_usage?.series,
        name,
        pctScale('cpu_limit_usage'),
      );
      const cpuRequestUsageRate = getLatestValue(
        metricData.value.cpu_request_usage?.series,
        name,
        pctScale('cpu_request_usage'),
      );
      const memUsage = getLatestValue(metricData.value.memory_usage?.series, name);
      const memLimitUsageRate = getLatestValue(
        metricData.value.memory_limit_usage?.series,
        name,
        pctScale('memory_limit_usage'),
      );
      const memRequestUsageRate = getLatestValue(
        metricData.value.memory_request_usage?.series,
        name,
        pctScale('memory_request_usage'),
      );
      const networkReceive = getLatestValue(metricData.value.network_receive?.series, name);
      const networkTransmit = getLatestValue(metricData.value.network_transmit?.series, name);
      return {
        name,
        cpuUsage,
        cpuLimitUsageRate,
        cpuRequestUsageRate,
        memUsage,
        memLimitUsageRate,
        memRequestUsageRate,
        networkReceive,
        networkTransmit,
        status: mapStatus(statusByName.value[name]),
      } as MonitorInstanceRow;
    });
    return rows.filter(row => committedInstances.value.includes(row.name));
  });

  const metricFilter = ref<'all' | 'cpu' | 'disk' | 'memory' | 'network'>('all');

  /** 根据 metricFilter 判断某分组是否可见（供 v-show 控制各模块显隐） */
  const isMetricGroupVisible = computed(
    () => (key: MetricGroup['key']) => metricFilter.value === 'all' || metricFilter.value === key,
  );

  const value = ref<DateValue | undefined>(['now/d', 'now']);
  const handleValueChange = (
    _val: DateValue | undefined,
    info: { dayjs: Dayjs | null; formatText: null | string }[],
  ) => {
    const start = info?.[0]?.dayjs;
    const end = info?.[1]?.dayjs;
    if (start && end) {
      startTime.value = start.unix();
      endTime.value = end.unix();
    }
  };
</script>
<style lang="postcss" scoped>
  :deep(.bk-modal-content) {
    scrollbar-gutter: auto !important;
    background-color: #f5f7fa;
  }
  :deep(.bk-select .bk-select-trigger .bk-select-tag:not(.collapse-tag)) {
    z-index: 999;
  }
  :deep(.bk-tree-node .bk-node-content) {
    gap: 6px;
  }
  .hide-tree-action :deep(.is-root.bk-tree-node .bk-node-action) {
    margin-right: 0 !important;
    min-width: 7px !important;
  }
</style>
