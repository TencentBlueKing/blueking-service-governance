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
  <div class="relative h-full flex flex-col">
    <FlexRow class="mt-[16px] px-[16px]">
      <template #left>
        <span class="text-[#191929] font-bold text-[14px]">{{ `Values ${$t('文件列表')}` }}</span>
        <InfoLine
          v-bk-tooltips="{
            content: $t(
              'Helm 应用部署时需指定 values 文件（默认使用 chart 自带），用户可创建多份 values 文件，以适配不同部署环境的个性化需求。',
            ),
            extCls: '!w-[400px]',
          }"
          class="ml-[6px] text-[16px] text-[#979BA5] transform translate-y-[2px]"
        />
      </template>
      <template #right>
        <div
          class="flex items-center cursor-pointer text-[#3A84FF] text-[12px]"
          @click="handleAddFile"
        >
          <Plus
            :height="24"
            :width="24"
          />
          {{ $t('添加文件') }}
        </div>
      </template>
    </FlexRow>

    <Loading
      class="h-full min-h-0"
      :loading="loading"
    >
      <Exception
        v-if="fileList.length === 0"
        class="pt-[120px] text-center normal-exception"
        scene="part"
        type="empty"
      >
        <template #type>
          <img src="/empty.svg" />
        </template>
        <span class="text-[#4D4F56]">{{ $t('暂无文件') }}，</span>
        <Button
          text
          theme="primary"
          @click="handleAddFile"
        >
          {{ $t('立即添加') }}
        </Button>
      </Exception>
      <div
        v-else
        class="h-full text-[12px] text-[#313238]"
      >
        <Tree
          :key="treeUpdateKey"
          children="children"
          class="py-[16px]"
          :data="treeFileList"
          expand-all
          :node-key="'id'"
          :prefix-icon="getPrefixIcon"
          :selected="defaultActiveFileIds"
        >
          <template #node="node">
            <div
              :class="['flex items-center justify-between pr-[12px] cursor-pointer group flex-1']"
              @click="handleSelectFile(node)"
            >
              <div
                class="flex-1 flex items-center ellipsis"
                :class="{ 'font-bold': activeFileId === node.id }"
              >
                <i
                  class="bkms-icon mr-[12px] text-[14px]"
                  :class="[
                    !node.baseAppConfigFileID ? 'bkms-icon-file' : 'bkms-icon-shezhi',
                    activeFileId === node.id ? 'text-[#3A84FF]' : 'text-[#979BA5]',
                  ]"
                ></i>
                <span class="ellipsis">
                  <OverflowTitle type="tips">{{ node.name }}</OverflowTitle>
                </span>
              </div>
              <div class="group-hover:hidden flex items-center">
                <Tag
                  v-if="node.bscpConfig"
                  class="!bg-[#FDEED8] ml-[4px] h-[16px] text-[10px] px-[4px]"
                  theme="warning"
                  >BSCP</Tag
                >
              </div>
              <div
                :class="[
                  'items-center gap-[6px]',
                  activePopConfirmId === node.id ? 'flex' : 'hidden group-hover:flex group-focus-within:flex',
                ]"
                @click.stop
              >
                <!-- 编辑 -->
                <EditLine
                  class="cursor-pointer text-[#979BA5] hover:text-[#3A84FF] p-[3px]"
                  @click.stop="handleEditFile(node)"
                >
                </EditLine>
                <!-- 删除 -->
                <PopConfirm
                  :title="$t('确认删除该文件？')"
                  trigger="click"
                  width="280"
                  @cancel="popConfirmHide"
                  @confirm="handleDeleteFile(node)"
                >
                  <template #content>
                    <div class="pb-[16px] text-[12px]">
                      <div class="mb-[4px]">
                        {{ $t('文件名称') }}：
                        <span class="text-[#313238]">{{ node?.name }}</span>
                      </div>
                      {{ $t('删除操作无法撤回，请谨慎操作！') }}
                    </div>
                  </template>
                  <Del
                    class="cursor-pointer text-[#979BA5] hover:text-[#EA3636] p-[3px]"
                    :class="{ 'opacity-100': activePopConfirmId === node.id }"
                    @click="popConfirmShow(node.id)"
                  >
                  </Del>
                </PopConfirm>
              </div>
            </div>
          </template>
        </Tree>
      </div>
    </Loading>

    <!-- 文件表单侧边栏 -->
    <FileFormSideslider
      v-model:visible="fileDialogVisible"
      :base-file-options="baseFileOptions"
      :current-file="currentFile"
      :is-edit="isEdit"
      :loading="submitLoading"
      :workspace-name="spaceStore.currentSpace"
      @cancel="handleCancel"
      @submit="handleFormSubmit"
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Button, Exception, Loading, Message, OverflowTitle, PopConfirm, Tag, Tree } from 'bkui-vue';
  import { Del, EditLine, InfoLine, Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import {
    AppConfigFileOutputObj,
    CreateAppConfigFileRequest,
    GetAppConfigFileDetailsOutput,
    ListAppConfigFilesOutput,
    UpdateAppConfigFileRequest,
  } from '~/@types/v1/app-config-files';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';

  import FileFormSideslider from './file-form-sideslider.vue';

  interface FileDetailChangeEvent {
    fileDetail: GetAppConfigFileDetailsOutput | null;
    fileInfo: AppConfigFileOutputObj;
  }

  interface FileFormData {
    baseAppConfigFileID?: string;
    contentSourceType: 'bscp' | 'local';
    name: string;
    type: 'normal' | 'overlay';
  }

  interface Props {
    hasUnsavedChanges?: boolean;
  }

  interface TreeFileNode extends AppConfigFileOutputObj {
    children: TreeFileNode[];
  }

  const props = withDefaults(defineProps<Props>(), {
    hasUnsavedChanges: false,
  });

  const emits = defineEmits<{
    'file-detail-change': [data: FileDetailChangeEvent];
    'file-list-change': [fileList: AppConfigFileOutputObj[]];
    'loading-change': [loading: boolean];
  }>();

  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  const spaceStore = useSpaceStore();
  const { confirmBox } = useLeaveConfirm();

  // 响应式数据
  const loading = ref(false);
  const fileList = ref<AppConfigFileOutputObj[]>([]);
  const fileDialogVisible = ref(false);
  const submitLoading = ref(false);
  const isEdit = ref(false);
  const currentFile = ref<AppConfigFileOutputObj | null>(null);
  const activeFileId = ref<null | string>(null); // 当前激活的文件ID
  const activePopConfirmId = ref<string>(''); // 当前显示PopConfirm的文件ID
  // 文件展示树数据
  const treeFileList = ref<TreeFileNode[]>([]);
  const defaultActiveFileIds = ref<string[]>([]);
  // fetchFileList的key，用于强制更新树形结构
  const treeUpdateKey = ref();

  // 基础values选项
  const baseFileOptions = computed(() =>
    fileList.value.filter(file => file.type === 'normal').map(file => ({ id: file.id || '', name: file.name || '' })),
  );

  const getPrefixIcon = (item: AppConfigFileOutputObj, renderType: string) => {
    return renderType === 'node_action' ? 'default' : item;
  };

  // 将文件列表转换为树形结构
  function convertToTreeStructure(files: AppConfigFileOutputObj[]): TreeFileNode[] {
    const fileMap = new Map<string, TreeFileNode>();

    files.forEach(file => {
      fileMap.set(file.id || '', { ...file, children: [] });
    });

    // 区分基础层和覆盖层
    const rootFiles: TreeFileNode[] = [];

    files.forEach(file => {
      const fileWithChildren = fileMap.get(file.id || '')!;
      if (file.type === 'overlay' && file.baseAppConfigFileID) {
        const parentFile = fileMap.get(file.baseAppConfigFileID);
        if (parentFile) {
          // 将覆盖层文件添加到基础层的 children 中
          parentFile.children.push(fileWithChildren);
        } else {
          rootFiles.push(fileWithChildren);
        }
      } else {
        rootFiles.push(fileWithChildren);
      }
    });
    return rootFiles;
  }

  // 获取文件详情
  async function fetchFileDetail(fileId: string) {
    const fileInfo = fileList.value.find(
      (item: AppConfigFileOutputObj) => item.id === fileId,
    ) as AppConfigFileOutputObj;
    try {
      const fileDetail: GetAppConfigFileDetailsOutput = await ApiServerService.GetAppConfigFileDetails(
        {
          appID: appDetailStore.appID,
          id: fileId,
        },
        { needRes: true },
      );

      // 发送数据给父组件
      emits('file-detail-change', {
        fileDetail,
        fileInfo,
      });
      return fileDetail;
    } catch {
      emits('file-detail-change', {
        fileDetail: null,
        fileInfo,
      });
    } finally {
      emits('loading-change', false);
    }
  }

  // 获取文件列表
  async function fetchFileList() {
    try {
      loading.value = true;
      const ret: ListAppConfigFilesOutput = await ApiServerService.ListAppConfigFiles(
        { appID: appDetailStore.appID },
        { needRes: true },
      );
      fileList.value = ret?.items || [];
      treeFileList.value = convertToTreeStructure(fileList.value);
      emits('file-list-change', fileList.value);
      // 默认激活第一项
      if (!activeFileId.value && fileList.value?.length) {
        handleSelectFile(fileList.value[0]);
        defaultActiveFileIds.value = [fileList.value[0]?.id || ''];
      } else {
        defaultActiveFileIds.value = activeFileId.value ? [activeFileId.value] : [];
      }
      treeUpdateKey.value += 1;
    } finally {
      loading.value = false;
    }
  }

  // 添加文件
  function handleAddFile() {
    isEdit.value = false;
    currentFile.value = null;
    fileDialogVisible.value = true;
  }

  // 添加文件
  async function handleAddFileSubmit(params: Partial<CreateAppConfigFileRequest>, formData: FileFormData) {
    const result = await ApiServerService.CreateAppConfigFile({
      ...params,
      type: formData.type,
    } as CreateAppConfigFileRequest)
      .then(() => true)
      .catch(() => false);
    if (result) {
      Message({
        theme: 'success',
        message: t('添加成功'),
      });
      fileDialogVisible.value = false;
      await fetchFileList();
      // 高亮新添加的文件
      const newFile = fileList.value.find(file => file.name === formData.name);
      if (newFile) {
        handleSelectFile(newFile);
      }
    }
  }

  // 取消操作
  function handleCancel() {
    fileDialogVisible.value = false;
    isEdit.value = false;
    currentFile.value = null;
  }

  // 删除文件
  async function handleDeleteFile(file: AppConfigFileOutputObj) {
    try {
      submitLoading.value = true;
      await ApiServerService.DeleteAppConfigFile({
        appID: appDetailStore.appID,
        id: file.id || '',
      });
      // 如果删除的是当前激活文件，清除激活状态
      if (activeFileId.value === file.id) {
        activeFileId.value = null;
      }
      Message({
        theme: 'success',
        message: t('删除成功'),
      });
      // 刷新列表
      await fetchFileList();
    } finally {
      submitLoading.value = false;
      // 重置PopConfirm状态
      activePopConfirmId.value = '';
    }
  }

  // 编辑文件
  function handleEditFile(file: AppConfigFileOutputObj) {
    currentFile.value = file;
    isEdit.value = true;
    fileDialogVisible.value = true;
  }

  // 编辑文件
  async function handleEditFileSubmit(params: Partial<CreateAppConfigFileRequest>) {
    const result = await ApiServerService.UpdateAppConfigFile({
      ...params,
      id: currentFile.value?.id || '',
    } as UpdateAppConfigFileRequest)
      .then(() => true)
      .catch(() => false);
    if (result) {
      Message({
        theme: 'success',
        message: t('更新成功'),
      });
      fileDialogVisible.value = false;
      await fetchFileList();
      await fetchFileDetail(currentFile.value?.id || '');
    }
  }

  // 表单提交处理
  async function handleFormSubmit(formData: FileFormData) {
    submitLoading.value = true;
    const baseAppConfigFileID = formData.type === 'overlay' ? formData.baseAppConfigFileID || '' : '';
    const params: Partial<CreateAppConfigFileRequest> = {
      appID: appDetailStore.appID,
      baseAppConfigFileID,
      fileFormat: appDetailStore.appType === 'taf' ? 'taf' : 'yaml',
      ...formData,
    };

    try {
      if (isEdit.value) {
        await handleEditFileSubmit(params);
      } else {
        await handleAddFileSubmit(params, formData);
      }
    } finally {
      submitLoading.value = false;
    }
  }

  // 选择文件（激活状态）
  async function handleSelectFile(file: AppConfigFileOutputObj) {
    // 当前文件有未保存的更改，显示确认对话框
    if (props.hasUnsavedChanges && activeFileId.value !== file.id) {
      const shouldLeave = await confirmBox();
      if (!shouldLeave) return;
    }

    activeFileId.value = file.id || null;
    emits('loading-change', true);
    try {
      // BSCP-来源文件
      if (file.contentSourceType === 'bscp' && file.bscpConfig) {
        await loadConfigContent(file.bscpConfig);
      } else {
        await fetchFileDetail(file.id || '');
      }
    } finally {
      emits('loading-change', false);
    }
  }

  // 加载覆盖层-bcsp配置项内容
  async function loadConfigContent(config: NonNullable<AppConfigFileOutputObj['bscpConfig']>) {
    const fileInfo = fileList.value.find(item => item.id === activeFileId.value) as AppConfigFileOutputObj;
    try {
      const res = await ApiServerService.GetBSCPConfig({
        bizID: config.bizID,
        serviceID: config.serviceID,
        configID: config.id,
      });
      const fileDetail = {
        editableContentField: 'overlayContent',
        content: res?.content || '',
        overlayContent: res?.content || '',
        baseContentInfo: {
          holderID: config.id,
          holderName: res?.name || '',
          holderContentSourceType: 'bscp',
          content: res?.content || '',
          isFromAnotherFile: false,
          bizName: res?.bizName,
          serviceName: res?.serviceName,
          // 配置项 name
          name: res?.name,
        },
      };
      emits('file-detail-change', {
        fileDetail,
        fileInfo,
      });
    } catch {
      emits('file-detail-change', {
        fileDetail: null,
        fileInfo,
      });
    } finally {
      emits('loading-change', false);
    }
  }

  function popConfirmHide() {
    activePopConfirmId.value = '';
  }

  // PopConfirm 显示和隐藏事件处理
  function popConfirmShow(fileId: string) {
    activePopConfirmId.value = fileId;
  }

  watch(
    () => appDetailStore.appID,
    newVal => {
      if (newVal) {
        fetchFileList();
      }
    },
    { immediate: true },
  );

  /** 刷新当前选中文件的详情（供外部调用，如版本回滚后） */
  function refetchCurrentFile() {
    if (activeFileId.value) {
      fetchFileDetail(activeFileId.value);
    }
  }

  defineExpose({
    refetchCurrentFile,
    fetchFileList,
  });
</script>

<style scoped lang="postcss">
  :deep(.bk-tree) {
    .bk-node-row {
      padding-left: 12px;
      &:hover {
        background-color: #f0f1f5;
      }
    }
    .bk-node-prefix {
      transform: translateY(2px);
    }
  }
</style>
