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
            v-model="dateRangeValue"
            behavior="simplicity"
            :format="dateFormat"
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
      class="monitor-toolbar flex-shrink-0 border-b border-[#DCDEE5] bg-[#fff] px-[24px] py-[8px] shadow-[0_3px_5.5px_0_#00000012]"
    >
      <div class="flex flex-wrap items-center gap-x-[16px] gap-y-[8px]">
        <div class="monitor-instance-selector flex items-center min-w-0">
          <span
            class="flex-shrink-0 whitespace-nowrap bg-[#FAFBFD] h-[32px] leading-[30px] text-[#4D4F56] text-[12px] px-[6px] b b-r-0 b-[#c4c6cc]"
            >{{ $t('实例名称') }}</span
          >
          <Select
            ref="selectRef"
            v-model="selectedInstance"
            class="flex-1 min-w-0"
            clearable
            collapse-tags
            custom-content
            display-key="name"
            id-key="id"
            multiple
            multiple-mode="tag"
            :placeholder="instancePlaceholder"
            @clear="handleClearSelection"
            @tag-remove="handleRemoveTag"
            @toggle="handleSelectBlur"
          >
            <div
              class="px-[12px] py-[6px]"
              @click.stop
              @keydown.stop
            >
              <Input
                v-model="searchText"
                clearable
                :placeholder="$t('输入实例名称搜索')"
              />
            </div>
            <Tree
              v-if="visibleInstances.length"
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
            <div
              v-else
              class="px-[12px] py-[8px] text-[12px] text-[#979BA5]"
            >
              {{ searchText.trim() ? $t('暂无数据') : $t('共 {0} 个实例，输入名称后再选择', [instanceCount]) }}
            </div>
            <div
              v-if="hasMoreSearchResults"
              class="px-[12px] py-[6px] text-[12px] text-[#979BA5]"
            >
              {{ $t('仅展示前 100 个结果，请缩小搜索范围') }}
            </div>
          </Select>
        </div>
        <div class="monitor-ranking flex flex-wrap items-center max-w-full gap-y-[8px]">
          <div class="flex items-center">
            <span class="monitor-ranking-label">
              {{ $t('排序') }}
              <i
                v-bk-tooltips="rankingTooltip"
                :aria-label="rankingTooltip"
                class="bkms-icon bkms-icon-help-document text-[#979BA5] ml-[4px]"
                role="img"
              />
            </span>
            <Select
              v-model="rankBy"
              class="monitor-ranking-metric flex-shrink-0"
              :clearable="false"
              :disabled="isRankingDisabled"
            >
              <Select.Option
                v-for="column in rankColumns"
                :id="column.key"
                :key="column.key"
                :name="t(column.metricLabel)"
              />
            </Select>
          </div>
          <Select
            v-model="rankType"
            class="monitor-ranking-direction flex-shrink-0"
            :clearable="false"
            :disabled="isRankingDisabled"
          >
            <Select.Option
              id="top"
              :name="$t('降序')"
            />
            <Select.Option
              id="bottom"
              :name="$t('升序')"
            />
          </Select>
          <div class="flex items-center">
            <span class="monitor-ranking-label">Limit</span>
            <!-- 数字步进按钮通过 change 提交，即使输入框未获得焦点。 -->
            <Input
              v-model="limitDraft"
              class="monitor-ranking-limit focus-within:z-1"
              :disabled="isRankingDisabled"
              :max="50"
              :min="1"
              placeholder="1–50"
              :precision="0"
              type="number"
              @blur="commitLimit"
              @change="commitLimit"
              @enter="commitLimit"
            />
          </div>
        </div>
      </div>
      <div class="monitor-toolbar-options flex items-center gap-[16px] mt-[8px]">
        <!-- 汇聚周期 -->
        <Select
          v-model="interval"
          class="w-[220px] flex-shrink-0"
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
        <Select
          v-model="method"
          class="w-[220px] flex-shrink-0"
          :clearable="false"
        >
          <template #prefix>
            <span class="bg-[#FAFBFD] leading-[32px] text-[#4D4F56] text-[12px] px-[6px] b-r b-[#c4c6cc]">{{
              $t('汇聚方法')
            }}</span>
          </template>
          <Select.Option
            v-for="option in methodOptions"
            :id="option"
            :key="option"
            :name="`${option.toUpperCase()}(PromQL)`"
          />
        </Select>
        <i
          v-if="method === 'count'"
          v-bk-tooltips="$t('COUNT 下使用率指标的分子、分母分别计数，结果恒为 100%')"
          class="bkms-icon bkms-icon-help-document text-[#979BA5] ml-[-8px]"
        />
        <!-- 时间对比 -->
        <div class="flex items-center flex-shrink-0">
          <span class="text-[#4D4F56] text-[12px] mr-[8px]">{{ $t('时间对比') }}</span>
          <Switcher
            v-model="timeCompareEnabled"
            v-bk-tooltips="{ content: $t('多环境不支持时间对比'), disabled: !isMultiEnv, placement: 'top' }"
            :disabled="isMultiEnv"
            theme="primary"
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
                    >{{ $t('自定义') }}</span
                  >
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
                        <span
                          class="text-[#3A84FF] cursor-pointer"
                          @click="handleCustomOffsetConfirm"
                          >{{ $t('确定') }}</span
                        >
                      </template>
                    </Input>
                  </template>
                </div>
              </template>
            </Select>
          </template>
        </div>
      </div>
    </div>
    <div
      v-bkloading="{ loading: isShow && chartPlotWidth > 0 && !queryReady }"
      class="min-h-0 flex-1 overflow-auto px-[24px] pt-[24px] pb-[24px] bg-[#F5F7FA]"
    >
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
          v-for="column in rankColumns"
          :key="column.key"
          :field="column.field"
          :label="t(column.tableLabel)"
          :min-width="column.width"
        >
          <template #header>
            <button
              :aria-label="
                !isRankingDisabled && rankBy === column.key
                  ? `${t(column.tableLabel)} (${rankDirectionLabel})`
                  : undefined
              "
              :aria-pressed="!isRankingDisabled && rankBy === column.key"
              class="monitor-rank-header"
              :disabled="isRankingDisabled"
              :title="!isRankingDisabled && rankBy === column.key ? rankingColumnTooltip : undefined"
              type="button"
              @click.stop="sortByColumn(column.key)"
            >
              {{ t(column.tableLabel) }}
              <span
                aria-hidden="true"
                class="monitor-rank-arrows"
              >
                <span :class="{ active: !isRankingDisabled && rankBy === column.key && rankType === 'bottom' }" />
                <span :class="{ active: !isRankingDisabled && rankBy === column.key && rankType === 'top' }" />
              </span>
            </button>
          </template>
          <template #default="{ row }">
            <div
              class="flex items-center"
              :class="{ 'gap-[4px]': column.key === 'cpu_limit_usage' }"
            >
              <Progress
                v-if="column.key === 'cpu_limit_usage' && row.cpuLimitUsageRate !== undefined"
                color="#3A84FF"
                :percent="Math.min(row.cpuLimitUsageRate, 100)"
                :show-text="false"
                :stroke-width="6"
                type="line"
              />
              <span class="text-[#63656E] text-[12px]">{{ column.format(row[column.field]) }}</span>
            </div>
          </template>
        </TableColumn>
      </Table>

      <!-- 图表 -->
      <template v-if="isShow">
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
            :area="item.area !== false"
            :compare-offsets="chartParams.compareOffsets"
            :end-time="chartParams.endTime"
            :env-names="chartParams.envNames"
            :instances="chartParams.instances"
            :interval="chartParams.interval"
            :method="chartParams.method"
            :metric-key="item.metricKey"
            :query-enabled="queryReady"
            :query-id="chartParams.queryId"
            :start-time="chartParams.startTime"
            :title="item.title"
            @metric-data="onMetricData"
            @plot-width-change="handlePlotWidthChange"
          >
            <template #title>
              <div class="flex items-center">
                <span class="text-[#313238] text-[14px] font-[500] mr-[8px]">{{ item.title }}</span>
                <Tag v-bk-tooltips="$t('数据步长')">{{ formatIntervalText(chartParams.interval || 0) }}</Tag>
              </div>
            </template>
          </chart-card-item>
        </ToggleCard>
      </template>
    </div>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, defineAsyncComponent, nextTick, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue';

  import DatePicker, { DateRange } from '@blueking/date-picker';
  import { Table, TableColumn } from '@blueking/table';
  import { Divider, Input, Message, Progress, Radio, Select, Sideslider, Switcher, Tag, Tree } from 'bkui-vue';
  import dayjs from 'dayjs';
  import { ceil, debounce } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import IntervalSelector from '~/components/interval-selector.vue';
  import useInterval from '~/composables/use-interval';
  import {
    fetchRankedMonitorInstances,
    getMonitorInstanceKey,
    getMonitorInstanceLabel,
    invalidateMonitorChartQueries,
    isPercentUnit,
  } from '~/pages/application/detail/deploy/monitor-chart-bridge';
  import { useAppDetail } from '~/stores/app-detail';
  import { useDeployEnvStore } from '~/stores/deploy-env';

  import type { MonitorInstanceGroup } from './instance-list/types';
  import type {
    MonitorInstance,
    MonitorMetricQuery,
    MonitorMetricTimeSeries,
    MonitorRankBy,
    RankedMonitorInstance,
  } from './monitor-chart-bridge';
  import type { DateValue } from '@blueking/date-picker';
  import type { Dayjs } from 'dayjs';
  import type { TimeSeriesItem } from '~/@types/v1/bkintegrations-bkmonitor';

  import '@blueking/date-picker/vue3/vue3.css';

  const ChartCardItem = defineAsyncComponent(() => import('~/pages/application/detail/deploy/chart-card-item.vue'));

  type ChartParams = Omit<MonitorMetricQuery, 'appID' | 'metricKey'> & {
    compareOffsets: number[];
    envNames: string[];
    queryId: number;
    ranking?: {
      by: MonitorRankBy;
      instances: Map<string, RankedMonitorInstance>;
    };
  };
  const props = defineProps<{
    envName?: string;
    envNames?: string[];
    initialSelection?: Record<string, string[]>;
    instanceGroups?: MonitorInstanceGroup[];
  }>();
  interface MonitorInstanceRow {
    cpuLimitUsageRate: number | undefined;
    cpuUsage: number | undefined;
    memLimitUsageRate: number | undefined;
    memUsage: number | undefined;
    name: string;
  }

  const { t } = useI18n();

  const isShow = defineModel<boolean>('isShow');

  const appDetailStore = useAppDetail();
  const envStore = useDeployEnvStore();
  const sliderWidth = ref<number | string>(1200);
  const queryReady = ref(false);
  let queryGeneration = 0;

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

  const queryEnvNames = computed(() =>
    Array.from(
      new Set(props.envNames?.length ? props.envNames : [props.envName || envStore.currentEnv].filter(Boolean)),
    ),
  );
  const envGroups = computed(() =>
    (props.instanceGroups || []).filter(group => queryEnvNames.value.includes(group.envName)),
  );
  const instanceCount = computed(() => envGroups.value.reduce((total, group) => total + group.instances.length, 0));
  const instancePlaceholder = computed(() =>
    envGroups.value.length === queryEnvNames.value.length ? t('全部实例（{0}）', [instanceCount.value]) : t('全部实例'),
  );
  const isMultiEnv = computed(() => queryEnvNames.value.length > 1);
  // 独立保留已选实例及其环境，快照中实例消失后仍可刷新查询历史数据。
  const selectedInstanceDetails = ref<Record<string, MonitorInstance>>({});
  const instancesByKey = computed(() => {
    const map = new Map<string, MonitorInstance>();
    for (const group of envGroups.value) {
      for (const instance of group.instances) {
        const item = { envName: group.envName, instance };
        map.set(getMonitorInstanceKey(item), item);
      }
    }
    for (const [key, item] of Object.entries(selectedInstanceDetails.value)) map.set(key, item);
    return map;
  });
  function handleClose() {
    isShow.value = false;
    queryReady.value = false;
    chartPlotWidth.value = 0;
    selectedInstance.value = [];
    selectedInstanceDetails.value = {};
    committedInstances.value = [];
    searchText.value = '';
    metricData.value = {};
    chartParams.value.instances = [];
    sliderWidth.value = 1200;
    metricFilter.value = 'all';
    timeCompareEnabled.value = false;
    compareOffsetSecondsList.value = [];
    showCustomInput.value = false;
    customOffsetText.value = '';
    compareOffsetOptions.value = compareOffsetOptions.value.filter(opt => typeof opt.id === 'number');
    onManualRefresh.cancel();
  }
  const metricData = ref<Partial<Record<string, MonitorMetricTimeSeries>>>({});

  // 与日期选择器默认的「今天（至今）」保持一致。
  const dateFormat = 'YYYY-MM-DD HH:mm:ss';
  const dateRangeValue = ref<DateValue | undefined>(['now/d', 'now']);
  const now = dayjs();
  const endTime = ref(now.unix());
  const startTime = ref(now.startOf('day').unix());
  const refreshRevision = ref(0);

  function setQueryTimeRange(start?: Dayjs | null, end?: Dayjs | null) {
    if (!start?.isValid() || !end?.isValid()) return;
    startTime.value = start.unix();
    endTime.value = end.unix();
  }

  function syncPickerTimeRange() {
    if (!dateRangeValue.value) return;
    const { startDate, endDate } = new DateRange(dateRangeValue.value, dateFormat);
    setQueryTimeRange(startDate, endDate);
  }

  // 供 ExploreChart 内部 useEcharts 消费的全局注入：timeRange 控制图表 X 轴显示范围，
  // 必须与实际选择的时间范围同步，否则图表会按默认值裁剪显示窗口
  const timeRangeRef = ref<string[]>([
    dayjs.unix(startTime.value).format(dateFormat),
    dayjs.unix(endTime.value).format(dateFormat),
  ]);
  provide('timeRange', timeRangeRef);
  // ExploreChart 依赖这项注入。侧栏通过 queryId 更新图表，不再额外触发组件内部刷新。
  provide('refreshImmediate', ref(0));

  const selectedInstance = ref<string[]>([]);
  const committedInstances = ref<string[]>([]);
  const isRankingDisabled = computed(() => selectedInstance.value.length > 0);
  const searchText = ref('');
  const treeRef = ref<InstanceType<typeof Tree>>();
  const selectRef = ref<InstanceType<typeof Select>>();
  const DEFAULT_RANK_BY: MonitorRankBy = 'cpu_limit_usage';
  const DEFAULT_RANKING_LIMIT = 10;
  // 查询只使用已提交的 Limit，避免键入过程中的中间值重复触发排名和时序请求。
  const limitDraft = ref<number | string>(DEFAULT_RANKING_LIMIT);
  const committedLimit = ref(DEFAULT_RANKING_LIMIT);
  function commitLimit() {
    const parsed = Number(limitDraft.value);
    committedLimit.value = Number.isFinite(parsed)
      ? Math.min(50, Math.max(1, Math.floor(parsed)))
      : DEFAULT_RANKING_LIMIT;
    limitDraft.value = committedLimit.value;
  }

  const cpuUsageFormatter = new Intl.NumberFormat('en-US', {
    maximumFractionDigits: 2,
    minimumFractionDigits: 2,
    useGrouping: false,
  });
  const rankColumns = [
    {
      key: 'cpu_limit_usage',
      field: 'cpuLimitUsageRate',
      metricLabel: 'CPU 使用率',
      tableLabel: 'CPU 使用率',
      width: 200,
      format: formatPercent,
    },
    {
      key: 'memory_limit_usage',
      field: 'memLimitUsageRate',
      metricLabel: '内存使用率',
      tableLabel: '内存使用率',
      width: 120,
      format: formatPercent,
    },
    {
      key: 'cpu_usage',
      field: 'cpuUsage',
      metricLabel: 'CPU 用量',
      tableLabel: 'CPU 用量（cores）',
      width: 120,
      format: formatCpuUsage,
    },
    {
      key: 'memory_usage',
      field: 'memUsage',
      metricLabel: '内存用量',
      tableLabel: '内存用量（MiB）',
      width: 120,
      format: formatBytes,
    },
  ] as const;
  const rankBy = ref<MonitorRankBy>(DEFAULT_RANK_BY);
  const rankType = ref<'bottom' | 'top'>('top');
  const methodOptions = ['sum', 'max', 'min', 'count', 'avg'] as const;
  const method = ref<(typeof methodOptions)[number]>('sum');
  const rankDirectionLabel = computed(() => t(rankType.value === 'top' ? '降序' : '升序'));
  const rankingColumnTooltip = computed(() => t('排名值为所选时段内 {0} 汇聚结果的均值', [method.value.toUpperCase()]));
  const rankingTooltip = computed(() => {
    if (isRankingDisabled.value) return t('选择实例后展示全部已选实例');
    const metric = rankColumns.find(column => column.key === rankBy.value);
    return t('所有图表按「{0}」的 {1} 汇聚结果在所选时段内的均值{2}，取前 {3} 个实例', [
      t(metric?.metricLabel ?? 'CPU 使用率'),
      method.value.toUpperCase(),
      rankDirectionLabel.value,
      committedLimit.value,
    ]);
  });

  interface TreeNodeData {
    children?: TreeNodeData[];
    id: string;
    name: string;
  }

  function sortByColumn(key: MonitorRankBy) {
    if (isRankingDisabled.value) return;
    rankType.value = rankBy.value === key && rankType.value === 'top' ? 'bottom' : 'top';
    rankBy.value = key;
  }

  // 只构建有限的搜索候选，第 101 项用于判断是否需要缩小范围。
  const searchMatches = computed(() => {
    const keyword = searchText.value.trim().toLowerCase();
    if (!keyword) return [];
    const matches: string[] = [];
    for (const group of envGroups.value) {
      for (const instance of group.instances) {
        if (instance.toLowerCase().includes(keyword))
          matches.push(getMonitorInstanceKey({ envName: group.envName, instance }));
        if (matches.length > 100) return matches;
      }
    }
    return matches;
  });
  const visibleInstances = computed(() => searchMatches.value.slice(0, 100));
  const hasMoreSearchResults = computed(() => searchMatches.value.length > 100);
  const treeData = computed<TreeNodeData[]>(() => {
    if (envGroups.value.length <= 1)
      return visibleInstances.value.map(id => ({ id, name: instancesByKey.value.get(id)?.instance || '' }));
    const visible = new Set(visibleInstances.value);
    return envGroups.value
      .map(group => ({
        id: 'env:' + group.envName,
        name: group.envName,
        children: group.instances
          .map(instance => ({ id: getMonitorInstanceKey({ envName: group.envName, instance }), name: instance }))
          .filter(item => visible.has(item.id)),
      }))
      .filter(group => group.children.length);
  });
  const isInternalSync = ref(false);
  async function syncSelection() {
    await nextTick();
    selectRef.value?.setSelected(
      selectedInstance.value.map(id => ({
        id,
        name: getMonitorInstanceLabel(instancesByKey.value.get(id) || {}, isMultiEnv.value),
      })),
    );
    if (!treeRef.value) return;
    isInternalSync.value = true;
    treeRef.value.setChecked(visibleInstances.value, false);
    treeRef.value.setChecked(
      selectedInstance.value.filter(id => visibleInstances.value.includes(id)),
      true,
    );
    isInternalSync.value = false;
  }
  watch(treeData, syncSelection);
  function handleClearSelection() {
    updateSelection([]);
    handleFiltersChange();
  }
  function handleNodeChecked(nodes: TreeNodeData[]) {
    if (isInternalSync.value) return;
    const visible = new Set(visibleInstances.value);
    updateSelection([
      ...selectedInstance.value.filter(id => !visible.has(id)),
      ...nodes.filter(node => visible.has(node.id)).map(node => node.id),
    ]);
  }
  function handleRemoveTag(id: string) {
    updateSelection(selectedInstance.value.filter(instance => instance !== id));
    handleFiltersChange();
  }
  function handleSelectBlur(open: boolean) {
    if (open) return;
    const current = selectedInstance.value;
    const last = committedInstances.value;
    if (last.length !== current.length || !last.every(id => current.includes(id))) handleFiltersChange();
  }
  function updateSelection(ids: string[]) {
    const uniqueIds = Array.from(new Set(ids));
    if (uniqueIds.length > 50) {
      Message({ theme: 'warning', message: t('最多选择 50 个实例') });
      void syncSelection();
      return;
    }
    selectedInstanceDetails.value = Object.fromEntries(
      uniqueIds.flatMap(id => {
        const instance = instancesByKey.value.get(id);
        return instance ? [[id, instance]] : [];
      }),
    );
    selectedInstance.value = uniqueIds;
    void syncSelection();
  }

  /** 自动刷新间隔（毫秒），-1 表示关闭；来自 IntervalSelector，用于定时触发图表重拉 */
  const autoRefreshInterval = ref<number>(-1);
  /** 汇聚周期（秒），默认 auto（按时间范围与图表宽度自动计算） */
  const interval = ref<number>(1);
  /** 卡片先测量容器，再启用查询，避免用默认宽度请求后又按实际宽度重查。 */
  const chartPlotWidth = ref(0);
  function handlePlotWidthChange(width: number) {
    if (isShow.value && width > 0) chartPlotWidth.value = width;
  }

  /** auto 模式下以图表实际可显示像素数为基准计算数据步长，确保每个像素约对应一个数据点 */
  const autoInterval = computed((): number => {
    if (!chartPlotWidth.value || !startTime.value || !endTime.value || startTime.value >= endTime.value) return 60;
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

  const chartParams = ref<ChartParams>({
    endTime: endTime.value,
    startTime: startTime.value,
    envNames: [],
    instances: [],
    method: 'sum',
    interval: effectiveInterval.value,
    compareOffsets: [],
    queryId: 0,
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
      m: 60,
      h: 3600,
      d: 86400,
      w: 7 * 86400,
      M: 30 * 86400,
      y: 365 * 86400,
    };
    const result = Number.parseInt(text) * unitMap[match[1]];
    if (result > MAX_COMPARE_OFFSET) return 0;
    return result;
  }
  /** 实际生效的偏移秒数列表 */
  const effectiveCompareOffsets = computed(() =>
    timeCompareEnabled.value && !isMultiEnv.value
      ? compareOffsetSecondsList.value.map(v => (typeof v === 'string' ? parseOffset(v) : v))
      : [],
  );

  function handleCompareOffsetClear() {
    compareOffsetSecondsList.value = [];
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
  async function handleFiltersChange() {
    if (!isShow.value || !chartPlotWidth.value) return;
    const generation = ++queryGeneration;
    const params = {
      endTime: endTime.value,
      startTime: startTime.value,
      envNames: [...queryEnvNames.value],
      method: method.value,
      interval: effectiveInterval.value,
    };
    committedInstances.value = [...selectedInstance.value];
    invalidateMonitorChartQueries();
    queryReady.value = false;
    // 排名只在当前期做一次。收到名单后再用同一份查询快照挂载所有图表。
    const currentRankBy = rankBy.value;
    const ranked = committedInstances.value.length
      ? undefined
      : await fetchRankedMonitorInstances({
          ...params,
          appID: appDetailStore.appID,
          limit: committedLimit.value,
          rankBy: currentRankBy,
          rankType: rankType.value,
        });
    if (!isShow.value || generation !== queryGeneration) return;
    const instances = ranked
      ? ranked.map(({ envName, instance }) => ({ envName, instance }))
      : committedInstances.value.flatMap(id => {
          const instance = selectedInstanceDetails.value[id];
          return instance && params.envNames.includes(instance.envName) ? [instance] : [];
        });
    metricData.value = {};
    timeRangeRef.value = [
      dayjs.unix(params.startTime).format(dateFormat),
      dayjs.unix(params.endTime).format(dateFormat),
    ];
    chartParams.value = {
      ...params,
      instances,
      compareOffsets: effectiveCompareOffsets.value,
      queryId: generation,
      ranking: ranked && {
        by: currentRankBy,
        instances: new Map(ranked.map(item => [getMonitorInstanceKey(item), item])),
      },
    };
    queryReady.value = true;
  }
  function handleShowCustomInput() {
    showCustomInput.value = true;
    customOffsetText.value = '';
  }
  // ─── 时间对比 end ───────────────────────────────────

  function handleTimeCompareToggle(val: boolean) {
    if (!val) {
      showCustomInput.value = false;
      compareOffsetSecondsList.value = [];
    }
  }
  // 逐项比较查询条件，固定汇聚周期时，单纯尺寸变化不应再次请求。
  watch(
    [
      () => startTime.value,
      () => endTime.value,
      () => effectiveInterval.value,
      () => chartPlotWidth.value > 0,
      () => committedLimit.value,
      () => rankBy.value,
      () => rankType.value,
      () => method.value,
      () => appDetailStore.appID,
      () => JSON.stringify(queryEnvNames.value),
      () => refreshRevision.value,
    ],
    handleFiltersChange,
  );
  watch(
    () => JSON.stringify(effectiveCompareOffsets.value),
    () => {
      // 对比时段沿用当前期名单。排名尚未返回时由该轮查询提交最新偏移量。
      if (!isShow.value || !queryReady.value) return;
      chartParams.value = {
        ...chartParams.value,
        compareOffsets: effectiveCompareOffsets.value,
        queryId: ++queryGeneration,
      };
    },
  );
  function refresh() {
    // 重新解析 now 等相对时间。由同一个 watcher 提交时间和刷新请求，避免重复查询。
    syncPickerTimeRange();
    refreshRevision.value += 1;
  }
  const onManualRefresh = debounce(refresh, 300);
  const { start: startPolling, stop: stopPolling } = useInterval(async () => {
    // 排名仍在进行时跳过本次刷新，避免慢请求累积。
    if (queryReady.value) refresh();
  }, autoRefreshInterval);
  onBeforeUnmount(() => {
    queryGeneration += 1;
    if (isShow.value) invalidateMonitorChartQueries();
    stopPolling();
    onManualRefresh.cancel();
  });

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

  function onMetricData(payload: { data: MonitorMetricTimeSeries | undefined; metricKey: string; queryId: number }) {
    if (!isShow.value || !queryReady.value || payload.queryId !== chartParams.value.queryId) return;
    metricData.value = { ...metricData.value, [payload.metricKey]: payload.data };
  }
  watch(
    isShow,
    (show, previousShow) => {
      queryReady.value = false;
      chartPlotWidth.value = 0;
      if (!show) {
        queryGeneration += 1;
        if (previousShow) invalidateMonitorChartQueries();
        onManualRefresh.cancel();
        return;
      }
      syncPickerTimeRange();
      const selection: Record<string, MonitorInstance> = {};
      for (const [envName, names] of Object.entries(props.initialSelection || {})) {
        for (const instance of names) {
          const item = { envName, instance };
          selection[getMonitorInstanceKey(item)] = item;
        }
      }
      selectedInstanceDetails.value = selection;
      selectedInstance.value = Object.keys(selection);
      searchText.value = '';
      limitDraft.value = DEFAULT_RANKING_LIMIT;
      committedLimit.value = DEFAULT_RANKING_LIMIT;
      rankBy.value = DEFAULT_RANK_BY;
      rankType.value = 'top';
      method.value = 'sum';
      void syncSelection();
    },
    { immediate: true },
  );

  /** 将 bytes 统一转换为 MiB（1 MiB = 1024^2 bytes）；非有限值返回 '-'。 */
  function formatBytes(value: number | undefined): string {
    if (value === undefined || !Number.isFinite(value) || value < 0) return '-';
    return formatDecimal(value / 1024 / 1024);
  }
  function formatCpuUsage(value: number | undefined): string {
    if (value === undefined || !Number.isFinite(value)) return '-';
    const roundedUp = Number.isInteger(value) ? value : ceil(value, 2);
    return roundedUp === 0 ? '0.00' : cpuUsageFormatter.format(roundedUp);
  }
  /** 常规值截断至两位小数，极小的非零值用科学计数法。 */
  function formatDecimal(value: number | undefined): string {
    if (value === null || value === undefined || !Number.isFinite(value)) return '-';
    const abs = Math.abs(value);
    if (abs > 0 && abs < 0.01) return value.toExponential();
    const decimal = value.toString();
    if (decimal.includes('e')) return decimal;
    const [intPart, decPart = ''] = decimal.split('.');
    const trimmed = decPart.slice(0, 2);
    return trimmed ? `${intPart}.${trimmed}` : intPart;
  }
  function formatPercent(value: number | undefined): string {
    return value === undefined ? '-' : `${formatDecimal(value)}%`;
  }

  function getRankValue(item: RankedMonitorInstance | undefined): number | undefined {
    const { unit, value } = item ?? {};
    if (typeof value !== 'number' || !Number.isFinite(value)) return undefined;
    if (unit === 'percent') return ratioToPercent(value);
    // 内存列在渲染时将原始 bytes 统一换算为 MiB。
    if (unit === 'cores' || unit === 'bytes') return value;
    return undefined;
  }

  function ratioToPercent(value: number): number {
    // 用十进制指数移动小数点，避免 0.29 * 100 的误差在截断时显示成 28.99%。
    const [mantissa, exponent = '0'] = value.toString().split('e');
    return Number(`${mantissa}e${Number(exponent) + 2}`);
  }

  const tableData = computed<MonitorInstanceRow[]>(() => {
    if (!queryReady.value) return [];
    const metrics = metricData.value;
    const { instances, ranking } = chartParams.value;
    const values = new Map<string, Map<string, TimeSeriesItem>>();
    for (const { key } of rankColumns) {
      values.set(key, new Map((metrics[key]?.series || []).map(series => [getMonitorInstanceKey(series), series])));
    }
    return instances.map(instance => {
      const instanceKey = getMonitorInstanceKey(instance);
      const getValue = (key: MonitorRankBy) => {
        if (ranking?.by === key) return getRankValue(ranking.instances.get(instanceKey));
        const item = values.get(key)?.get(instanceKey);
        const value = item?.stat?.last?.[1];
        if (typeof value !== 'number' || !Number.isFinite(value)) return undefined;
        return isPercentUnit(metrics[key]?.unit) ? ratioToPercent(value) : value;
      };
      return {
        name: getMonitorInstanceLabel(instance, isMultiEnv.value),
        cpuLimitUsageRate: getValue('cpu_limit_usage'),
        memLimitUsageRate: getValue('memory_limit_usage'),
        cpuUsage: getValue('cpu_usage'),
        memUsage: getValue('memory_usage'),
      };
    });
  });

  const metricFilter = ref<'all' | 'cpu' | 'disk' | 'memory' | 'network'>('all');

  /** 根据 metricFilter 判断某分组是否可见（供 v-show 控制各模块显隐） */
  const isMetricGroupVisible = computed(
    () => (key: MetricGroup['key']) => metricFilter.value === 'all' || metricFilter.value === key,
  );

  const handleValueChange = (
    _val: DateValue | undefined,
    info: { dayjs: Dayjs | null; formatText: null | string }[],
  ) => {
    setQueryTimeRange(info?.[0]?.dayjs, info?.[1]?.dayjs);
  };
</script>
<style lang="postcss" scoped>
  :deep(.bk-modal-body) {
    display: flex;
    flex-direction: column;
  }

  :deep(.bk-modal-content) {
    display: flex;
    flex: 1;
    min-height: 0;
    overflow: hidden !important;
    scrollbar-gutter: auto !important;
    background-color: #f5f7fa;
  }

  :deep(.bk-sideslider-content) {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  :deep(.bk-select .bk-select-trigger .bk-select-tag:not(.collapse-tag)) {
    z-index: 999;
  }

  :deep(.bk-tree-node .bk-node-content) {
    gap: 6px;
  }

  .monitor-toolbar-options {
    overflow: auto hidden;
  }

  .monitor-instance-selector {
    flex: 1 1 320px;
  }

  .monitor-ranking-label {
    @apply flex h-[32px] shrink-0 items-center whitespace-nowrap b b-r-0 b-[#c4c6cc] bg-[#FAFBFD] px-[6px] text-[12px] text-[#4D4F56];
  }

  .monitor-ranking-metric {
    width: 128px;
  }

  .monitor-ranking-direction {
    width: 112px;
  }

  .monitor-ranking-limit {
    width: 88px;
  }

  .monitor-rank-header {
    font: inherit;
    color: inherit;

    @apply inline-flex cursor-pointer items-center gap-[6px] border-0 bg-transparent p-0;

    &[aria-pressed='true'] {
      color: #3a84ff;
    }

    &:disabled {
      cursor: default;
    }
  }

  .monitor-rank-arrows {
    @apply flex flex-col gap-[3px];

    span {
      border: 4px solid transparent;

      &:first-child {
        border-top: 0;
        border-bottom-color: #c4c6cc;

        &.active {
          border-bottom-color: #3a84ff;
        }
      }

      &:last-child {
        border-top-color: #c4c6cc;
        border-bottom: 0;

        &.active {
          border-top-color: #3a84ff;
        }
      }
    }
  }

  .hide-tree-action :deep(.is-root.bk-tree-node .bk-node-action) {
    min-width: 7px !important;
    margin-right: 0 !important;
  }
</style>
