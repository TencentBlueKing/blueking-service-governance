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
  <MsHeader
    back-color="#3A84FF"
    class="env-header !fixed w-full z-[999]"
    :title="$t('集群健康诊断')"
    :trigger-back="handlePageBack"
  >
    <Divider
      class="h-[16px] mx-[16px]"
      direction="vertical"
      type="solid"
    />
    <span class="text-[#4D4F56]">{{ $t('集群') }}：</span>
    <span class="text-[#4D4F56]">
      {{ clusterData?.clusterID || '--' }}
    </span>
  </MsHeader>
  <Skeleton
    :full-height="false"
    :loading="isLoading"
    theme="gray"
  >
    <template #loading>
      <div class="grid grid-cols-2 gap-[16px] p-[16px] mt-[52px] responsive-container">
        <Layout.shape
          :height="252"
          width="100%"
        ></Layout.shape>
        <Layout.shape
          :height="252"
          width="100%"
        ></Layout.shape>
        <div class="col-span-2 bg-[#fff] min-h-[calc(100vh_-_404px)]">
          <FlexRow>
            <template #left>
              <Layout.shape
                class="mr-[12px]"
                :height="42"
              >
              </Layout.shape>
              <Layout.shape
                class="mr-[12px]"
                :height="42"
              ></Layout.shape>
              <Layout.shape :height="42"></Layout.shape>
            </template>
            <template #right>
              <Layout.shape></Layout.shape>
            </template>
          </FlexRow>
          <FlexRow class="py-[12px] w-full">
            <template #left>
              <Layout.shape
                class="mr-[2px]"
                :width="102"
              ></Layout.shape>
              <Layout.shape
                class="mr-[2px]"
                :width="102"
              ></Layout.shape>
              <Layout.shape
                class="mr-[2px]"
                :width="102"
              ></Layout.shape>
              <Layout.shape :width="102"></Layout.shape>
            </template>
            <template #right>
              <Layout.shape
                class="mr-[6px]"
                :width="102"
              ></Layout.shape>
              <Layout.shape
                class="mr-[6px]"
                :width="102"
              ></Layout.shape>
              <Layout.shape
                class="mr-[6px]"
                :width="102"
              ></Layout.shape>
              <Layout.shape
                class="mr-[6px]"
                :width="102"
              ></Layout.shape>
              <Layout.shape :width="300"></Layout.shape>
            </template>
          </FlexRow>
          <Layout.table
            :rows="9"
            width="100%"
          ></Layout.table>
        </div>
      </div>
    </template>
    <div class="responsive-container">
      <div class="grid grid-cols-2 gap-[16px]">
        <!-- 集群健康分数 -->
        <ScoreCard
          :active="curScoreCardActive as string"
          class="score-card rounded-[2px] shadow-[0_2px_4px_0_rgba(25,25,41,0.05)]"
          :count="levelCount"
          :value="clusterData?.score || 0"
          @change="handleScoreCardChange"
        />
        <!-- 基础信息 -->
        <BasicInfoCard
          class="rounded-[2px] shadow-[0_2px_4px_0_rgba(25,25,41,0.05)]"
          :cluster-info="clusterData?.clusterInfo"
        />
      </div>
      <div class="col-span-2 bg-[#fff] px-[16px] mt-[16px] rounded-[2px] shadow-[0_2px_4px_0_rgba(25,25,41,0.05)]">
        <Tab
          :key="tabKey"
          :active="filterData.isRecovered"
          class="action-tab"
          type="unborder-card"
          @change="handleTabChange"
        >
          <template #setting>
            <Button
              v-bkloading="{ loading: exportLoading, size: 'small' }"
              class="mr-[16px]"
              text
              theme="primary"
              @click="handleDownloadReport"
            >
              <i class="bkms-icon bkms-icon-download text-[16px] mr-[4px]"></i>
              {{ $t('导出诊断报告') }}
            </Button>
          </template>
          <Tab.TabPanel
            v-for="item in tabList"
            :key="item.id"
            :label="item.label"
            :name="item.id"
          >
            <template #label>
              <div class="flex items-center">
                <span>{{ item.label }}</span>
                <div class="bg-[#E1ECFF] h-[16px] leading-[16px] px-[8px] ml-[4px] rounded-[8px]">
                  {{ item.count }}
                </div>
              </div>
            </template>
          </Tab.TabPanel>
        </Tab>
        <div class="py-[12px] w-full">
          <FlexRow
            class="w-full mb-[16px] action-bar"
            rclass="flex"
          >
            <template #left>
              <ButtonGroup
                v-model="filterData.category"
                :list="categoryTabs"
              >
                <template #suffix="{ data }: { data: { label: string; value: number | string; count?: number } }">
                  ({{ data.count }})
                </template>
              </ButtonGroup>
            </template>
            <template #right>
              <Select
                v-model="filterData.level"
                class="mr-[8px] min-w-[108px] w-[108px]"
                :clearable="false"
                :list="levelSelectOptions"
                :placeholder="$t('请选择级别')"
              >
              </Select>
              <Select
                v-model="filterData.resourceType"
                class="mr-[8px] min-w-[108px]"
                :clearable="false"
                filterable
                :list="resourceTypeSelectOptions"
                :placeholder="$t('请选择资源类型')"
              ></Select>
              <Select
                v-model="filterData.namespace"
                class="mr-[8px] min-w-[108px]"
                :clearable="false"
                filterable
                :list="namespaceSelectOptions"
                :placeholder="$t('请选择命名空间')"
              ></Select>
              <Select
                v-model="filterData.resourceName"
                class="mr-[8px] min-w-[108px]"
                :clearable="false"
                filterable
                :list="resourceNameSelectOptions"
                :placeholder="$t('请选择资源名称')"
              ></Select>
              <Input
                v-model.trim="filterData.keyword"
                class="max-w-[320px] min-w-[240px]"
                clearable
                :placeholder="createPlaceholder({ labels: ['告警资源', '告警内容'] })"
              />
            </template>
          </FlexRow>
          <Table
            v-bkloading="{ loading: filterLoading }"
            :data="filteredList"
            :expand-config="expandConfig"
            :max-height="maxHeight"
            :pagination="pagination"
            :row-config="{
              isHover: true,
              isCurrent: true,
            }"
            show-header-overflow
            show-overflow
            :sort-config="sortConfig"
            :virtual-y-config="{ enabled: true, gt: 10 }"
            @page-limit-change="handlePageLimitChange"
            @page-value-change="handlePageValueChange"
            @sort-change="handleSortChange"
          >
            <template #empty>
              <TableException
                :type="curExceptionType"
                @clear="initFilterData"
                @refresh="handleGetList"
              >
              </TableException>
            </template>
            <TableColumn
              type="expand"
              width="30"
            >
              <template #content="{ row }: { row: RowData }">
                <DetailCard :data="row" />
              </template>
            </TableColumn>
            <TableColumn
              field="resourceKey"
              :label="$t('告警资源')"
              min-width="240"
              show-overflow="tooltip"
            >
              <template #default="{ row }: { row: RowData }">
                <div class="flex items-center">
                  <span
                    v-if="row.level && LEVEL_FOR_UI?.[row.level]?.resourceKeyColumnBg"
                    class="inline-block w-[2px] h-[12px]"
                    :style="{ backgroundColor: LEVEL_FOR_UI?.[row.level].resourceKeyColumnBg }"
                  >
                  </span>
                  <Button
                    class="ml-[8px]"
                    text
                    theme="primary"
                    >{{ row.resourceKey }}</Button
                  >
                </div>
              </template>
            </TableColumn>
            <TableColumn
              field="resourceType"
              :label="$t('类型')"
              show-overflow="tooltip"
              width="120"
            >
            </TableColumn>
            <TableColumn
              field="recordCount"
              :label="$t('记录数')"
              show-overflow="tooltip"
              sortable
              width="100"
            >
            </TableColumn>
            <TableColumn
              field="timestamp"
              :label="$t('开始时间')"
              show-overflow="tooltip"
              sortable
              width="160"
            >
              <template #default="{ row }">
                {{ formatTimeByTimezone(row.timestamp) }}
              </template>
            </TableColumn>
            <TableColumn
              field="contextMsg"
              :label="$t('告警内容')"
              min-width="300"
              show-overflow="tooltip"
              sortable
            >
            </TableColumn>
            <TableColumn
              field="isRecovered"
              :label="$t('状态')"
              show-overflow="tooltip"
              width="100"
            >
              <template #default="{ row }: { row: RowData }">
                <Tag
                  v-if="row.level"
                  :theme="LEVEL_FOR_UI?.[row.level].tagTheme"
                >
                  {{ LEVEL_FOR_UI?.[row.level].isRecovered ? $t('已恢复') : $t('未恢复') }}
                </Tag>
              </template>
            </TableColumn>
          </Table>
        </div>
      </div>
    </div>
  </Skeleton>
