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
  <!-- 全局视图容器：使用 flex 布局实现高度自适应，overflow-hidden 防止内容溢出 -->
  <div class="flex h-full flex-col overflow-hidden">
    <Skeleton
      :loading="isLoading"
      theme="gray"
    >
      <!-- 骨架屏加载状态：展示表格结构的占位符 -->
      <template #loading>
        <div class="p-[16px]">
          <FlexRow class="mb-[12px] w-full">
            <template #left>
              <Layout.shape :width="260" />
            </template>
            <template #right>
              <Layout.shape :width="320" />
            </template>
          </FlexRow>
          <Layout.table :rows="10" />
        </div>
      </template>

      <!-- 筛选工具栏：包含镜像 Tag 筛选和部署环境过滤选项 -->
      <div
        class="flex h-[40px] items-center justify-between bg-[#fff] px-[16px] border-t border-l border-r border-[#e8eaec]"
      >
        <!-- 左侧筛选控件 -->
        <div class="flex items-center gap-[16px]">
          <!-- 镜像 Tag 下拉筛选框 -->
          <Select
            v-model="selectedImageTag"
            class="w-[200px] bg-[#fff]"
            :clearable="false"
            filterable
            :placeholder="$t('全部')"
            size="small"
          >
            <template #prefix>
              <span
                class="inline-flex h-full items-center border-r border-[#dcdee5] px-[8px] text-[12px] text-[#4D4F56] bg-[#FAFBFD]"
              >
                {{ $t('镜像 Tag') }}
              </span>
            </template>
            <Select.Option
              id="__all__"
              :name="$t('全部')"
            />
            <Select.Option
              v-for="tag in imageTagOptions"
              :id="tag"
              :key="tag"
              :name="tag"
            />
          </Select>
          <!-- 仅显示有部署环境的复选框 -->
          <Checkbox v-model="onlyDeployedEnvs">
            <span class="text-[12px]">{{ $t('仅显示有部署的环境') }}</span>
          </Checkbox>
        </div>
        <!-- 右侧图例说明：展示不同部署状态的颜色标识 -->
        <div class="flex items-center gap-[12px] text-[12px] text-[#4d4f56]">
          <span>{{ $t('Tag 图例') }}：</span>
          <span class="inline-flex items-center gap-[6px]">
            <i class="inline-block h-[14px] w-[14px] rounded-[2px] bg-[#c8f2d8]"></i>{{ $t('部署成功') }}
          </span>
          <span class="inline-flex items-center gap-[6px]">
            <i class="inline-block h-[14px] w-[14px] rounded-[2px] bg-[#ffe1e1]"></i>{{ $t('部署失败') }}
          </span>
          <span class="inline-flex items-center gap-[6px]">
            <i class="inline-block h-[14px] w-[14px] rounded-[2px] bg-[#d9e8ff]"></i>{{ $t('部署中') }}
          </span>
        </div>
      </div>

      <!-- 矩阵表格区域 -->
      <div
        ref="matrixContentRef"
        class="min-h-0 flex-1 overflow-hidden"
      >
        <Table
          :border="'full'"
          class="global-view-table"
          :columns="matrixColumns"
          :data="filteredApps"
          :header-cell-class-name="handleHeaderCellClassName"
          :max-height="matrixTableHeight"
          :pagination="tablePagination"
          :row-config="{
            keyField: 'id',
            isHover: true,
            isCurrent: false,
          }"
          :row-height="42"
          show-overflow="tooltip"
          stripe
          @page-limit-change="handlePageLimitChange"
          @page-value-change="handlePageValueChange"
        >
          <template #empty>
            <TableException
              :type="curExceptionType"
              @clear="handleClearFilters"
              @refresh="handleLoadMatrix"
            />
          </template>
        </Table>
      </div>
    </Skeleton>
  </div>
</template>

