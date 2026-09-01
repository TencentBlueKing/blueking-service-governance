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
    <!-- 自定义 Header -->
    <div class="h-[52px] flex items-center justify-between px-[24px] bg-[#FFF] shadow-[0_3px_4px_0_#0000000a]">
      <div class="text-[16px] text-[#313238]">{{ $t('构建管理') }}</div>
      <Button
        text
        theme="primary"
        @click="handleBeforeEditBuilder"
      >
        <i class="bkms-icon bkms-icon-setting-line mr-[4px] text-[14px]"></i>
        <span class="text-[14px]">{{ $t('构建配置') }}</span>
      </Button>
    </div>
    <div class="px-[24px] py-[20px]">
      <Skeleton
        :loading="loading"
        theme="gray"
      >
        <template #loading>
          <Layout.shape
            class="mb-[16px]"
            :height="34"
            width="100%"
          />
          <Layout.shape class="mr-[10px]" />
          <Layout.shape :width="400" />
          <Layout.table class="mt-[10px]" />
        </template>
        <FlexRow>
          <template #left>
            <div class="flex">
              <Button
                v-if="isImageRegistry"
                v-bk-tooltips="$t('应用的镜像来源为镜像仓库，无需执行构建')"
                class="mr-[10px] w-[80px]"
                disabled
                theme="primary"
              >
                {{ $t('执行构建') }}
              </Button>
              <Popover
                v-else
                hide-ignore-reference
                :is-show="showPopConfirm"
                placement="bottom-start"
                theme="light"
                trigger="manual"
              >
                <Button
                  class="mr-[10px] w-[80px]"
                  theme="primary"
                  @click="showPopConfirm = true"
                >
                  {{ $t('执行构建') }}
                </Button>
                <template #content>
                  <div @mousedown="dismissBranchSelectOnOutsideMouseDown">
                    <div class="pb-[16px] text-[14px]">{{ $t('执行配置') }}</div>
                    <Form
                      ref="executeFormRef"
                      form-type="vertical"
                      :model="formData"
                      :rules="rules"
                    >
                      <Form.FormItem
                        :description="$t('值会通过变量 {0} 传给流水线', ['BKMS_REPO_REVISION'])"
                        :label="$t('代码分支')"
                        property="branch"
                        :required="isBuildFieldRequired"
                      >
                        <!-- 代码分支 -->
                        <RepoRefSelect
                          ref="branchSelectRef"
                          v-model="formData.branch"
                          class="w-[400px]"
                          :repository-id="repoAlias"
                          :workspace-id="workspaceId"
                          @branch-commit="handleBranchSelect"
                        />
                      </Form.FormItem>
                      <Form.FormItem
                        :description="$t('值会通过变量 {0} 传给流水线', ['BKMS_IMAGE_TAG'])"
                        :label="$t('版本号（镜像 Tag）')"
                        property="tag"
                        :required="isBuildFieldRequired"
                      >
                        <Input
                          v-model.trim="formData.tag"
                          class="w-[400px]"
                          clearable
                          :placeholder="`${$t('请输入，例如')}v1.0.0-alpha.1`"
                        />
                        <p
                          v-if="recommendTag"
                          class="text-[#979BA5]"
                        >
                          {{ $t('推荐版本号') }}：{{ recommendTag }}
                        </p>
                      </Form.FormItem>
                    </Form>
                    <div class="flex justify-end mt-[10px]">
                      <Button
                        :loading="isLoading"
                        size="small"
                        theme="primary"
                        @click="handleExecuteSource"
                      >
                        {{ $t('确认') }}
                      </Button>
                      <Button
                        class="ml-[8px]"
                        size="small"
                        @click="showPopConfirm = false"
                      >
                        {{ $t('取消') }}
                      </Button>
                    </div>
                  </div>
                </template>
              </Popover>
              <Input
                v-model.trim="searchValue"
                class="w-[400px]"
                clearable
                :placeholder="
                  createPlaceholder({
                    labels: ['制品', '触发人'],
                  })
                "
                type="search"
                @focus="showPopConfirm = false"
              />
            </div>
          </template>
        </FlexRow>
        <Table
          auto-resize
          class="mt-[16px] w-full build-table"
          :data="buildList"
          :pagination="pagination"
          :row-config="{
            isHover: true,
            isCurrent: true,
          }"
          :settings="settings"
          :show-settings="true"
          :sort-config="sortConfig"
          sync-resize
          :tooltip-config="tooltipConfig"
          @page-limit-change="pageSizeChange"
          @page-value-change="pageChange"
          @setting-change="handleSettingChange"
        >
          <template #empty>
            <TableException
              :type="curExceptionType"
              @clear="handleClearFilters"
              @refresh="fetchBuildList"
            >
            </TableException>
          </template>
          <TableColumn
            field="buildNum"
            fixed="left"
            :label="$t('构建号')"
            show-overflow="tooltip"
            :width="100"
          >
            <template #default="{ row }">
              <div
                class="cursor-pointer flex items-center"
                @click.stop="handleOpenBuildLog(row)"
              >
                <span
                  v-if="row.buildNum"
                  :class="[getColor(row.status)]"
                >
                  {{ `#${row.buildNum}` }}
                </span>
                <span v-else>--</span>
                <StatusIcon
                  v-if="isRunning(row.status)"
                  class="ml-[6px]"
                  hide-text
                  pending
                />
              </div>
            </template>
          </TableColumn>
          <TableColumn
            field="sourceMaterial"
            :label="$t('源材料')"
            show-overflow="tooltip"
            :width="200"
          >
            <template #default="{ row }">
              <div
                v-if="row.sourceMaterial"
                class="flex items-center"
              >
                <span class="bkms-icon bkms-icon-branchs mr-[5px]"></span>
                <span>{{ row.sourceMaterial.revision }}</span>
                <span class="bkms-icon bkms-icon-commit mx-[5px] text-[18px]"></span>
                <Button
                  :disabled="!row.sourceMaterial.commitID"
                  text
                  theme="primary"
                  @click.stop="handleToCommit(row.sourceMaterial.commitUrl)"
                >
                  {{ row.sourceMaterial.commitID.slice(0, 8) }}
                </Button>
              </div>
              <span v-else>--</span>
            </template>
          </TableColumn>
          <TableColumn
            field="creator"
            :label="$t('触发人')"
            show-overflow="tooltip"
            :width="120"
          >
            <template #default="{ row }">
              <span class="text-[12px]">{{ row?.creator || '--' }}</span>
            </template>
          </TableColumn>
          <TableColumn
            field="createAt"
            :label="$t('构建开始时间')"
            show-overflow="tooltip"
            sortable
            :width="180"
          >
          </TableColumn>
          <TableColumn
            field="updatedAt"
            :label="$t('构建结束时间')"
            show-overflow="tooltip"
            sortable
            :width="180"
          >
          </TableColumn>
          <TableColumn
            field="constructionTime"
            :label="$t('构建耗时')"
            show-overflow="tooltip"
            sortable
            :width="120"
          >
            <template #default="{ row }">
              {{ row?.constructionTime || '--' }}
            </template>
          </TableColumn>
          <TableColumn
            field="products"
            :label="$t('制品')"
            :min-width="200"
            show-overflow="tooltip"
          >
            <template #default="{ row }">
              <HoverCopy
                :copy-value="row?.products"
                :text="row?.products"
              />
            </template>
          </TableColumn>
        </Table>
      </Skeleton>
    </div>

    <!-- 编辑构建配置侧栏 -->
    <HelmBuildConfigSideslider
      v-if="isHelmLike"
      :data="helmBuildConfigData"
      :is-show="showBuilderSideslider"
      @close="showBuilderSideslider = false"
      @update="handleHelmBuildConfigUpdate"
    />
    <EditBuilderConfig
      v-else
      v-model:builder-data="builderData"
      v-model:is-show="showBuilderSideslider"
      :app-name="appDetailStore.appDetail?.name"
      :language="appDetailStore.appDetail?.appModelSpec?.trpcSpec?.language"
      :type="builderType"
      @confirm="handleUpdateBuildConfig(builderData)"
    />
    <ViewBuildLog
      v-model:visible="showBuildLog"
      :build-info="buildLogInfo"
    />
  </div>
