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
    v-model:is-show="visible"
    quick-close
    render-directive="if"
    :width="960"
    @hidden="handleHidden"
  >
    <template #header>
      <div class="flex items-center">
        <span class="mr-[16px]">{{ $t('版本列表') }}</span>
        <div
          v-if="isFileMode && currentFile"
          class="mr-[12px] flex items-center"
        >
          <Divider
            class="h-[16px] mx-[0] text-[12px]"
            color="#DCDEE5"
            direction="vertical"
            type="solid"
          />
          <span class="text-[14px] ml-[16px] text-[#979BA5]">{{ currentFile.name }}</span>
        </div>
      </div>
    </template>
    <div class="flex flex-col w-full p-[24px] h-full gap-[16px]">
      <!-- 环境切换（按环境模式） -->
      <OverflowCtrl v-if="!isFileMode && enableEnvConfig">
        <Radio.Group
          v-model="curEnvID"
          type="capsule"
        >
          <Radio.Button :label="DEFAULT_ENV_ID">
            {{ $t('默认配置') }}
          </Radio.Button>
          <Radio.Button
            v-for="item in envList"
            :key="item.id"
            :label="item.id"
          >
            {{ item.displayName }}
          </Radio.Button>
        </Radio.Group>
      </OverflowCtrl>

      <!-- 搜索框 -->
      <SearchSelect
        v-model="searchValue"
        class="w-full bg-[#fff]"
        :data="versionSearchData"
        :placeholder="
          createPlaceholder({
            type: 'searchSelect',
            labels: ['版本号', '版本描述', '创建人'],
          })
        "
        unique-select
        value-behavior="need-key"
      />

      <!-- 版本列表表格 -->
      <Table
        v-bkloading="{ loading: versionListLoading }"
        :data="versionListWithCurrent"
        :pagination="pagination"
        @page-limit-change="pageSizeChange"
        @page-value-change="pageChange"
      >
        <template #empty>
          <div class="min-h-[200px]">
            <TableException
              v-show="!versionListLoading"
              :type="curExceptionType"
              @clear="() => (searchValue = [])"
              @refresh="fetchVersionList"
            />
          </div>
        </template>
        <TableColumn
          field="version"
          :min-width="120"
          show-overflow-tooltip
        >
          <template #header>
            <div class="flex items-center gap-[4px]">
              <span>{{ $t('版本号') }}</span>
              <Popover
                placement="bottom-start"
                theme="dark"
              >
                <div class="flex items-center cursor-pointer">
                  <i class="bkms-icon bkms-icon-warning-circle text-[16px] text-[#979BA5]"></i>
                </div>
                <template #content>
                  <div class="max-w-[238px] text-[#fff]">
                    <div class="leading-[20px] text-[#979BA5]">
                      {{ $t('版本号自动生成规则') }}
                    </div>
                    <div class="text-[12px] leading-[20px]">
                      {{ $t('每次保存或回滚将自动递增生成新版本号，格式为') }}
                      <span
                        class="inline-block h-[16px] leading-[16px] bg-[#4D4F56] text-[#AD7A6B] text-[10px] px-[4px] rounded-[2px]"
                        >VN
                      </span>
                      {{ $t('。') }}
                    </div>
                    <div class="text-[12px] leading-[20px] mb-[8px] text-[#979BA5]">
                      {{ $t('示例：（ V1:V2:V3... ）') }}
                    </div>
                    <div class="border-t border-[#4D4F56] pt-[8px] text-[12px] leading-[20px] text-[#979BA5]">
                      {{ $t('回滚操作也会生成新版本号，原历史记录始终保留，不会被覆盖。') }}
                    </div>
                  </div>
                </template>
              </Popover>
            </div>
          </template>
          <template #default="{ row }">
            <div class="flex items-center gap-[8px]">
              <span>V{{ row.version }}</span>
              <Tag
                v-if="row.isCurrent"
                size="small"
                theme="success"
              >
                {{ $t('当前版本') }}
              </Tag>
            </div>
          </template>
        </TableColumn>
        <TableColumn
          field="description"
          :label="$t('版本描述')"
          :min-width="100"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            <span>{{ row.description || '--' }}</span>
          </template>
        </TableColumn>
        <TableColumn
          field="creator"
          :label="$t('创建人')"
          :min-width="100"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            {{ row.creator || '--' }}
          </template>
        </TableColumn>
        <TableColumn
          field="createdAt"
          :label="$t('创建时间')"
          :min-width="160"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            {{ row.createdAt ? dayjs(row.createdAt).format('YYYY-MM-DD HH:mm:ss') : '--' }}
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('操作')"
          :min-width="140"
        >
          <template #default="{ row }">
            <template v-if="row.isCurrent">--</template>
            <div
              v-else
              class="flex items-center gap-[12px]"
            >
              <Button
                :disabled="row.isCurrent"
                text
                theme="primary"
                @click="handleRollback(row)"
              >
                {{ $t('回滚到此版本') }}
              </Button>
              <Button
                text
                theme="primary"
                @click="handleCompare(row)"
              >
                {{ $t('版本对比') }}
              </Button>
              <PopConfirm
                :content="$t('确认删除该历史版本？')"
                trigger="click"
                width="288"
                @confirm="handleDelete(row)"
              >
                <Button
                  text
                  theme="primary"
                >
                  {{ $t('删除') }}
                </Button>
              </PopConfirm>
            </div>
          </template>
        </TableColumn>
      </Table>
    </div>
  </Sideslider>

  <!-- 回滚确认弹窗 -->
  <RollbackConfirmDialog
    v-model:is-show="rollbackDialogVisible"
    :current-version="currentVersionNum"
    :version-data="currentRollbackRow"
    @rollback-success="handleRollbackSuccess"
  />

  <!-- 版本对比弹窗 -->
  <VersionCompareDialog
    v-model:is-show="compareDialogVisible"
    :app-config-file-i-d="currentFile?.id"
    :app-i-d="appDetailStore.appID"
    :current-version-num="currentVersionNum"
    :env-name="curEnvName"
    :previous-version="comparePreviousVersion"
  />
