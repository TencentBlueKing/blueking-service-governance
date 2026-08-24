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
    quick-close
    render-directive="if"
    :width="960"
    @closed="handleClosed"
  >
    <template #header>
      <DividerHeader>
        <template #title>
          <span class="text-[16px]">{{ $t('应用关联的特性环境') }}</span>
        </template>
        <span>{{ $t('共 {0} 个', [list.length]) }}</span>
      </DividerHeader>
    </template>

    <div class="h-full p-[24px] flex flex-col gap-[16px]">
      <Alert theme="info">
        <div class="text-[12px] leading-[20px]">
          <div>{{ $t('特性环境只属于当前应用，其他应用无法使用。来源环境仅为创建时快照，之后互不影响。') }}</div>
          <div>{{ $t('环境创建后不会自动部署，需要单独执行部署；销毁前需先移除部署。') }}</div>
        </div>
      </Alert>

      <div class="flex items-center gap-[12px]">
        <Button
          theme="primary"
          @click="emit('create')"
        >
          <Plus class="text-[18px]" />
          {{ $t('新建') }}
        </Button>
        <Input
          v-model.trim="searchKeyword"
          class="ml-auto w-[320px]"
          clearable
          :placeholder="$t('搜索环境展示名称')"
        >
          <template #suffix>
            <Search class="text-[16px] text-[#979BA5] mr-[6px] mt-[2px] hover:text-[#3A84FF]" />
          </template>
        </Input>
      </div>

      <Table
        ref="tableRef"
        v-bkloading="{ loading }"
        class="min-h-0 flex-1"
        :data="tableData"
        :filter-config="{ remote: true }"
        @filter-change="handleFilterChange"
      >
        <template #empty>
          <TableException
            v-show="!loading"
            :type="curExceptionType"
            @clear="handleClearFilters"
            @refresh="emit('refresh')"
          />
        </template>

        <TableColumn
          field="displayName"
          fixed="left"
          :label="$t('环境展示名称')"
          min-width="140"
          show-overflow-tooltip
        >
          <template #default="{ row }: { row: FeatureEnvRow }">
            {{ row.displayName }}
          </template>
        </TableColumn>

        <TableColumn
          field="deployStatus"
          :filters="deployStatusFilterOptions"
          min-width="110"
        >
          <template #header>
            <CustomFilter
              field="deployStatus"
              :filters="deployStatusFilterOptions"
              :label="$t('部署状态')"
              :table-ref="tableRef"
            />
          </template>
          <template #default="{ row }: { row: FeatureEnvRow }">
            <div class="flex items-center">
              <ColorIcon
                class="mr-[6px]"
                :icon="row.deployStatusIcon"
                :size="12"
              />
              <span>{{ row.deployStatusText }}</span>
            </div>
          </template>
        </TableColumn>

        <TableColumn
          field="sourceEnvText"
          :label="$t('来源环境')"
          min-width="140"
          show-overflow-tooltip
        >
          <template #default="{ row }: { row: FeatureEnvRow }">
            <span>{{ row.sourceEnvText }}</span>
            <!-- 特性环境使用创建时的来源环境分类快照。 -->
            <Tag
              v-if="row.envTypeText !== '--'"
              :class="['ml-[4px]', envTypeTagClassMap[row.envType]]"
              size="small"
            >
              {{ row.envTypeText }}
            </Tag>
          </template>
        </TableColumn>

        <TableColumn
          field="namespace"
          :label="$t('命名空间')"
          min-width="180"
          show-overflow-tooltip
        />

        <TableColumn
          field="createdAtText"
          :label="$t('创建时间')"
          min-width="170"
          show-overflow-tooltip
          sortable
        />

        <TableColumn
          fixed="right"
          :label="$t('操作')"
          min-width="160"
        >
          <template #default="{ row }: { row: FeatureEnvRow }">
            <div class="flex items-center gap-[12px]">
              <Button
                v-if="row.canDeploy"
                text
                theme="primary"
                @click="handleDeploy(row)"
              >
                {{ $t('部署') }}
              </Button>
              <Button
                text
                theme="primary"
                @click="handleGoConfig(row)"
              >
                {{ $t('去配置') }}
              </Button>
              <Popover
                :disabled="!row.destroyDisabled"
                ext-cls="feature-env-destroy-popover"
                placement="top"
                theme="dark"
                trigger="hover"
              >
                <span class="inline-flex">
                  <Button
                    :disabled="row.destroyDisabled"
                    text
                    theme="primary"
                    @click="handleShowDestroyDialog(row)"
                  >
                    {{ $t('销毁') }}
                  </Button>
                </span>
                <template #content>
                  <div
                    v-if="row.destroyDisabledReason === 'deployed'"
                    class="text-[12px] leading-[20px]"
                  >
                    <div class="font-bold text-[14px] leading-[22px]">{{ $t('无法销毁特性环境') }}</div>
                    <div>• {{ $t('该特性环境有部署实例，请先移除部署后再销毁环境') }}</div>
                    <Button
                      class="mt-[6px] !text-[#3A84FF]"
                      :loading="preparingRemoveDeployRowID === row.id"
                      text
                      @click.stop.prevent="handleShowRemoveDeployDialog(row)"
                    >
                      • {{ $t('立即移除部署') }}
                    </Button>
                  </div>
                  <span
                    v-else
                    class="whitespace-nowrap text-[12px] leading-[20px]"
                  >
                    {{ $t('部署中或卸载中的特性环境暂不支持销毁') }}
                  </span>
                </template>
              </Popover>
            </div>
          </template>
        </TableColumn>
      </Table>
    </div>
  </Sideslider>

  <!-- 销毁环境是不可逆操作，要求输入环境内部名称进行二次确认。 -->
  <Dialog
    v-model:is-show="isShowDestroyDialog"
    render-directive="if"
    :width="580"
    @closed="resetDestroyDialog"
  >
    <template #header>
      <div class="flex flex-col items-center">
        <SvgIcon
          :height="42"
          icon="bkms-icon-tishi"
          :width="42"
        />
        <span class="text-[#313238] text-[20px] leading-[32px] text-center mt-[18px]">
          {{ $t('确认销毁特性环境') }} ?
        </span>
      </div>
    </template>
    <div class="mb-[16px] px-[16px] py-[12px] bg-[#F5F6FA] rounded-[2px]">
      {{ $t('将永久删除特性环境及其命名空间，并清理应用配置 / 北极星 / 组件中的引用。') }}
    </div>
    <i18n-t keypath="该操作不可恢复，请输入特性环境命名空间：{0} 进行确认">
      <span
        class="font-bold text-[#EA3636] px-[3px] cursor-pointer hover:bg-[#FFEBEB]"
        @click="copyText(destroyTargetRow?.name || '')"
      >
        {{ destroyTargetRow?.name || '--' }}
      </span>
    </i18n-t>
    <Form
      ref="destroyFormRef"
      class="mt-[8px]"
      form-type="vertical"
      :model="destroyFormData"
      :rules="destroyRules"
    >
      <Form.FormItem
        property="confirmName"
        required
      >
        <Input
          v-model.trim="destroyFormData.confirmName"
          clearable
          :placeholder="$t('请输入{0}', [$t('环境名称')])"
        />
      </Form.FormItem>
    </Form>
    <template #footer>
      <div class="flex justify-center">
        <Button
          class="mr-[8px]"
          :disabled="!isDestroyConfirmValid"
          :loading="destroyLoading"
          theme="danger"
          @click="handleDestroyFeatureEnv"
        >
          {{ $t('销毁') }}
        </Button>
        <Button
          :loading="destroyLoading"
          @click="handleCloseDestroyDialog"
        >
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Dialog>

  <!-- 已部署的特性环境需要先移除部署，再销毁环境。 -->
  <Dialog
    v-model:is-show="isShowRemoveDeployDialog"
    :width="480"
    @closed="handleRemoveDeployDialogClosed"
  >
    <template #header>
      <div class="flex flex-col items-center">
        <SvgIcon
          :height="42"
          icon="bkms-icon-tishi"
          :width="42"
        />
        <span class="text-[#313238] text-[20px] leading-[32px] text-center mt-[18px]">
          {{ $t('确认移除部署') }} ?
        </span>
      </div>
    </template>
    <div class="text-[14px] leading-[24px] mb-[20px]">
      <div>
        <span>{{ $t('环境') }}：</span>
        <span class="font-bold">{{ removeDeployTargetRow?.name || '--' }}</span>
        <Tag class="ml-[8px] text-[#3A9EAA] bg-[#E2F5F7]">
          {{ $t('特性') }}
        </Tag>
      </div>
      <div>
        <span>{{ $t('影响实例') }}：</span>
        <span class="font-bold">{{ removeDeployInstanceCount }}</span>
        <span>&nbsp;{{ $t('个') }}</span>
      </div>
    </div>
    <div class="mb-[16px] px-[16px] py-[12px] bg-[#F5F6FA] rounded-[2px] text-[14px] leading-[24px]">
      {{ $t('将移除当前环境下的全部运行实例，服务将不可访问。特性环境本身会保留，之后仍可再次「部署」到该环境。') }}
    </div>
    <template #footer>
      <div class="flex justify-center">
        <Button
          class="mr-[8px]"
          :loading="removeDeployLoading"
          theme="danger"
          @click="handleRemoveDeploy"
        >
          {{ $t('移除部署') }}
        </Button>
        <Button
          :loading="removeDeployLoading"
          @click="isShowRemoveDeployDialog = false"
        >
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
  import { computed, ref, shallowRef, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Button, Dialog, Form, Input, Message, Popover, Sideslider, Tag } from 'bkui-vue';
  import { Plus, Search } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { FeatureEnvDeployStatusOutput, FeatureEnvOutput } from '~/@types/v1/env';
  import { EnvService, InstanceService } from '~/api/modules/v1';
  import { APP_DEPLOY_STATUS } from '~/common/enums/deploy';
  import { copyText } from '~/common/util';
  import ColorIcon from '~/components/color-icon.vue';
  import CustomFilter from '~/components/custom-filter.vue';
  import DividerHeader from '~/components/divider-header.vue';
  import TableException from '~/components/table-exception.vue';
  import { useDeployStatusMap } from '~/composables/use-deploy-status';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';
  import { IInputKey, ISelectKey, useTableSearchInput, useTableSearchSelect } from '~/composables/use-search';
  import useTableEmpty from '~/composables/use-table-empty';
  import useTime from '~/composables/use-time';
  import { useAppDetail } from '~/stores/app-detail';

  import { type DeployableAppType, useDeployAPIs } from './use-deploy';

  import type { AppType } from '~/composables/app-type';

  const DEFAULT_DEPLOY_ICON = 'status-unknown';

  type DestroyDisabledReason = '' | 'deployed';

  interface FeatureEnvRow {
    canDeploy: boolean;
    createdAtText: string;
    deployStatus: string;
    deployStatusIcon: string;
    deployStatusText: string;
    destroyDisabled: boolean;
    destroyDisabledReason: DestroyDisabledReason;
    displayName: string;
    envType: string;
    envTypeText: string;
    id: string;
    name: string;
    namespace: string;
    sourceEnvName: string;
    sourceEnvText: string;
  }

  type FeatureEnvTableRef = {
    getVxeTableInstance?: () => {
      clearFilter?: () => void;
      setFilter?: (field: string, filters: FilterOption[], checked?: boolean) => void;
    };
  };

  type FilterOption = {
    checked?: boolean;
    label: number | string;
    value: number | string;
  };

  const props = withDefaults(
    defineProps<{
      error?: boolean;
      list?: FeatureEnvOutput[];
      loading?: boolean;
    }>(),
    {
      error: false,
      list: () => [],
      loading: false,
    },
  );

  const emit = defineEmits<{
    create: [];
    deleted: [payload: { envName: string; sourceEnvName?: string }];
    deploy: [env: FeatureEnvOutput];
    'deploy-removed': [];
    refresh: [];
  }>();

  const isShow = defineModel<boolean>('isShow');
  const { t } = useI18n();
  const router = useRouter();
  const route = useRoute();
  const appDetailStore = useAppDetail();
  const { getDeployStatusInfo } = useDeployStatusMap();
  const { formatDateString } = useTime();

  const tableRef = ref<FeatureEnvTableRef>({});

  // 销毁弹窗状态：要求输入环境内部名称，避免误删不可恢复的特性环境。
  const isShowDestroyDialog = ref(false);
  const destroyLoading = ref(false);
  const destroyTargetRow = ref<FeatureEnvRow | null>(null);
  const destroyFormRef = ref<InstanceType<typeof Form> | null>(null);
  const destroyFormData = ref({
    confirmName: '',
  });

  // 移除部署弹窗状态：已部署的特性环境必须先清理实例，才能继续销毁环境。
  const isShowRemoveDeployDialog = ref(false);
  const removeDeployLoading = ref(false);
  const preparingRemoveDeployRowID = ref('');
  const removeDeployTargetRow = ref<FeatureEnvRow | null>(null);
  const removeDeployInstanceCount = ref(0);
  const isDestroyConfirmValid = computed(
    () => !!destroyTargetRow.value?.name && destroyFormData.value.confirmName === destroyTargetRow.value.name,
  );
  const destroyRules = computed(() => ({
    confirmName: [
      {
        trigger: 'blur',
        required: true,
        message: t('请输入{0}', [t('环境名称')]),
      },
      {
        trigger: 'blur',
        validator: (value: string) => value === destroyTargetRow.value?.name,
        message: t('{0}填写错误', [t('环境名称')]),
      },
    ],
  }));

  // 统一生成表格行，模板只消费展示字段和操作状态。
  const normalizedList = computed<FeatureEnvRow[]>(() =>
    props.list.map(env => {
      // listFeatureEnvs(with_deploy_status=true) 返回的 deployStatuses 是侧栏唯一状态源。
      const deployStatuses = env.deployStatuses || [];
      const deployStatus = deployStatuses[0]?.deployStatus || '';
      const deployStatusInfo = getFeatureEnvDeployStatusInfo(deployStatus);
      const envType = env.type || '';
      const envTypeInfo = envTypeMap[envType];
      const destroyDisabledReason = getDestroyDisabledReason(deployStatuses);
      // 未配置集群资源的环境无法部署；仅就绪且没有部署记录的环境展示首次部署入口。
      const canDeploy = !!env.name && env.status !== 'NotReady' && deployStatuses.length === 0;

      return {
        canDeploy,
        id: env.id || '',
        name: env.name || '',
        displayName: env.displayName || env.name || '--',
        deployStatus,
        deployStatusIcon: deployStatusInfo.icon,
        deployStatusText: deployStatusInfo.text || t('未知'),
        envType,
        envTypeText: envTypeInfo?.name || '--',
        sourceEnvText: env.sourceEnv?.displayName || env.sourceEnv?.name || '--',
        sourceEnvName: env.sourceEnv?.name || '',
        namespace: env.cluster?.namespace || '--',
        createdAtText: env.createdAt ? formatDateString(env.createdAt) : '--',
        destroyDisabled: !env.id || !!destroyDisabledReason,
        destroyDisabledReason,
      };
    }),
  );

  const searchKeys = ref<IInputKey[]>([{ id: 'displayName', field: 'displayName', fuzzy: true }]);
  const { searchValue: searchKeyword, tableDataMatchSearch: tableDataMatchKeyword } = useTableSearchInput(
    normalizedList,
    searchKeys,
  );

  const deployStatusSearchData = shallowRef<ISelectKey<FeatureEnvRow>[]>([
    {
      id: 'deployStatus',
      name: t('部署状态'),
      field: 'deployStatus',
      multiple: true,
      children: [],
      // 行内 deployStatus 为空时等价于未部署，筛选值使用枚举值保持和表头过滤项一致。
      handleFilter: (row, values) => {
        const deployStatus = row.deployStatus || APP_DEPLOY_STATUS.UNINSTALLED;
        return values.some(value => value.id === deployStatus);
      },
    },
  ]);

  const {
    filterChangeEvent: handleFilterChange,
    filterOptions,
    searchValue: filterSearchValue,
    tableDataMatchSearch: tableData,
  } = useTableSearchSelect(tableDataMatchKeyword, deployStatusSearchData, {
    tableRef,
  });
  const deployStatusFilterOptions = computed(() => filterOptions.value.deployStatus || []);

  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: [searchKeyword, filterSearchValue],
  });

  function buildDeployStatusOptions(rows: FeatureEnvRow[]) {
    const optionMap = new Map<string, string>();
    rows.forEach(row => {
      optionMap.set(row.deployStatus || APP_DEPLOY_STATUS.UNINSTALLED, row.deployStatusText);
    });
    return Array.from(optionMap, ([id, name]) => ({
      id,
      name,
    }));
  }

  function getDestroyDisabledReason(deployStatuses: FeatureEnvDeployStatusOutput[]): DestroyDisabledReason {
    // 只要后端返回任意部署记录，就必须先移除部署，再允许销毁环境。
    if (deployStatuses.length > 0) return 'deployed';
    return '';
  }

  function getFeatureEnvDeployStatusInfo(deployStatus: string) {
    // 未部署状态在特性环境侧栏中展示为“待部署”，和应用部署页的空态保持一致。
    const isPending = !deployStatus || deployStatus === APP_DEPLOY_STATUS.UNINSTALLED;
    const info = deployStatus
      ? getDeployStatusInfo((appDetailStore.appType || undefined) as AppType | undefined, deployStatus)
      : null;

    return {
      icon: info?.icon || DEFAULT_DEPLOY_ICON,
      text: isPending ? t('待部署') : info?.text || t('未知'),
    };
  }

  function handleClearFilters() {
    searchKeyword.value = '';
    filterSearchValue.value = [];
    tableRef.value?.getVxeTableInstance?.()?.clearFilter?.();
  }

  function handleClosed() {
    handleClearFilters();
  }

  function handleCloseDestroyDialog() {
    isShowDestroyDialog.value = false;
    resetDestroyDialog();
  }

  function handleDeploy(row: FeatureEnvRow) {
    const env = props.list.find(item => item.name === row.name);
    if (env) emit('deploy', env);
  }

  async function handleDestroyFeatureEnv() {
    const targetRow = destroyTargetRow.value;
    const valid = await destroyFormRef.value
      ?.validate()
      .then(() => true)
      .catch(() => false);
    if (!valid || !targetRow?.id) return;

    destroyLoading.value = true;
    try {
      await EnvService.deleteEnv({
        envID: targetRow.id,
      });
      Message({
        theme: 'success',
        message: t('销毁成功'),
      });
      isShowDestroyDialog.value = false;
      emit('deleted', {
        envName: targetRow.name,
        sourceEnvName: targetRow.sourceEnvName || undefined,
      });
    } catch (err) {
      console.error(err);
    } finally {
      destroyLoading.value = false;
    }
  }

  // 跳到应用配置页后，deploy-config.vue 会消费 envName 并选中对应特性环境。
  function handleGoConfig(row: FeatureEnvRow) {
    const query = {
      ...route.query,
      activeTab: 'deploy-config',
      envName: row.name,
    };
    const resolved = router.resolve({
      name: 'detail',
      params: {
        ...route.params,
        menuName: 'appConfig',
      },
      query,
    });
    window.open(resolved.href, '_blank');
  }

  async function handleRemoveDeploy() {
    if (!appDetailStore.appID || !removeDeployTargetRow.value?.name) return;

    removeDeployLoading.value = true;
    try {
      const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
      await deployAPIs.deleteDeploy({
        appID: appDetailStore.appID,
        envName: removeDeployTargetRow.value.name,
      });
      Message({
        theme: 'success',
        message: t('移除部署成功'),
      });
      isShowRemoveDeployDialog.value = false;
      emit('deploy-removed');
    } catch (err) {
      console.error(err);
    } finally {
      removeDeployLoading.value = false;
    }
  }

  function handleRemoveDeployDialogClosed() {
    removeDeployTargetRow.value = null;
    removeDeployInstanceCount.value = 0;
  }

  // 前端仅拦截明确不可销毁的状态，最终以后端校验为准。
  function handleShowDestroyDialog(row: FeatureEnvRow) {
    if (row.destroyDisabled) return;
    destroyTargetRow.value = row;
    isShowDestroyDialog.value = true;
  }

  async function handleShowRemoveDeployDialog(row: FeatureEnvRow) {
    if (!appDetailStore.appID || !row.name || preparingRemoveDeployRowID.value) return;

    preparingRemoveDeployRowID.value = row.id;
    removeDeployTargetRow.value = row;
    try {
      // 实例数只在打开确认弹窗时获取，避免侧栏列表批量请求。
      const res = await InstanceService.listAppInstances({
        appID: appDetailStore.appID,
        envName: row.name,
        page: 1,
        pageSize: 5,
      });
      removeDeployInstanceCount.value = Number(res.count) || 0;
      isShowRemoveDeployDialog.value = true;
    } catch (err) {
      console.error(err);
    } finally {
      preparingRemoveDeployRowID.value = '';
    }
  }

  function resetDestroyDialog() {
    destroyTargetRow.value = null;
    destroyFormData.value.confirmName = '';
    destroyFormRef.value?.clearValidate?.();
  }

  // 部署状态筛选项基于当前列表动态生成，避免表头展示后端未返回的状态。
  watch(
    normalizedList,
    rows => {
      const deployStatusOptions = buildDeployStatusOptions(rows);
      deployStatusSearchData.value = deployStatusSearchData.value.map(item => {
        if (item.id === 'deployStatus') {
          return {
            ...item,
            children: deployStatusOptions,
          };
        }
        return item;
      });
    },
    { immediate: true },
  );

  // 父组件负责请求列表，侧栏只同步错误态用于 TableException 展示。
  watch(
    () => props.error,
    error => {
      if (error) {
        setTypeToError();
      } else {
        clearErrorType();
      }
    },
    { immediate: true },
  );
</script>

<style lang="postcss" scoped>
  :deep(.bk-modal-body) {
    .bk-modal-header {
      .bk-dialog-header {
        padding-top: 48px;
      }
    }
    .bk-modal-footer {
      .bk-dialog-footer {
        border: none;
        background-color: unset;
        padding-top: 0;
        padding-bottom: 24px;
      }
    }
  }
</style>
