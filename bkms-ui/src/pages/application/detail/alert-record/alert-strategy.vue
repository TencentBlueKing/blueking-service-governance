<template>
  <div
    ref="containerRef"
    class="rounded-[2px] flex flex-col gap-[16px]"
  >
    <!-- 顶部工具栏 -->
    <div
      ref="searchBarRef"
      class="flex items-center justify-between gap-[350px]"
    >
      <Button
        theme="primary"
        @click="handleCreate"
      >
        <Plus
          class="mr-[4px]"
          :height="24"
          :width="24"
        />
        {{ $t('新建策略') }}
      </Button>
      <SearchSelect
        v-model="searchValue"
        class="flex-1 bg-[#fff]"
        :data="searchData"
        :placeholder="placeholder"
        unique-select
        value-behavior="need-key"
      />
    </div>

    <!-- 表格：max-height 限制在父容器剩余空间内，筛选栏高度 + 表格高度不超过父元素 -->
    <div class="flex-1">
      <Table
        v-bkloading="{ loading: isLoading, zIndex: 6 }"
        :data="filteredData"
        :max-height="tableMaxHeight"
        :pagination="pagination"
        :row-height="56"
        :row-config="{ isHover: true }"
        @filter-change="handleFilterChange"
        @page-limit-change="pageSizeChange"
        @page-value-change="pageChange"
      >
        <template #empty>
          <TableException
            :type="filteredData.length === 0 && hasFilter ? 'search' : 'empty'"
            @clear="handleClearFilters"
          />
        </template>
        <!-- 策略名称 -->
        <TableColumn
          field="displayName"
          :label="$t('策略名称')"
          :min-width="200"
          show-overflow="tooltip"
        >
          <template #default="{ row }">
            <span
              class="text-[#3a84ff] cursor-pointer"
              @click="handleEdit(row)"
              >{{ row.displayName || '--' }}</span
            >
          </template>
        </TableColumn>
        <!-- 监控指标 -->
        <TableColumn
          field="monitorMetric"
          filter-multiple
          :filters="filterOptions.monitorMetric"
          :label="$t('监控指标')"
          :min-width="160"
          show-overflow="tooltip"
        >
          <template #default="{ row }">
            <span class="flex flex-col">
              <span>{{ monitorMetricNameMap.get(row.monitorMetric || '') || row.monitorMetric || '--' }}</span>
              <span class="text-[#979BA5] text-[12px] font-mono">{{ row.monitorMetric }}</span>
            </span>
          </template>
        </TableColumn>
        <!-- 告警条件 -->
        <TableColumn
          field="threshold"
          :label="$t('告警条件')"
          :min-width="160"
        >
          <template #default="{ row }">
            {{ formatThreshold(row.threshold, row.monitorMetric) }}
          </template>
        </TableColumn>
        <!-- 告警级别 -->
        <TableColumn
          field="severity"
          filter-multiple
          :filters="filterOptions.severity"
          :label="$t('告警级别')"
          :width="120"
        >
          <template #default="{ row }">
            <SeverityLabel :severity="row.severity" />
          </template>
        </TableColumn>
        <!-- 生效环境 -->
        <TableColumn
          field="effectiveScope.type"
          filter-multiple
          :filters="filterOptions['effectiveScope.type']"
          :label="$t('生效环境')"
          :min-width="200"
        >
          <template #default="{ row }">
            <span v-if="row.effectiveScope?.type === 'all'">{{ $t('所有环境') }}</span>
            <span v-else-if="row.effectiveScope?.type === 'env_type'">{{ $t('按环境类型') }}</span>
            <span v-else-if="row.effectiveScope?.type === 'specific_envs'">{{ $t('部分环境') }}</span>
            <span v-else>--</span>
          </template>
        </TableColumn>
        <!-- 告警组 -->
        <TableColumn
          field="noticeGroupIDs"
          filter-multiple
          :filters="filterOptions.noticeGroupIDs"
          :label="$t('告警组')"
          :min-width="220"
        >
          <template #default="{ row }">
            <MoreTag
              v-if="row.noticeGroupIDs?.length"
              :data="row.noticeGroupIDs"
              overflow-mode="popover"
            >
              <template #default="{ item }">
                <span
                  v-bk-tooltips="{
                    content: getUserGroupUsers(item)
                      .map(u => u.display_name || u.id || '--')
                      .join('，'),
                    delay: 300,
                    disabled: !getUserGroupUsers(item).length,
                  }"
                  class="flex items-center gap-[4px] rounded-[2px] px-[8px] bg-[#F0F1F5] cursor-default"
                >
                  <i class="bkms-icon bkms-icon-usergroup text-[#979BA5] text-[14px]"></i>
                  {{ userGroupMap.get(String(item))?.name || item }}
                </span>
              </template>
            </MoreTag>
            <span v-else>--</span>
          </template>
        </TableColumn>
        <!-- 启/停用 -->
        <TableColumn
          field="enabled"
          fixed="right"
          :label="$t('启/停用')"
          :width="100"
        >
          <template #default="{ row }">
            <PopConfirm
              v-if="row.enabled"
              :confirm-text="$t('停用')"
              :content="$t('确定停用该策略？')"
              placement="top"
              trigger="click"
              width="280"
              @confirm="() => handleSwitch(row)"
            >
              <Switcher
                theme="primary"
                :value="row.enabled"
              />
            </PopConfirm>
            <Switcher
              v-else
              :before-change="() => handleSwitch(row)"
              theme="primary"
              :value="row.enabled"
            />
          </template>
        </TableColumn>
        <!-- 操作 -->
        <TableColumn
          fixed="right"
          :label="$t('操作')"
          :width="140"
        >
          <template #default="{ row }">
            <div class="flex items-center gap-[12px]">
              <Button
                text
                theme="primary"
                @click="handleEdit(row)"
                >{{ $t('编辑') }}</Button
              >
              <Button
                text
                theme="primary"
                @click="handleDelete(row)"
                >{{ $t('删除') }}</Button
              >
            </div>
          </template>
        </TableColumn>
      </Table>
    </div>

    <!-- 新建/编辑/查看策略侧滑 -->
    <StrategyForm
      v-model:is-show="formVisible"
      :data="editingRow"
      :mode="formMode"
      :user-groups="userGroups"
      @success="onFormSuccess"
    />
  </div>
