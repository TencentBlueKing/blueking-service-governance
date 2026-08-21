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
  <Orchestrate
    ref="editorRef"
    class="h-full"
    :editable="editable"
    :workspace="spaceStore.currentSpace"
    @current-file-change="handleCurFileInfoChange"
    @validate-change="handleValidateChange"
  >
    <template #footer>
      <div class="flex items-center mt-[14px] pl-[24px] bg-[#FFFFFF] h-[48px] border border-[1px_solid_#EAEBF0]">
        <Button
          v-bk-tooltips="{
            content: isBscp ? $t('来源于 BSCP 的文件仅能查看，不能编辑') : $t('校验未通过'),
            disabled: !isSaveDisabled,
            placement: 'left',
          }"
          :disabled="isSaveDisabled"
          theme="primary"
          @click="showSaveVersionDialog = true"
          >{{ $t('保存') }}</Button
        >
      </div>
    </template>
  </Orchestrate>

  <!-- 保存版本确认弹窗 -->
  <SaveVersionConfirmDialog
    ref="saveVersionDialogRef"
    v-model:is-show="showSaveVersionDialog"
    :next-version="nextVersion"
    @confirm="handleSaveVersionConfirm"
  />
</template>
<script setup lang="ts">
  import { computed, ref } from 'vue';

  import { Button, Message } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { AppConfigFileOutputObj, UpdateAppConfigFileContentOutput } from '~/@types/v1/app-config-files';
  import { AppConfigFilesService } from '~/api/modules/v1';
  import { hasErrorCode } from '~/common/util';
  import SaveVersionConfirmDialog from '~/pages/application/detail/app-config/components/save-version-confirm-dialog.vue';
  import Orchestrate from '~/pages/application/template/helm-chart/orchestrate.vue';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';

  const { t } = useI18n();
  const spaceStore = useSpaceStore();
  const appDetailStore = useAppDetail();

  const editable = ref<boolean>(true);
  const editorRef = ref<InstanceType<typeof Orchestrate>>();
  const curFileInfo = ref<AppConfigFileOutputObj | null>(null);

  // 保存版本确认弹窗
  const showSaveVersionDialog = ref(false);
  const saveVersionDialogRef = ref<InstanceType<typeof SaveVersionConfirmDialog> | null>(null);

  function handleCurFileInfoChange(fileInfo: AppConfigFileOutputObj | null) {
    curFileInfo.value = fileInfo;
  }

  /** 下一版本号：当前文件的 currentVersion + 1 */
  const nextVersion = computed(() => {
    return Number(curFileInfo.value?.currentVersion || 0) + 1;
  });

  async function handleSave(description: string = '') {
    const value = editorRef.value?.getValue() || '';
    // 本地编辑覆盖层
    if (curFileInfo.value?.type === 'overlay') {
      await updateValuesFileOverlayContent(curFileInfo.value?.id ?? '', value, description);
    } else if (curFileInfo.value?.contentSourceType === 'bscp' && curFileInfo.value?.type === 'normal') {
      // BSCP 覆盖层
      await updateValuesFileOverlayContent(curFileInfo.value?.id ?? '', value, description);
    } else {
      await updateValuesFileContent(curFileInfo.value!.id ?? '', value, description);
    }
  }

  /** 保存版本确认回调 */
  async function handleSaveVersionConfirm(description: string) {
    try {
      await handleSave(description);
      showSaveVersionDialog.value = false;
    } catch (err) {
      saveVersionDialogRef.value?.stopLoading();
      if (hasErrorCode(err, 'APP_CONFIG_FILE_VERSION_CONFLICT')) {
        Message({
          theme: 'error',
          message: t('当前配置已被他人更新。为避免数据被覆盖，请刷新页面获取最新版本后重新编辑。'),
        });
      }
    }
  }

  // 更新 values 文件
  async function updateValuesFile(
    fileId: string,
    updateData: { content?: string; overlayContent?: string },
    description = '',
  ) {
    const isOverlayUpdate = 'overlayContent' in updateData;
    // 普通、覆盖层
    const apiMethod: typeof AppConfigFilesService.updateAppConfigFileContent = isOverlayUpdate
      ? AppConfigFilesService.updateAppConfigFileOverlayContent
      : AppConfigFilesService.updateAppConfigFileContent;

    const curFileInfoRef = curFileInfo.value;
    const requestParams = {
      appID: appDetailStore.appID,
      id: fileId,
      currentVersion: curFileInfoRef?.currentVersion ?? 0,
      description,
      ...updateData,
    };

    const res = (await apiMethod(requestParams, {
      needRes: true,
      interceptorErr: false,
    })) as UpdateAppConfigFileContentOutput;
    Message({
      theme: 'success',
      message: t('文件内容更新成功'),
    });
    editorRef.value?.markAsSaved();
    // 刷新文件列表以更新 currentVersion
    await editorRef.value?.refreshFileList();
    if (res?.arrgData) {
      editorRef.value?.updateValidationData(res.arrgData);
    }
  }

  // 修改文件 values 的 Content
  async function updateValuesFileContent(fileId: string, content: string, description = '') {
    return updateValuesFile(fileId, { content }, description);
  }

  // 修改文件 values 的 overlayContent
  async function updateValuesFileOverlayContent(fileId: string, overlayContent: string, description = '') {
    return updateValuesFile(fileId, { overlayContent }, description);
  }

  // 校验
  const isValidate = ref(false);
  function handleValidateChange(validate: boolean) {
    isValidate.value = validate;
  }

  // 当前文件是否为 BSCP 来源
  const isBscp = computed(() => {
    return !!curFileInfo.value?.bscpConfig;
  });

  const isSaveDisabled = computed(() => {
    return !isValidate.value || isBscp.value;
  });
</script>
