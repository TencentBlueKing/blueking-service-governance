/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

import { computed, ref, watch } from 'vue';
import type { Ref } from 'vue';

import { useI18n } from 'vue-i18n';
import { AppService } from '~/api/modules/v1';
import { APP_DEPLOY_STATUS, DEPLOY_FAILED_STATUSES } from '~/common/enums/deploy';
import { useDeployStatusMap } from '~/composables/use-deploy-status';
import { envTypeMap } from '~/composables/use-env-manager';
import { useResourceSpecDisplay } from '~/composables/use-resource-spec-display';
import { type ISearchValue, type ISelectKey, useTableSearchSelect } from '~/composables/use-search';
import useSearchFilter from '~/composables/use-search-filter';
import { useAppDetail } from '~/stores/app-detail';

import type { VxeTableDefines } from '@blueking/vxe-table';
import type { AppDeployOverviewEnvObj, DeployOverviewInstancesObj, GetAppDeployOverviewRequest } from '~/@types/v1/app';
import type { EnvOutput } from '~/@types/v1/env';
import type { GPAConfigOutputObj, GPAStatusOutput } from '~/@types/v1/gpa';

const ENV_TYPE_ORDER = ['production', 'staging', 'test', 'development'];
const TYPE_FILTER_ALL = '__all__';
// GPA 未返回 phase 不代表异常；只有已启用且明确处于正常集合之外时，才展示异常状态。
const NORMAL_AUTOSCALING_PHASES = new Set(['active', 'limited', 'initializing']);
// 状态筛选按业务关注度排序，接口中出现但未列出的状态会追加到末尾。
const DEPLOY_STATUS_FILTER_ORDER: string[] = [
  APP_DEPLOY_STATUS.FAILED,
  APP_DEPLOY_STATUS.POLLING_TIMEOUT,
  APP_DEPLOY_STATUS.POLLING_BROKEN,
  APP_DEPLOY_STATUS.DEPLOYING,
  APP_DEPLOY_STATUS.DEPLOYED,
  APP_DEPLOY_STATUS.UNINSTALLING,
  APP_DEPLOY_STATUS.UNINSTALLED,
  APP_DEPLOY_STATUS.CANCELED,
  APP_DEPLOY_STATUS.UNKNOWN,
];

export interface DeployOverviewAutoScale {
  /** 已启用且后端明确返回异常 phase。 */
  abnormal: boolean;
  /** 区分 autoscaling=null（未配置）与 enabled=false（已配置但关闭）。 */
  configured: boolean;
  enabled: boolean;
  status: GPAStatusOutput | null;
  tips: string[];
}

export interface DeployOverviewDeployTarget {
  /** 总览接口返回的期望实例数，用作新增部署侧栏的默认副本数。 */
  effectiveReplicas?: number;
  /** 来自环境列表接口的完整环境对象，部署表单需要其中的环境类型等信息。 */
  env: EnvOutput;
}

/**
 * 后台总览数据转换后的页面模型。
 * 实例数保留 null，页面据此展示“--”，不能把接口的未知状态误显示成 0。
 */
export interface DeployOverviewRow {
  abnormalCount: null | number;
  autoScale: DeployOverviewAutoScale;
  cpuLimits: string;
  cpuRequests: string;
  deployedAt: string;
  deployStatus: string;
  desiredCount: null | number;
  displayName: string;
  isFeature: boolean;
  memoryLimits: string;
  memoryRequests: string;
  name: string;
  runningCount: null | number;
  type: string;
}

export interface DeployOverviewStat {
  color: string;
  /** 数字后的补充说明，为空时不渲染。 */
  desc: string;
  iconBg: string;
  iconColor: string;
  key: DeployOverviewStatKey;
  label: string;
  value: number;
}

export type DeployOverviewStatKey = 'abnormalInstance' | 'deploying' | 'failed' | 'total';

/** 补充接口字段的 null 语义，并复用现有 GPA 类型描述完整 autoscaling 数据。 */
type DeployOverviewApiRow = Omit<AppDeployOverviewEnvObj, 'autoscalingEnabled' | 'instances'> & {
  autoscaling?: DeployOverviewAutoscaling | null;
  instances?: DeployOverviewInstancesObj | null;
};