</template>

<script lang="ts" setup>
  import { computed, onMounted, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Divider, Input, Message, Select, Tab, Tag } from 'bkui-vue';
  import { countBy, debounce } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { CheckItemOutput, ClusterReportOutput } from '~/@types/v1/bkintegrations-kubeinsight';
  import { BkintegrationsKubeinsightService } from '~/api/modules/v1';
  import { downloadBase64File, formatTimeByTimezone, generateFieldOptions } from '~/common/util';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import useDynamicsHeight from '~/composables/use-table-height';

  import BasicInfoCard from './card/basic-info-card.vue';
  import DetailCard from './card/detail-card.vue';
  import ScoreCard from './card/score-card.vue';
  import { LEVEL_FOR_UI, LEVEL_VALUE, LevelType } from './levelMap';

  import type { VxeTablePropTypes } from '@blueking/vxe-table';

  /** 过滤项类型 */
  type FilterOptions = 'all' | number | string;
  /** Table行数据类型 */
  type RowData = CheckItemOutput;

  const props = defineProps<{
    envId: string;
  }>();

  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const router = useRouter();
  // 引入表格动态高度
  // 1. 响应式更新表格最大高度
  // 2. 让虚拟滚动生效，优化表格性能，解决limit较大时有明显卡顿问题
  const { maxHeight, refresh } = useDynamicsHeight(88, [
    '.bk-navigation-header',
    '.env-header',
    '.score-card',
    '.action-tab',
    '.action-bar',
    '.bk-notice-component-alert',
  ]);

  /** table折叠配置项 */
  const expandConfig = {
    showIcon: true,
    iconOpen: 'bkms-icon bkms-icon-down-shape !text-[#C4C6CC] hover:!text-[#63656E]',
    iconClose: 'bkms-icon bkms-icon-right-shape !text-[#C4C6CC] hover:!text-[#63656E]',
  };
  /** Table排序配置 */
  const sortConfig = {
    multiple: false,
    trigger: 'cell',
  };

  /** 级别下拉框Options */
  const levelSelectOptions = [
    {
      value: 'all',
      label: t('所有级别'),
    },
    {
      value: LEVEL_VALUE.RISK,
      label: t('致命问题'),
    },
    {
      value: LEVEL_VALUE.WARN,
      label: t('预警问题'),
    },
    {
      value: LEVEL_VALUE.INFO,
      label: t('提醒问题'),
    },
  ];

  /** 当前页面所有数据的载体 */
  const clusterData = ref<ClusterReportOutput>();

  const defaultFilterData = {
    level: 'all',
    resourceType: 'all',
    namespace: 'all',
    resourceName: 'all',
    isRecovered: 0,
    category: 'all',
    keyword: '',
  };

  /** 当前过滤参数 */
  const filterData = ref<{
    category: FilterOptions;
    isRecovered: FilterOptions;
    keyword: string;
    level: FilterOptions;
    namespace: FilterOptions;
    resourceName: FilterOptions;
    resourceType: FilterOptions;
  }>({ ...defaultFilterData });
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: filterData,
  });

  // 排序参数单独管理
  const sortParams = ref<{
    field?: string;
    order: VxeTablePropTypes.SortOrder;
  }>({
    field: 'timestamp',
    order: 'desc',
  });
  // 分页参数
  const pagination = ref({
    current: 1,
    count: 0,
    limit: 10,
  });
  /** 页面首次loading 用于骨架屏 */
  const isLoading = ref(false);
  /** 导出出诊报告loading */
  const exportLoading = ref(false);
  /** 基于前端筛选的loading */
  const filterLoading = ref(false);
  // 基于 filterData 筛选后的列表
  const filteredList = ref<RowData[]>([]);
  /** 用于refresh Tab组件 */
  const tabKey = ref(0);

  // 集群健康分数 - 基于当前 是否选中（恢复/未恢复）
  const curScoreCardActive = computed(() => {
    // * 因为filterData.level与Select绑定，不能存已恢复的level，这里需要区分处理
    if (filterData.value.isRecovered === 0) {
      // 未恢复的情况，需要返回当前级别
      return filterData.value.level;
    } else if (filterData.value.isRecovered === 1) {
      // 已恢复的情况，需排除 未恢复、全部
      return LEVEL_VALUE.RECOVERED;
    }
    return '';
  });

  // 各级别数量统计
  const levelCount = computed(() => {
    const items = clusterData.value?.abnormalItems || [];
    const counted = countBy(items, 'level');
    return {
      RISK: counted[LEVEL_VALUE.RISK] || 0,
      WARN: counted[LEVEL_VALUE.WARN] || 0,
      INFO: counted[LEVEL_VALUE.INFO] || 0,
      RECOVERED: counted[LEVEL_VALUE.RECOVERED] || 0,
    };
  });

  const tabList = computed(() => {
    const unRecoveredCount = levelCount.value.RISK + levelCount.value.WARN + levelCount.value.INFO;
    const recoveredCount = levelCount.value.RECOVERED;
    const totalCount = unRecoveredCount + recoveredCount;
    return [
      {
        id: 0,
        label: t('未恢复'),
        count: unRecoveredCount,
      },
      {
        id: 1,
        label: t('已恢复'),
        count: recoveredCount,
      },
      {
        id: 'all',
        label: t('全部'),
        count: totalCount,
      },
    ];
  });

  // 问题分类tabs
  const categoryTabs = computed(() =>
    generateFieldOptions<RowData>(filteredList.value, 'category', {
      allItem: { label: t('全部问题'), value: 'all' },
      valueStrategy: 'field',
    }),
  );
  // 资源类型下拉选项
  const resourceTypeSelectOptions = computed(() =>
    generateFieldOptions<RowData>(clusterData.value?.abnormalItems || [], 'resourceType', {
      allItem: { label: t('所有类型'), value: 'all' },
      valueStrategy: 'field',
    }),
  );
  // 命名空间下拉选项
  const namespaceSelectOptions = computed(() => {
    const items = clusterData.value?.abnormalItems || [];
    const namespaceCount: Record<string, number> = {};

    // 统计每个命名空间的数量
    items.forEach(item => {
      const parts = item?.resourceKey?.split('/') || [];
      if (parts.length > 1) {
        // 存在命名空间
        const namespace = parts[0];
        namespaceCount[namespace] = (namespaceCount[namespace] || 0) + 1;
      }
    });

    // 生成选项列表
    const options = Object.keys(namespaceCount).map(namespace => ({
      label: namespace,
      value: namespace,
    }));

    return [{ label: t('所有命名空间'), value: 'all' }, ...options];
  });

  // 资源名称下拉选项
  const resourceNameSelectOptions = computed(() => {
    const items = clusterData.value?.abnormalItems || [];
    const resourceNameCount: Record<string, number> = {};

    // 统计每个资源名称的数量
    items.forEach(item => {
      const parts = item?.resourceKey?.split('/') || [];
      const resourceName = parts.length > 1 ? parts[1] : parts[0];
      resourceNameCount[resourceName] = (resourceNameCount[resourceName] || 0) + 1;
    });

    // 生成选项列表
    const options = Object.keys(resourceNameCount).map(resourceName => ({
      label: resourceName,
      value: resourceName,
    }));

    // 添加"所有资源名称"选项
    return [{ label: t('所有资源名称'), value: 'all' }, ...options];
  });

  // 封装 debounce 更新方法，支持自定义 callback 和 loading
  function debouncedUpdate(callback?: () => void, showLoading = true, delay = 300) {
    return debounce(() => {
      if (showLoading) {
        filterLoading.value = true;
      }
      // 此处的setTimeout用于延迟前端筛选逻辑的执行
      // filterData的变更同样会触发前端交互，为避免交互卡顿，此处增加延迟
      setTimeout(() => {
        if (callback) {
          callback();
        } else {
          filteredList.value = filterItems();
        }
        if (showLoading) {
          filterLoading.value = false;
        }
      }, 100);
    }, delay);
  }

  // 前端筛选逻辑函数
  function filterItems() {
    const items = clusterData.value?.abnormalItems || [];
    const filtered = items.filter(item => {
      // 筛选：是否已恢复
      if (filterData.value.isRecovered !== 'all' && item.level) {
        const isRecovered = LEVEL_FOR_UI[item.level].isRecovered;
        const expectRecovered = filterData.value.isRecovered === 1;
        if (isRecovered !== expectRecovered) return false;
      }

      // 筛选：问题分类
      if (filterData.value.category !== 'all' && item.category !== filterData.value.category) {
        return false;
      }

      // 筛选：级别
      if (filterData.value.level !== 'all' && item.level !== filterData.value.level) {
        return false;
      }

      // 筛选：资源类型
      if (filterData.value.resourceType !== 'all' && item.resourceType !== filterData.value.resourceType) {
        return false;
      }

      // 筛选：命名空间
      if (filterData.value.namespace !== 'all') {
        const parts = item?.resourceKey?.split('/') || [];
        const namespace = parts.length > 1 ? parts[0] : '';
        if (namespace !== filterData.value.namespace) return false;
      }

      // 筛选：资源名称
      if (filterData.value.resourceName !== 'all') {
        const parts = item?.resourceKey?.split('/') || [];
        const resourceName = parts.length > 1 ? parts[1] : parts[0];
        if (resourceName !== filterData.value.resourceName) return false;
      }

      // 筛选：关键词模糊查询（告警资源 + 告警内容）
      if (filterData.value.keyword) {
        const keyword = filterData.value.keyword.trim().toLowerCase();
        const resourceKey = item.resourceKey?.toLowerCase() || '';
        const contextMsg = item.contextMsg?.toLowerCase() || '';
        if (!resourceKey.includes(keyword) && !contextMsg.includes(keyword)) {
          return false;
        }
      }

      return true;
    });

    // 排序逻辑
    const { field, order } = sortParams.value;
    if (field && order) {
      filtered.sort((a, b) => {
        let aVal = a[field as keyof RowData];
        let bVal = b[field as keyof RowData];

        // recordCount 字段先转换为 number
        if (field === 'recordCount') {
          aVal = Number(aVal);
          bVal = Number(bVal);
        }

        // 处理 null/undefined
        if (aVal == null && bVal == null) return 0;
        if (aVal == null) return order === 'asc' ? -1 : 1;
        if (bVal == null) return order === 'asc' ? 1 : -1;

        // 比较值
        let result = 0;
        if (field === 'timestamp') {
          // 时间排序
          result = new Date(aVal as string).getTime() - new Date(bVal as string).getTime();
        } else if (typeof aVal === 'number' && typeof bVal === 'number') {
          result = aVal - bVal;
        } else {
          result = String(aVal).localeCompare(String(bVal));
        }

        return order === 'asc' ? result : -result;
      });
    }

    return filtered;
  }

  // 导出诊断报告
  async function handleDownloadReport() {
    exportLoading.value = true;
    const res = await BkintegrationsKubeinsightService.getLatestEnvReport({
      envID: props.envId,
      generatePDF: true,
    }).finally(() => (exportLoading.value = false));
    const mimeType = 'application/pdf';
    const fileName = `${res.clusterID}-${t('诊断报告')}.pdf`;
    const pdfData = (res?.pdfData as unknown as string) || '';
    downloadBase64File(pdfData, fileName, mimeType);
    Message({
      theme: 'success',
      message: t('导出成功'),
    });
  }

  async function handleGetList() {
    try {
      isLoading.value = true;
      clusterData.value = await BkintegrationsKubeinsightService.getLatestEnvReport({
        envID: props.envId,
      });
      if (clusterData.value?.abnormalItems) {
        pagination.value.count = clusterData.value?.abnormalItems.length;
      }
      clearErrorType();
    } catch (error) {
      console.error(error);
      setTypeToError();
    } finally {
      isLoading.value = false;
      refresh();
    }
  }

  // 返回上一页
  function handlePageBack() {
    // 显式指定 fallback：环境管理页未配置为当前页的父子级路由，resolveParent 无法推导到
    router.back({ name: 'env' });
  }

  function handlePageLimitChange(val: number) {
    pagination.value.limit = val;
  }

  function handlePageValueChange(val: number) {
    pagination.value.current = val;
  }

  // 集群健康分数卡片 change事件
  function handleScoreCardChange(level: LevelType, isRecovered: boolean) {
    // 如果输出level为空，表示取消选择，则默认选中所有级别
    let curLevel = level === '' ? 'all' : level;
    // 级别下拉框不包含已恢复选项，此处需要过滤掉
    if (curLevel !== LEVEL_VALUE.RECOVERED) {
      filterData.value.level = curLevel;
    } else {
      initLevel();
    }
    // 集群分数卡片包含已恢复/未恢复，change后需要更新isRecovered状态
    filterData.value.isRecovered = Number(isRecovered);
  }

  // tableSort Change事件
  function handleSortChange({ field, order }: { field: string; order: VxeTablePropTypes.SortOrder }) {
    sortParams.value = {
      field,
      order,
    };
  }

  // Tab Change事件
  // 此处没有使用v-model目的在于 同时更改level与isRecovered 避免watch filterData多次筛选
  function handleTabChange(value: FilterOptions) {
    if (value === filterData.value.isRecovered) return;
    if (value === 1) {
      initLevel();
    }
    filterData.value.isRecovered = value;
  }

  function initFilterData() {
    Object.assign(filterData.value, defaultFilterData);
  }

  // 重置级别下拉框为 所有级别
  function initLevel() {
    filterData.value.level = 'all';
  }

  // 初始化 pagination
  function initPagination() {
    pagination.value.current = 1;
    pagination.value.count = filteredList.value.length;
  }

  // Tab 组件可能没有处理TabList的深度监听，因此label发生变化不会响应式更新
  watch(
    tabList,
    () => {
      tabKey.value += 1;
    },
    { deep: true },
  );

  // filterData 变更，触发前端筛选逻辑
  watch(
    filterData,
    debouncedUpdate(
      () => {
        filteredList.value = filterItems();
        initPagination();
      },
      true,
      300,
    ),
    { deep: true },
  );

  // 排序参数变更，触发前端筛选逻辑，但不需要重置 pagination
  watch(sortParams, debouncedUpdate(), { deep: true });

  // 监听数据加载完成，初始化 filteredList 并重置 pagination
  watch(
    () => clusterData.value?.abnormalItems,
    debouncedUpdate(
      () => {
        if (clusterData.value?.abnormalItems) {
          filteredList.value = filterItems();
          initPagination();
        }
      },
      false,
      0,
    ),
  );

  onMounted(() => {
    handleGetList();
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-tab-content) {
    display: none;
  }

  .responsive-container {
    width: 100%;
    height: calc(100% - 52px);
    padding: 16px;
    margin-top: 52px;
  }

  @media (min-width: 1600px) {
    .responsive-container {
      max-width: 1600px;
      margin-left: auto;
      margin-right: auto;
    }
  }
</style>