</template>

<script lang="ts" setup>
  import { computed, h, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, InfoBox, Message, PopConfirm, SearchSelect, Switcher } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRoute } from 'vue-router';
  import { BkintegrationsBkmonitorService } from '~/api/modules/v1/bkintegrations-bkmonitor';
  import { COUNT_UNIT_METRICS } from '~/common/const';
  import MoreTag from '~/components/more-tag.vue';
  import TableException from '~/components/table-exception.vue';
  import { useElementHeight } from '~/composables/use-element-height';
  import usePageConf from '~/composables/use-page';
  import useSearchFilter from '~/composables/use-search-filter';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';

  import SeverityLabel from './components/severity-label.vue';
  import StrategyForm, { strategyCodeOptions } from './components/strategy-form.vue';

  import type { ISearchValue } from 'bkui-vue/lib/search-select/utils';
  import type { AlertStrategyOutput, ThresholdConfigInput, UserGroup } from '~/@types/v1/bkintegrations-bkmonitor';
  import type { ISelectKey } from '~/composables/use-search';

  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();

  const route = useRoute();
  const appDetailStore = useAppDetail();
  const spaceStore = useSpaceStore();

  /** 策略表单侧滑显隐 */
  const formVisible = ref(false);
  /** 当前编辑的策略行（新建态为 undefined） */
  const editingRow = ref<AlertStrategyOutput>();
  /** 策略表单模式：create 新建 / edit 编辑 */
  const formMode = ref<'create' | 'edit'>('create');

  /** 阈值比较方法符号映射 */
  const methodTextMap: Record<string, string> = {
    gte: '≥',
    gt: '>',
    lte: '≤',
    lt: '<',
    eq: '=',
    neq: '≠',
  };

  /**
   * 监控指标 Prometheus 名称 → 中文展示名映射
   * 从 strategy-form.vue 的 strategyCodeOptions 中提取（同一 monitorMetric 可能对应多个 strategyCode，
   * 去重时首个 name 优先）。
   */
  const monitorMetricNameMap = computed(() => {
    const map = new Map<string, string>();
    strategyCodeOptions.forEach(item => {
      if (item.monitorMetric && !map.has(item.monitorMetric)) {
        map.set(item.monitorMetric, item.name);
      }
    });
    return map;
  });

  /** 搜索栏配置 */
  const searchData = shallowRef<ISelectKey<AlertStrategyOutput>[]>([
    { id: 'displayName', name: t('策略名称'), field: 'displayName', fuzzy: true },
    { id: 'monitorMetric', name: t('监控指标'), field: 'monitorMetric', multiple: true, children: [] },
    {
      id: 'severity',
      name: t('告警级别'),
      field: 'severity',
      multiple: true,
      children: [
        { id: '1', name: t('致命') },
        { id: '2', name: t('预警') },
        { id: '3', name: t('提醒') },
      ],
    },
    {
      id: 'effectiveScope.type',
      name: t('生效环境'),
      field: 'effectiveScope.type',
      multiple: true,
      children: [
        { id: 'all', name: t('所有环境') },
        { id: 'env_type', name: t('按环境类型') },
        { id: 'specific_envs', name: t('部分环境') },
      ],
    },
    { id: 'noticeGroupIDs', name: t('告警组'), field: 'noticeGroupIDs', multiple: true, children: [] },
  ]);

  const placeholder = createPlaceholder({
    type: 'searchSelect',
    labels: [t('策略名称'), t('监控指标'), t('告警级别'), t('告警条件'), t('生效环境'), t('告警组')],
  });

  const searchValue = ref<ISearchValue[]>([]);
  /** 表格筛选与 SearchSelect 联动（TableColumn field 需与 searchData 的 id 一致） */
  const { filterOptions, handleFilterChange } = useSearchFilter(searchData, searchValue, [
    'monitorMetric',
    'severity',
    'effectiveScope.type',
    'noticeGroupIDs',
  ] as const);

  /** 告警组列表（用于列展示与筛选项动态生成） */
  const userGroups = ref<UserGroup[]>([]);
  /** 告警组 ID -> UserGroup 映射 */
  const userGroupMap = computed(() => {
    const map = new Map<string, UserGroup>();
    userGroups.value.forEach(g => {
      if (g.id != null) map.set(String(g.id), g);
    });
    return map;
  });

  /** 获取告警组的通知接收人员列表（用于 Popover tooltips） */
  function getUserGroupUsers(item: unknown) {
    return userGroupMap.value.get(String(item))?.users ?? [];
  }

  const tableData = ref<AlertStrategyOutput[]>([]);
  const isLoading = ref(false);
  /** 当前正在切换启停状态的策略 ID（用于 Switcher loading） */
  const switchingId = ref<string>('');

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
  const hasFilter = computed(() => searchValue.value.length > 0);

  /** 表格最大高度：父容器内容区高度 - 根容器内边距(24×2) - 筛选栏高度 - 间距(16)，确保不超出父元素 */
  const tableMaxHeight = computed(() => Math.max(parentContentHeight.value - searchBarHeight.value - 24 * 2 - 16, 0));

  /** 前端筛选后的数据（后端 listAlertStrategies 暂不支持搜索参数，采用前端过滤） */
  const filteredData = computed(() => {
    if (searchValue.value.length === 0) return tableData.value;

    return tableData.value.filter(row => {
      return searchValue.value.every(item => {
        const values = (item.values || []).map(v => v.id);
        if (values.length === 0) return true;

        if (item.id === 'displayName') {
          const keyword = values[0].toLowerCase();
          return String(row.displayName || '')
            .toLowerCase()
            .includes(keyword);
        }
        if (item.id === 'monitorMetric') {
          return values.includes(String(row.monitorMetric || ''));
        }
        if (item.id === 'severity') {
          return values.includes(String(row.severity ?? ''));
        }
        if (item.id === 'effectiveScope.type') {
          return values.includes(String(row.effectiveScope?.type || ''));
        }
        if (item.id === 'noticeGroupIDs') {
          return (row.noticeGroupIDs || []).some(id => values.includes(String(id)));
        }
        return true;
      });
    });
  });

  const filteredCount = computed(() => filteredData.value.length);

  const { pagination, pageChange, pageSizeChange, handleResetPage } = usePageConf(
    filteredData,
    {
      current: 1,
      limit: 10,
      remote: false,
    },
    filteredCount,
  );

  /** 拉取告警策略列表 */
  async function fetchList() {
    if (!appDetailStore.appID || !spaceStore.currentSpace) return;
    isLoading.value = true;
    try {
      const res = await BkintegrationsBkmonitorService.listAlertStrategies({
        workspaceID: spaceStore.currentSpace,
        appID: appDetailStore.appID,
      });
      tableData.value = res?.results ?? [];
      // 根据返回数据动态更新监控指标筛选项
      updateMonitorMetricChildren();
    } catch {
      tableData.value = [];
    } finally {
      isLoading.value = false;
    }
  }

  /** 拉取告警组列表（用于列展示与筛选项） */
  async function fetchUserGroups() {
    if (!spaceStore.currentSpace) return;
    try {
      const res = await BkintegrationsBkmonitorService.listUserGroups({
        workspaceID: spaceStore.currentSpace,
      });
      userGroups.value = res?.results ?? [];
      updateUserGroupChildren();
      // 必须在 children 填充完成后调用，否则 SearchSelect value-behavior=need-key 无法识别预填值
      applyGroupIdFromQuery();
    } catch {
      userGroups.value = [];
    }
  }

  /** 是否已应用过 URL 中的 groupID 预筛选（避免重复 push） */
  const groupIDApplied = ref(false);
  /**
   * 从 URL query.groupID 读取并预填告警组筛选条件。
   * 由「告警组 Tab 关联策略数」链接跳转触发（router.push 时携带 groupID）。
   */
  function applyGroupIdFromQuery() {
    if (groupIDApplied.value) return;
    const groupID = route.query.groupID;
    if (groupID == null || groupID === '') return;
    const idStr = String(groupID);
    const group = userGroups.value.find(g => String(g.id) === idStr);
    // 仅追加，未曾存在过 noticeGroupIDs 筛选时才设置（防止覆盖用户已有的筛选）
    const already = searchValue.value.some(item => item.id === 'noticeGroupIDs');
    if (!already) {
      searchValue.value = [
        ...searchValue.value,
        {
          id: 'noticeGroupIDs',
          name: t('告警组'),
          values: [{ id: idStr, name: group?.name || idStr }],
        },
      ];
    }
    groupIDApplied.value = true;
  }

  /** 格式化阈值条件为可读文案，如 "≥ 80%"、"≥ 5次" */
  function formatThreshold(threshold?: ThresholdConfigInput, monitorMetric?: string): string {
    if (!threshold) return '--';
    const method = methodTextMap[threshold.method] || threshold.method;
    const value = threshold.value ?? '--';
    const unit = monitorMetric && COUNT_UNIT_METRICS.has(monitorMetric) ? t('次') : '%';
    return `${method} ${value}${unit}`;
  }

  /** 清空所有筛选 */
  function handleClearFilters() {
    searchValue.value = [];
    handleResetPage();
  }

  /** 新建策略 */
  function handleCreate() {
    editingRow.value = undefined;
    formMode.value = 'create';
    formVisible.value = true;
  }

  /** 删除策略 */
  function handleDelete(row: AlertStrategyOutput) {
    if (!appDetailStore.appID || !spaceStore.currentSpace || !row.id) return;
    InfoBox({
      title: `${t('确认删除该策略')}?`,
      headerAlign: 'center',
      footerAlign: 'center',
      content: h('div', { class: 'text-left' }, [
        h('div', [t('策略名称: {name}', { name: row.displayName || row.id })]),
        h('div', { class: 'mt-[14px] bg-[#F5F7FA] py-[12px] px-[16px]' }, [t('删除后，将不可恢复，请谨慎操作！')]),
      ]),
      confirmButtonTheme: 'danger',
      confirmText: t('删除'),
      cancelText: t('取消'),
      async onConfirm() {
        if (!appDetailStore.appID || !spaceStore.currentSpace || !row.id) return;
        try {
          await BkintegrationsBkmonitorService.deleteAlertStrategy({
            workspaceID: spaceStore.currentSpace,
            appID: appDetailStore.appID,
            strategyID: row.id,
          });
          Message({ theme: 'success', message: t('删除成功') });
          await fetchList();
        } catch {
          // 错误由拦截器统一处理
        }
      },
    });
  }

  /** 编辑策略 */
  function handleEdit(row: AlertStrategyOutput) {
    editingRow.value = row;
    formMode.value = 'edit';
    formVisible.value = true;
  }

  /** 切换策略启停状态，返回是否切换成功 */
  async function handleSwitch(row: AlertStrategyOutput): Promise<boolean> {
    if (!appDetailStore.appID || !spaceStore.currentSpace || !row.id) return false;
    switchingId.value = row.id;
    const targetEnabled = !row.enabled;
    try {
      await BkintegrationsBkmonitorService.switchAlertStrategy({
        workspaceID: spaceStore.currentSpace,
        appID: appDetailStore.appID,
        strategyID: row.id,
        enabled: targetEnabled,
      });
      row.enabled = targetEnabled;
      Message({
        theme: 'success',
        message: targetEnabled ? t('启用成功') : t('停用成功'),
      });
      return true;
    } catch {
      return false;
    } finally {
      switchingId.value = '';
    }
  }

  /** 表单保存成功后刷新列表 */
  function onFormSuccess() {
    fetchList();
  }

  /** 根据当前表格数据动态生成监控指标筛选项 */
  function updateMonitorMetricChildren() {
    const set = new Set<string>();
    tableData.value.forEach(row => {
      if (row.monitorMetric) set.add(row.monitorMetric);
    });
    const children = Array.from(set).map(metric => ({ id: metric, name: metric }));
    const next = [...searchData.value];
    const idx = next.findIndex(item => item.id === 'monitorMetric');
    if (idx > -1) {
      next[idx] = { ...next[idx], children };
      searchData.value = next;
    }
  }

  /** 根据告警组列表动态生成告警组筛选项 */
  function updateUserGroupChildren() {
    const children = userGroups.value.map(g => ({
      id: String(g.id ?? ''),
      name: g.name || String(g.id ?? ''),
    }));
    const next = [...searchData.value];
    const idx = next.findIndex(item => item.id === 'noticeGroupIDs');
    if (idx > -1) {
      next[idx] = { ...next[idx], children };
      searchData.value = next;
    }
  }

  // 监听搜索值变化，重置分页
  watch(
    searchValue,
    () => {
      handleResetPage();
    },
    { deep: true },
  );

  // 监听 appID 变化，首次加载
  watch(
    () => appDetailStore.appID,
    async appID => {
      if (appID) {
        handleResetPage();
        // 先拉取告警组列表，建立 ID->名称 映射后再拉取策略列表
        // 避免表格先渲染出告警组 ID，再切换为名称的闪烁
        isLoading.value = true;
        try {
          await fetchUserGroups();
          await fetchList();
        } finally {
          // fetchList() 内部已设置 isLoading=false 兜底，此处再确保一次
          isLoading.value = false;
        }
      } else {
        tableData.value = [];
        userGroups.value = [];
      }
    },
    { immediate: true },
  );
</script>

<style lang="postcss" scoped></style>
