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
  <div>
    <Skeleton
      :loading="loading"
      theme="gray"
    >
      <template #loading>
        <FlexRow class="w-full mb-[16px]">
          <template #left>
            <Layout.shape :width="110" />
          </template>
          <template #right>
            <Layout.shape :width="400" />
          </template>
        </FlexRow>
        <Layout.table />
      </template>
      <!-- 北极星待重部署提示 -->
      <PolarisPendingRedeployAlert
        :configs="originalList"
        :deploying-env-names="deployingEnvNames"
        :env-list="envList"
        :refreshing="pendingRedeployRefreshing"
        @go-deploy="handleGoDeploy"
        @refresh="handleRefreshPendingRedeploy"
      />
      <FlexRow class="mb-[16px]">
        <template #left>
          <Button
            theme="primary"
            @click="handleCreatePolaris"
          >
            <Plus
              :height="24"
              :width="24"
            />
            {{ $t('添加北极星') }}
          </Button>
        </template>
        <template #right>
          <SearchSelect
            v-model="searchValue"
            class="w-[520px] bg-[#fff] relative z-[100]"
            :data="searchSelectData"
            :placeholder="
              createPlaceholder({
                type: 'searchSelect',
                labels: ['环境类型', '可用环境'],
              })
            "
            unique-select
            value-behavior="need-key"
          >
          </SearchSelect>
        </template>
      </FlexRow>
      <Table
        ref="tableRef"
        v-bkloading="{ loading: filterLoading || loading }"
        class="bg-[#fff] mt-[16px]"
        :data="list"
        :filter-config="{ remote: true }"
        :max-height="maxHeight"
        :pagination="pagination"
        :row-config="{
          isHover: true,
          isCurrent: true,
        }"
        :row-height="40"
        :virtual-y-config="{ enabled: true, gt: 0 }"
        @filter-change="handleFilterChange"
        @page-limit-change="handlePageLimitChange"
        @page-value-change="handlePageValueChange"
      >
        <template #empty>
          <TableException
            :type="curExceptionType"
            @clear="handleClearFilters"
            @refresh="handleFetchList"
          >
          </TableException>
        </template>
        <TableColumn
          field="polarisName"
          :label="$t('服务名')"
          min-width="220"
        >
          <template #default="{ row }">
            <span
              class="inline-flex max-w-full cursor-pointer items-center text-[#3A84FF]"
              @click="handleToPolarisProject(row)"
            >
              <template v-if="row?.polarisName">
                <span
                  v-bk-tooltips="row.polarisName"
                  class="truncate"
                >
                  {{ row.polarisName }}
                </span>
                <Share class="ml-[4px] shrink-0 text-[12px]" />
              </template>
              <template v-else>--</template>
            </span>
          </template>
        </TableColumn>
        <TableColumn
          field="polarisNamespace"
          :filters="polarisNamespaceFilterOptions"
          width="200"
        >
          <template #header>
            <CustomFilter
              field="polarisNamespace"
              :filters="polarisNamespaceFilterOptions"
              :label="$t('环境类型')"
              :table-ref="tableRef"
            />
          </template>
          <template #default="{ row }">
            <span>
              {{ row.polarisNamespace }}
            </span>
          </template>
        </TableColumn>
        <TableColumn
          field="instanceKey"
          :label="$t('实例名称')"
          min-width="160"
        >
        </TableColumn>
        <TableColumn
          field="servicePort"
          :label="$t('服务端口')"
          width="200"
        >
        </TableColumn>
        <TableColumn
          field="scopeEnvNames"
          :filters="scopeEnvNamesFilterOptions"
          width="300"
        >
          <template #header>
            <CustomFilter
              field="scopeEnvNames"
              :filters="scopeEnvNamesFilterOptions"
              :label="$t('可用环境')"
              :table-ref="tableRef"
            />
          </template>
          <template #default="{ row }">
            <!-- 部署提示 -->
            <template v-if="row.scopeEnvNames?.length">
              <MoreTag
                :data="row.scopeEnvNames"
                popover-theme="popover-dark-translucent"
              >
                <template #default="{ item }">
                  <PolarisEnvRedeployTag
                    v-if="item && typeof item === 'string'"
                    :config="row"
                    :display-name="envNameMapping[item] || item"
                    :env-name="item"
                    @go-deploy="handleGoDeploy"
                  />
                </template>
                <template #more-content="{ items }">
                  <PolarisEnvRedeployMoreTip
                    :config="row"
                    :env-list="envList"
                    :env-names="getValidEnvNames(items)"
                    @go-deploy="handleGoDeploy"
                  />
                </template>
              </MoreTag>
            </template>
            <span v-else> -- </span>
          </template>
        </TableColumn>
        <TableColumn
          field="enableHealthCheck"
          :label="$t('健康检查')"
          width="200"
        >
          <template #default="{ row }">
            <div class="flex items-center">
              <ColorIcon
                class="mr-[4px]"
                :icon="row.enableHealthCheck ? 'normal' : 'status-unknown'"
              />
              {{ row.enableHealthCheck ? $t('已开启') : $t('未启用') }}
            </div>
          </template>
        </TableColumn>
        <TableColumn
          fixed="right"
          :label="$t('操作')"
          :show-overflow="false"
          :width="300"
        >
          <template #default="{ row }">
            <span class="inline-flex items-center">
              <Button
                text
                theme="primary"
                @click.stop="handleShowAssociatedEnvs(row)"
              >
                {{ $t('关联环境') }}
              </Button>
              <Tag
                class="ml-[6px]"
                :theme="getAssociatedEnvNames(row)?.length ? 'info' : 'default'"
              >
                {{ getAssociatedEnvNames(row)?.length }}
              </Tag>
            </span>
            <Button
              class="ml-[16px]"
              text
              theme="primary"
              @click.stop="handleShowCertificate(row)"
            >
              {{ $t('查看凭证') }}
            </Button>
            <Button
              class="mx-[16px]"
              text
              theme="primary"
              @click.stop="handleEdit(row)"
            >
              {{ $t('编辑') }}
            </Button>
            <Button
              text
              theme="primary"
              @click.stop="handleDelete(row)"
            >
              {{ $t('删除') }}
            </Button>
          </template>
        </TableColumn>
      </Table>
    </Skeleton>

    <!-- 创建/编辑北极星侧栏 -->
    <EditPolaris
      v-model:is-show="isShowEditPolaris"
      :edit-data="editPolarisData"
      :env-list="envList"
      :redeploy-checking="redeployChecking"
      :redeploy-config="redeployConfig"
      @confirm="handleConfirmEdit"
      @go-deploy="handleGoDeploy"
      @no-redeploy="handleNoRedeploy"
    />

    <!-- 查看凭证侧栏 -->
    <ViewCertificate
      v-model:is-show="isShowViewCertificate"
      :config-name="certificateConfig?.name || ''"
      :polaris-name="certificateConfig?.polarisName"
      :polaris-namespace="certificateConfig?.polarisNamespace"
      :register-mode="certificateConfig?.registerMode"
    />

    <!-- 关联环境及按环境权重侧栏 -->
    <PolarisEnvSideslider
      v-model:is-show="isShowAssociatedEnvs"
      :config="associatedEnvConfig"
      :env-list="envList"
      @updated="handleAssociatedConfigUpdated"
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, h, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, InfoBox, Message, SearchSelect, Tag } from 'bkui-vue';
  import { Plus, Share } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { PolarisConfigOutputObj } from '~/@types/v1/polaris-config';
  import { AppService, PolarisConfigService } from '~/api/modules/v1';
  import { APP_DEPLOY_STATUS } from '~/common/enums/deploy';
  import FlexRow from '~/components/flex-row.vue';
  import MoreTag from '~/components/more-tag.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import SvgIcon from '~/components/svg-icon.vue';
  import useEnvManager, { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';
  import { useErrorHandler } from '~/composables/use-error-handler';
  import useSearchFilter from '~/composables/use-search-filter';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import useDynamicsHeight from '~/composables/use-table-height';
  import { useAppDetail } from '~/stores/app-detail';
  import { useDeployEnvStore } from '~/stores/deploy-env';
  import { useSpaceStore } from '~/stores/space';

  import EditPolaris from './edit-polaris.vue';
  import PolarisEnvRedeployMoreTip from './polaris-env-redeploy-more-tip.vue';
  import PolarisEnvRedeployTag from './polaris-env-redeploy-tag.vue';
  import PolarisEnvSideslider from './polaris-env-sideslider.vue';
  import PolarisPendingRedeployAlert from './polaris-pending-redeploy-alert.vue';
  import { isImmediateRegister } from './redeploy-utils';
  import ViewCertificate from './view-certificate.vue';

  import type { ISearchItem, ISearchValue } from 'bkui-vue/lib/search-select/utils';
  import type { AppDeployedEnvOutputObj } from '~/@types/v1/app';

  interface ApiErrorResponse {
    error: Record<string, unknown>;
    status: number;
  }

  interface ConfirmEditPayload {
    configName?: string;
    mode: 'create' | 'edit';
    needsRedeployTip?: boolean;
  }

  interface DeployedEnv {
    displayName: string;
    envType?: string;
    name: string;
  }

  type PolarisEnvStateWithStatus = NonNullable<PolarisConfigOutputObj['envStates']>[string] & {
    status?: string;
  };

  const POLARIS_DELETE_BLOCKING_STATUSES = ['deployed', 'pendingdelete'];

  const { t } = useI18n();
  const currentRoute = useRoute();
  const router = useRouter();
  const { createPlaceholder } = useSearchPlaceholder();
  const appDetailStore = useAppDetail();
  const deployEnvStore = useDeployEnvStore();
  const spaceStore = useSpaceStore();
  const { handleError } = useErrorHandler();

  const { envList, envNameMapping, getAppEnvList, generateEnvNameMapping } = useEnvManager();
  const { maxHeight } = useDynamicsHeight(106, ['.header-right', '.container-header']);
  const list = ref<PolarisConfigOutputObj[]>([]);
  const originalList = ref<PolarisConfigOutputObj[]>([]); // 保存原始数据

  /** SearchSelect 数据配置 */
  const searchSelectData = ref<ISearchItem[]>([
    {
      name: t('环境类型'),
      id: 'polarisNamespace',
      multiple: true,
      children: ['Development', 'Test', 'Pre-release', 'Production'].map(item => ({
        id: item,
        name: item,
      })),
    },
    {
      name: t('可用环境'),
      id: 'scopeEnvNames',
      multiple: true,
      children: [], // 动态填充
    },
  ]);

  /** SearchSelect 选中值 */
  const searchValue = ref<ISearchValue[]>([]);
  const pagination = ref({
    count: 0,
    limit: 10,
    current: 1,
  });
  const isShowEditPolaris = ref(false);
  const isShowViewCertificate = ref(false);
  const isShowAssociatedEnvs = ref(false);
  const editPolarisData = ref<PolarisConfigOutputObj | undefined>(undefined); // 编辑时的数据
  const associatedEnvConfig = ref<PolarisConfigOutputObj>();
  const certificateConfig = ref<PolarisConfigOutputObj>(); // 查看凭证时的配置
  const loading = ref(true);
  const filterLoading = ref(false);
  const pendingRedeployRefreshing = ref(false);
  const deployingEnvNames = ref<string[]>([]);
  const redeployChecking = ref(false);
  const redeployConfig = ref<PolarisConfigOutputObj | undefined>(undefined);
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });

  /** 使用 useSearchFilter hook 实现 TableColumn filter 与 SearchSelect 联动 */
  const { filterOptions, handleFilterChange } = useSearchFilter(searchSelectData, searchValue, [
    'polarisNamespace',
    'scopeEnvNames',
  ] as const);

  /** 北极星环境类型 筛选配置 */
  const polarisNamespaceFilterOptions = computed(() => filterOptions.value.polarisNamespace);
  /** 可用环境 筛选配置 */
  const scopeEnvNamesFilterOptions = computed(() => filterOptions.value.scopeEnvNames);
  const tableRef = ref();

  /** 统一更新北极星配置列表及其分页、错误状态。 */
  function applyPolarisConfigList(configs: PolarisConfigOutputObj[]) {
    originalList.value = configs;
    list.value = configs;
    pagination.value.count = configs.length;
    clearErrorType();
  }

  /** 删除分支 1：将可用环境 scope 清空，用户重新部署后再进入后续删除流程。 */
  async function clearPolarisScope(row: PolarisConfigOutputObj) {
    await PolarisConfigService.patchAppPolarisConfig({
      appID: appDetailStore.appID,
      configName: row?.name ?? '',
      scopeEnvNames: [],
    });
    Message({
      message: t('操作成功'),
      theme: 'success',
    });
    await handleFetchList();
  }

  async function deletePolarisConfig(row: PolarisConfigOutputObj) {
    try {
      await PolarisConfigService.deleteAppPolarisConfig(
        {
          appID: appDetailStore.appID,
          configName: row?.name ?? '',
        },
        { interceptorErr: false, originalResponse: true, needStatus: true },
      );

      Message({
        message: t('操作成功'),
        theme: 'success',
      });
      await handleFetchList();
    } catch (err: unknown) {
      const errorResponse = err as ApiErrorResponse;
      const { error, status } = errorResponse;
      const errorObj = { ...error, status };
      // 北极星配置管理操作指南文档地址
      const POLARIS_DELETE_FAQ_PATH =
        '/p/4017310012#Q:-%E5%B9%B3%E5%8F%B0%E7%94%9F%E6%88%90%E7%9A%84%E5%8C%97%E6%9E%81%E6%98%9F%E6%9C%8D%E5%8A%A1%E5%88%A0%E9%99%A4%E5%A4%B1%E8%B4%A5%EF%BC%8C%E6%8F%90%E7%A4%BA%E2%80%9Csome-instances-existed-in-service%E2%80%9D';

      // 只有 409 错误才显示自定义 UI，其他错误使用默认处理
      if (status === 409) {
        handleError(errorObj, 409, {
          theme: 'error',
          extCls: 'view-solutions-message',
          delay: 0,
          actions: [
            {
              id: 'details',
              render: () =>
                h(
                  Button,
                  {
                    theme: 'primary',
                    text: true,
                    onClick: () => window.open(window.BK_DOC_URL + POLARIS_DELETE_FAQ_PATH, '_blank'),
                  },
                  {
                    default: () => [h(Share, { class: 'mr-[6px]' }), t('查看解决方案')],
                  },
                ),
            },
            {
              id: 'assistant',
              disabled: true, // 不显示助手
            },
            {
              id: 'fix',
              disabled: true, // 不显示固定
            },
          ],
          message: {
            overview: (error?.message as string) || t('请求异常'),
          },
        });
      } else {
        // 其他错误使用默认处理
        handleError(errorObj, status);
      }
    }
  }

  /** 获取应用当前正在部署的环境；同一环境的任意泳道部署中即视为该环境部署中。 */
  async function fetchDeployingEnvNames() {
    const res = await AppService.getAppDeployStatuses({
      appID: appDetailStore.appID,
    });
    return Array.from(new Set((res || []).filter(isDeployingEnv).map(item => item.name)));
  }

  /** 权重可管理环境 = 当前 scope 与已完成首次部署的环境并集。 */
  function getAssociatedEnvNames(config: PolarisConfigOutputObj) {
    const envNames = Array.from(new Set(config.scopeEnvNames || []));
    Object.entries(config.envStates || {}).forEach(([envName, state]) => {
      if (state?.appliedFields && !envNames.includes(envName)) {
        envNames.push(envName);
      }
    });
    return envNames;
  }

  /** 根据环境名补齐环境展示名和类型，接口无环境详情时回退展示原环境名。 */
  function getEnvInfo(envName: string): DeployedEnv {
    const env = envList.value.find(item => item.name === envName);
    return {
      name: envName,
      displayName: env?.displayName || envName,
      envType: env?.type,
    };
  }

  /** 兼容 envStates.status 的大小写和空格差异，供删除弹窗判断复用。 */
  function getPolarisEnvStatus(status?: string) {
    return status?.trim().toLowerCase() || '';
  }

  /** 删除分支 2：筛出仍需要重新部署摘除实例的环境。 */
  function getRedeployBlockingEnvs(row: PolarisConfigOutputObj): DeployedEnv[] {
    return Object.entries(row.envStates || {}).reduce<DeployedEnv[]>((result, [envName, state]) => {
      const status = getPolarisEnvStatus((state as PolarisEnvStateWithStatus).status);
      if (!status || !POLARIS_DELETE_BLOCKING_STATUSES.includes(status)) return result;

      result.push(getEnvInfo(envName));
      return result;
    }, []);
  }

  /** 删除分支 1：将 scopeEnvNames 转换成弹窗可展示的环境信息。 */
  function getScopeEnvs(row: PolarisConfigOutputObj): DeployedEnv[] {
    return (row.scopeEnvNames || []).map(getEnvInfo);
  }

  /** 过滤出合法环境名，避免搜索筛选项混入非字符串值。 */
  function getValidEnvNames(envNames: unknown[]) {
    return envNames.filter((envName): envName is string => typeof envName === 'string');
  }

  /** 使用权重接口返回的完整配置同步列表与当前侧栏，避免额外刷新请求。 */
  function handleAssociatedConfigUpdated(config: PolarisConfigOutputObj) {
    const replaceConfig = (item: PolarisConfigOutputObj) => (item.name === config.name ? config : item);
    originalList.value = originalList.value.map(replaceConfig);
    list.value = list.value.map(replaceConfig);
    associatedEnvConfig.value = config;
  }

  /** 清除筛选条件 */
  function handleClearFilters() {
    // 清除筛选效果
    tableRef.value.getVxeTableInstance().clearFilter('polarisNamespace');
    tableRef.value.getVxeTableInstance().clearFilter('scopeEnvNames');
    // 清除 SearchSelect 选中值
    searchValue.value = [];
  }

  async function handleConfirmEdit(payload?: ConfirmEditPayload) {
    if (payload?.mode !== 'edit' || !payload.needsRedeployTip) {
      await handleFetchList();
      return;
    }

    redeployChecking.value = true;
    redeployConfig.value = undefined;
    try {
      // 刷新列表
      const latestList = await handleFetchList({ throwOnError: true });
      const latestConfig = latestList.find(item => item.name === payload.configName);
      if (!latestConfig) {
        Message({
          message: t('配置保存成功'),
          theme: 'success',
        });
        isShowEditPolaris.value = false;
        return;
      }

      redeployConfig.value = latestConfig;
    } finally {
      redeployChecking.value = false;
    }
  }

  function handleCreatePolaris() {
    editPolarisData.value = undefined; // 清空编辑数据，表示创建模式
    redeployChecking.value = false;
    redeployConfig.value = undefined;
    isShowEditPolaris.value = true;
  }

  /** 删除入口：立即生效配置由后端同步清理，其余配置沿用部署态前置检查。 */
  function handleDelete(row: PolarisConfigOutputObj) {
    if (isImmediateRegister(row)) {
      showDeleteConfirm(row);
      return;
    }

    const scopeEnvs = getScopeEnvs(row);
    if (scopeEnvs.length > 0) {
      showClearScopeInfoBox(row, scopeEnvs);
      return;
    }

    const redeployBlockingEnvs = getRedeployBlockingEnvs(row);
    if (redeployBlockingEnvs.length > 0) {
      showRedeployBlockingInfoBox(row, redeployBlockingEnvs);
      return;
    }

    showDeleteConfirm(row);
  }

  function handleEdit(row: PolarisConfigOutputObj) {
    editPolarisData.value = row; // 设置编辑数据
    redeployChecking.value = false;
    redeployConfig.value = undefined;
    isShowEditPolaris.value = true;
  }

  /** 初始化部署状态，不影响页面主体的加载和错误状态。 */
  async function handleFetchDeployingEnvNames() {
    try {
      deployingEnvNames.value = await fetchDeployingEnvNames();
    } catch (error) {
      console.error(error);
    }
  }

  // 初始化列表数据
  async function handleFetchList(options?: { throwOnError?: boolean }) {
    try {
      loading.value = true;
      const res = await PolarisConfigService.listAppPolarisConfigs({
        appID: appDetailStore.appID,
      });

      applyPolarisConfigList(res || []);
      return res || [];
    } catch (error) {
      console.error(error);
      setTypeToError();
      if (options?.throwOnError) {
        throw error;
      }
      return [];
    } finally {
      loading.value = false;
    }
  }

  /** 前端筛选逻辑 */
  function handleFilterList() {
    filterLoading.value = true;
    setTimeout(() => {
      pagination.value.current = 1;

      // 处理过滤逻辑
      let filteredList = [...originalList.value];

      // 从 searchValue 中提取筛选条件
      const polarisNamespaceFilter = searchValue.value.find(item => item.id === 'polarisNamespace');
      const scopeEnvNamesFilter = searchValue.value.find(item => item.id === 'scopeEnvNames');

      // 筛选 polarisNamespace
      if (polarisNamespaceFilter?.values && polarisNamespaceFilter.values.length > 0) {
        const selectedValues = polarisNamespaceFilter.values.map(v => v.id);
        filteredList = filteredList.filter(
          item => item.polarisNamespace && selectedValues.includes(item.polarisNamespace),
        );
      }

      // 筛选 scopeEnvNames (string[])
      if (scopeEnvNamesFilter?.values && scopeEnvNamesFilter.values.length > 0) {
        const selectedValues = scopeEnvNamesFilter.values.map(v => v.id);
        filteredList = filteredList.filter(item => item?.scopeEnvNames?.some(env => selectedValues.includes(env)));
      }

      list.value = filteredList;
      pagination.value.count = filteredList.length;
      filterLoading.value = false;
    }, 200);
  }

  function handleGoDeploy(envName?: string) {
    const spaceParam = currentRoute.params.space;
    const space = (Array.isArray(spaceParam) ? spaceParam[0] : spaceParam) || spaceStore.currentSpace;
    if (envName) {
      deployEnvStore.updateCurrentEnv(envName);
      // 部署页会按 `${space}:${appName}` 恢复环境模式；指定环境跳转时同步写成单环境，
      // 避免新标签页先读取历史 multi 缓存，再异步切回单环境导致闪烁。
      const scopeKey = space && appDetailStore.app ? `${space}:${appDetailStore.app}` : '';
      deployEnvStore.updateAppEnvSelection(scopeKey, {
        mode: 'single',
        selectedEnvs: [envName],
      });
    }
    isShowEditPolaris.value = false;
    const route = router.resolve({
      name: 'detail',
      params: {
        space,
        name: appDetailStore.app,
        menuName: 'deployment',
        type: appDetailStore.appType,
      },
      query: {
        activeTab: 'instance',
        // envName 是部署页的一次性环境定位参数，消费后会从 URL 中移除
        ...(envName ? { envName } : {}),
      },
    });
    window.open(route.href, '_blank');
  }

  /** 初始化可用环境筛选配置 */
  function handleInitScopeEnvNamesFilterOptions() {
    // 更新 searchSelectData 中的 scopeEnvNames children
    const scopeEnvItem = searchSelectData.value.find(item => item.id === 'scopeEnvNames');
    if (scopeEnvItem) {
      scopeEnvItem.children = envList.value.map(env => ({
        id: env.name!,
        name: env.displayName!,
      }));
    }
  }

  function handleNoRedeploy() {
    Message({
      message: t('配置保存成功'),
      theme: 'success',
    });
    isShowEditPolaris.value = false;
  }

  function handlePageLimitChange(val: number) {
    pagination.value.limit = val;
  }

  function handlePageValueChange(val: number) {
    pagination.value.current = val;
  }

  /** 同步刷新北极星配置和部署状态；任一请求失败时保留刷新前的数据。 */
  async function handleRefreshPendingRedeploy() {
    if (pendingRedeployRefreshing.value) return;

    pendingRedeployRefreshing.value = true;
    try {
      const [configs, latestDeployingEnvNames] = await Promise.all([
        PolarisConfigService.listAppPolarisConfigs({
          appID: appDetailStore.appID,
        }),
        fetchDeployingEnvNames(),
      ]);
      const latestConfigs = configs || [];
      applyPolarisConfigList(latestConfigs);
      deployingEnvNames.value = latestDeployingEnvNames;
      if (searchValue.value.length) {
        handleFilterList();
      }
    } catch (error) {
      console.error(error);
    } finally {
      pendingRedeployRefreshing.value = false;
    }
  }

  /** 打开当前北极星配置的关联环境侧栏。 */
  function handleShowAssociatedEnvs(row: PolarisConfigOutputObj) {
    associatedEnvConfig.value = row;
    isShowAssociatedEnvs.value = true;
  }

  function handleShowCertificate(row: PolarisConfigOutputObj) {
    certificateConfig.value = row;
    isShowViewCertificate.value = true;
  }

  function handleToPolarisProject(row: PolarisConfigOutputObj) {
    const { polarisNamespace, polarisName } = row;
    const url = `${import.meta.env.BK_POLARIS_URL}/#/services/info/instance/${polarisNamespace}/${polarisName}`;
    window.open(url);
  }

  /** 部署状态为 deploying 且包含合法环境名时，收窄为可直接读取 name 的类型。 */
  function isDeployingEnv(item: AppDeployedEnvOutputObj): item is AppDeployedEnvOutputObj & {
    name: string;
  } {
    return !!item.name && item.deployStatus === APP_DEPLOY_STATUS.DEPLOYING;
  }

  /** 删除分支 1 弹窗内容：提示先置空可用环境，且不展示去部署操作。 */
  function renderClearScopeInfoBoxContent(row: PolarisConfigOutputObj, scopeEnvs: DeployedEnv[]) {
    return h('div', { class: 'text-[14px] leading-[22px] text-[#4D4F56]' }, [
      h('div', { class: 'mb-[16px] bg-[#F5F7FA] p-[12px]' }, [
        h('div', { class: 'mb-[12px]' }, [
          t('北极星 {0} 已关联以下环境，需要按以下顺序完成后才能删除：', [row.polarisName || '--']),
        ]),
        h('div', { class: 'mb-[8px] flex items-center' }, [
          h(
            'span',
            { class: 'mr-[8px] flex h-[20px] w-[20px] items-center justify-center rounded-full bg-[#EAEBF0]' },
            '1',
          ),
          h('span', t('可用环境置空')),
        ]),
        h('div', { class: 'flex items-center' }, [
          h(
            'span',
            { class: 'mr-[8px] flex h-[20px] w-[20px] items-center justify-center rounded-full bg-[#EAEBF0]' },
            '2',
          ),
          h('span', t('重新部署已关联的环境')),
        ]),
      ]),
      renderRedeployEnvTable(scopeEnvs, { showDeployAction: false }),
    ]);
  }

  /** 删除确认说明：立即生效配置由后端同步摘除，其他配置沿用未部署场景说明。 */
  function renderDeleteInfoBoxContent(row: PolarisConfigOutputObj) {
    const serviceName = h('div', { class: 'mb-[16px] text-[14px] text-[#313238]' }, [
      `${t('北极星服务名')}：${row.polarisName || '--'}`,
    ]);

    const description = isImmediateRegister(row)
      ? t('该配置为立即生效，删除后将同步从已关联环境中摘除北极星实例并移除配置。')
      : t('该配置对应环境尚未部署，北极星侧无已注册实例，删除后将直接移除。');

    return h('div', [
      serviceName,
      h('div', { class: 'bg-[#F5F7FA] px-[16px] py-[12px] text-[14px] leading-[22px]' }, [description]),
    ]);
  }

  function renderPolarisDeleteWarningTitle(title: string) {
    return h('div', { class: 'polaris-delete-warning-title' }, [
      h(SvgIcon, {
        class: 'polaris-delete-warning-svg',
        height: 42,
        icon: 'bkms-icon-tishi',
        width: 42,
      }),
      h('div', { class: 'polaris-delete-warning-title-text' }, title),
    ]);
  }

  /** 删除分支 2 弹窗内容：提示重新部署已关联环境后才能删除配置。 */
  function renderRedeployBlockingInfoBoxContent(row: PolarisConfigOutputObj, redeployBlockingEnvs: DeployedEnv[]) {
    return h('div', { class: 'text-[14px] leading-[22px] text-[#4D4F56]' }, [
      h('div', { class: 'mb-[24px]' }, [
        t('北极星 '),
        h('span', { class: 'font-bold' }, row.polarisName || '--'),
        t(' 的实例仍注册在以下环境中，请先重新部署这些环境，实例摘除后才可删除该配置。'),
      ]),
      renderRedeployEnvTable(redeployBlockingEnvs),
    ]);
  }

  /** 渲染删除弹窗中的环境列表，可按弹窗类型控制是否展示去部署入口。 */
  function renderRedeployEnvTable(envs: DeployedEnv[], options: { showDeployAction?: boolean } = {}) {
    const { showDeployAction = true } = options;
    return h('div', { class: 'overflow-hidden rounded-[2px] border border-[#EAEBF0]' }, [
      h('div', { class: 'flex h-[32px] items-center bg-[#EAEBF0] px-[16px] text-[14px] text-[#313238]' }, [
        t('需要重新部署的 {0} 个环境', [envs.length]),
      ]),
      ...envs.map((env, index) =>
        h(
          'div',
          {
            class: ['flex h-[36px] items-center px-[16px] text-[12px]', index % 2 === 1 ? 'bg-[#FAFBFD]' : 'bg-[#fff]'],
          },
          [
            h('span', { class: 'truncate' }, env.displayName),
            env.envType && envTypeMap[env.envType]
              ? h(
                  Tag,
                  {
                    class: ['ml-[8px] shrink-0', envTypeTagClassMap[env.envType]],
                    size: 'small',
                  },
                  { default: () => envTypeMap[env.envType!].name },
                )
              : null,
            showDeployAction
              ? h(
                  Button,
                  {
                    class: 'ml-auto shrink-0',
                    text: true,
                    theme: 'primary',
                    onClick: () => handleGoDeploy(env.name),
                  },
                  {
                    default: () => [t('去部署'), h(Share, { class: 'ml-[4px]' })],
                  },
                )
              : null,
          ].filter(Boolean),
        ),
      ),
    ]);
  }

  /** 打开删除分支 1 弹窗，确认后调用 PATCH 接口清空 scope。 */
  function showClearScopeInfoBox(row: PolarisConfigOutputObj, scopeEnvs: DeployedEnv[]) {
    InfoBox({
      width: 480,
      extCls: 'polaris-delete-warning-infobox',
      title: renderPolarisDeleteWarningTitle(t('删除前需要先完成以下操作')),
      content: renderClearScopeInfoBoxContent(row, scopeEnvs),
      headerAlign: 'center',
      footerAlign: 'center',
      contentAlign: 'left',
      confirmText: t('置空可用环境'),
      cancelText: t('取消'),
      onConfirm: () => clearPolarisScope(row),
    });
  }

  // 确认删除北极星配置。
  function showDeleteConfirm(row: PolarisConfigOutputObj) {
    InfoBox({
      title: t('确认删除该北极星配置？'),
      content: renderDeleteInfoBoxContent(row),
      headerAlign: 'center',
      footerAlign: 'center',
      contentAlign: 'left',
      confirmButtonTheme: 'danger',
      confirmText: t('删除'),
      cancelText: t('取消'),
      onConfirm: () => deletePolarisConfig(row),
    });
  }

  /** 打开删除分支 2 弹窗，仅提示重新部署，不直接删除配置。 */
  function showRedeployBlockingInfoBox(row: PolarisConfigOutputObj, redeployBlockingEnvs: DeployedEnv[]) {
    InfoBox({
      width: 480,
      extCls: 'polaris-delete-warning-infobox',
      title: renderPolarisDeleteWarningTitle(t('删除前需要重新部署已关联环境')),
      content: renderRedeployBlockingInfoBoxContent(row, redeployBlockingEnvs),
      headerAlign: 'center',
      footerAlign: 'center',
      contentAlign: 'left',
      confirmText: t('我知道了'),
    });
  }

  // 监听 searchValue 变化，触发前端筛选
  watch(
    searchValue,
    () => {
      handleFilterList();
    },
    { deep: true },
  );

  watch(
    () => appDetailStore.appID,
    async val => {
      originalList.value = [];
      list.value = [];
      deployingEnvNames.value = [];
      pagination.value.count = 0;
      pagination.value.current = 1;

      if (val) {
        await Promise.all([handleFetchList(), handleFetchDeployingEnvNames(), getAppEnvList(val)]);
        generateEnvNameMapping();
        handleInitScopeEnvNamesFilterOptions();
      }
    },
    { immediate: true },
  );
</script>

<style lang="postcss">
  .view-solutions-message {
    .overview {
      align-items: center !important;
      .describe {
        max-width: 600px !important;
        width: 600px !important;
      }
    }
  }

  .polaris-delete-warning-infobox {
    .bk-infobox-title {
      margin-top: 0;
    }

    .polaris-delete-warning-title {
      display: flex;
      flex-direction: column;
      align-items: center;
    }

    .polaris-delete-warning-title-text {
      margin-top: 22px;
    }
  }
</style>
