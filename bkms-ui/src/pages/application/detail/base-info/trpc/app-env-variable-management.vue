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
  <div class="flex flex-col gap-[16px] p-[16px] bg-[#FFF] shadow-[0_2px_4px_0_#1919290d]">
    <Alert theme="info">
      {{ $t('应用变量优先级最高，同名时将覆盖该应用继承的环境级别变量。') }}
      <Button
        text
        theme="primary"
        @click="showEnvBgVarsSlider = true"
      >
        {{ $t('查看环境级变量') }}
      </Button>
    </Alert>
    <div class="flex items-center justify-between">
      <div class="flex gap-[8px]">
        <Button
          theme="primary"
          @click="handleAddVariable"
        >
          <Plus
            :height="24"
            :width="24"
          />
          {{ $t('新增应用变量') }}
        </Button>
        <Button @click="showImportSlider = true">
          <i class="bkms-icon bkms-icon-daoru mr-[6px] text-[#979BA5]"></i>
          {{ $t('导入') }}
        </Button>
        <Button @click="handleShowExportDialog">
          <i class="bkms-icon bkms-icon-daochu mr-[6px] text-[#979BA5]"></i>
          {{ $t('导出') }}
        </Button>
      </div>
      <Input
        v-model.trim="searchKeyword"
        class="w-[420px]"
        clearable
        :placeholder="$t('搜索变量名、变量值、描述')"
      >
        <template #suffix>
          <Search class="text-[16px] text-[#979BA5] mr-[6px] mt-[2px] hover:text-[#3A84FF]" />
        </template>
      </Input>
    </div>
    <EditableVariableTable
      ref="editableTableRef"
      :editable="true"
      :list="variableList"
      @add="handleAdd"
      @delete="handleDelete"
      @edit="handleEdit"
    />

    <!-- 环境级变量侧栏 -->
    <EnvBgVarsSideslider
      v-model:visible="showEnvBgVarsSlider"
      :alert-text="$t('应用从环境继承的所有生效变量（包含环境自定义、公共及系统内置变量）。')"
      :app-id="appDetailStore.appID"
      source="app"
      :title="$t('环境级变量')"
    />

    <!-- 应用环境变量导入侧栏 -->
    <EnvVarsImportSideslider
      v-model:visible="showImportSlider"
      :download-template-request="handleDownloadTemplateRequest"
      :import-request="handleImportRequest"
      :preview-request="handlePreviewRequest"
      :target-label="$t('应用')"
      :target-name="appName"
      template-filename="app-env-vars-template.env"
      @success="fetchAppEnvVarList"
    />

    <!-- 应用环境变量导出弹窗 -->
    <Dialog
      v-model:is-show="showExportDialog"
      :width="560"
      @closed="resetExportForm"
    >
      <template #header>
        <div class="flex items-center gap-[12px]">
          <span>{{ $t('导出环境变量') }}</span>
          <Divider
            class="h-[16px] mx-[0] text-[12px]"
            color="#DCDEE5"
            direction="vertical"
            type="solid"
          />
          <span class="text-[14px] font-normal text-[#979BA5]">{{ $t('应用') }}：{{ appName }}</span>
        </div>
      </template>
      <Form
        ref="exportFormRef"
        class="mt-[8px]"
        form-type="vertical"
        :model="exportForm"
      >
        <Form.FormItem
          :label="$t('导出范围')"
          property="scope"
          required
        >
          <Radio.Group v-model="exportForm.scope">
            <Radio label="appDefined">{{ $t('应用自定义变量') }}</Radio>
            <Radio
              class="ml-[24px]"
              label="effectiveByEnv"
            >
              {{ $t('导出指定环境生效全集') }}
            </Radio>
          </Radio.Group>
        </Form.FormItem>
        <Form.FormItem
          v-if="exportForm.scope === 'effectiveByEnv'"
          :label="$t('选择环境')"
          property="envName"
          required
        >
          <Select
            v-model="exportForm.envName"
            :loading="envListLoading"
            :placeholder="$t('请选择环境')"
          >
            <Select.Option
              v-for="env in envList"
              :key="env.id || env.name"
              :label="env.displayName || env.name"
              :value="env.name"
            />
          </Select>
        </Form.FormItem>
      </Form>
      <template #footer>
        <Button
          :loading="isExporting"
          theme="primary"
          @click="handleExport"
        >
          {{ $t('确定') }}
        </Button>
        <Button
          class="ml-[8px]"
          :disabled="isExporting"
          @click="showExportDialog = false"
        >
          {{ $t('取消') }}
        </Button>
      </template>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Alert, Button, Dialog, Divider, Form, Input, Message, Radio, Select } from 'bkui-vue';
  import { Plus, Search } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { EnvService, EnvvarsService } from '~/api/modules/v1';
  import { sortByDate } from '~/common/util';
  import EditableVariableTable, { type EnvVariableConfig } from '~/components/editable-variable-table/index.vue';
  import EnvVarsImportSideslider from '~/components/env-vars-import-sideslider.vue';
  import { useFileExport } from '~/composables/use-file-export';
  import { type IInputKey, useTableSearchInput } from '~/composables/use-search';
  import { useAppDetail } from '~/stores/app-detail';

  import EnvBgVarsSideslider from './env-bg-vars-sideslider.vue';

  import type { EnvOutput } from '~/@types/v1/env';
  import type {
    AppEnvVarDetailedOutputObj,
    DownloadAppEnvVarTemplateRequest,
    ExportAppEnvVarsRequest,
  } from '~/@types/v1/envvars';

  const appDetailStore = useAppDetail();
  const { t } = useI18n();
  const appName = computed(() => appDetailStore.app || appDetailStore.appID);
  const showImportSlider = ref(false);
  const { exportFile, isExporting } = useFileExport();
  type ExportScope = 'appDefined' | 'effectiveByEnv';
  const showExportDialog = ref(false);
  const exportFormRef = ref<InstanceType<typeof Form>>();
  const exportForm = ref<{ envName: string; scope: ExportScope }>({
    scope: 'appDefined',
    envName: '',
  });
  const envList = ref<EnvOutput[]>([]);
  const envListLoading = ref(false);

  // 自定义环境变量列表 - 通过 ListDetailedAppEnvVars 接口获取
  const customEnvVarList = ref<EnvVariableConfig[]>([]);
  async function fetchAppEnvVarList() {
    if (!appDetailStore.appID) return;
    const list: AppEnvVarDetailedOutputObj[] = await EnvvarsService.listDetailedAppEnvVars({
      appID: appDetailStore.appID,
    }).catch(() => []);
    sortByDate(list, item => item.appEnvVar?.createdAt);
    customEnvVarList.value = (list || []).map((item: AppEnvVarDetailedOutputObj) => ({
      key: item.appEnvVar?.key || '',
      value: item.appEnvVar?.value || '',
      description: item.appEnvVar?.description || '',
      isSensitive: item.appEnvVar?.isSensitive ?? false,
      conflictedInfo: item.conflictedInfo || undefined,
    }));
  }

  async function handleAdd(data: EnvVariableConfig) {
    if (!appDetailStore.appID) return;
    await EnvvarsService.createAppEnvVars({
      appID: appDetailStore.appID,
      key: data.key,
      value: data.value,
      description: data.description,
      isSensitive: data.isSensitive,
    });
    Message({ theme: 'success', message: t('新增成功') });
    await fetchAppEnvVarList();
  }
  async function handleDelete(data: EnvVariableConfig) {
    if (!appDetailStore.appID) return;
    await EnvvarsService.deleteAppEnvVars({
      appID: appDetailStore.appID,
      key: data.key,
    });
    Message({ theme: 'success', message: t('删除成功') });
    await fetchAppEnvVarList();
  }
  async function handleEdit(data: EnvVariableConfig, originalData: EnvVariableConfig) {
    if (!appDetailStore.appID) return;
    try {
      await EnvvarsService.updateAppEnvVars({
        appID: appDetailStore.appID,
        key: originalData.key, // 旧 key（路径参数）
        updatedKey: data.key, // 新 key（可能重命名）
        value: data.value,
        description: data.description,
        isSensitive: data.isSensitive,
      });
      Message({ theme: 'success', message: t('修改成功') });
      await fetchAppEnvVarList();
    } catch (error: unknown) {
      console.warn('error', error);
      // 更新失败，还原该行为编辑前的原始数据
      editableTableRef.value?.restoreData(customEnvVarList.value);
    }
  }

  // 暴露刷新方法供父组件调用
  function refreshData() {
    fetchAppEnvVarList();
  }

  const searchKeys = ref<IInputKey[]>([
    { field: 'key', id: 'key', fuzzy: true },
    { field: 'value', id: 'value', fuzzy: true },
    { field: 'description', id: 'description', fuzzy: true },
  ]);
  const { searchValue: searchKeyword, tableDataMatchSearch: filteredEnvVars } = useTableSearchInput(
    customEnvVarList,
    searchKeys,
  );

  // 展示的变量列表
  const variableList = computed(() => filteredEnvVars.value);

  const editableTableRef = ref<InstanceType<typeof EditableVariableTable>>();
  // 添加变量
  function handleAddVariable() {
    editableTableRef.value?.addVariable();
  }

  function handleDownloadTemplateRequest() {
    return EnvvarsService.downloadAppEnvVarTemplate<DownloadAppEnvVarTemplateRequest, Response>(undefined, {
      originalResponse: true,
    });
  }

  // 导出应用直接定义的环境变量
  async function handleExport() {
    if (!appDetailStore.appID) return;
    const valid = await exportFormRef.value?.validate().catch(() => false);
    if (!valid) return;
    const { envName, scope } = exportForm.value;
    const success = await exportFile({
      request: () =>
        EnvvarsService.exportAppEnvVars<ExportAppEnvVarsRequest, Response>(
          {
            appID: appDetailStore.appID,
            scope,
            ...(scope === 'effectiveByEnv' ? { envName } : {}),
          },
          { originalResponse: true },
        ),
      fallbackFilename: `app-${appName.value}-env-vars.env`,
    });
    if (success) showExportDialog.value = false;
  }

  function handleImportRequest(file: File) {
    return EnvvarsService.importAppDefinedEnvVar(
      { appID: appDetailStore.appID, file },
      { interceptorErr: false, multipart: true },
    );
  }

  function handlePreviewRequest(file: File) {
    return EnvvarsService.previewAppDefinedEnvVar(
      { appID: appDetailStore.appID, file },
      { interceptorErr: false, multipart: true },
    );
  }

  async function handleShowExportDialog() {
    if (!appDetailStore.appID) return;
    showExportDialog.value = true;
    envListLoading.value = true;
    envList.value = await EnvService.listAppEnvs({ appID: appDetailStore.appID }).catch(() => []);
    envListLoading.value = false;
  }

  function resetExportForm() {
    exportForm.value = { scope: 'appDefined', envName: '' };
    exportFormRef.value?.clearValidate();
  }

  // 查看环境级变量侧栏
  const showEnvBgVarsSlider = ref(false);

  // 监听 appID 变化后请求环境变量列表，确保 appID 已就绪
  watch(
    () => appDetailStore.appID,
    newAppID => {
      if (newAppID) fetchAppEnvVarList();
    },
    { immediate: true },
  );

  defineExpose({ refreshData });
</script>
