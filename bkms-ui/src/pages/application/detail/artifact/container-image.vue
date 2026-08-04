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
  <div class="w-full">
    <Skeleton
      :full-height="false"
      :loading="isLoading"
      theme="gray"
    >
      <template #loading>
        <FlexRow class="flex">
          <template #left>
            <Layout.shape
              :height="32"
              :width="120"
            />
          </template>
          <template #right>
            <Layout.shape
              :height="32"
              :width="400"
            />
          </template>
        </FlexRow>
        <Layout.shape
          class="mt-[12px]"
          :height="32"
          width="100%"
        />
        <Layout.table class="mt-[16px]" />
      </template>
      <FlexRow lclass="flex">
        <template #left>
          <Button
            v-bk-tooltips="$t('将远程镜像源中的镜像同步至服务治理平台')"
            class="mr-[8px]"
            :loading="isRefreshing"
            theme="primary"
            @click="handleRefreshImages"
          >
            <i class="bkms-icon bkms-icon-sync-data text-[14px] mr-[8px]"></i>
            {{ $t('一键同步') }}
          </Button>
        </template>
        <template #right>
          <div class="flex">
            <SearchSelect
              v-model="searchValue"
              class="w-[400px] bg-[#fff] relative z-[100]"
              :data="artifactSearchData"
              :placeholder="
                createPlaceholder({
                  type: 'searchSelect',
                  labels: [' Tag'],
                })
              "
              unique-select
              value-behavior="need-key"
            />
          </div>
        </template>
      </FlexRow>
      <Alert
        v-if="hasSnapshotLastError"
        class="my-[12px]"
        theme="warning"
      >
        {{ t('部分制品的镜像信息同步失败，大小、更新时间、摘要可能显示不完整。') }}
        <Button
          class="ml-[6px]"
          text
          theme="primary"
          @click="handleRefreshImages"
        >
          {{ $t('重新同步') }}
        </Button>
      </Alert>
      <Table
        v-bkloading="{ loading: isLoading }"
        auto-resize
        class="mt-[16px] w-full artifact-table"
        :data="artifactList"
        :expand-config="expandConfig"
        :pagination="pagination"
        :row-config="{
          isHover: true,
          isCurrent: true,
        }"
        :sort-config="sortConfig"
        sync-resize
        @page-limit-change="pageSizeChange"
        @page-value-change="pageChange"
      >
        <template #empty>
          <TableException
            :type="curExceptionType"
            @clear="handleClearFilters"
            @refresh="fetchArtifactList"
          />
        </template>
        <!-- 展开行 -->
        <TableColumn
          fixed="left"
          type="expand"
          width="30"
        >
          <template #content="{ row }">
            <ExpandContent
              :env-name-display-map="envNameDisplayMap"
              :row="row"
            />
          </template>
        </TableColumn>
        <TableColumn
          field="tag"
          fixed="left"
          :label="t('镜像 Tag')"
          show-overflow="tooltip"
          :width="160"
        >
          <template #default="{ row }">
            {{ row.tag || '--' }}
            <Tag
              v-if="row.isPromoted"
              v-bk-tooltips="{
                content: getPromotedTooltipContent(row),
                extCls: 'promoted-tag-tooltip',
              }"
              class="ml-[8px]"
              theme="success"
            >
              <div class="flex items-center gap-[4px] relative">
                <Done
                  class="absolute top-[-2px] left-[-7px]"
                  :height="26"
                  :width="26"
                ></Done>
                <span class="ml-[18px]">{{ $t('已晋级') }}</span>
              </div>
            </Tag>
          </template>
        </TableColumn>
        <TableColumn
          field="size"
          :label="t('大小')"
          show-overflow="tooltip"
          :width="120"
        >
          <template #default="{ row }">
            {{ formatSize(row.size) }}
          </template>
        </TableColumn>
        <TableColumn
          field="builtAt"
          :label="t('构建时间')"
          show-overflow="tooltip"
          :width="180"
        >
          <template #default="{ row }">
            {{ row.builtAt ? formatDateString(row.builtAt) : '--' }}
          </template>
        </TableColumn>
        <!-- 摘要 -->
        <TableColumn
          field="digest"
          :label="t('摘要')"
          :min-width="100"
        >
          <template #default="{ row }">
            <HoverCopy
              :copy-value="row.digest"
              :text="formatDigest(row.digest)"
              :tooltip="row.digest"
            />
          </template>
        </TableColumn>
        <TableColumn
          field="deployedEnvs"
          :label="t('已部署环境')"
          :min-width="200"
        >
          <template #default="{ row }">
            <div
              v-if="row.deployedEnvs?.length"
              class="flex flex-wrap gap-[4px]"
            >
              <Tag
                v-for="item in getDeployedEnvTags(row.deployedEnvs)"
                :key="item.key"
                v-bk-tooltips="{ content: getEnvTooltipContent(item.envNames) }"
                :class="envTypeTagClassMap[item.key]"
              >
                {{ item.label }}（{{ item.count }}）
              </Tag>
            </div>
            <span v-else>--</span>
          </template>
        </TableColumn>
        <!-- 操作列 -->
        <TableColumn
          fixed="right"
          :label="t('操作')"
          :width="200"
        >
          <template #default="{ row }">
            <div class="flex items-center gap-[16px]">
              <Button
                :disabled="row.isPromoted"
                text
                theme="primary"
                @click.stop="handlePromote(row)"
              >
                {{ t('晋级') }}
              </Button>
              <Button
                text
                theme="primary"
                @click.stop="handleDelete(row)"
              >
                {{ t('删除') }}
              </Button>
            </div>
          </template>
        </TableColumn>
      </Table>
    </Skeleton>

    <!-- 晋级确认弹窗 -->
    <Dialog
      v-model:is-show="isPromoteDialogShow"
      footer-align="center"
      header-align="center"
      :title="t('确认晋级')"
      :width="480"
    >
      <Alert
        v-if="showPromoteRiskAlert"
        class="mb-[24px]"
        theme="warning"
      >
        {{ t('该镜像尚未部署到任何测试环境，请确认是否仍要晋级') }}
      </Alert>
      <div class="rounded bg-[#F5F7FA] px-[16px] py-[12px]">
        <FieldItem
          :container-height="32"
          :field-value="t('镜像 Tag')"
          :field-width="90"
          :value="promoteRow?.tag || '--'"
          value-color="#313238"
        />
        <FieldItem
          class="!h-auto !items-start"
          :container-height="32"
          :field-value="t('已部署环境')"
          :field-width="90"
          value-color="#313238"
        >
          <template #value>
            <div class="flex flex-wrap gap-[4px]">
              <Tag
                v-for="env in promoteRow?.deployedEnvs || []"
                :key="env.envName"
                :class="envTypeTagClassMap[env.envType || '']"
              >
                {{ getEnvDisplayName(env?.envName ?? '') }}
              </Tag>
              <span v-if="!promoteRow?.deployedEnvs?.length">--</span>
            </div>
          </template>
        </FieldItem>
      </div>
      <div class="mt-[12px] text-[12px] text-[#979BA5]">
        {{ t('制品仅在手动晋级后，才允许被部署到生产类型环境中：') }}
      </div>
      <div class="mt-[8px] flex flex-wrap gap-[4px]">
        <Tag
          v-for="envName in productionEnvNames"
          :key="envName"
          :class="getEnvTagClass(envName)"
        >
          {{ getEnvDisplayName(envName) }}
        </Tag>
        <span
          v-if="!productionEnvNames.length"
          class="text-[12px] text-[#979BA5]"
          >--</span
        >
      </div>
      <template #footer>
        <Button
          :loading="isPromoting"
          theme="primary"
          @click="confirmPromote"
        >
          {{ t('确定') }}
        </Button>
        <Button
          class="ml-[8px]"
          :disabled="isPromoting"
          @click="isPromoteDialogShow = false"
        >
          {{ t('取消') }}
        </Button>
      </template>
    </Dialog>

    <!-- 删除镜像 Tag 确认弹窗 -->
    <DeleteImageTagDialog
      v-model:is-show="isDeleteDialogShow"
      :get-env-display-name="getEnvDisplayName"
      :image-usages="imageUsages"
      :is-loading-usages="isLoadingUsages"
      :row="deleteRow"
      @closed="handleDeleteDialogClose"
      @perm-error="handlePermErrorShow"
      @success="fetchArtifactList"
    />

    <!-- 无删除权限提示弹窗 -->
    <PermErrorDialog v-model:is-show="isPermErrorShow" />
  </div>
