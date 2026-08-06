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
  <div class="h-full flex overflow-auto">
    <!-- 文件列表面板 -->
    <CollapsiblePanel
      v-model:collapsed="isFileListCollapsed"
      panel-class="box-shadow-[#191929] bg-[#fff]"
      position="left"
      :width="260"
    >
      <HelmFileList
        ref="helmFileListRef"
        class="h-full"
        :has-unsaved-changes="hasUnsavedChanges"
        @file-detail-change="handleFileDetailChange"
        @file-list-change="handleFileListChange"
        @loading-change="handleLoadingChange"
      />
    </CollapsiblePanel>
    <!-- 空状态 -->
    <div
      v-if="!fileList?.length"
      class="flex-1 h-full flex items-center min-w-0 bg-[#F5F7FA]"
    >
      <Exception
        class="large-exception"
        scene="part"
        type="empty"
      >
        <template #type>
          <img src="/empty.svg" />
        </template>
        <template #description>
          <div class="text-[#4D4F56] text-[14px] leading-[22px]">{{ $t('请先在左侧添加文件') }}</div>
        </template>
      </Exception>
    </div>
    <div
      v-else
      class="flex-1 h-full pt-[12px] flex flex-col min-w-0 bg-[#F5F7FA]"
    >
      <FlexRow class="mb-[8px] mx-[24px]">
        <template #left>
          <span class="font-bold text-[14px]">{{ curFileInfo?.name || '' }}</span>
          <span
            v-if="isReadonlyFile"
            class="text-[#F59500] text-[12px]"
            >{{ $t('（来源于 BSCP 的文件仅能查看，不能编辑）') }}</span
          >
          <Tag
            v-else
            v-bk-tooltips="{
              content: $t('当前版本 {0}，{1}', [
                `V${curFileInfo?.currentVersion}`,
                formatRelativeTimeWithTooltip(curFileInfo?.updatedAt).text,
              ]),
            }"
            class="ml-[8px]"
            theme="info"
          >
            V{{ curFileInfo?.currentVersion }}
          </Tag>
        </template>
        <template #right>
          <!-- 覆盖层展示 -->
          <template v-if="curFileInfo?.type === 'overlay'">
            <IconTextButton
              :active="showType === 'completeValues'"
              icon="bkms-icon bkms-icon-yanjing-kejian"
              :text="$t('完整 values')"
              @click="toggleAsideShow('completeValues')"
            />
            <Divider
              class="h-[12px]"
              color="#C4C6CC"
              direction="vertical"
              type="solid"
            />
          </template>
          <IconTextButton
            :active="showType === 'values'"
            icon="bkms-icon bkms-icon-setting"
            :text="$t('默认 values')"
            @click="toggleAsideShow('values')"
          />
          <Divider
            class="h-[12px]"
            color="#C4C6CC"
            direction="vertical"
            type="solid"
          />
          <IconTextButton
            :active="showType === 'variables'"
            icon="bkms-icon bkms-icon-variable"
            :text="$t('变量')"
            @click="toggleAsideShow('variables')"
          />
          <Divider
            class="h-[12px]"
            color="#C4C6CC"
            direction="vertical"
            type="solid"
          />
          <span v-bk-tooltips="{ content: $t('BSCP 文件不支持在此查看版本列表'), disabled: !isReadonlyFile }">
            <IconTextButton
              :active="showVersionListSideslider"
              :disabled="isReadonlyFile"
              icon="bkms-icon bkms-icon-time-2"
              :text="$t('版本列表')"
              @click="handleShowVersionList"
            />
          </span>
        </template>
      </FlexRow>
      <!-- 代码编辑器 -->
      <ResizeLayout
        ref="editorLayoutRef"
        :border="false"
        class="!min-w-[600px] flex-1 min-h-[0px] mx-[24px]"
        initial-divide="50%"
        placement="right"
      >
        <template #aside>
          <div class="pl-[12px] h-full">
            <MsHighlightjs
              v-if="showType === 'completeValues'"
              class="h-full"
              :code="completeValues"
              :loading="completeValuesLoading"
            >
              <template #title>
                {{ $t('完整 Values') }}
                <InfoLine
                  v-bk-tooltips="{
                    content: $t(
                      '当前 values 文件为“覆盖层”类型，将与基础 values 文件（default）基于 Patch 算法合并，生成完整的 values 配置。',
                    ),
                    extCls: '!w-[400px]',
                    theme: 'light',
                  }"
                  class="ml-[6px] text-[14px] text-[#979BA5] transform translate-y-[2px]"
                />
              </template>
            </MsHighlightjs>
            <MsHighlightjs
              v-else-if="showType === 'values'"
              class="h-full"
              :code="values"
              :loading="versionContentLoading"
              :title="$t('默认 Values')"
            >
              <template #tools>
                <Select
                  v-model="version"
                  class="mr-[20px]"
                  :clearable="false"
                  display-key="name"
                  filterable
                  id-key="name"
                  :list="versionList"
                  :prefix="$t('版本')"
                  size="small"
                  @change="handleVersionChange"
                />
              </template>
            </MsHighlightjs>
            <Variables v-else-if="showType === 'variables'" />
          </div>
        </template>
        <template #main>
          <div class="flex flex-col overflow-x-hidden h-full">
            <div class="w-full flex-1 min-h-0 relative mr-[10px]">
              <Loading
                class="h-full min-h-0"
                :loading="isLoading"
                :opacity="0.3"
              >
                <!-- BSCP 为只读状态 -->
                <MsEditor
                  v-model="overrideValues"
                  class="h-full"
                  :readonly="isReadonlyFile"
                  @error="handleYamlError"
                >
                  <template #title>
                    {{ curFileInfo?.name || '' }}
                    <Popover
                      v-if="curFileInfo?.type === 'overlay'"
                      allow-html
                      ext-cls="multi-values-tag-popover"
                      placement="top"
                      render-type="auto"
                      theme="light"
                      width="240"
                    >
                      <Tag
                        class="!text-[#CDDFFE] !bg-[#2759AE] ml-[4px] h-[16px] text-[10px] px-[4px]"
                        theme="info"
                      >
                        {{ $t('覆盖层') }}
                      </Tag>
                      <template #content>
                        <DetailItem
                          v-for="item in overlayPopoverData"
                          :key="item.label"
                          class="items-center !h-[20px]"
                          :label="item.label"
                          :label-width="65"
                          :value="item.value"
                        />
                      </template>
                    </Popover>
                    <Popover
                      v-if="curFileInfo?.bscpConfig"
                      allow-html
                      ext-cls="multi-values-tag-popover"
                      placement="top"
                      render-type="auto"
                      theme="light"
                    >
                      <Tag
                        class="!text-[#FDF4E8] !bg-[#6D5B42] ml-[4px] h-[16px] text-[10px] px-[4px]"
                        theme="warning"
                      >
                        BSCP
                      </Tag>
                      <template #content>
                        <DetailItem
                          v-for="item in bscpPopoverData"
                          :key="item.label"
                          class="items-center !h-[20px]"
                          :label="item.label"
                          :label-width="65"
                          :value="item.value"
                        />
                      </template>
                    </Popover>
                  </template>
                  <template #tools>
                    <IconButton
                      class="mr-[8px]"
                      :desc="$t('支持上传 .yaml 和 .yml 格式的文件')"
                      @click="handleUpload"
                    >
                      <template #icon>
                        <Upload
                          color="#979BA5"
                          height="16"
                          width="16"
                        />
                      </template>
                    </IconButton>
                  </template>
                </MsEditor>
              </Loading>
            </div>
            <HelmErrorStatus
              v-if="editable && (!!validateData?.length || !!yamlErrorLines.length)"
              ref="errorStatusRef"
              :data="validateData"
              :error-lines="yamlErrorLines"
            />
          </div>
        </template>
      </ResizeLayout>
      <slot name="footer"></slot>
    </div>
  </div>
  <!-- 文件选择 -->
  <input
    ref="fileInputRef"
    accept=".yaml,.yml"
    style="display: none"
    type="file"
    @change="handleFileSelect"
  />

  <!-- 版本列表侧边栏 -->
  <VersionListSideslider
    v-model:visible="showVersionListSideslider"
    :config-file-list="fileList"
    :current-file="latestFileInfo"
    @refresh="handleVersionRefresh"
    @rollback="handleVersionRollback"
  />