type DeployOverviewAutoscaling = Pick<
  GPAConfigOutputObj,
  'computeByLimits' | 'enabled' | 'maxReplicas' | 'metrics' | 'minReplicas'
> & {
  status?: GPAStatusOutput | null;
};

/**
 * 部署总览的数据层：负责接口请求、字段适配、筛选、排序和分页。
 * 组件本身只负责布局与事件转发，避免接口的 null 语义和表格展示逻辑混在模板中。
 */
export function useDeployOverview(envList: Ref<EnvOutput[]>) {
  const { t } = useI18n();
  const { getResourceText, getResourceTips } = useResourceSpecDisplay();
  const appDetailStore = useAppDetail();
  const { getDeployStatusInfo, getDeployStatusMaps } = useDeployStatusMap();

  const rows = ref<DeployOverviewRow[]>([]);
  const isLoading = ref(false);
  const isError = ref(false);
  const globalTypeFilter = ref(TYPE_FILTER_ALL);
  // 指标卡是独立的一级筛选；默认展示当前环境类型下的全部部署环境。
  const activeStat = ref<DeployOverviewStatKey>('total');
  const filterKeys = ['deployStatus'] as const;
  // 每次请求递增；应用快速切换时，仅最后一次请求可以更新页面状态。
  let loadToken = 0;

  const deployStatusMaps = computed(() => getDeployStatusMaps(appDetailStore.appType || null));

  // 顶部“全部 / 生产 / 预发布 / 测试 / 开发”及对应数量。
  const envTypeOptions = computed(() => {
    const counts = Object.fromEntries(ENV_TYPE_ORDER.map(type => [type, 0])) as Record<string, number>;
    rows.value.forEach(row => {
      if (row.type in counts) counts[row.type] += 1;
    });
    return [
      { count: rows.value.length, label: t('全部'), value: TYPE_FILTER_ALL },
      ...ENV_TYPE_ORDER.map(type => ({
        count: counts[type],
        label: envTypeMap[type]?.name || type,
        value: type,
      })),
    ];
  });

  const typeScopedRows = computed(() =>
    globalTypeFilter.value === TYPE_FILTER_ALL
      ? rows.value
      : rows.value.filter(row => row.type === globalTypeFilter.value),
  );

  /**
   * 指标卡只统计当前环境类型范围，不受搜索框、表头筛选和卡片自身筛选影响。
   * 实例汇总仅累加接口已返回的数字，instances=null 的行仍在表格中保持“--”语义。
   */
  const stats = computed<DeployOverviewStat[]>(() => {
    const scoped = typeScopedRows.value;
    const deployingCount = scoped.filter(row => row.deployStatus === APP_DEPLOY_STATUS.DEPLOYING).length;
    const failedCount = scoped.filter(row => isFailedStatus(row.deployStatus)).length;
    const abnormalEnvCount = scoped.filter(row => (row.abnormalCount ?? 0) > 0).length;
    const runningCount = scoped.reduce((total, row) => total + (row.runningCount ?? 0), 0);
    const desiredCount = scoped.reduce((total, row) => total + (row.desiredCount ?? 0), 0);
    const abnormalCount = scoped.reduce((total, row) => total + (row.abnormalCount ?? 0), 0);

    return [
      {
        key: 'total',
        label: t('部署环境'),
        value: scoped.length,
        desc: t('运行实例 {0} / {1}', [runningCount, desiredCount]),
        color: '#313238',
        iconColor: '#63656E',
        iconBg: '#F0F1F5',
      },
      {
        key: 'deploying',
        label: t('部署中环境'),
        value: deployingCount,
        desc: '',
        color: deployingCount > 0 ? '#3A84FF' : '#313238',
        iconColor: '#3A84FF',
        iconBg: '#E1ECFF',
      },
      {
        key: 'failed',
        label: t('部署失败环境'),
        value: failedCount,
        desc: '',
        color: failedCount > 0 ? '#EA3636' : '#313238',
        iconColor: '#EA3636',
        iconBg: '#FFEEEE',
      },
      {
        key: 'abnormalInstance',
        label: t('异常实例'),
        value: abnormalCount,
        desc: abnormalEnvCount > 0 ? t('涉及 {0} 个环境', [abnormalEnvCount]) : '',
        color: abnormalCount > 0 ? '#EA3636' : '#313238',
        iconColor: '#FF9C01',
        iconBg: '#FFF3E1',
      },
    ];
  });

  /**
   * 新增部署不能直接使用总览行：总览接口不保证返回部署表单所需的完整环境信息。
   * 因此以 EnvSelect 的标准环境列表为准，再用总览行补充默认副本数。
   */
  const deployTargets = computed<DeployOverviewDeployTarget[]>(() => {
    const expectedByEnv = new Map(rows.value.map(row => [row.name, row.desiredCount ?? undefined]));
    return envList.value
      .filter(env => !!env.name && env.status !== 'NotReady' && (env.kind || 'standard') === 'standard')
      .map(env => ({ env, effectiveReplicas: expectedByEnv.get(env.name || '') }));
  });

  /** 将部署状态码转换为当前应用类型对应的展示文案，未知状态保留原值。 */
  function getStatusText(status: string) {
    return deployStatusMaps.value.statusTextMap[status] || status || '--';
  }

  // 只展示当前环境类型范围内实际存在的部署状态，避免出现选择后必然为空的选项。
  const statusFilterOptions = computed(() => {
    const presentTexts = new Set(typeScopedRows.value.map(row => getStatusText(row.deployStatus)));
    const options: { label: string; value: string }[] = [];
    const added = new Set<string>();
    // 仅追加当前数据中存在且尚未加入的状态文案。
    const push = (text: string) => {
      if (!text || added.has(text) || !presentTexts.has(text)) return;
      added.add(text);
      options.push({ label: text, value: text });
    };
    DEPLOY_STATUS_FILTER_ORDER.forEach(status => push(getStatusText(status)));
    presentTexts.forEach(push);
    return options;
  });

  // SearchSelect 与表头状态筛选共用同一份搜索条件，保证两处筛选结果同步。
  const searchData = ref<ISelectKey<DeployOverviewRow>[]>([
    {
      id: 'envName',
      name: t('环境名称'),
      field: 'displayName',
      fuzzy: true,
      handleFilter: (row, values) => matchKeyword(values, [row.displayName, row.name]),
    },
    {
      id: 'deployStatus',
      name: t('部署状态'),
      field: 'deployStatus',
      multiple: true,
      children: [],
      handleFilter: (row, values) => values.some(value => value.id === getStatusText(row.deployStatus)),
    },
    {
      id: 'resource',
      name: t('资源规格'),
      field: 'resource',
      fuzzy: true,
      handleFilter: (row, values) => matchKeyword(values, [getResourceText(row)]),
    },
  ]);

  const { searchValue, tableDataMatchSearch: searchFilteredRows } = useTableSearchSelect(typeScopedRows, searchData);
  const { filterOptions, handleFilterChange } = useSearchFilter(searchData, searchValue, filterKeys);

  watch(
    statusFilterOptions,
    options => {
      const target = searchData.value.find(item => item.id === 'deployStatus');
      if (target) target.children = options.map(option => ({ id: option.value, name: option.label }));
    },
    { immediate: true },
  );

  const hasFilter = computed(() => activeStat.value !== 'total' || searchValue.value.length > 0);

  // 未主动点击排序时，默认按最近部署时间倒序；时间相同时按环境类型排序。
  const visibleRows = computed(() =>
    searchFilteredRows.value
      .filter(row => matchStat(row, activeStat.value))
      .sort((a, b) => toTimestamp(b.deployedAt) - toTimestamp(a.deployedAt) || compareByEnvType(a, b)),
  );

  const pagination = ref({ count: 0, limit: 10, current: 1 });

  watch(
    visibleRows,
    list => {
      pagination.value.count = list.length;
      const maxPage = Math.max(1, Math.ceil(list.length / pagination.value.limit));
      if (pagination.value.current > maxPage) pagination.value.current = maxPage;
    },
    { immediate: true },
  );

  watch([globalTypeFilter, activeStat, searchValue], () => {
    pagination.value.current = 1;
  });

  // Table 使用远程筛选模式，但数据均已由总览接口一次返回，因此排序仍在前端完成。
  const sortConfig = {
    // 最近优先对应设计稿中的上三角高亮，因此页面排序方向与组件 order 做反向映射。
    defaultSort: { field: 'deployedAt', order: 'asc' },
    multiple: false,
    trigger: 'cell',
    /** 根据表格传入的首个排序条件返回新数组，避免直接修改原始总览数据。 */
    sortMethod({ data, sortList }: { data: DeployOverviewRow[]; sortList: VxeTableDefines.SortCheckedParams[] }) {
      const sort = sortList[0];
      if (!sort) return data;
      const factor = sort.order === 'asc' ? -1 : 1;
      return [...data].sort((a, b) => compareByField(a, b, String(sort.field)) * factor);
    },
  };

  /** 获取部署状态对应的文案、颜色和图标配置，供状态列统一渲染。 */
  function getStatusInfo(status: string) {
    return getDeployStatusInfo(appDetailStore.appType || null, status);
  }

  /** 清空指标卡、搜索及表头筛选，并将分页恢复到第一页。 */
  function clearFilters() {
    activeStat.value = 'total';
    searchValue.value = [];
    pagination.value.current = 1;
  }

  /**
   * 应用指标卡筛选：点击当前卡片或“部署环境”恢复全部，其余卡片进入对应筛选。
   * 所有卡片始终可点击，数量为 0 时仍保留选中态并让表格展示筛选空态。
   */
  function handleStatClick(key: DeployOverviewStatKey) {
    const shouldReset = key === 'total' || activeStat.value === key;
    searchValue.value = [];
    activeStat.value = shouldReset ? 'total' : key;
    pagination.value.current = 1;
  }

  /** 判断部署状态是否属于失败、轮询超时或轮询中断。 */
  function isFailedStatus(status: string) {
    return (DEPLOY_FAILED_STATUSES as readonly string[]).includes(status);
  }

  /** 根据当前指标卡判断环境行是否应进入表格结果。 */
  function matchStat(row: DeployOverviewRow, stat: DeployOverviewStatKey) {
    if (stat === 'deploying') return row.deployStatus === APP_DEPLOY_STATUS.DEPLOYING;
    if (stat === 'failed') return isFailedStatus(row.deployStatus);
    if (stat === 'abnormalInstance') return row.abnormalCount !== null && row.abnormalCount > 0;
    return true;
  }

  /** 对 SearchSelect 输入执行不区分大小写的模糊匹配，任一关键词命中即保留该行。 */
  function matchKeyword(values: ISearchValue['values'], candidates: string[]) {
    const keywords = values.map(value => (value.id || value.name || '').trim().toLowerCase()).filter(Boolean);
    if (!keywords.length) return true;
    return keywords.some(keyword => candidates.some(candidate => candidate.toLowerCase().includes(keyword)));
  }

  /** 请求总览唯一数据源，并用 token 丢弃应用切换前发出的过期响应。 */
  async function load() {
    const appID = appDetailStore.appID;
    const token = (loadToken += 1);
    if (!appID) {
      rows.value = [];
      isError.value = false;
      return;
    }
    isLoading.value = true;
    isError.value = false;
    try {
      const list = await AppService.getAppDeployOverview<GetAppDeployOverviewRequest, DeployOverviewApiRow[]>(
        { appID },
        { interceptorErr: false },
      );
      if (token !== loadToken) return;
      rows.value = (list || []).map(buildRow);
    } catch (error) {
      if (token !== loadToken) return;
      console.error(error);
      rows.value = [];
      isError.value = true;
    } finally {
      if (token === loadToken) isLoading.value = false;
    }
  }

  /** 将接口结构收敛为模板需要的稳定模型，集中处理缺失字段和 null。 */
  function buildRow(item: DeployOverviewApiRow): DeployOverviewRow {
    const resources = item.resources || {};
    return {
      // instances=null 表示后台无法提供实例数据，三个数量必须统一显示“--”。
      abnormalCount: item.instances ? (item.instances.abnormal ?? 0) : null,
      autoScale: createAutoScale(item.autoscaling),
      cpuLimits: resources.cpuLimits || '',
      cpuRequests: resources.cpuRequests || '',
      deployedAt: item.lastDeployStartedAt || '',
      deployStatus: item.deployStatus || APP_DEPLOY_STATUS.UNKNOWN,
      desiredCount: item.instances ? (item.instances.expected ?? 0) : null,
      displayName: item.envDisplayName || item.envName || '--',
      isFeature: item.envKind === 'feature',
      memoryLimits: resources.memoryLimits || '',
      memoryRequests: resources.memoryRequests || '',
      name: item.envName || '',
      runningCount: item.instances ? (item.instances.running ?? 0) : null,
      type: item.envType || '',
    };
  }

  /**
   * 统一扩缩容展示语义：
   * - null：未配置，不展示标签；
   * - enabled=false：展示“已关闭”；
   * - enabled=true：展示实例区间、指标及后端状态。
   */
  function createAutoScale(config?: DeployOverviewAutoscaling | null): DeployOverviewAutoScale {
    if (!config) return { abnormal: false, configured: false, enabled: false, status: null, tips: [] };
    const enabled = !!config.enabled;
    const status = config.status || null;
    const phase = status?.phase?.trim().toLowerCase() || '';
    const basis = config.computeByLimits ? 'Limits' : 'Requests';
    const tips: string[] = [];
    if (config.minReplicas !== undefined && config.maxReplicas !== undefined) {
      tips.push(t('实例数区间：{0} ~ {1} 个', [config.minReplicas, config.maxReplicas]));
    }
    (config.metrics || []).forEach(metric => {
      if (metric.averageUtilization === undefined) return;
      const label = metric.resource === 'memory' ? t('内存') : (metric.resource || '').toUpperCase();
      tips.push(t('{0} {1} 使用率：{2}%', [label || '--', basis, metric.averageUtilization]));
    });
    return {
      // status/phase 为空不能判异常，避免接口暂未上报状态时出现误报。
      abnormal: enabled && !!phase && !NORMAL_AUTOSCALING_PHASES.has(phase),
      configured: true,
      enabled,
      status,
      tips,
    };
  }

  /** 更新每页条数，并回到第一页避免当前页超出新的页数范围。 */
  function handlePageLimitChange(limit: number) {
    pagination.value.limit = limit;
    pagination.value.current = 1;
  }

  /** 同步表格分页器选择的当前页。 */
  function handlePageValueChange(current: number) {
    pagination.value.current = current;
  }

  /** 按生产、预发布、测试、开发的固定业务顺序比较两条环境数据。 */
  function compareByEnvType(a: DeployOverviewRow, b: DeployOverviewRow) {
    return getEnvTypeWeight(a.type) - getEnvTypeWeight(b.type);
  }

  /** 执行表格主动排序；目前仅最近部署时间可排序，其余字段回退到环境类型顺序。 */
  function compareByField(a: DeployOverviewRow, b: DeployOverviewRow, field: string) {
    if (field === 'deployedAt') return toTimestamp(a.deployedAt) - toTimestamp(b.deployedAt);
    return compareByEnvType(a, b);
  }

  /** 获取环境类型的排序权重，未知类型统一排在标准类型之后。 */
  function getEnvTypeWeight(type: string) {
    const index = ENV_TYPE_ORDER.indexOf(type);
    return index === -1 ? ENV_TYPE_ORDER.length : index;
  }

  /** 将接口时间转换为可排序时间戳，空值或非法时间按 0 处理。 */
  function toTimestamp(time: string) {
    if (!time) return 0;
    const timestamp = new Date(time).getTime();
    return Number.isNaN(timestamp) ? 0 : timestamp;
  }

  return {
    activeStat,
    clearFilters,
    deployTargets,
    envTypeOptions,
    filterOptions,
    getResourceText,
    getResourceTips,
    getStatusInfo,
    globalTypeFilter,
    handleFilterChange,
    handlePageLimitChange,
    handlePageValueChange,
    handleStatClick,
    hasFilter,
    isError,
    isLoading,
    load,
    pagination,
    searchData,
    searchValue,
    sortConfig,
    stats,
    visibleRows,
  };
}
