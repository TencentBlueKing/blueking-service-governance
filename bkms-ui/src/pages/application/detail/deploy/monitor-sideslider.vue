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
      class="monitor-toolbar border-b border-[#DCDEE5] bg-[#fff] px-[24px] py-[8px] shadow-[0_3px_5.5px_0_#00000012]"
    >
      <div class="flex items-center gap-[16px] min-w-0">
        <!-- 实例名称 -->
        <div class="flex items-center flex-1 min-w-0">
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
            <div
              class="flex items-center px-[12px] py-[6px] text-[#63656E] cursor-pointer hover:bg-[#f5f7fa]"
              @click.stop="handleToggleAll"
            >
              <Checkbox
                class="pointer-events-none"
                :disabled="!visibleInstances.length"
                :indeterminate="isPartiallySelected"
                :model-value="isAllSelected"
                size="small"
              />
              <span class="ml-[6px]">{{ $t('全选') }}</span>
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
        <div
          v-bk-tooltips="{ content: $t('选择实例后展示全部已选实例'), disabled: !selectedInstance.length }"
          class="flex items-center flex-shrink-0"
        >
          <span class="bg-[#FAFBFD] h-[32px] leading-[30px] text-[#4D4F56] text-[12px] px-[6px] b b-r-0 b-[#c4c6cc]"
            >Limit</span
          >
          <Select
            v-model="limitType"
            class="w-[96px]"
            :clearable="false"
            :disabled="!!selectedInstance.length"
          >
            <Select.Option
              id="top"
              name="top"
            />
            <Select.Option
              id="bottom"
              name="bottom"
            />
          </Select>
          <Input
            v-model="limit"
            class="w-[88px] ml-[-1px]"
            :disabled="!!selectedInstance.length"
            :max="50"
            :min="1"
            :precision="0"
            type="number"
            @blur="normalizeLimit"
          />
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
    <div class="overflow-auto px-[24px] pt-[24px] pb-[24px] bg-[#F5F7FA] h-[calc(100vh-140px)]">
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
            :all="chartParams.all"
            :area="item.area !== false"
            :compare-offsets="chartParams.compareOffsets"
            :env-name="chartParams.envName"
            :env-names="chartParams.envNames"
            :instances="chartParams.instances"
            :interval="chartParams.interval"
            :limit="chartParams.limit"
            :limit-type="chartParams.limitType"
            :method="chartParams.method"
            :metric-key="item.metricKey"
            :query-enabled="queryReady"
            :query-id="chartParams.queryId"
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

  import DatePicker from '@blueking/date-picker';
  import { Table, TableColumn } from '@blueking/table';
  import {
    Checkbox,
    Divider,
    Input,
    Message,
    Progress,
    Radio,
    Select,
    Sideslider,
    Switcher,
    Tag,
    Tree,
  } from 'bkui-vue';
  import dayjs from 'dayjs';
  import { debounce } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import IntervalSelector from '~/components/interval-selector.vue';
  import useInterval from '~/composables/use-interval';
  import {
    fetchMonitorMetric,
    getMonitorInstanceKey,
    getMonitorInstanceLabel,
    isPercentUnit,
    setChartTimeRange,
  } from '~/pages/application/detail/deploy/monitor-chart-bridge';
  import { useAppDetail } from '~/stores/app-detail';
  import { useDeployEnvStore } from '~/stores/deploy-env';

  import type { MonitorInstanceGroup } from './instance-list/types';
  import type { MonitorInstance, MonitorMetricQuery, MonitorMetricTimeSeries } from './monitor-chart-bridge';
  import type { DateValue } from '@blueking/date-picker';
  import type { Dayjs } from 'dayjs';
  import type { TimeSeriesItem } from '~/@types/v1/bkintegrations-bkmonitor';

  import '@blueking/date-picker/vue3/vue3.css';

  const ChartCardItem = defineAsyncComponent(() => import('~/pages/application/detail/deploy/chart-card-item.vue'));

  type ChartParams = Omit<MonitorMetricQuery, 'appID' | 'metricKey'> & {
    all: boolean;
    compareOffsets: number[];
    instances: MonitorInstance[];
    queryId: number;
  };
  const props = defineProps<{
    envName?: string;
    envNames?: string[];
    initialSelection?: Record<string, string[]>;
    instanceGroups?: MonitorInstanceGroup[];
  }>();
  interface MonitorInstanceRow {
    cpuLimitUsageRate: number;
    cpuUsage: number;
    memLimitUsageRate: number;
    memUsage: number;
    name: string;
  }

  const { t } = useI18n();

  const isShow = defineModel<boolean>('isShow');

  const appDetailStore = useAppDetail();
  const envStore = useDeployEnvStore();
  const sliderWidth = ref<number | string>(1200);
  const queryReady = ref(false);

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
    summaryMetricData.value = {};
    chartParams.value.queryId += 1;
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
  const summaryMetricData = ref<Partial<Record<string, MonitorMetricTimeSeries>>>({});

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
  const committedInstances = ref<string[]>([]);
  const searchText = ref('');
  const treeRef = ref<InstanceType<typeof Tree>>();
  const selectRef = ref<InstanceType<typeof Select>>();
  const limit = ref<number | string>(10);
  const limitType = ref<'bottom' | 'top'>('top');
  const methodOptions = ['sum', 'max', 'min', 'count', 'avg'] as const;
  const method = ref<(typeof methodOptions)[number]>('sum');
  interface TreeNodeData {
    children?: TreeNodeData[];
    id: string;
    name: string;
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
  const isAllSelected = computed(
    () => visibleInstances.value.length > 0 && visibleInstances.value.every(id => selectedInstance.value.includes(id)),
  );
  const isPartiallySelected = computed(
    () => !isAllSelected.value && visibleInstances.value.some(id => selectedInstance.value.includes(id)),
  );
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
  function handleToggleAll() {
    if (!visibleInstances.value.length) return;
    const visible = new Set(visibleInstances.value);
    updateSelection(
      isAllSelected.value
        ? selectedInstance.value.filter(id => !visible.has(id))
        : [...selectedInstance.value, ...visibleInstances.value],
    );
  }
  function normalizeLimit() {
    const parsed = Number(limit.value);
    limit.value = Number.isFinite(parsed) ? Math.min(50, Math.max(1, Math.floor(parsed))) : 10;
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
    instances: [],
    all: true,
    limit: 10,
    limitType: 'top',
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
    compareOffsetSecondsList.value.map(v => (typeof v === 'string' ? parseOffset(v) : v)),
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
  function handleFiltersChange() {
    if (!isShow.value || !chartPlotWidth.value) return;
    const all = !selectedInstance.value.length;
    // 提交时固定查询快照，列表的 SSE 更新只改变搜索候选，不触发图表重新请求。
    chartParams.value = {
      endTime: endTime.value,
      startTime: startTime.value,
      envName: queryEnvNames.value[0],
      envNames: [...queryEnvNames.value],
      instances: selectedInstance.value.flatMap(id => {
        const instance = instancesByKey.value.get(id);
        return instance ? [{ ...instance }] : [];
      }),
      all,
      ...(all
        ? { limit: Math.min(50, Math.max(1, Math.floor(Number(limit.value) || 10))), limitType: limitType.value }
        : {}),
      method: method.value,
      interval: effectiveInterval.value,
      compareOffsets: timeCompareEnabled.value && !isMultiEnv.value ? effectiveCompareOffsets.value : [],
      queryId: chartParams.value.queryId + 1,
    };
    queryReady.value = true;
    committedInstances.value = [...selectedInstance.value];
    metricData.value = {};
    summaryMetricData.value = {};
    setChartTimeRange(startTime.value, endTime.value);
    timeRangeRef.value = [
      dayjs.unix(startTime.value).format('YYYY-MM-DD HH:mm:ss'),
      dayjs.unix(endTime.value).format('YYYY-MM-DD HH:mm:ss'),
    ];
    refreshImmediateRef.value += 1;
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
      () => limit.value,
      () => limitType.value,
      () => method.value,
      () => JSON.stringify(queryEnvNames.value),
      () => JSON.stringify(timeCompareEnabled.value ? effectiveCompareOffsets.value : []),
    ],
    handleFiltersChange,
  );
  const onManualRefresh = debounce(handleFiltersChange, 300);
  const { start: startPolling, stop: stopPolling } = useInterval(async () => {
    handleFiltersChange();
  }, autoRefreshInterval);
  onBeforeUnmount(() => {
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

  let summaryRequestedQueryId = -1;
  async function onMetricData(payload: {
    data: MonitorMetricTimeSeries | undefined;
    metricKey: string;
    queryId: number;
  }) {
    if (!isShow.value || payload.queryId !== chartParams.value.queryId) return;
    metricData.value = { ...metricData.value, [payload.metricKey]: payload.data };
    if (
      !chartParams.value.all ||
      payload.metricKey !== 'cpu_limit_usage' ||
      summaryRequestedQueryId === payload.queryId ||
      !payload.data?.series?.length
    )
      return;
    summaryRequestedQueryId = payload.queryId;
    const params = chartParams.value;
    const cpuSeries = payload.data.series;
    // 其余三列沿用 CPU 当前期的实例及环境，避免独立排名后各列对应不同实例。
    const instances = cpuSeries.flatMap(series =>
      series.instance ? [{ envName: series.envName || params.envName || '', instance: series.instance }] : [],
    );
    if (!instances.length) return;
    const results = await Promise.all(
      ['memory_limit_usage', 'cpu_usage', 'memory_usage'].map(async metricKey => {
        const data = await fetchMonitorMetric({
          appID: appDetailStore.appID,
          envName: params.envName,
          envNames: params.envNames,
          instances,
          metricKey,
          method: params.method,
          interval: params.interval,
          startTime: params.startTime,
          endTime: params.endTime,
        });
        return [metricKey, data] as const;
      }),
    );
    if (isShow.value && chartParams.value.queryId === payload.queryId)
      summaryMetricData.value = Object.fromEntries(results);
  }
  watch(
    isShow,
    show => {
      if (!show) return;
      queryReady.value = false;
      chartPlotWidth.value = 0;
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
      limit.value = 10;
      limitType.value = 'top';
      method.value = 'sum';
      handleFiltersChange();
      void syncSelection();
    },
    { immediate: true },
  );

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

  const tableData = computed<MonitorInstanceRow[]>(() => {
    const metrics = chartParams.value.all
      ? { ...summaryMetricData.value, cpu_limit_usage: metricData.value.cpu_limit_usage }
      : metricData.value;
    const keys = ['cpu_limit_usage', 'memory_limit_usage', 'cpu_usage', 'memory_usage'] as const;
    const values = new Map<string, Map<string, TimeSeriesItem>>();
    for (const key of keys) {
      values.set(key, new Map((metrics[key]?.series || []).map(series => [getMonitorInstanceKey(series), series])));
    }
    const rows = chartParams.value.all
      ? (metricData.value.cpu_limit_usage?.series || []).map(series => ({
          instance: series.instance || '',
          envName: series.envName,
        }))
      : chartParams.value.instances;
    return rows.map(instance => {
      const latest = (key: (typeof keys)[number]) => {
        const item = values.get(key)?.get(getMonitorInstanceKey(instance));
        const value = item?.stat?.last?.[1] ?? item?.dataPoints?.at(-1)?.[0] ?? 0;
        return value * (isPercentUnit(metrics[key]?.unit) ? 100 : 1);
      };
      return {
        name: getMonitorInstanceLabel(instance, isMultiEnv.value),
        cpuLimitUsageRate: latest('cpu_limit_usage'),
        memLimitUsageRate: latest('memory_limit_usage'),
        cpuUsage: latest('cpu_usage'),
        memUsage: latest('memory_usage'),
      };
    });
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

  .monitor-toolbar-options {
    overflow: auto hidden;
  }

  .hide-tree-action :deep(.is-root.bk-tree-node .bk-node-action) {
    min-width: 7px !important;
    margin-right: 0 !important;
  }
</style>
