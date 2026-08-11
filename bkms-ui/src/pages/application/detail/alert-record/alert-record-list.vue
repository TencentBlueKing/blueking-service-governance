<template>
  <div
    ref="containerRef"
    class="rounded-[2px] flex flex-col gap-[16px]"
  >
    <!-- 顶部搜索栏 -->
    <div
      ref="searchBarRef"
      class="flex items-center gap-[12px]"
    >
      <EnvSelectPanel
        v-model="selectedEnv"
        class="shrink-0"
        init-first-env-when-empty
      />
      <SearchSelect
        v-model="searchValue"
        class="flex-1 bg-[#fff] relative z-[100]"
        :data="searchData"
        :placeholder="placeholder"
        unique-select
        value-behavior="need-key"
      />
      <DatePicker
        v-model="dateRange"
        class="shrink-0 bg-[#fff]"
        format="YYYY-MM-DD HH:mm:ss"
        :need-timezone="false"
        :version="2"
        @update:model-value="handleDateChange"
      />
    </div>

    <!-- 表格：max-height 限制在父容器剩余空间内，筛选栏高度 + 表格高度不超过父元素 -->
    <div class="flex-1">
      <Table
        v-bkloading="{ loading: isLoading, zIndex: 6 }"
        :data="tableData"
        :expand-config="expandConfig"
        :max-height="tableMaxHeight"
        :pagination="pagination"
        :row-class-name="getAlertRowClass"
        :row-config="{ isHover: true }"
        :sort-config="sortConfig"
        @filter-change="handleFilterChange"
        @page-limit-change="pageSizeChange"
        @sort-change="handleSortChange"
        @page-value-change="pageChange"
        @toggle-row-expand="handleExpandChange"
      >
        <template #empty>
          <TableException
            :type="tableData.length === 0 && hasFilter ? 'search' : 'empty'"
            @clear="handleClearFilters"
          />
        </template>
        <!-- 展开行（type="expand" 列不能加 fixed，否则展开内容会被左侧固定列遮挡） -->
        <TableColumn
          type="expand"
          width="30"
        >
          <template #content="{ row }">
            <AlertRecordExpand :row="row" />
          </template>
        </TableColumn>
        <!-- 开始时间 -->
        <TableColumn
          field="beginTime"
          :label="$t('开始时间')"
          sortable
          :width="180"
        >
          <template #default="{ row }">
            {{ row.beginTime ? formatDateString(row.beginTime * 1000) : '--' }}
          </template>
        </TableColumn>
        <!-- 告警级别 -->
        <TableColumn
          field="severity"
          :label="$t('告警级别')"
          sortable
          :width="120"
        >
          <template #default="{ row }">
            <SeverityLabel :severity="row.severity" />
          </template>
        </TableColumn>
        <!-- 告警名称 -->
        <TableColumn
          field="alertDisplayName"
          :label="$t('告警名称')"
          :min-width="200"
          show-overflow="tooltip"
        >
          <template #default="{ row }">
            {{ row.alertDisplayName || '--' }}
          </template>
        </TableColumn>
        <!-- 持续时间 -->
        <TableColumn
          field="duration"
          :label="$t('持续时间')"
          sortable
          :width="120"
        >
          <template #default="{ row }">
            {{ row.duration || '--' }}
          </template>
        </TableColumn>
        <!-- 告警内容 -->
        <TableColumn
          field="description"
          :label="$t('告警内容')"
          :min-width="200"
          show-overflow="tooltip"
        >
          <template #default="{ row }">
            {{ row.description || '--' }}
          </template>
        </TableColumn>
        <!-- 状态 -->
        <TableColumn
          field="status"
          :filter-multiple="true"
          :filters="statusFilterOptions"
          :label="$t('状态')"
          :width="120"
        >
          <template #default="{ row }">
            <StatusIcon
              :status="normalizeStatus(row.status)"
              :status-color-map="statusColorMap"
              :status-text-map="statusTextMap"
            />
          </template>
        </TableColumn>
        <!-- 操作 -->
        <TableColumn
          fixed="right"
          :label="$t('操作')"
          :width="80"
        >
          <template #default="{ row }">
            <Button
              text
              theme="primary"
              @click="handleToAlertDetail(row)"
            >
              {{ $t('详情') }}
            </Button>
          </template>
        </TableColumn>
      </Table>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import type { Component } from 'vue';
  import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue';

  import DatePicker from '@blueking/date-picker';
  import { Table, TableColumn } from '@blueking/table';
  import { Button, Message, SearchSelect } from 'bkui-vue';
  import dayjs from 'dayjs';
  import { useI18n } from 'vue-i18n';
  import { BkintegrationsBkmonitorService } from '~/api/modules/v1/bkintegrations-bkmonitor';
  import EnvSelectPanel from '~/components/env-select-panel.vue';
  import StatusIcon from '~/components/status-icon.vue';
  import TableException from '~/components/table-exception.vue';
  import { useElementHeight } from '~/composables/use-element-height';
  import usePageConf from '~/composables/use-page';
  import useSearchFilter from '~/composables/use-search-filter';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTime from '~/composables/use-time';
  import useToLink from '~/composables/use-to-link';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';

  import SeverityLabel from './components/severity-label.vue';

  import type { DateValue } from '@blueking/date-picker';
  import type { VxeTableDefines } from '@blueking/vxe-table';
  import type { ISearchValue } from 'bkui-vue/lib/search-select/utils';
  import type { Dayjs } from 'dayjs';
  import type { AlertEventOutput, ListAlertEventsRequest } from '~/@types/v1/bkintegrations-bkmonitor';
  import type { ISelectKey } from '~/composables/use-search';

  import '@blueking/date-picker/vue3/vue3.css';

  const AlertRecordExpand: Component = defineAsyncComponent(() => import('./components/alert-record-expand.vue'));

  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const { formatDateString } = useTime();

  const appDetailStore = useAppDetail();
  const spaceStore = useSpaceStore();
  const { handleToLink } = useToLink();

  /** 状态色 + 文案映射 */
  const statusColorMap = {
    RECOVERED: 'green',
    ABNORMAL: 'red',
    CLOSED: 'gray',
  };
  const statusTextMap = {
    RECOVERED: t('已恢复'),
    ABNORMAL: t('未恢复'),
    CLOSED: t('已失效'),
  };

  /** 表格展开配置 */
  const expandConfig = {
    showIcon: true,
    trigger: 'row',
    iconOpen: 'bkms-icon bkms-icon-down-shape !text-[#C4C6CC] hover:!text-[#63656E] p-[2px]',
    iconClose: 'bkms-icon bkms-icon-right-shape !text-[#C4C6CC] hover:!text-[#63656E] p-[2px]',
  };

  /** 服务端排序：remote 模式下点击表头仅触发 @sort-change，排序交给后端 ordering 处理 */
  const sortConfig = ref({
    multiple: false,
    remote: true,
    trigger: 'cell',
  });

  /** VxeTable 排序方向：'' 表示未排序（与 remote 配合使用） */
  type SortOrder = '' | 'asc' | 'desc';

  /** 当前排序字段（表格列 field）与方向 */
  const sortBy = ref('');
  const sortOrder = ref<SortOrder>('');

  /** 表格列 field → 蓝鲸监控 search_alert 排序字段名映射 */
  const sortFieldMap: Record<string, string> = {
    beginTime: 'begin_time',
    severity: 'severity',
    duration: 'duration',
  };

  /** 搜索栏配置 */
  const searchData = shallowRef<ISelectKey<AlertEventOutput>[]>([
    { id: 'alertDisplayName', name: t('告警名称'), field: 'alertDisplayName', fuzzy: true },
    // { id: 'alertID', name: t('告警 ID'), field: 'alertID', fuzzy: true },
    { id: 'description', name: t('告警内容'), field: 'description', fuzzy: true },
    {
      id: 'status',
      name: t('状态'),
      field: 'status',
      multiple: false,
      children: [
        { id: 'RECOVERED', name: t('已恢复') },
        { id: 'ABNORMAL', name: t('未恢复') },
        { id: 'CLOSED', name: t('已失效') },
      ],
    },
  ]);

  const placeholder = createPlaceholder({
    type: 'searchSelect',
    labels: [t('告警名称'), t('告警内容'), t('状态')],
  });

  const searchValue = ref<ISearchValue[]>([]);
  /** 表格状态列 filter 与 SearchSelect 联动（field 需与 searchData 的 id 一致） */
  const { filterOptions, handleFilterChange } = useSearchFilter(searchData, searchValue, ['status'] as const);

  /** 状态列筛选配置（由 filterOptions 派生，随 searchValue 同步 checked） */
  const statusFilterOptions = computed(() => filterOptions.value.status);

  /** 时间选择器展示值：默认选中"今天（至今）" */
  const dateRange = ref<DateValue | undefined>(['now/d', 'now']);
  /** 查询时间范围（Unix 秒级时间戳），由 DatePicker 事件同步；默认"今天（至今）"，清空为 undefined */
  const timeRange = ref<[number, number] | undefined>([dayjs().startOf('day').unix(), dayjs().unix()]);
  const selectedEnv = ref<string>('');
  /** 当前展开的行 key 集合（用于行背景色标识） */
  const expandedRowKeys = ref<Set<string>>(new Set());
  const tableData = ref<AlertEventOutput[]>([]);
  const count = ref(0);
  const isLoading = ref(false);

  const containerRef = ref<HTMLElement>();
  const searchBarRef = ref<HTMLElement>();
  /** 父容器（index.vue 的 flex-1 容器）内容区高度：边框盒高度 - 垂直内边距 */
  const parentContentHeight = ref(400);
  const { height: searchBarHeight } = useElementHeight(searchBarRef, {
    watchSource: isLoading,
    defaultHeight: 40,
  });

  /** 测量父容器内容区高度：getBoundingClientRect 为边框盒（含 padding），需扣除父元素垂直内边距 */
  function updateParentHeight() {
    const parent = containerRef.value?.parentElement;
    if (!parent) return;
    const rect = parent.getBoundingClientRect();
    const parentStyle = getComputedStyle(parent);
    const padTop = Number.parseFloat(parentStyle.paddingTop) || 0;
    const padBottom = Number.parseFloat(parentStyle.paddingBottom) || 0;
    parentContentHeight.value = rect.height - padTop - padBottom;
  }

  onMounted(() => {
    updateParentHeight();
    window.addEventListener('resize', updateParentHeight);
  });

  onBeforeUnmount(() => {
    window.removeEventListener('resize', updateParentHeight);
  });

  /** 是否有任意筛选条件 */
  const hasFilter = computed(() => searchValue.value.length > 0 || (dateRange.value?.length ?? 0) > 0);

  /** 表格最大高度：父容器内容区高度 - 根容器内边距(24×2) - 筛选栏高度 - 间距(16)，确保不超出父元素 */
  const tableMaxHeight = computed(() => Math.max(parentContentHeight.value - searchBarHeight.value - 24 * 2 - 16, 0));

  const { pagination, pageChange, pageSizeChange, handleResetPage } = usePageConf(
    tableData,
    {
      current: 1,
      limit: 10,
      remote: true,
      onPageChange: () => fetchList(),
      onPageSizeChange: () => fetchList(),
    },
    count,
  );

  /** 当前空间绑定的蓝鲸监控业务 ID（可能为空：未配置蓝鲸监控的空间） */
  const bkMonitorProjectID = computed(() => spaceStore.workspaceDetail?.bkSystems?.bkMonitorProjectID ?? '');

  /** 跳转到蓝鲸监控告警详情页：`${BK_MONITOR}/?bizId=${bkMonitorProjectID}/#/trace/alarm-center/detail/${alertID}`
   *  字段来源：
   *   - bkMonitorProjectID 来自 workspace.bkSystems（不在 alert 事件行上）
   *   - alertID 是告警事件主键
   */
  function handleToAlertDetail(row: AlertEventOutput) {
    if (!bkMonitorProjectID.value) {
      Message({ theme: 'warning', message: t('当前空间未配置蓝鲸监控业务，无法跳转') });
      return;
    }
    const alertID = (row as unknown as Record<string, string>).alertID;
    if (!alertID) {
      Message({ theme: 'warning', message: t('告警详情链接信息不完整，无法跳转') });
      return;
    }
    handleToLink('monitor-alert', bkMonitorProjectID.value, alertID);
  }

  /** 拉取告警事件列表 */
  async function fetchList() {
    if (!appDetailStore.appID || !spaceStore.currentSpace || !selectedEnv.value) return;
    // 数据刷新（切页/搜索/环境/时间变化）后旧展开状态失效，清空展开行集合
    expandedRowKeys.value = new Set();
    isLoading.value = true;
    try {
      const [startTime, endTime] = timeRange.value ?? [];
      const hasValidDateRange = Boolean(startTime && endTime);
      const statusValues = (searchValue.value.find(it => it.id === 'status')?.values || []).map(v => v.id);

      const params: ListAlertEventsRequest = {
        workspaceID: spaceStore.currentSpace,
        appID: appDetailStore.appID,
        page: pagination.value.current,
        pageSize: pagination.value.limit,
        ...(selectedEnv.value && { envName: selectedEnv.value }),
        ...(pickSearchValue('alertDisplayName') && { alertDisplayName: pickSearchValue('alertDisplayName') }),
        ...(pickSearchValue('alertID') && { alertID: pickSearchValue('alertID') }),
        ...(pickSearchValue('description') && { description: pickSearchValue('description') }),
        ...(statusValues.length > 0 && { status: statusValues }),
        ...(hasValidDateRange && { startTime, endTime }),
        ...(sortBy.value && sortOrder.value
          ? { ordering: [`${sortOrder.value === 'desc' ? '-' : ''}${sortFieldMap[sortBy.value] ?? sortBy.value}`] }
          : {}),
      };

      const res = await BkintegrationsBkmonitorService.listAlertEvents(params);
      tableData.value = res?.results ?? [];
      count.value = Number(res?.count ?? 0);
    } catch {
      tableData.value = [];
      count.value = 0;
    } finally {
      isLoading.value = false;
    }
  }

  /** 为行附加展开背景色类名（用于识别当前行是否处于展开态） */
  function getAlertRowClass({ row }: { row: AlertEventOutput }): string {
    const key = row.eventID || '';
    if (key && expandedRowKeys.value.has(key)) return 'alert-expanded-row';
    return '';
  }

  /** 清空所有筛选 */
  function handleClearFilters() {
    searchValue.value = [];
    dateRange.value = undefined;
    timeRange.value = undefined;
    handleResetPage();
    fetchList();
  }

  /** 处理日期变化：从 DatePicker 事件中提取秒级时间戳并刷新 */
  function handleDateChange(_val: DateValue | undefined, info: { dayjs: Dayjs | null; formatText: null | string }[]) {
    const start = info?.[0]?.dayjs;
    const end = info?.[1]?.dayjs;
    timeRange.value = start && end ? [start.unix(), end.unix()] : undefined;
    handleResetPage();
    fetchList();
  }

  /** 处理表头排序：更新排序状态并重新拉取列表（服务端排序，携带 ordering） */
  function handleSortChange({ field, order }: { field: string; order: null | SortOrder }) {
    sortBy.value = order ? field : '';
    sortOrder.value = order || '';
    handleResetPage();
    fetchList();
  }

  /** 处理行展开/折叠，维护展开行 key 集合 */
  function handleExpandChange({ row, expanded }: VxeTableDefines.ToggleRowExpandEventParams<AlertEventOutput>) {
    const key = row.eventID || '';
    if (!key) return;
    const next = new Set(expandedRowKeys.value);
    if (expanded) {
      next.add(key);
    } else {
      next.delete(key);
    }
    expandedRowKeys.value = next;
  }

  /** 标准化 status 为 status-icon 颜色 key（兼容后端 RECOVERED / ABNORMAL / CLOSED 三态） */
  function normalizeStatus(status?: string): 'ABNORMAL' | 'CLOSED' | 'RECOVERED' {
    const s = (status || '').toLowerCase();
    if (s === 'recovered') return 'RECOVERED';
    if (s === 'abnormal') return 'ABNORMAL';
    if (s === 'closed') return 'CLOSED';
    return 'ABNORMAL';
  }

  /** 从 searchValue 提取字段值 */
  function pickSearchValue(id: string): string {
    const item = searchValue.value.find(it => it.id === id);
    return item?.values?.[0]?.id ?? '';
  }

  // 监听搜索值变化，触发刷新
  watch(
    searchValue,
    () => {
      if (pagination.value.current === 1) {
        fetchList();
      } else {
        pageChange(1);
      }
    },
    { deep: true },
  );

  // 监听环境变化
  watch(selectedEnv, () => {
    handleResetPage();
    fetchList();
  });

  // 监听 appID 变化，首次加载
  watch(
    () => appDetailStore.appID,
    async appID => {
      if (appID) {
        handleResetPage();
        await fetchList();
      } else {
        tableData.value = [];
        count.value = 0;
      }
    },
    { immediate: true },
  );
</script>

<style lang="postcss" scoped>
  /* 展开行的数据行背景色 */
  :deep(.vxe-body--row.alert-expanded-row) {
    background-color: #f5f7fa;
  }
</style>