</template>
<script setup lang="ts">
  import { computed, nextTick, onMounted, ref, watch } from 'vue';

  import { Divider, Exception, Loading, Message, Popover, ResizeLayout, Select, Tag } from 'bkui-vue';
  import { InfoLine, Upload } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import CollapsiblePanel from '~/components/collapsible-panel.vue';
  import { isHelmLikeAppType } from '~/composables/app-type';
  import { formatRelativeTimeWithTooltip } from '~/composables/use-time';
  import { useAppDetail } from '~/stores/app-detail';

  import VersionListSideslider from '../../detail/app-config/version-list-sideslider.vue';
  import HelmErrorStatus from './helm-error-status.vue';
  import HelmFileList from './helm-file-list.vue';
  import Variables from './variables.vue';

  import type {
    AppConfigFileOutputObj,
    ArrgResultItemOutputObj,
    GetAppConfigFileDetailsOutput,
    ValidateArrgValuesYAMLOutputObj,
  } from '~/@types/v1/app-config-files';
  import type { ChartVersionOutputObj } from '~/@types/v1/helm-charts';
  import type { IMonacoEditorErrorMarkerItem } from '~/common/util';

  interface ExtendedBaseContentInfo {
    [key: string]: boolean | string | undefined;
    bizName?: string;
    name?: string;
    serviceName?: string;
  }

  interface FileDetailChangeEvent {
    fileDetail: GetAppConfigFileDetailsOutput | null;
    fileInfo: AppConfigFileOutputObj;
  }

  interface IEmit {
    (e: 'validate-change', isValidate: boolean): void;
    (e: 'current-file-change', fileInfo: AppConfigFileOutputObj | null): void;
  }

  const props = defineProps({
    editable: {
      type: Boolean,
      default: true,
    },
    workspace: {
      type: String,
      default: '',
    },
  });
  const emits = defineEmits<IEmit>();
  const appDetailStore = useAppDetail();
  const { t } = useI18n();

  const isLoading = ref<boolean>(false);
  const errorStatusRef = ref<InstanceType<typeof HelmErrorStatus>>();
  const editorLayoutRef = ref<InstanceType<typeof ResizeLayout> | null>(null);
  const helmFileListRef = ref<InstanceType<typeof HelmFileList> | null>(null);
  const showType = ref<'completeValues' | 'history' | 'values' | 'variables'>();
  const isFileListCollapsed = ref<boolean>(false);
  const isReadonlyFile = computed(() => {
    return !!curFileInfo.value?.bscpConfig;
  });

  // 版本列表侧边栏
  const showVersionListSideslider = ref(false);

  // 覆盖层 Popover
  const overlayPopoverData = computed(() => {
    const baseFile = fileList.value.find(file => file.id === curFileInfo.value?.baseAppConfigFileID);
    return [
      {
        label: t('基础 values'),
        value: baseFile?.name,
      },
    ];
  });

  // BSCP Popover
  const bscpPopoverData = computed(() => {
    const {
      bizName = '',
      serviceName = '',
      name = '',
    } = (curFileDetail.value?.baseContentInfo as ExtendedBaseContentInfo) || {};
    const { bizID = '' } = curFileInfo.value?.bscpConfig || {};
    return [
      {
        label: t('内容来源'),
        value: t('服务配置中心（BSCP）'),
      },
      {
        label: t('业务'),
        value: `${bizName} (${bizID})`,
      },
      {
        label: t('服务'),
        value: `${t('服务名称')} (${serviceName})`,
      },
      {
        label: t('配置项'),
        value: name,
      },
    ];
  });

  function handleHideAside() {
    if (!showType.value && editorLayoutRef.value) {
      editorLayoutRef.value.asideRef.hidden = true;
    }
  }

  function handleLoadingChange(loading: boolean) {
    isLoading.value = loading;
  }

  function toggleAsideShow(type: typeof showType.value) {
    const shouldHideAside = showType.value === type;
    showType.value = shouldHideAside ? undefined : type;

    if (editorLayoutRef.value?.asideRef) {
      editorLayoutRef.value.asideRef.hidden = shouldHideAside;
    }
    if (type === 'completeValues' && !shouldHideAside) {
      getCompleteValues();
    }
  }

  // 默认values版本列表
  const version = ref<string>('');
  const versionLoading = ref<boolean>(false);
  const versionList = ref<Array<ChartVersionOutputObj>>([]);
  const values = ref<string>('');

  // 获取默认values版本
  async function getListChartVersions() {
    version.value = '';
    values.value = '';
    versionList.value = [];
    if (!props.workspace || !appDetailStore.app || !isHelmLikeAppType(appDetailStore.appDetail?.type)) return;
    versionLoading.value = true;
    versionList.value = await ApiServerService.ListChartVersions({ appID: appDetailStore.appID }).catch(() => []);
    versionLoading.value = false;
    version.value = versionList.value[0]?.name ?? '';
    if (version.value) {
      handleVersionChange();
    }
  }

  // 切换默认values版本
  const versionContentLoading = ref(false);
  async function handleVersionChange() {
    if (!version.value) return;
    const versionItem = versionList.value.find(item => item.name === version.value);
    if (!versionItem) {
      values.value = '';
      return;
    }

    versionContentLoading.value = true;
    values.value = await ApiServerService.GetValuesFile({
      appID: appDetailStore.appID,
      chartVersion: versionItem.name || '',
    }).catch(() => '');
    versionContentLoading.value = false;
  }

  const overrideValues = ref('');
  // 存储未修改内容用于比较
  const originalOverrideValues = ref('');

  // 内容是否有变化
  const hasUnsavedChanges = computed(() => {
    return overrideValues.value !== originalOverrideValues.value;
  });

  // 点击校验
  const validateData = ref<ArrgResultItemOutputObj[]>([]);

  // yaml校验
  const yamlErrorLines = ref<IMonacoEditorErrorMarkerItem[]>([]);
  const isValidate = computed(() => !yamlErrorLines.value?.length);
  // 清空校验异常信息
  function clearValidationMessage() {
    validateData.value = [];
    yamlErrorLines.value = [];
  }

  function getValue() {
    return overrideValues.value;
  }

  function handleYamlError(data: IMonacoEditorErrorMarkerItem[]) {
    yamlErrorLines.value = data;
  }

  // 滚动到异常信息组件
  function scrollErrorView() {
    nextTick(() => {
      if (!!validateData.value?.length || !!yamlErrorLines.value.length) {
        errorStatusRef.value?.$el?.scrollIntoView();
      }
    });
  }

  // 文件列表
  const fileList = ref<AppConfigFileOutputObj[]>([]);
  // 文件信息
  const curFileInfo = ref<AppConfigFileOutputObj | null>(null);
  const curFileDetail = ref({} as GetAppConfigFileDetailsOutput | null);

  /** 从 fileList 实时获取当前文件最新数据（保证 currentVersion 等字段始终最新，用于版本列表等需要乐观锁的场景） */
  const latestFileInfo = computed(() => {
    if (!curFileInfo.value?.id) return null;
    return fileList.value.find(file => file.id === curFileInfo.value?.id) ?? null;
  });

  function handleFileDetailChange(data: FileDetailChangeEvent) {
    curFileInfo.value = data.fileInfo;
    curFileDetail.value = data.fileDetail;
    updateEditorContent(data.fileDetail);
    clearValidationMessage();

    if (showType.value === 'completeValues') {
      if (data.fileInfo?.type === 'overlay') {
        getCompleteValues();
      } else {
        showType.value = undefined;
        if (editorLayoutRef.value?.asideRef) {
          editorLayoutRef.value.asideRef.hidden = true;
        }
      }
    }
  }

  // 文件列表变化
  function handleFileListChange(list: AppConfigFileOutputObj[]) {
    fileList.value = list;
    if (!list.length) {
      curFileInfo.value = null;
      overrideValues.value = '';
      originalOverrideValues.value = '';
    }
    nextTick(() => {
      handleHideAside();
    });
  }

  // 更新 values 内容
  function updateEditorContent(fileDetail: GetAppConfigFileDetailsOutput | null) {
    overrideValues.value = '';
    if (fileDetail) {
      const { editableContentField, content, overlayContent } = fileDetail;
      switch (editableContentField) {
        case 'content':
          overrideValues.value = content || '';
          break;
        case 'overlayContent':
          overrideValues.value = overlayContent || '';
          break;
      }
    }
    originalOverrideValues.value = overrideValues.value;
  }

  // 完整 values
  const completeValues = ref<string>('');
  const completeValuesLoading = ref<boolean>(false);
  // 传入覆盖层的值，获取完整 values
  async function getCompleteValues() {
    completeValues.value = '';
    completeValuesLoading.value = true;
    try {
      const res = await ApiServerService.PreviewOverlayMerge({
        appID: appDetailStore.appID,
        id: curFileInfo.value?.baseAppConfigFileID || '',
        overlayContent: curFileInfo.value?.bscpConfig ? '' : overrideValues.value,
      });
      if (res) {
        completeValues.value = res;
      }
    } catch (error) {
      console.error(error);
      completeValues.value = '';
    } finally {
      completeValuesLoading.value = false;
    }
  }

  const fileInputRef = ref<HTMLInputElement>();

  // 处理文件选择
  async function handleFileSelect(event: Event) {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];

    if (file) {
      try {
        const content = await file.text();
        overrideValues.value = content;
      } catch (err) {
        Message({
          theme: 'error',
          message: t('上传失败') + err,
        });
      }
      input.value = '';
    }
  }

  /** 打开历史版本侧边栏 */
  function handleShowVersionList() {
    if (isReadonlyFile.value) return;
    showVersionListSideslider.value = true;
  }

  function handleUpload() {
    fileInputRef.value?.click();
  }

  /** 版本列表刷新回调，刷新文件列表以更新 currentVersion */
  function handleVersionRefresh() {
    helmFileListRef.value?.fetchFileList();
  }

  /** 版本回滚成功回调，刷新当前文件内容 */
  function handleVersionRollback() {
    helmFileListRef.value?.fetchFileList();
    helmFileListRef.value?.refetchCurrentFile();
  }

  watch(
    isValidate,
    (newValue, oldValue) => {
      if (newValue === oldValue) return;
      emits('validate-change', isValidate.value);
    },
    { immediate: true },
  );

  watch(
    latestFileInfo,
    newValue => {
      emits('current-file-change', newValue ?? curFileInfo.value);
    },
    { immediate: true },
  );

  watch(
    [() => appDetailStore.appDetail, () => props.workspace],
    async () => {
      if (isHelmLikeAppType(appDetailStore.appDetail?.type) && appDetailStore.appID) {
        await getListChartVersions();
      }
    },
    { immediate: true },
  );

  onMounted(() => {
    handleHideAside();
  });

  // 保存成功，重置状态
  function markAsSaved() {
    originalOverrideValues.value = overrideValues.value;
  }

  /** 刷新文件列表（供外部调用，如保存后） */
  function refreshFileList() {
    return helmFileListRef.value?.fetchFileList();
  }

  function updateValidationData(data: ValidateArrgValuesYAMLOutputObj) {
    validateData.value = (Object.keys(data) as Array<keyof typeof data>)
      .map(key => data[key])
      .filter((item): item is ArrgResultItemOutputObj => !!item?.skippedReason);
    scrollErrorView();
  }

  defineExpose({
    getValue,
    markAsSaved,
    updateValidationData,
    refreshFileList,
  });
</script>
<style scoped>
  :deep(.multi-values-tag-popover) {
    padding: 8px !important;
  }
  :deep(.bk-resize-layout-aside) {
    border: 0px;
  }
</style>