<script lang="ts" setup>
  import { computed, h, onBeforeMount, reactive, ref, watch } from 'vue';
  import type { VNodeChild } from 'vue';

  import { Table } from '@blueking/table';
  import { Button, Checkbox, Popover, Select, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import { APP_DEPLOY_STATUS, HELM_DEPLOY_STATUS } from '~/common/enums/deploy';
  import ColorIcon from '~/components/color-icon.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import TableException from '~/components/table-exception.vue';
  import { useDeployStatusMap } from '~/composables/use-deploy-status';
  import { useElementHeight } from '~/composables/use-element-height';
  import useTableEmpty from '~/composables/use-table-empty';
  import { useDeployEnvStore } from '~/stores/deploy-env';
  import { useSpaceStore } from '~/stores/space';

  import TypeIcon from './components/type-icon.vue';

  import type { ISearchValue } from 'bkui-vue/lib/search-select/utils';
  import type { AppDeployedEnvOutputObj, AppInfoOutputObj } from '~/@types/v1/app';
  import type { EnvOutput } from '~/@types/v1/env';
  import type { AppType } from '~/composables/app-type';

  /** 应用表头部署状态筛选值 */
  type DeployStatusFilter = 'all' | Exclude<MatrixStatusTone, 'unknown'>;

  interface DeployStatusFilterOption {
    icon?: string;
    label: string;
    value: DeployStatusFilter;
  }

  /** 环境分组接口：用于将环境按类型（开发/测试/生产）分组展示 */
  interface EnvGroup {
    envs: EnvOutput[];
    label: string;
    type: string;
  }

  /** 矩阵列配置接口：定义表格列的结构和渲染方式 */
  interface MatrixColumn {
    children?: MatrixColumn[];
    className?: string;
    field?: string;
    fixed?: boolean | string;
    label?: boolean | number | string;
    minWidth?: number;
    width?: number;
    render?: (args: { row: AppInfoOutputObj }) => VNodeChild;
    renderHead?: () => VNodeChild;
  }

  /** 矩阵部署信息类型：扩展应用部署环境对象，增加镜像 Tag 字段 */
  type MatrixDeployInfo = AppDeployedEnvOutputObj & {
    imageTag?: string;
  };

  /** 部署状态色调类型：用于区分不同状态的标签样式 */
  type MatrixStatusTone = 'failed' | 'running' | 'success' | 'unknown';

  interface Props {
    apps: AppInfoOutputObj[];
    /** 应用列表页的搜索条件 */
    searchValue?: ISearchValue[];
    /** 外部传入的工作空间 ID，不传则使用 store 中的当前空间 */
    space?: string;
  }

  /** 表格单元格参数接口：用于表头样式处理 */
  interface TableCellParams {
    column?: {
      field?: string;
      prop?: string;
      title?: string;
    };
  }

  const props = defineProps<Props>();
  const emit = defineEmits<{
    clear: [];
  }>();

  const router = useRouter();
  const { t } = useI18n();
  const deployEnvStore = useDeployEnvStore();
  const spaceStore = useSpaceStore();
  const { getDeployStatusInfo } = useDeployStatusMap();

  /** 环境类型排序权重：确保表头按开发→测试→生产的顺序展示 */
  const ENV_TYPE_ORDER: Record<string, number> = {
    development: 1,
    production: 3,
    test: 2,
  };

  /** 环境类型展示文案映射 */
  const ENV_TYPE_LABEL: Record<string, string> = {
    development: '开发',
    production: '生产',
    test: '测试',
    unknown: '未知',
  };

  /** 失败状态集合：用于判断部署是否为失败状态 */
  const FAILED_STATUSES = new Set<string>([
    APP_DEPLOY_STATUS.FAILED,
    APP_DEPLOY_STATUS.POLLING_TIMEOUT,
    APP_DEPLOY_STATUS.POLLING_BROKEN,
    HELM_DEPLOY_STATUS.FAILED,
  ]);

  /** 运行中状态集合：用于判断部署是否正在进行中 */
  const RUNNING_STATUSES = new Set<string>([
    APP_DEPLOY_STATUS.DEPLOYING,
    APP_DEPLOY_STATUS.UNINSTALLING,
    HELM_DEPLOY_STATUS.UNINSTALLING,
    HELM_DEPLOY_STATUS.PENDING_INSTALL,
    HELM_DEPLOY_STATUS.PENDING_UPGRADE,
    HELM_DEPLOY_STATUS.PENDING_ROLLBACK,
  ]);

  /** 数据加载状态 */
  const isLoading = ref(false);

  /** 所有环境列表 */
  const envList = ref<EnvOutput[]>([]);

  /** 镜像 Tag 筛选默认值：'__all__' 表示显示全部 */
  const IMAGE_TAG_ALL = '__all__';

  /** 镜像 Tag 筛选值 */
  const selectedImageTag = ref(IMAGE_TAG_ALL);

  /** 是否仅显示有部署的环境 */
  const onlyDeployedEnvs = ref(false);

  /** 应用部署状态筛选值 */
  const selectedDeployStatus = ref<DeployStatusFilter>('all');

  /** 部署状态筛选弹层 */
  const statusFilterPopoverRef = ref<InstanceType<typeof Popover>>();

  /** 表格容器 DOM 引用，用于计算表格高度 */
  const matrixContentRef = ref<HTMLElement>();

  /** 表格分页配置 */
  const pagination = reactive({
    current: 1,
    limit: 20,
  });

  /** 部署状态 */
  const DEPLOY_TONE_THEME_MAP: Record<MatrixStatusTone, '' | 'danger' | 'success' | 'warning'> = {
    failed: 'danger',
    running: 'warning',
    success: 'success',
    unknown: '',
  };

  /** 应用部署状态筛选选项 */
  const deployStatusFilterOptions = computed<DeployStatusFilterOption[]>(() => [
    { label: t('全部'), value: 'all' },
    { icon: 'normal', label: t('部署成功'), value: 'success' },
    { icon: 'abnormal', label: t('部署失败'), value: 'failed' },
    { icon: 'loading', label: t('部署中'), value: 'running' },
  ]);

  /** 环境分组 → 表头样式映射：不同环境类型使用不同的背景色和文字颜色 */
  const ENV_GROUP_HEADER_CLASS: Record<string, string> = {
    development: '!text-[#3a84ff] bg-[#f0f5ff]',
    production: '!text-[#2caf5e] bg-[#ebfaf0]',
    test: '!text-[#f59500] bg-[#fdf4e8]',
  };

  /** 自适应表格高度：监听容器尺寸和加载状态变化 */
  const { height: matrixTableHeight } = useElementHeight(matrixContentRef, {
    watchSource: isLoading,
    defaultHeight: 600,
  });

  /** 表格空状态管理：根据当前筛选项判断展示「无数据」还是「筛选无结果」 */
  const filters = computed(() => ({
    onlyDeployedEnvs: onlyDeployedEnvs.value ? true : '',
    searchValue: props.searchValue || [],
    selectedDeployStatus: selectedDeployStatus.value === 'all' ? '' : selectedDeployStatus.value,
    selectedImageTag: selectedImageTag.value === IMAGE_TAG_ALL ? '' : selectedImageTag.value,
  }));
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({ filters });

  /** 工作空间 ID：优先使用 props 传入，否则使用 store 中的当前空间 */
  const workspaceID = computed(() => props.space || spaceStore.currentSpace);

  /** 所有可选的镜像 Tag：从所有应用的部署环境中提取并去重 */
  const imageTagOptions = computed(() => getUniqueImageTags(props.apps));

  /** 按镜像 Tag 筛选后的应用列表 */
  const imageTagFilteredApps = computed(() => filterAppsByImageTag(props.apps, selectedImageTag.value));

  /** 按部署状态筛选后的应用列表 */
  const deployStatusFilteredApps = computed(() =>
    filterAppsByDeployStatus(imageTagFilteredApps.value, selectedDeployStatus.value),
  );

  /** 最终展示的应用列表：可选仅展示存在部署环境数据的应用 */
  const filteredApps = computed(() =>
    onlyDeployedEnvs.value ? deployStatusFilteredApps.value.filter(hasDeployedEnvs) : deployStatusFilteredApps.value,
  );

  /** 当前展示应用中存在部署记录的环境名称集合 */
  const deployedEnvNames = computed(
    () =>
      new Set(
        filteredApps.value.flatMap(app =>
          (app.deployedEnvs || []).map(env => env.name).filter((name): name is string => Boolean(name)),
        ),
      ),
  );

  /** 最终展示的环境列表：可选仅展示当前应用中存在部署记录的环境 */
  const filteredEnvs = computed(() =>
    onlyDeployedEnvs.value
      ? envList.value.filter(env => typeof env.name === 'string' && deployedEnvNames.value.has(env.name))
      : envList.value,
  );

  /** 按类型分组的环境列表 */
  const envGroups = computed(() => buildEnvGroups(filteredEnvs.value));

  /** 总页数计算 */
  const pageCount = computed(() => Math.max(1, Math.ceil(filteredApps.value.length / pagination.limit)));

  /** 表格分页配置：包含总数、当前页、每页条数和分页大小选项 */
  const tablePagination = computed(() => ({
    count: filteredApps.value.length,
    current: pagination.current,
    limit: pagination.limit,
    limitList: [10, 20, 50, 100],
  }));

  /**
   * 矩阵列配置：第一列固定为应用名，后续为环境分组列（每组含多个环境子列）
   * 结构：[{ app 列 }, { 环境分组 1: [环境列...] }, { 环境分组 2: [环境列...] }]
   */
  const matrixColumns = computed<MatrixColumn[]>(() => [
    {
      field: 'name',
      fixed: 'left',
      minWidth: 100,
      render: ({ row }) => renderAppCell(row),
      renderHead: renderAppHeader,
      width: 240,
    },
    ...envGroups.value.map(group => ({
      children: group.envs.map(env => ({
        field: `env-${env.name}`,
        label: env.displayName || env.name,
        minWidth: 100,
        render: ({ row }: { row: AppInfoOutputObj }) => renderDeployCell(row, env),
        renderHead: () => renderEnvHeader(env),
      })),
      field: `group-${group.type}`,
      label: `${group.label}（${group.envs.length}）`,
      minWidth: group.envs.length * 100,
      renderHead: () => renderEnvGroupHeader(group.label, group.envs.length),
    })),
  ]);

  /**
   * 构建环境分组：将环境按类型（开发/测试/生产）分组
   * @param envs - 环境列表
   */
  function buildEnvGroups(envs: EnvOutput[]): EnvGroup[] {
    // 按环境类型分组
    const groupMap = envs.reduce<Record<string, EnvOutput[]>>((acc, env) => {
      const type = env.type || 'unknown';
      if (!acc[type]) acc[type] = [];
      acc[type].push(env);
      return acc;
    }, {});

    // 按预定义顺序排序并转换为数组
    return Object.entries(groupMap)
      .sort(([typeA], [typeB]) => {
        const orderA = ENV_TYPE_ORDER[typeA] ?? 99;
        const orderB = ENV_TYPE_ORDER[typeB] ?? 99;
        if (orderA !== orderB) return orderA - orderB;
        return typeA.localeCompare(typeB);
      })
      .map(([type, groupEnvs]) => ({
        type,
        label: ENV_TYPE_LABEL[type] || type,
        envs: [...groupEnvs].sort((a, b) => (a.name || '').localeCompare(b.name || '')),
      }));
  }

  /** 按部署状态筛选应用，任一环境命中即保留 */
  function filterAppsByDeployStatus(apps: AppInfoOutputObj[], status: DeployStatusFilter) {
    if (status === 'all') return apps;
    return apps.filter(app =>
      (app.deployedEnvs || []).some(deployment => getDeployTagTone(deployment.deployStatus) === status),
    );
  }

  /**
   * 按镜像 Tag 筛选应用
   * @param apps - 应用列表
   * @param imageTag - 镜像 Tag
   */
  function filterAppsByImageTag(apps: AppInfoOutputObj[], imageTag: string) {
    if (!imageTag || imageTag === IMAGE_TAG_ALL) return apps;
    return apps.filter(app => (app.deployedEnvs || []).some(env => (env as MatrixDeployInfo).imageTag === imageTag));
  }

  /**
   * 获取应用在指定环境下的部署记录
   * @param app - 应用对象
   * @param envName - 环境名称
   */
  function getCellDeployments(app: AppInfoOutputObj, envName: string) {
    return getMatrixCellDeployments(app, envName);
  }

  /**
   * 获取部署状态的展示文案
   * @param appType - 应用类型
   * @param status - 部署状态
   */
  function getDeployStatusText(appType: string | undefined, status: string | undefined) {
    if (!status) return '';
    return getDeployStatusInfo(appType as AppType, status).text;
  }

  /**
   * 获取部署标签的展示文本：优先展示镜像 Tag，其次展示状态文案
   * @param appType - 应用类型
   * @param status - 部署状态
   * @param imageTag - 镜像 Tag
   */
  function getDeployTagText(appType: string | undefined, status: string | undefined, imageTag?: string) {
    return imageTag || getDeployStatusText(appType, status) || '--';
  }

  /**
   * 获取部署标签的色调类型：根据状态判断标签样式
   * @param status - 部署状态
   */
  function getDeployTagTone(status?: string): MatrixStatusTone {
    if (status === APP_DEPLOY_STATUS.DEPLOYED || status === HELM_DEPLOY_STATUS.DEPLOYED) return 'success';
    if (status && FAILED_STATUSES.has(status)) return 'failed';
    if (status && RUNNING_STATUSES.has(status)) return 'running';
    return 'unknown';
  }

  /**
   * 获取矩阵单元格中的部署记录：筛选出指定环境的部署信息
   * @param app - 应用对象
   * @param envName - 环境名称
   */
  function getMatrixCellDeployments(app: AppInfoOutputObj, envName: string) {
    return ((app.deployedEnvs || []) as MatrixDeployInfo[]).filter(env => env.name === envName);
  }

  /**
   * 提取所有应用的镜像 Tag 并去重排序
   * @param apps - 应用列表
   */
  function getUniqueImageTags(apps: AppInfoOutputObj[]) {
    return Array.from(
      new Set(
        apps.flatMap(app =>
          (app.deployedEnvs || [])
            .map(env => (env as MatrixDeployInfo).imageTag)
            .filter((tag): tag is string => Boolean(tag)),
        ),
      ),
    ).sort((a, b) => a.localeCompare(b));
  }

  /** 重置所有筛选条件：清空镜像 Tag、部署环境过滤和父组件搜索条件 */
  function handleClearFilters() {
    selectedImageTag.value = IMAGE_TAG_ALL;
    onlyDeployedEnvs.value = false;
    selectedDeployStatus.value = 'all';
    emit('clear');
  }

  /**
   * 跳转到应用详情页
   * @param row - 应用对象
   */
  function handleGoAppDetail(row: AppInfoOutputObj) {
    router.push({
      name: 'detail',
      params: {
        name: row.name,
        menuName: 'info',
        type: row?.type || '',
      },
    });
  }

  /** 跳转到应用指定环境的部署管理页 */
  function handleGoDeployment(row: AppInfoOutputObj, env: EnvOutput) {
    if (env.name) {
      deployEnvStore.updateCurrentEnv(env.name);
    }
    router.push({
      name: 'detail',
      params: {
        name: row.name,
        menuName: 'deployment',
        type: row.type || '',
      },
    });
  }

  /**
   * 根据列的 label 识别分组类型并返回对应样式
   * @param { TableCellParams } params - 单元格参数
   */
  function handleHeaderCellClassName({ column }: TableCellParams) {
    const defaultStyle = 'bg-[#FAFBFD] !font-bold';
    if (column?.field) {
      return defaultStyle;
    }
    // 环境分类表头样式：根据标题前缀判断环境类型
    const title = column?.title || '';
    const type = (['开发', '测试', '生产'] as const).find(prefix => title.startsWith(prefix));
    return type
      ? [
          '!font-bold text-center env-group-header-cls',
          ENV_GROUP_HEADER_CLASS[({ 开发: 'development', 测试: 'test', 生产: 'production' } as const)[type]],
        ]
      : defaultStyle;
  }

  /** 加载环境列表数据 */
  async function handleLoadMatrix() {
    if (!workspaceID.value) return;
    isLoading.value = true;
    try {
      const envs = await ApiServerService.ListEnvs({ workspaceID: workspaceID.value }, { validateCode: false });
      envList.value = envs || [];
      clearErrorType();
    } catch {
      envList.value = [];
      setTypeToError();
    } finally {
      isLoading.value = false;
    }
  }

  /**
   * 处理分页大小变化
   * @param limit - 每页条数
   */
  function handlePageLimitChange(limit: number) {
    pagination.limit = limit;
    pagination.current = 1;
  }

  /**
   * 处理页码变化
   * @param current - 当前页码
   */
  function handlePageValueChange(current: number) {
    pagination.current = current;
  }

  /** 应用部署状态筛选 */
  function handleSelectDeployStatus(status: DeployStatusFilter) {
    selectedDeployStatus.value = status;
    statusFilterPopoverRef.value?.hide();
  }

  /**
   * 判断应用是否存在部署环境数据
   * @param app - 应用对象
   */
  function hasDeployedEnvs(app: AppInfoOutputObj): app is AppInfoOutputObj & { deployedEnvs: MatrixDeployInfo[] } {
    return Array.isArray(app.deployedEnvs) && app.deployedEnvs.length > 0;
  }

  // ==================== 渲染函数（矩阵表格列/单元格） ====================

  /**
   * 渲染应用列单元格：类型图标 + 应用名链接
   * @param row - 应用对象
   */
  function renderAppCell(row: AppInfoOutputObj) {
    return h('div', { class: 'flex min-w-0 items-center gap-[8px]' }, [
      h(TypeIcon, {
        classes: 'inline-flex shrink-0',
        showLabel: false,
        type: row.type,
      }),
      h(
        Button,
        {
          class: 'global-view-app-link min-w-0 text-ellipsis whitespace-nowrap',
          text: true,
          theme: 'primary',
          onClick: () => handleGoAppDetail(row),
        },
        () => row.name || '--',
      ),
    ]);
  }

  /** 渲染应用列表头 */
  function renderAppHeader() {
    return h(
      Popover,
      {
        ref: statusFilterPopoverRef,
        arrow: false,
        placement: 'bottom-start',
        theme: 'light global-view-status-filter-popover',
        trigger: 'click',
      },
      // 应用状态表头过滤
      {
        default: () =>
          h('span', { class: 'inline-flex cursor-pointer items-center' }, [
            h('span', t('应用')),
            h('i', {
              class: [
                'relative left-[4px] top-[1px] vxe-filter--btn vxe-table-icon-funnel',
                selectedDeployStatus.value === 'all' ? 'text-[#c0c4cc]' : 'text-[#3a84ff]',
              ],
            }),
          ]),
        content: () =>
          h(
            'div',
            { class: 'w-[150px] py-[4px]' },
            deployStatusFilterOptions.value.map(option =>
              h(
                'div',
                {
                  class: [
                    'flex h-[36px] cursor-pointer items-center gap-[8px] px-[12px] text-[12px] hover:bg-[#f5f7fa]',
                    selectedDeployStatus.value === option.value ? 'bg-[#e1ecff] text-[#3a84ff]' : 'text-[#63656e]',
                  ],
                  onClick: () => handleSelectDeployStatus(option.value),
                },
                [
                  option.icon
                    ? h(ColorIcon, { class: 'shrink-0', icon: option.icon, size: 12 })
                    : h('span', { class: 'inline-block w-[12px] shrink-0' }),
                  h('span', option.label),
                ],
              ),
            ),
          ),
      },
    );
  }

  /**
   * 渲染部署状态单元格：展示镜像 Tag 或状态标签
   * @param row - 应用对象
   * @param env - 环境对象
   */
  function renderDeployCell(row: AppInfoOutputObj, env: EnvOutput) {
    const deployments = getCellDeployments(row, env.name || '');
    if (!deployments.length) return h('span', { class: 'text-[#979ba5]' }, '--');

    // 渲染该环境下所有部署记录（一个应用可能在同一环境多次部署）
    return h(
      'div',
      { class: 'inline-flex max-w-full items-center gap-[6px]' },
      deployments.map((deploy, index) => {
        const tone = getDeployTagTone(deploy.deployStatus);
        const tagText = getDeployTagText(row.type, deploy.deployStatus, deploy.imageTag);
        return h(
          Popover,
          {
            key: `${deploy.name}-${tagText}-${index}`,
            popoverDelay: [100, 0],
            theme: 'popover-dark-translucent',
            trigger: 'hover',
          },
          {
            default: () =>
              h(
                Tag,
                {
                  class:
                    'inline-flex h-[24px] max-w-[136px] cursor-pointer items-center gap-[6px] overflow-hidden text-ellipsis border-0 align-middle',
                  onClick: () => handleGoDeployment(row, env),
                  theme: DEPLOY_TONE_THEME_MAP[tone],
                },
                // 部署中使用 ColorIcon 添加 loading icon
                () =>
                  tone === 'running'
                    ? [
                        h('span', { class: 'inline-flex min-w-0 items-center gap-[4px]' }, [
                          h(ColorIcon, {
                            class: 'shrink-0',
                            icon: 'loading',
                            size: 12,
                          }),
                          h('span', { class: 'min-w-0 overflow-hidden text-ellipsis whitespace-nowrap' }, tagText),
                        ]),
                      ]
                    : [tagText],
              ),
            content: () =>
              h('div', { class: 'leading-[20px]' }, [
                h('div', `镜像 Tag：${tagText}`),
                h('div', { class: 'text-[#c4c6cc]' }, [
                  h('div', `环境：${env.displayName || env.name}`),
                  h(
                    'div',
                    `部署状态：${getDeployStatusText(row.type, deploy.deployStatus) || deploy.deployStatus || '--'}`,
                  ),
                ]),
              ]),
          },
        );
      }),
    );
  }

  /**
   * 渲染环境分组表头：展示分组名 + 环境数量
   * @param label - 分组标签
   * @param count - 环境数量
   */
  function renderEnvGroupHeader(label: string, count: number) {
    return h('div', { class: 'flex h-full items-center justify-center' }, `${label}（${count}）`);
  }

  /**
   * 渲染环境列表头：展示 displayName，hover 时显示完整名称
   * @param env - 环境对象
   */
  function renderEnvHeader(env: EnvOutput) {
    return h(
      Popover,
      {
        popoverDelay: [100, 0],
        theme: 'popover-dark-translucent',
        trigger: 'hover',
      },
      {
        default: () => h('span', env.displayName || env.name),
        content: () => [h('div', env.displayName || env.name), h('div', { class: 'text-[#c4c6cc]' }, env.name)],
      },
    );
  }

  // 筛选条件变化时重置到第一页
  watch([selectedImageTag, onlyDeployedEnvs, selectedDeployStatus], () => {
    pagination.current = 1;
  });

  watch(
    () => props.apps,
    () => {
      pagination.current = 1;
    },
  );

  // 每页条数变化时重置到第一页
  watch(
    () => pagination.limit,
    () => {
      pagination.current = 1;
    },
  );

  // 筛选项导致总页数减少时，修正当前页码
  watch(pageCount, count => {
    if (pagination.current > count) pagination.current = count;
  });

  // 切换工作空间时重新加载数据
  watch(workspaceID, async (newWorkspace, oldWorkspace) => {
    if (!newWorkspace || newWorkspace === oldWorkspace) return;
    // 重置筛选条件
    selectedImageTag.value = IMAGE_TAG_ALL;
    onlyDeployedEnvs.value = false;
    selectedDeployStatus.value = 'all';
    pagination.current = 1;
    await handleLoadMatrix();
  });

  // 组件挂载前加载数据
  onBeforeMount(handleLoadMatrix);
</script>

<style lang="postcss" scoped>
  /**
   * 恢复表格最后一行的纵向边框
   * 不绘制底部横线，避免与分页 footer 的顶部边框叠加
   */
  :deep(.global-view-table .vxe-table--body .vxe-body--row:nth-last-child(1) .vxe-body--column) {
    background-size:
      1px 100%,
      100% 0 !important;
  }

  :deep(.global-view-table .vxe-header--column .vxe-cell) {
    font-size: 12px;
  }

  :deep(.global-view-app-link .bk-button-text) {
    line-height: 20px;
  }

  :deep(.env-group-header-cls) .vxe-cell--title {
    margin: 0 auto;
  }
</style>

<style lang="postcss">
  .global-view-status-filter-popover {
    padding: 0 !important;
  }
</style>