</template>

<script setup lang="ts">
  import { computed, nextTick, ref, shallowRef, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Divider, PopConfirm, Popover, Radio, SearchSelect, Sideslider, Tag } from 'bkui-vue';
  import dayjs from 'dayjs';
  import { useI18n } from 'vue-i18n';
  import { AppConfigFileOutputObj, AppConfigFileVersionOutputObj } from '~/@types/v1/app-config-files';
  import { EnvOutput } from '~/@types/v1/env';
  import { AppConfigFilesService } from '~/api/modules/v1';
  import OverflowCtrl from '~/components/overflow-ctrl.vue';
  import TableException from '~/components/table-exception.vue';
  import usePageConf from '~/composables/use-page';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import { useAppDetail } from '~/stores/app-detail';

  import RollbackConfirmDialog from './components/rollback-confirm-dialog.vue';
  import VersionCompareDialog from './components/version-compare-dialog.vue';

  import type { ISearchValue } from 'bkui-vue/lib/search-select/utils';

  /** 默认配置环境 ID */
  const DEFAULT_ENV_ID = '__default__';

  /** 版本项类型（接口数据 + isCurrent 计算字段） */
  type VersionItem = AppConfigFileVersionOutputObj & { isCurrent: boolean };

  const props = defineProps<{
    configFileList: AppConfigFileOutputObj[];
    /** 当前环境名（按环境模式时使用） */
    currentEnvName?: string;
    /** 当前选中的文件（Helm 按文件模式时使用） */
    currentFile?: AppConfigFileOutputObj | null;
    /** 是否启用环境配置（按环境模式时使用） */
    enableEnvConfig?: boolean;
    /** 环境列表（按环境模式时使用） */
    envList?: EnvOutput[];
    visible: boolean;
  }>();

  const emit = defineEmits<{
    refresh: [];
    rollback: [];
    'update:visible': [value: boolean];
  }>();

  const appDetailStore = useAppDetail();
  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();

  /** 控制侧栏显示/隐藏 */
  const visible = computed({
    get: () => props.visible,
    set: (val: boolean) => emit('update:visible', val),
  });

  /** 是否为按文件模式（Helm 应用场景） */
  const isFileMode = computed(() => !!props.currentFile);

  /** 当前选中的环境 ID */
  const curEnvID = ref(DEFAULT_ENV_ID);

  /** 当前选中环境对应的 envName（默认配置时为空字符串） */
  const curEnvName = computed(() => {
    if (isFileMode.value) return '';
    if (!props.enableEnvConfig || curEnvID.value === DEFAULT_ENV_ID) return '';
    return props.envList?.find(e => e.id === curEnvID.value)?.name ?? '';
  });

  /** 搜索值 */
  const searchValue = ref<ISearchValue[]>([]);

  /** 搜索配置 */
  const versionSearchData = shallowRef([
    { name: t('版本号'), id: 'version', multiple: false, placeholder: t('请输入版本号') },
    { name: t('版本描述'), id: 'description', multiple: false, placeholder: t('请输入版本描述') },
    { name: t('创建人'), id: 'creator', multiple: false, placeholder: t('请输入创建人') },
  ]);

  /** 版本列表加载状态 */
  const versionListLoading = ref(false);

  /** 版本列表数据 */
  const versionList = ref<AppConfigFileVersionOutputObj[]>([]);

  /** 分页总数 */
  const total = ref(0);

  /** 分页配置 */
  const { pagination, pageChange, pageSizeChange, pageConf, handleResetPage } = usePageConf(
    versionList,
    {
      current: 1,
      limit: 10,
      remote: true,
      onPageChange: fetchVersionList,
      onPageSizeChange: fetchVersionList,
    },
    total,
  );

  /** 表格空状态管理 */
  const { curExceptionType, setTypeToError, clearErrorType } = useTableEmpty({ filters: searchValue });

  /** 回滚弹窗 */
  const rollbackDialogVisible = ref(false);
  const currentRollbackRow = ref<null | VersionItem>(null);

  /** 版本对比弹窗 */
  const compareDialogVisible = ref(false);
  const comparePreviousVersion = ref<null | VersionItem>(null);

  /** 是否发生了回滚操作（关闭侧边栏时通知父组件刷新文件内容） */
  const hasRollback = ref(false);

  /** 当前生效版本号（按文件模式从 currentFile 获取，按环境模式从 configFileList 获取） */
  const currentVersionNum = computed(() => {
    if (isFileMode.value) {
      return props.currentFile?.currentVersion ?? 0;
    }
    const configFile = props.configFileList.find(item => item.envName === curEnvName.value);
    return configFile?.currentVersion ?? 0;
  });

  /** 版本列表（附加 isCurrent 标记，当前生效版本置顶） */
  const versionListWithCurrent = computed<VersionItem[]>(() => {
    return versionList.value
      .map(v => ({ ...v, isCurrent: v.version === currentVersionNum.value }))
      .sort((a, b) => Number(b.isCurrent) - Number(a.isCurrent));
  });

  /** 获取版本列表 */
  async function fetchVersionList() {
    if (!appDetailStore.appID || (isFileMode.value && !props.currentFile?.id)) return;

    versionListLoading.value = true;
    try {
      const res = await AppConfigFilesService.listAppConfigFileVersions({
        appID: appDetailStore.appID,
        // 按文件模式使用 appConfigFileID，按环境模式使用 envName
        ...(isFileMode.value ? { appConfigFileID: props.currentFile!.id } : { envName: curEnvName.value }),
        page: pageConf.current,
        pageSize: pageConf.limit,
        ...getSearchParams(),
      });
      versionList.value = res.results ?? [];
      total.value = res.count ?? 0;
      clearErrorType();
    } catch {
      versionList.value = [];
      setTypeToError();
    } finally {
      versionListLoading.value = false;
    }
  }

  /** 根据 envName 获取对应的环境 ID */
  function getEnvIDByEnvName(envName: string): string {
    if (!envName) return DEFAULT_ENV_ID;
    return props.envList?.find(e => e.name === envName)?.id ?? DEFAULT_ENV_ID;
  }

  /** 从 searchValue 中提取搜索参数 */
  function getSearchParams(): Record<string, number | string | undefined> {
    const params: Record<string, number | string | undefined> = {};
    for (const item of searchValue.value) {
      const value = item.values?.[0]?.id;
      if (value !== undefined && value !== '') {
        // 版本号需要去掉 V 前缀并转为数字
        params[item.id] = item.id === 'version' ? Number(String(value).replace(/^V/i, '')) : value;
      }
    }
    return params;
  }

  /** 对比操作 */
  function handleCompare(row: VersionItem) {
    if (!currentVersionNum.value) return;
    comparePreviousVersion.value = row;
    compareDialogVisible.value = true;
  }

  /** 删除历史版本 */
  async function handleDelete(row: VersionItem) {
    const result = await AppConfigFilesService.deleteAppConfigFileVersion({
      appID: appDetailStore.appID,
      id: row?.id ?? '',
    })
      .then(() => true)
      .catch(() => false);
    if (result) {
      refreshAndNotify();
    }
  }

  /** 面板关闭时重置状态 */
  function handleHidden() {
    if (hasRollback.value) {
      emit('rollback');
      hasRollback.value = false;
    }
    // 重置所有状态
    curEnvID.value = DEFAULT_ENV_ID;
    searchValue.value = [];
    versionList.value = [];
    total.value = 0;
    handleResetPage();
  }

  /** 打开回滚确认弹窗 */
  function handleRollback(row: VersionItem) {
    currentRollbackRow.value = row;
    rollbackDialogVisible.value = true;
  }

  /** 确认回滚成功回调 */
  function handleRollbackSuccess() {
    hasRollback.value = true;
    refreshAndNotify();
  }

  /** 初始化环境选择 */
  function initEnvSelection() {
    if (!props.enableEnvConfig) return;

    const targetID = getEnvIDByEnvName(props.currentEnvName ?? '');
    // 先重置再赋值，确保 curEnvID 变更能触发 watch
    if (curEnvID.value === targetID) {
      curEnvID.value = '';
    }
    curEnvID.value = targetID;
  }

  /** 刷新列表并通知父组件 */
  function refreshAndNotify() {
    fetchVersionList();
    emit('refresh');
  }

  /** 监听侧栏显示状态 */
  watch(
    () => props.visible,
    visible => {
      if (!visible) return;

      // 按文件模式或未开启环境配置时，直接请求版本列表
      if (isFileMode.value || !props.enableEnvConfig) {
        fetchVersionList();
        return;
      }

      // 开启环境配置时，初始化环境选择
      const targetID = getEnvIDByEnvName(props.currentEnvName ?? '');
      const needManualFetch = curEnvID.value === targetID;

      initEnvSelection();

      // 如果初始化前后 curEnvID 没有变化，则需要手动触发请求
      if (needManualFetch) {
        nextTick(fetchVersionList);
      }
    },
  );

  /** 监听环境切换 */
  watch(curEnvID, () => {
    if (props.visible && curEnvID.value) {
      handleResetPage();
      fetchVersionList();
    }
  });

  /** 监听搜索条件变化，重新请求数据 */
  watch(searchValue, () => {
    if (props.visible) {
      handleResetPage();
      fetchVersionList();
    }
  });

  /** 监听环境列表变化 */
  watch(
    () => props.envList,
    () => {
      if (props.visible && !isFileMode.value) {
        initEnvSelection();
      }
    },
    { deep: true },
  );
</script>