</template>

<script lang="ts" setup>
  import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { useDebounce } from '@vueuse/core';
  import { Button, Form, Input, Message, Popover } from 'bkui-vue';
  import { cloneDeep } from 'lodash-es';
  import { ISettings } from 'node_modules/@blueking/table/typings/components/setting-column/Index.vue';
  import { useI18n } from 'vue-i18n';
  import { type BuildConfigOutputObj } from '~/@types/v1/app';
  import { type BuildRecordOutputObj, type UpdateBuildConfigRequest } from '~/@types/v1/builds';
  import { BuildsService } from '~/api/modules/v1';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { isHelmLikeAppType } from '~/composables/app-type';
  import { useAppRepoRefSelect } from '~/composables/use-app-repo-ref-select';
  import { useErrorHandler } from '~/composables/use-error-handler';
  import useInterval from '~/composables/use-interval';
  import usePageConf from '~/composables/use-page';
  import { useRecommendTag } from '~/composables/use-recommend-tag';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import useTableSort from '~/composables/use-table-sort';
  import { normalizeTagConfig } from '~/composables/use-tag-config';
  import useTime from '~/composables/use-time';
  import HelmBuildConfigSideslider from '~/pages/application/detail/base-info/helm/helm-build-config.vue';
  import EditBuilderConfig from '~/pages/application/detail/base-info/trpc/edit-builder-config.vue';
  import ViewBuildLog from '~/pages/application/detail/components/view-build-log/index.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import type { BuildInfo, BuildStatus } from '~/pages/application/detail/components/view-build-log/type';

  import 'tippy.js/dist/tippy.css';
  import 'tippy.js/themes/light.css';

  // 引入国际化
  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const appDetailStore = useAppDetail();
  const { formatDateString, calculateTimeDifference, parseTimeToSeconds } = useTime();
  const isHelmLike = computed(() => isHelmLikeAppType(appDetailStore.appType));

  const {
    workspaceId,
    repoAlias,
    branchSelectRef,
    prepareBranchAfterMount,
    resetBranchSelect,
    dismissBranchSelectOnOutsideMouseDown,
  } = useAppRepoRefSelect(() => appDetailStore.appDetail?.buildConfig?.repoBuildConfig?.repoAlias || '');
  // 来自镜像删除页「去配置」跳转时通过 store 瞬态标记（pendingBuilderSource）携带的来源，
  // 用于自动打开构建配置侧栏并默认选中指定来源。标记被消费后即清空，避免重入重复弹出，
  // 同时不影响 helmBuildConfigData 的预选（否则 sourceType 被重置会导致侧栏表单闪烁）
  const lockedBuilderSource = ref<'codeRepository' | 'imageRegistry' | undefined>(undefined);
  const helmBuildConfigData = computed(() => {
    const cfg = appDetailStore.appDetail?.buildConfig;
    if (!cfg || !lockedBuilderSource.value) return cfg;
    return { ...cfg, sourceType: lockedBuilderSource.value };
  });
  const { start, stop } = useInterval(fetchBuildList, 10000); // 轮询
  const { sortMethod } = useTableSort<BuildRecordOutputObj>();

  type BuildListType = {
    activeRowIndex: number;
    buildInfo: BuildInfo;
    buildNum: string;
    constructionTime: string;
    createAt: string;
    creator: string;
    pipeline: string;
    pipelineBuildID: string;
    products: string;
    sourceMaterial: null | {
      commitID: string;
      commitUrl: string;
      revision: string;
    };
    status: string;
    updatedAt: string;
  };
  type statusType = 'FAILED' | 'RUNNING' | 'SUCCEED' | 'UNKNOWN';
  const statusColor: Record<statusType, string> = {
    RUNNING: 'text-[#3d86ff]',
    SUCCEED: 'text-[#30b061]',
    FAILED: 'text-[#ec4343]',
    UNKNOWN: 'text-[#ffb848]',
  };
  const failedStatus = ['failed', 'pollingBroken'];

  /** 从构建记录提取日志侧滑所需信息，fallback 用于接口字段暂未返回时兜底。 */
  function createBuildInfo(record: BuildRecordOutputObj, fallback: Partial<BuildInfo> = {}): BuildInfo {
    const artifact = record.artifact || '';
    const tagSeparatorIndex = artifact.lastIndexOf(':');
    const artifactTag = tagSeparatorIndex > artifact.lastIndexOf('/') ? artifact.slice(tagSeparatorIndex + 1) : '';
    return {
      buildID: record.buildID || fallback.buildID || '',
      imageTag: record.params?.BKMS_IMAGE_TAG || artifactTag || fallback.imageTag || '',
      operator: record.operator || fallback.operator || '',
      pipelineID: record.pipelineID || fallback.pipelineID || '',
      revision: record.revision || fallback.revision || '',
      status: record.status ? normalizeBuildStatus(record.status) : fallback.status || 'warning',
    };
  }

  /** 将接口构建状态收窄为构建日志组件支持的状态。 */
  function normalizeBuildStatus(status?: string): BuildStatus {
    if (status === 'failed' || status === 'pollingBroken' || status === 'running' || status === 'success') {
      return status;
    }
    return 'warning';
  }

  /**
   * 表格 tooltip 配置
   * - 仅对 sourceMaterial 列生效
   * - 仅在单元格内容溢出（scrollWidth > clientWidth）时显示 tooltip
   */
  const tooltipConfig = {
    contentMethod: ({ cell, column, row }: { cell: HTMLElement; column: { field: string }; row: BuildListType }) => {
      const overflowElement = cell.children[0] as HTMLElement | undefined;
      // 自定义内容会强制显示 tooltip，未溢出时需返回 undefined 以保留组件默认行为
      if (
        column.field !== 'sourceMaterial' ||
        !row.sourceMaterial ||
        !overflowElement ||
        overflowElement.scrollWidth <= overflowElement.clientWidth
      ) {
        return undefined;
      }
      // 返回自定义 tooltip 内容：版本号 + commitID 前 8 位
      return `${row.sourceMaterial.revision} ${row.sourceMaterial.commitID.slice(0, 8)}`;
    },
  };

  // 获取状态颜色
  function getColor(status: string) {
    if (status === 'running') return statusColor.RUNNING;
    if (status === 'success') return statusColor.SUCCEED;
    if (failedStatus.includes(status)) return statusColor.FAILED;
    return statusColor.UNKNOWN;
  }
  // 是否运行中
  function isRunning(status: string) {
    return status === 'running';
  }

  const buildList = ref<BuildListType[]>([]);
  // 分页数据
  // const pagination = ref({ count: 0, limit: 10, current: 1 });
  const count = ref(0);
  const { pagination, pageChange, pageSizeChange, handleResetPage } = usePageConf(
    buildList,
    {
      current: 1,
      limit: 10,
      remote: true,
      onPageChange: fetchBuildList,
      onPageSizeChange: fetchBuildList,
    },
    count,
  );

  const searchValue = ref('');
  const debounceSearch = useDebounce(searchValue, 300);
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });

  // 执行操作相关的表单数据
  const isBuildFieldRequired = computed(() => appDetailStore.appDetail?.buildConfig?.pipelineBuildConfig === null);
  const formData = ref<{
    branch: string;
    tag: string;
  }>({
    branch: '',
    tag: '',
  });
  const rules = {
    branch: [
      {
        validator: (value: string) => value.length,
        message: t('分支不能为空'),
        trigger: ['blur', 'change'],
      },
    ],
    tag: [
      {
        validator: (value: string) => value.length,
        message: t('tag不能为空'),
        trigger: 'blur',
      },
    ],
  };

  const executeFormRef = ref<InstanceType<typeof Form>>();

  // 源码执行
  const showPopConfirm = ref<boolean>(false);

  // 构建日志侧滑
  const showBuildLog = ref<boolean>(false);
  // 仅跟踪通过“确认”新建的构建，打开历史日志时不持续同步状态。
  const shouldTrackCreatedBuildStatus = ref(false);
  const buildLogInfo = ref<BuildInfo>({
    buildID: '',
    imageTag: '',
    operator: '',
    pipelineID: '',
    revision: '',
    status: 'running',
  });
  // 推荐版本号（模板展示用）
  const recommendTag = ref('');
  // 获取推荐版本号
  const { getDefaultBranch, fetchRecommendTag } = useRecommendTag(() => formData.value.branch, {
    manualFetchOnly: computed(() => !repoAlias.value),
    onRecommend: tag => {
      if (!showPopConfirm.value) return;
      formData.value.tag = tag;
      recommendTag.value = tag;
    },
  });

  /** 分支确认后拉取推荐 Tag（Select 选中 / Input change 防抖后 / prepare 回填） */
  function handleBranchSelect(branch: string) {
    if (branch) fetchRecommendTag(branch);
  }

  // 打开：复位表单并回填默认分支；关闭：收起分支下拉
  watch(showPopConfirm, async visible => {
    if (!visible) {
      resetBranchSelect();
      return;
    }
    formData.value = { branch: '', tag: '' };
    recommendTag.value = '';
    await prepareBranchAfterMount(getDefaultBranch());
    // 流水线模式无 repoAlias，prepare 不会触发 branch-commit，需主动拉取推荐版本号
    if (!repoAlias.value) {
      await fetchRecommendTag('');
    }
    executeFormRef.value?.clearValidate?.();
  });

  const isLoading = ref<boolean>(false);
  // 获取构建列表
  async function fetchBuildList() {
    if (!appDetailStore.appID) return;

    const res = await BuildsService.listBuildRecords(
      {
        appID: appDetailStore.appID,
        page: pagination.value.current,
        pageSize: pagination.value.limit,
        keyword: debounceSearch.value,
      },
      { validateCode: false },
    )
      .then(data => {
        clearErrorType();
        return data;
      })
      .catch(() => {
        setTypeToError();
        return { count: 0, results: [] };
      });
    count.value = Number(res.count);
    buildList.value = (res?.results ?? []).map((item: BuildRecordOutputObj, index): BuildListType => ({
      activeRowIndex: index,
      buildInfo: createBuildInfo(item),
      buildNum: item.num || '',
      sourceMaterial: item.extras?.BK_CI_GIT_REPO_URL
        ? {
            revision: item.revision || '',
            commitID: item.commitID || '',
            commitUrl: `${item.extras.BK_CI_GIT_REPO_URL.replace(/\.git$/, '')}/commit/${item.extras?.BK_CI_GIT_REPO_HEAD_COMMIT_ID}`,
          }
        : null,
      creator: item.operator || '',
      createAt: item.startedAt ? formatDateString(item.startedAt) : '--',
      updatedAt:
        item.endedAt && !isRunning(item.status ?? '') && String(item.endedAt) !== '0001-01-01T00:00:00Z'
          ? formatDateString(item.endedAt)
          : '--',
      constructionTime: calculateTimeDifference(String(item.startedAt), String(item.endedAt)),
      products: item.artifact || '',
      status: item.status || '',
      pipeline: item.pipelineID || '',
      pipelineBuildID: item.buildID || '',
    }));
    await syncCurrentBuildLogInfo(res?.results ?? []);
  }

  async function handleExecuteSource() {
    const validate = await executeFormRef.value?.validate().catch(() => false);
    if (!validate) return;
    isLoading.value = true;

    const result = await BuildsService.createBuild(
      {
        appID: appDetailStore.appID,
        branch: formData.value.branch,
        imageTag: formData.value.tag,
      },
      { interceptorErr: false },
    )
      .then(record => {
        Message({
          theme: 'success',
          message: t('操作成功'),
        });
        // API 成功后再写入构建信息，保证侧滑打开时已有 buildID 可拉取日志
        buildLogInfo.value = createBuildInfo(record);
        return true;
      })
      .catch(err => {
        const { handleError } = useErrorHandler();
        handleError(err.error ?? err, 409, {
          theme: 'error',
          message: t('构建失败，已有应用构建任务进行中'),
        });
        return false;
      });
    isLoading.value = false;

    // 仅成功时关闭「执行构建」弹窗，再打开「构建日志」侧滑，避免两者叠在一起
    if (result) {
      shouldTrackCreatedBuildStatus.value = true;
      showPopConfirm.value = false;
      await nextTick();
      showBuildLog.value = true;
      // 点击构建重新开启轮询
      stop();
      await fetchBuildList();
      start();
    }
  }
  // 执行流水线
  // const showPipelineConfig = ref(false);
  // const params = ref<Record<string, string|number>>({});
  // function handleBeforeExecutePipeline() {
  //   if (builderType.value === 'source') return;

  //   params.value = appDetail.value?.builder.pipeline.params || {};
  //   showPipelineConfig.value = true;
  // }
  // 跳转到commit地址
  function handleToCommit(url: string) {
    if (url) window.open(url, '_blank');
  }

  /** 使用构建列表轮询结果同步当前日志侧滑的构建信息。 */
  async function syncCurrentBuildLogInfo(records: BuildRecordOutputObj[]) {
    const appID = appDetailStore.appID;
    const buildID = buildLogInfo.value.buildID;
    if (!shouldTrackCreatedBuildStatus.value || !showBuildLog.value || !appID || !buildID) return;

    let currentRecord = records.find(item => item.buildID === buildID);
    // 搜索或分页可能使当前构建不在表格结果中；构建中时额外从最新记录中查找。
    if (!currentRecord && buildLogInfo.value.status === 'running') {
      const result = await BuildsService.listBuildRecords({
        appID,
        page: 1,
        pageSize: 100,
      }).catch(() => null);
      currentRecord = result?.results?.find(item => item.buildID === buildID);
    }

    if (currentRecord && appID === appDetailStore.appID && buildID === buildLogInfo.value.buildID) {
      buildLogInfo.value = createBuildInfo(currentRecord, buildLogInfo.value);
      const terminalStatuses = ['canceled', 'failed', 'pollingBroken', 'pollingTimeout', 'success'];
      if (currentRecord.status && terminalStatuses.includes(currentRecord.status)) {
        shouldTrackCreatedBuildStatus.value = false;
      }
    }
  }

  const settings = reactive({
    checked: ['buildNum', 'sourceMaterial', 'creator', 'createAt', 'updatedAt', 'constructionTime', 'products'],
    disabled: ['buildNum'],
    size: 'small',
  });

  const sortConfig = ref({
    multiple: false,
    trigger: 'cell',
    sortMethod: ({
      data,
      sortList,
    }: {
      data: BuildRecordOutputObj[];
      sortList: {
        field: string;
        order: 'asc' | 'desc';
      }[];
    }) =>
      sortMethod(
        { data, sortList },
        (itemA, itemB) => parseTimeToSeconds(itemA as string) - parseTimeToSeconds(itemB as string),
      ),
  });

  function handleSettingChange(data: ISettings) {
    settings.size = data.size as string;
  }

  // 应用详情
  const loading = ref(false);
  // const builderType = ref<'imageRegistry'|'codeRepository'>('codeRepository');

  // 判断是否为镜像仓库来源
  const isImageRegistry = computed(() => {
    return appDetailStore.appDetail?.buildConfig?.sourceType === 'imageRegistry';
  });

  // 清除筛选并搜索
  function handleClearFilters() {
    searchValue.value = '';
    fetchBuildList();
  }

  // 打开构建日志侧滑
  function handleOpenBuildLog(row: BuildListType) {
    shouldTrackCreatedBuildStatus.value = false;
    buildLogInfo.value = { ...row.buildInfo };
    showBuildLog.value = true;
  }

  watch(showBuildLog, visible => {
    if (!visible) shouldTrackCreatedBuildStatus.value = false;
  });

  // 搜索（防抖）
  watch(debounceSearch, () => {
    handleResetPage();
    fetchBuildList();
  });

  // app变化时，重新获取列表
  watch(
    () => appDetailStore.appID,
    async () => {
      await fetchBuildList();
      if (buildList.value.length) {
        start();
      }
      showPopConfirm.value = false;
    },
    { immediate: true },
  );

  onBeforeUnmount(() => {
    stop();
  });

  // 编辑构建配置侧栏
  const showBuilderSideslider = ref(false);
  const builderData = ref<BuildConfigOutputObj>({} as BuildConfigOutputObj);
  const builderType = ref<'codeRepository' | 'pipeline'>('codeRepository');

  /**
   * 格式化构建配置为 API 请求参数
   */
  function formatUpdateBuildConfig(value: BuildConfigOutputObj): UpdateBuildConfigRequest {
    return {
      appID: appDetailStore.appID,
      sourceType: value.sourceType,
      tagConfig: normalizeTagConfig(value.tagConfig),
      codeRepo: value.repoBuildConfig || null,
      pipeline: value.pipelineBuildConfig || null,
    } as UpdateBuildConfigRequest;
  }

  /**
   * 打开编辑构建配置侧栏
   * appDetail 已由 detail.vue 的 detailLoading 保证就绪，直接读取 buildConfig 作为编辑初始值
   */
  function handleBeforeEditBuilder() {
    showPopConfirm.value = false; // 关闭执行构建的 Popover
    const buildConfig = (appDetailStore.appDetail?.buildConfig || {}) as BuildConfigOutputObj;
    builderData.value = cloneDeep(buildConfig);
    builderType.value = (buildConfig.sourceType as typeof builderType.value) || 'codeRepository';
    showBuilderSideslider.value = true;
  }

  // 从镜像删除页「去配置」跳转过来时（通过 store 的瞬态标记 pendingBuilderSource 携带来源），
  // 自动打开编辑构建配置侧栏。
  // appDetail 已由 detail.vue 的 detailLoading 保证就绪，挂载时直接打开不会先渲染空面板
  watch(
    [() => appDetailStore.pendingBuilderSource, isHelmLike],
    () => {
      const source = appDetailStore.pendingBuilderSource;
      if (!source || !isHelmLike.value) return;
      // 仅当真正要打开时才消费标记（清空），避免 isHelmLike 尚未就绪时提前清空导致丢失
      appDetailStore.consumePendingBuilderSource();
      lockedBuilderSource.value = source as 'codeRepository' | 'imageRegistry';
      nextTick(() => (showBuilderSideslider.value = true));
    },
    { immediate: true },
  );

  // 侧栏关闭后重置锁定的来源，避免影响后续手动打开时默认不选来源
  watch(showBuilderSideslider, isShow => {
    if (!isShow) lockedBuilderSource.value = undefined;
  });

  /** Helm 构建配置更新回调 */
  async function handleHelmBuildConfigUpdate() {
    showBuilderSideslider.value = false;
    await Promise.all([appDetailStore.fetchAppDetail(), fetchBuildList()]);
  }

  /**
   * 更新构建配置
   * @param value 构建配置数据
   */
  async function handleUpdateBuildConfig(value: BuildConfigOutputObj) {
    const params = formatUpdateBuildConfig(value);
    const result = await BuildsService.updateBuildConfig(params, { validateCode: false })
      .then(() => true)
      .catch(() => false);

    if (result) {
      Message({
        message: t('操作成功'),
        theme: 'success',
        delay: 1500,
      });
      showBuilderSideslider.value = false;
      await Promise.all([appDetailStore.fetchAppDetail(), fetchBuildList()]);
    }
    return result;
  }
</script>

<style lang="postcss" scoped>
  /* 隐藏setting Tab的滚动条 */
  .action-tab-wrapper {
    overflow-y: auto !important;
  }
  :deep(.build-table) {
    ::-webkit-scrollbar {
      height: 8px !important;
    }
    .vxe-cell--sort {
      height: 20px;
      padding: 0 3px;
    }
  }
</style>
