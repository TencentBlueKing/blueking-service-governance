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
        <template v-if="previousVersion && currentVersion">
          (V{{ previousVersion.version }}：V{{ currentVersion.version }})
        </template>
        <Tag v-if="!compareLoading && previousContent && currentContent && previousContent === currentContent">
          {{ $t('内容无差异') }}
        </Tag>
        <template v-else-if="!compareLoading">
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
      </div>
    </template>

    <MsEditor
      v-bkloading="{ loading: compareLoading, zIndex: 1000, opacity: 0.1 }"
      class="!h-[70vh]"
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
              V{{ previousVersion?.version }}
              <span class="text-[#979BA5] ml-[8px]">
                {{ previousVersion?.createdAt ? dayjs(previousVersion?.createdAt).format('YYYY-MM-DD HH:mm:ss') : '' }}
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
  import { ref, watch } from 'vue';

  import { Button, Dialog, Tag } from 'bkui-vue';
  import dayjs from 'dayjs';
  import { AppConfigFileVersionOutputObj, CompareAppConfigFileVersionsOutput } from '~/@types/v1/app-config-files';
  import { AppConfigFilesService } from '~/api/modules/v1';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';

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

  /** 对比接口加载状态 */
  const compareLoading = ref(false);

  /** 当前生效版本（内部获取） */
  const currentVersion = ref<null | VersionItem>(null);

  /** 旧版本内容 */
  const previousContent = ref<string>('');
  /** 新版本内容 */
  const currentContent = ref<string>('');

  /** diff 行数统计（由 Monaco diff editor 计算得出） */
  const diffStats = ref({ added: 0, deleted: 0 });

  /** 调用对比接口 */
  async function fetchCompareResult() {
    if (!props.previousVersion || !currentVersion.value || !props.appID) return;

    compareLoading.value = true;
    previousContent.value = '';
    currentContent.value = '';

    try {
      const res = (await AppConfigFilesService.compareAppConfigFileVersions(
        {
          appID: props.appID,
          previousVersionID: props.previousVersion?.id ?? '',
          currentVersionID: currentVersion.value!.id ?? '',
        },
        { needRes: true },
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      )) as any as CompareAppConfigFileVersionsOutput;

      const getVersionContent = (v?: AppConfigFileVersionOutputObj) =>
        v?.type === 'overlay' ? v?.overlayContent || '' : v?.content || '';

      previousContent.value = getVersionContent(res?.previous);
      currentContent.value = getVersionContent(res?.current);
    } catch {
      previousContent.value = '';
      currentContent.value = '';
    } finally {
      compareLoading.value = false;
    }
  }

  /** 获取当前生效版本详情 */
  async function fetchCurrentVersion() {
    if (!props.appID || !props.currentVersionNum) return;

    try {
      const res = await AppConfigFilesService.listAppConfigFileVersions({
        appID: props.appID,
        // 按文件模式使用 appConfigFileID，按环境模式使用 envName
        ...(props.appConfigFileID ? { appConfigFileID: props.appConfigFileID } : { envName: props.envName ?? '' }),
        version: props.currentVersionNum,
        page: 1,
        pageSize: 10,
      });
      const currentVersionData = res.results?.[0];
      if (currentVersionData) {
        currentVersion.value = { ...currentVersionData, isCurrent: true };
      }
    } catch (error) {
      console.error('获取当前生效版本详情失败', error);
    }
  }

  /** 关闭弹窗 */
  function handleClose() {
    isShow.value = false;
    resetState();
  }

  /** 弹窗关闭动画结束后重置状态 */
  function handleClosed() {
    resetState();
  }

  /** 处理 Monaco diff-stats 事件 */
  function handleDiffStats(stats: { added: number; deleted: number }) {
    diffStats.value = stats;
  }

  /** 重置状态 */
  function resetState() {
    previousContent.value = '';
    currentContent.value = '';
    currentVersion.value = null;
    diffStats.value = { added: 0, deleted: 0 };
  }

  /** 弹窗打开时获取当前版本详情及对比数据 */
  watch(isShow, async val => {
    if (val && props.previousVersion) {
      resetState();
      compareLoading.value = true;
      await fetchCurrentVersion();
      if (currentVersion.value && props.previousVersion.id !== currentVersion.value.id) {
        await fetchCompareResult();
      } else {
        compareLoading.value = false;
        isShow.value = false;
      }
    }
  });
</script>