</template>

<script lang="ts" setup>
  import { computed, defineAsyncComponent, h, nextTick, onBeforeUnmount, ref, shallowRef, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Button, Dialog, Message, SearchSelect, Tag } from 'bkui-vue';
  import { Done } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { AppImageOutputObj } from '~/@types/v1/images';
  import { EnvService, ImagesService } from '~/api/modules/v1';
  import { formatSize } from '~/common/util';
  import FieldItem from '~/components/field-item.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';
  import usePageConf from '~/composables/use-page';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import useTime from '~/composables/use-time';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';

  import DeleteImageTagDialog from './components/delete-image-tag-dialog.vue';
  import PermErrorDialog from './components/perm-error-dialog.vue';

  import type { ISearchValue } from 'bkui-vue/lib/search-select/utils';
  import type { ImageTagUsagesOutputObj } from '~/@types/v1/images';

  import 'tippy.js/dist/tippy.css';
  import 'tippy.js/themes/light.css';

  const ARTIFACT_POLLING_INTERVAL = 60 * 1000;
  const DEPLOYED_ENV_TYPES = ['development', 'test', 'staging', 'production'] as const;

  type DeployedEnvTag = {
    count: number;
    envNames: string[];
    key: (typeof DEPLOYED_ENV_TYPES)[number];
    label: string;
  };

  type DeployedEnvType = (typeof DEPLOYED_ENV_TYPES)[number];

  const ExpandContent = defineAsyncComponent(() => import('./components/artifact-expand.vue'));

  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const { formatDateString, formatRelativeTime } = useTime();

  const appDetailStore = useAppDetail();
  const spaceStore = useSpaceStore();

  /** 环境 name → displayName 映射 */
  const envNameDisplayMap = ref<Record<string, string>>({});
  /** 环境 name → type 映射 */
  const envNameTypeMap = ref<Record<string, string>>({});

  const artifactList = ref<AppImageOutputObj[]>([]);
  const count = ref(0);
  const { pagination, pageChange, pageSizeChange } = usePageConf(
    artifactList,
    {
      current: 1,
      limit: 10,
      remote: true,
      onPageChange: () => fetchArtifactList(),
      onPageSizeChange: () => fetchArtifactList(),
    },
    count,
  );
  const sortConfig = ref({
    multiple: false,
    trigger: 'cell',
  });

  /** table 折叠配置项 */
  const expandConfig = {
    showIcon: true,
    trigger: 'row',
    iconOpen: 'bkms-icon bkms-icon-down-shape !text-[#C4C6CC] hover:!text-[#63656E] p-[2px]',
    iconClose: 'bkms-icon bkms-icon-right-shape !text-[#C4C6CC] hover:!text-[#63656E] p-[2px]',
  };

  const artifactSearchData = shallowRef([
    {
      name: 'Tag',
      id: 'tag',
      multiple: false,
    },
  ]);
  const searchValue = ref<ISearchValue[]>([]);
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });

  const searchKeyword = computed(() => {
    if (!searchValue.value || searchValue.value.length === 0) return '';
    const searchItem = searchValue.value[0];
    return searchItem?.values?.[0]?.id || '';
  });

  const isLoading = ref(false);
  const isRefreshing = ref(false);
  const snapshotLastError = ref('');
  const snapshotRefreshStatus = ref('idle');
  const hasSnapshotLastError = computed(() => Boolean(snapshotLastError.value));
  const shouldPollArtifactList = computed(() => snapshotRefreshStatus.value !== 'idle');

  let pollingTimer: null | ReturnType<typeof setTimeout> = null;

  /** 获取制品列表 */
  async function fetchArtifactList(showMessage = false) {
    if (!appDetailStore.appID) return;
    isLoading.value = true;
    try {
      const res = await ImagesService.listAppImages({
        appID: appDetailStore.appID,
        keyword: searchKeyword.value,
        page: pagination.value.current,
        pageSize: pagination.value.limit,
      });

      artifactList.value =
        res?.results?.map(item => ({
          ...item,
          repo: item.repository,
          artifact: `${item.repository}:${item.tag}`,
        })) || [];
      count.value = Number(res?.count) || 0;
      productionEnvNames.value = res.productionEnvNames || [];
      snapshotLastError.value = res.snapshotStatus?.lastError || '';
      snapshotRefreshStatus.value = res.snapshotStatus?.refreshStatus || 'idle';
      clearErrorType();

      if (shouldPollArtifactList.value) {
        startPolling();
      } else {
        stopPolling();
        if (showMessage) {
          Message({ message: t('镜像同步成功'), theme: 'success' });
        }
      }
    } catch {
      setTypeToError();
      artifactList.value = [];
      count.value = 0;
      snapshotRefreshStatus.value = 'idle';
      stopPolling();
    } finally {
      isLoading.value = false;
    }
  }

  /** 获取环境列表（仅页面初始化时调用一次） */
  async function fetchEnvList() {
    const envList = await EnvService.listEnvs({ workspaceID: spaceStore.currentSpace }).catch(() => []);
    const displayMap: Record<string, string> = {};
    const typeMap: Record<string, string> = {};
    (envList as Array<{ displayName: string; name: string; type?: string }>).forEach(env => {
      displayMap[env.name] = env.displayName || env.name;
      if (env.type) typeMap[env.name] = env.type;
    });
    envNameDisplayMap.value = displayMap;
    envNameTypeMap.value = typeMap;
  }

  /** 格式化摘要，截取冒号后前8位 */
  function formatDigest(digest?: string) {
    if (!digest) return '';
    return digest.includes(':') ? digest.split(':')[1].slice(0, 8) : digest.slice(0, 8);
  }
  function getDeployedEnvTags(deployedEnvs: AppImageOutputObj['deployedEnvs']): DeployedEnvTag[] {
    if (!deployedEnvs?.length) return [];

    const typeEnvNamesMap: Record<DeployedEnvType, string[]> = {
      development: [],
      staging: [],
      production: [],
      test: [],
    };

    deployedEnvs.forEach(({ envName, envType }) => {
      if (!envType || !DEPLOYED_ENV_TYPES.includes(envType as DeployedEnvType)) return;
      typeEnvNamesMap[envType as DeployedEnvType].push(envName ?? '');
    });

    return DEPLOYED_ENV_TYPES.filter(envType => typeEnvNamesMap[envType].length > 0).map(envType => ({
      count: typeEnvNamesMap[envType].length,
      envNames: typeEnvNamesMap[envType],
      key: envType,
      label: envTypeMap[envType]?.name || envType,
    }));
  }

  function getEnvDisplayName(envName: string) {
    return envNameDisplayMap.value[envName] || envName;
  }

  function getEnvTagClass(envName: string) {
    return envTypeTagClassMap[envNameTypeMap.value[envName]];
  }

  function getEnvTooltipContent(envNames: string[]) {
    return envNames.map(getEnvDisplayName).join('、');
  }

  function getPromotedTooltipContent(row: AppImageOutputObj) {
    const productionEnvText = productionEnvNames.value.length ? getEnvTooltipContent(productionEnvNames.value) : '--';
    const promotedByText = row.promotedBy || '--';
    const promotedAtText = row.promotedAt ? formatRelativeTime(row.promotedAt) : '--';

    return h('div', [
      h('div', `${t('可部署到生产类型环境')}：${productionEnvText}`),
      h(
        'div',
        {
          style: {
            color: '#979BA5',
          },
        },
        t('由 {0} 于 {1} 晋级', [promotedByText, promotedAtText]),
      ),
    ]);
  }

  function handleClearFilters() {
    searchValue.value = [];
    fetchArtifactList();
  }

  function startPolling() {
    stopPolling();
    pollingTimer = setTimeout(() => {
      void fetchArtifactList();
    }, ARTIFACT_POLLING_INTERVAL);
  }

  function stopPolling() {
    if (pollingTimer) {
      clearTimeout(pollingTimer);
      pollingTimer = null;
    }
  }

  /** 晋级弹窗 */
  const isPromoteDialogShow = ref(false);
  const isPromoting = ref(false);
  const promoteRow = ref<AppImageOutputObj | null>(null);
  const productionEnvNames = ref<string[]>([]);
  const showPromoteRiskAlert = computed(() => {
    const deployedEnvs = promoteRow.value?.deployedEnvs || [];
    return !deployedEnvs.some(env => env.envType === 'test');
  });

  /** 确认晋级 */
  async function confirmPromote() {
    if (!promoteRow.value) return;
    isPromoting.value = true;
    try {
      await ImagesService.promoteAppImage({
        appID: appDetailStore.appID,
        tag: promoteRow.value?.tag ?? '',
      });
      isPromoteDialogShow.value = false;
      await fetchArtifactList();
    } finally {
      isPromoting.value = false;
    }
  }

  /** 打开晋级确认弹窗 */
  function handlePromote(row: AppImageOutputObj) {
    promoteRow.value = row;
    isPromoteDialogShow.value = true;
  }

  /** 删除弹窗 */
  const isDeleteDialogShow = ref(false);
  const isLoadingUsages = ref(false);
  const deleteRow = ref<AppImageOutputObj | null>(null);
  const imageUsages = ref<ImageTagUsagesOutputObj | null>(null);
  /** 权限错误弹窗 */
  const isPermErrorShow = ref(false);

  /** 点击删除按钮：先查询占用情况，再打开弹窗 */
  async function handleDelete(row: AppImageOutputObj) {
    deleteRow.value = row;
    imageUsages.value = null;
    isDeleteDialogShow.value = true;
    isLoadingUsages.value = true;

    try {
      const res = await ImagesService.listAppImageUsages({
        appID: appDetailStore.appID,
        tag: row.tag ?? '',
      });
      imageUsages.value = res;
    } finally {
      isLoadingUsages.value = false;
    }
  }

  /** 关闭删除弹窗时重置状态 */
  function handleDeleteDialogClose() {
    imageUsages.value = null;
    deleteRow.value = null;
  }

  function handlePermErrorShow() {
    isDeleteDialogShow.value = false;
    nextTick(() => {
      isPermErrorShow.value = true;
    });
  }

  /** 刷新镜像快照 */
  async function handleRefreshImages() {
    if (isRefreshing.value) return;
    isRefreshing.value = true;
    try {
      await ImagesService.refreshAppImages({ appID: appDetailStore.appID });
      await fetchArtifactList(true);
    } catch (err) {
      console.error(err);
    } finally {
      isRefreshing.value = false;
    }
  }

  watch(
    () => appDetailStore.appID,
    async () => {
      await Promise.all([fetchEnvList(), fetchArtifactList()]);
    },
    {
      immediate: true,
    },
  );

  watch(searchKeyword, async () => {
    if (pagination.value.current === 1) {
      await fetchArtifactList();
    } else {
      pageChange(1);
    }
  });

  onBeforeUnmount(() => {
    stopPolling();
  });
</script>

<style lang="postcss" scoped>
  :deep(.artifact-table) {
    ::-webkit-scrollbar {
      height: 8px !important;
    }
    .vxe-cell--sort {
      height: 20px;
      padding: 0 3px;
    }
  }
  :deep(.bk-dialog-footer) {
    padding-top: 0;
    padding-bottom: 16px;
    background-color: #fff;
    border: none;
  }
  :deep(.promoted-tag-tooltip .tippy-content) {
    white-space: pre-line;
  }
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
