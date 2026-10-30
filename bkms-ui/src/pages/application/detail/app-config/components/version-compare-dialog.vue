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
  <Dialog
    v-model:is-show="isShow"
    render-directive="if"
    width="80%"
    @closed="handleClosed"
    @hidden="handleClosed"
  >
    <template #header>
      <div class="flex items-center gap-[8px]">
        <span>{{ $t('版本对比') }}</span>
        <template v-if="previousVersionData && currentVersion">
          (V{{ previousVersionData.version }}：V{{ currentVersion.version }})
        </template>
        <template v-if="comparisonReady">
          <Tag v-if="contentUnchanged && (isHelm || (previousContent && currentContent))">
            {{ isHelm && !configDiffRows.length ? $t('无差异') : $t('内容无差异') }}
          </Tag>
          <template v-else>
            <Tag
              v-if="diffStats.added"
              theme="success"
            >
              {{ $t('新增') }} +{{ diffStats.added }}
            </Tag>
            <Tag
              v-if="diffStats.deleted"
              theme="danger"
            >
              {{ $t('删除') }} -{{ diffStats.deleted }}
            </Tag>
          </template>
          <Tag
            v-if="configDiffRows.length"
            theme="warning"
          >
            {{ $t('配置变更 {0} 项', [configDiffRows.length]) }}
          </Tag>
        </template>
      </div>
    </template>

    <div
      v-bkloading="{ loading: compareLoading, zIndex: 1000, opacity: 0.1 }"
      class="flex flex-col h-[70vh] min-h-0"
    >
      <VersionConfigDiff
        v-if="comparisonReady && configDiffRows.length"
        class="mb-[12px] shrink-0"
        :current-version="currentVersion?.version"
        :previous-version="previousVersionData?.version"
        :rows="configDiffRows"
      />
      <MsEditor
        class="flex-1 min-h-0 !h-auto"
        :is-diff="true"
        :model-value="currentContent"
        :original="previousContent"
        readonly
        :show-copy="false"
        @diff-stats="handleDiffStats"
      >
        <template #title>
          <div class="flex items-center text-[#fff]">
            <div class="flex-1 flex justify-between">
              <div>
                V{{ previousVersionData?.version }}
                <span class="text-[#979BA5] ml-[8px]">
                  {{
                    previousVersionData?.createdAt
                      ? dayjs(previousVersionData.createdAt).format('YYYY-MM-DD HH:mm:ss')
                      : ''
                  }}
                </span>
              </div>
              <div
                v-if="diffStats.deleted"
                class="text-[#EA3636]"
              >
                -{{ diffStats.deleted }}
              </div>
            </div>
            <div class="flex-1 flex justify-between pl-[28px]">
              <div>
                V{{ currentVersion?.version }}
                <span class="text-[#979BA5]">（{{ $t('当前版本') }}）</span>
              </div>
              <div
                v-if="diffStats.added"
                class="text-[#65C389]"
              >
                +{{ diffStats.added }}
              </div>
            </div>
          </div>
        </template>
      </MsEditor>
    </div>

    <template #footer>
      <div class="flex justify-end gap-[8px]">
        <Button @click="handleClose">
          {{ $t('关闭') }}
        </Button>
      </div>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, ref, watch } from 'vue';

  import { Button, Dialog, Tag } from 'bkui-vue';
  import dayjs from 'dayjs';
  import { useI18n } from 'vue-i18n';
  import { AppConfigFilesService } from '~/api/modules/v1';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';
  import { APP_TYPES } from '~/composables/app-type';
  import { useAppDetail } from '~/stores/app-detail';

  import VersionConfigDiff from './version-config-diff.vue';

  import type { AppConfigFileVersionOutputObj, CompareAppConfigFileVersionsOutput } from '~/@types/v1/app-config-files';

  /** 版本项类型（接口数据 + isCurrent 计算字段） */
  interface VersionItem extends AppConfigFileVersionOutputObj {
    isCurrent: boolean;
  }

  const props = defineProps<{
    /** Helm 按文件模式时使用的文件 ID */
    appConfigFileID?: string;
    appID: string;
    currentVersionNum: number;
    /** 按环境模式时使用的环境名 */
    envName?: string;
    previousVersion: null | VersionItem;
  }>();

  const isShow = defineModel<boolean>('isShow', { default: false });
  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  const isHelm = computed(() => appDetailStore.appType === APP_TYPES.HELM);

  /** 对比接口加载状态 */
  const compareLoading = ref(false);
  let compareRequestID = 0;
  const comparison = ref<CompareAppConfigFileVersionsOutput | null>(null);
  const comparisonReady = computed(
    () => !compareLoading.value && !!comparison.value?.previous && !!comparison.value?.current,
  );
  const previousVersionData = computed(() => comparison.value?.previous ?? props.previousVersion);

  /** 当前生效版本（内部获取） */
  const currentVersion = ref<AppConfigFileVersionOutputObj | null>(null);

  /** 旧版本内容 */
  const previousContent = computed(() => getVersionContent(comparison.value?.previous));
  /** 新版本内容 */
  const currentContent = computed(() => getVersionContent(comparison.value?.current));
  const contentUnchanged = computed(() => previousContent.value === currentContent.value);
  const referenceNames = ref<Record<string, string>>({});

  const baseReferenceChanged = computed(() => {
    const { previous, current } = comparison.value ?? {};
    return (
      isHelm.value &&
      previous &&
      current &&
      (previous.type === 'overlay' || current.type === 'overlay') &&
      (previous.baseAppConfigFileID ?? '') !== (current.baseAppConfigFileID ?? '')
    );
  });

  const configDiffRows = computed(() => {
    const { previous, current } = comparison.value ?? {};
    if (!isHelm.value || !previous || !current) return [];

    const rows = [];
    if ((previous.name ?? '') !== (current.name ?? '')) {
      rows.push({ key: 'name', label: t('文件名称'), previous: previous.name || '--', current: current.name || '--' });
    }
    if (baseReferenceChanged.value) {
      rows.push({
        key: 'base',
        label: t('覆盖层'),
        previous: getReferenceName(previous),
        current: getReferenceName(current),
      });
    }
    return rows;
  });

  /** diff 行数统计（由 Monaco diff editor 计算得出） */
  const diffStats = ref({ added: 0, deleted: 0 });

  /** 获取当前版本及完整对比数据；关闭或重新打开后忽略旧请求。 */
  async function fetchComparison(requestID: number) {
    const { appID, appConfigFileID, envName, currentVersionNum, previousVersion } = props;
    if (!appID || !currentVersionNum || !previousVersion?.id) return;

    compareLoading.value = true;
    try {
      const res = await AppConfigFilesService.listAppConfigFileVersions({
        appID,
        // 按文件模式使用 appConfigFileID，按环境模式使用 envName
        ...(appConfigFileID ? { appConfigFileID } : { envName: envName ?? '' }),
        version: currentVersionNum,
        page: 1,
        pageSize: 10,
      });
      if (requestID !== compareRequestID) return;
      currentVersion.value = res.results?.[0] ?? null;
      if (!currentVersion.value?.id || previousVersion.id === currentVersion.value.id) {
        isShow.value = false;
        return;
      }

      const result = await AppConfigFilesService.compareAppConfigFileVersions(
        { appID, previousVersionID: previousVersion.id, currentVersionID: currentVersion.value.id },
        { needRes: true },
      );
      if (requestID !== compareRequestID || !result.previous || !result.current) return;
      comparison.value = result;
      currentVersion.value = result.current;
      // 内容已就绪，引用名称查询期间先展示文件 ID，不阻塞弹窗。
      compareLoading.value = false;
      if (baseReferenceChanged.value) {
        const names = await fetchReferenceNames(appID, result);
        if (requestID !== compareRequestID) return;
        referenceNames.value = names;
      }
    } catch {
      // 请求层已提示错误，失败时不显示差异结论。
    } finally {
      if (requestID === compareRequestID) compareLoading.value = false;
    }
  }

  /** 只查对应历史版本的名称，无法获取时由展示层回退为文件 ID。 */
  async function fetchReferenceNames(appID: string, result: CompareAppConfigFileVersionsOutput) {
    const references = new Map<string, { appConfigFileID: string; version: number }>();
    for (const version of [result.previous, result.current]) {
      if (version?.baseAppConfigFileID && version.baseVersion) {
        references.set(`${version.baseAppConfigFileID}:${version.baseVersion}`, {
          appConfigFileID: version.baseAppConfigFileID,
          version: version.baseVersion,
        });
      }
    }
    const results = await Promise.allSettled(
      [...references].map(async ([key, query]) => {
        const res = await AppConfigFilesService.listAppConfigFileVersions(
          { appID, ...query, page: 1, pageSize: 10 },
          { interceptorErr: false },
        );
        return [key, res.results?.[0]?.name || query.appConfigFileID] as const;
      }),
    );
    return Object.fromEntries(results.flatMap(result => (result.status === 'fulfilled' ? [result.value] : [])));
  }

  function getReferenceName(version: AppConfigFileVersionOutputObj) {
    const id = version.baseAppConfigFileID;
    return id ? referenceNames.value[`${id}:${version.baseVersion}`] || id : '--';
  }

  function getVersionContent(version?: AppConfigFileVersionOutputObj) {
    return version?.type === 'overlay' || (isHelm.value && version?.contentSourceType === 'bscp')
      ? (version?.overlayContent ?? '')
      : (version?.content ?? '');
  }

  /** 关闭弹窗 */
  function handleClose() {
    isShow.value = false;
  }

  /** 弹窗关闭动画结束后重置状态 */
  function handleClosed() {
    if (!isShow.value) resetState();
  }

  /** 处理 Monaco diff-stats 事件 */
  function handleDiffStats(stats: { added: number; deleted: number }) {
    if (isShow.value && comparison.value) diffStats.value = stats;
  }

  /** 重置状态 */
  function resetState() {
    compareRequestID += 1;
    compareLoading.value = false;
    comparison.value = null;
    referenceNames.value = {};
    currentVersion.value = null;
    diffStats.value = { added: 0, deleted: 0 };
  }

  /** 弹窗打开时获取当前版本详情及对比数据 */
  watch(
    [
      isShow,
      () => props.previousVersion?.id,
      () => props.appID,
      () => props.appConfigFileID,
      () => props.envName,
      () => props.currentVersionNum,
    ],
    ([visible]) => {
      resetState();
      if (visible) fetchComparison(compareRequestID);
    },
    { immediate: true, flush: 'sync' },
  );

  onBeforeUnmount(resetState);
</script>
