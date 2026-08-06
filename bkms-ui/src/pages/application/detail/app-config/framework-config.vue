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
  <div class="h-full px-[24px] pt-[20px] overflow-auto flex flex-col">
    <Skeleton
      :full-height="false"
      :loading="isLoading || appDetailStore.loading"
    >
      <template #loading>
        <Layout.shape
          :height="28"
          width="100%"
        />
        <div class="grid grid-cols-2 gap-4 gap-y-2 my-[16px] pl-[60px]">
          <Layout.formItem />
          <Layout.formItem />
        </div>
        <Layout.shape
          :height="28"
          width="100%"
        />
        <div class="flex-1 min-h-0 px-[24px] pb-[16px]">
          <div class="flex justify-between">
            <Layout.shape
              class="mt-[16px]"
              :height="28"
              :width="320"
            />
            <Layout.shape
              class="mt-[16px]"
              :height="28"
              :width="120"
            />
          </div>
          <Layout.shape
            class="mt-[16px]"
            :height="300"
            width="100%"
          />
        </div>
      </template>
      <!-- 框架配置文件名 + 路径 -->
      <div class="mb-[16px] flex-shrink-0">
        <BkmsContent
          class="info-title shadow-[0_2px_4px_0_#1919290d]"
          :show-edit-icon="!isFileInfoEditing"
          :title="$t('文件信息')"
          @edit="handleFileInfoEdit"
        >
          <div class="p-[16px] bg-[#fff]">
            <!-- 查看态 -->
            <template v-if="!isFileInfoEditing">
              <div class="grid grid-cols-2 gap-[12px] gap-y-2 pl-[44px]">
                <FieldItem
                  class="min-h-[30px]"
                  :container-height="20"
                  :field-value="$t('框架配置文件')"
                >
                  <template #value>
                    <span class="text-[12px] text-[#313238]">
                      {{ currentFileSpec?.fileName || '--' }}
                    </span>
                  </template>
                </FieldItem>
                <FieldItem
                  class="min-h-[30px]"
                  :container-height="20"
                  :field-value="$t('配置文件路径')"
                >
                  <template #value>
                    <div class="max-w-[220px] text-[#313238]">
                      <OverflowTitle type="tips">
                        {{ currentFileSpec?.filePath || '--' }}
                      </OverflowTitle>
                    </div>
                  </template>
                </FieldItem>
              </div>
            </template>
            <!-- 编辑态 -->
            <div v-else>
              <Form
                ref="fileInfoFormRef"
                form-type="vertical"
                :model="fileInfoFormData"
                :rules="fileInfoFormRules"
              >
                <div class="grid grid-cols-2 gap-[12px] gap-y-2 pl-[44px]">
                  <Form.FormItem
                    :label="$t('框架配置文件')"
                    :property="isFileNameEditable ? 'fileName' : undefined"
                    :required="isFileNameEditable"
                  >
                    <Input
                      v-model.trim="fileInfoFormData.fileName"
                      class="w-full"
                      :maxlength="100"
                      :placeholder="isFileNameEditable ? $t('请输入') : ''"
                      :readonly="!isFileNameEditable"
                    />
                  </Form.FormItem>
                  <Form.FormItem
                    :label="$t('配置文件路径')"
                    property="filePath"
                    required
                  >
                    <Input
                      v-model.trim="fileInfoFormData.filePath"
                      class="w-full"
                      clearable
                      :placeholder="$t('请输入')"
                    />
                  </Form.FormItem>
                </div>
                <div class="!mb-0 pl-[44px]">
                  <Button
                    :loading="isFileInfoSaving"
                    size="small"
                    theme="primary"
                    @click="handleFileInfoSave"
                  >
                    {{ $t('保存') }}
                  </Button>
                  <Button
                    class="ml-[8px]"
                    size="small"
                    @click="handleFileInfoCancel"
                  >
                    {{ $t('取消') }}
                  </Button>
                </div>
              </Form>
            </div>
          </div>
        </BkmsContent>
      </div>

      <!-- YAML 编辑器区域 -->
      <div class="flex-1 min-h-0 pb-[16px]">
        <BkmsContent
          class="info-title shadow-[0_2px_4px_0_#1919290d] h-full flex flex-col yaml-content"
          :show-edit-icon="!isEditing"
          :title="$t('文件内容')"
          @edit="handleStartEdit"
        >
          <template #action>
            <Button
              text
              theme="primary"
              @click="showVersionListSideslider = true"
            >
              <i class="bkms-icon bkms-icon-time-2 text-[14px] mr-[4px] mt-[-1px]"></i>
              {{ $t('版本列表') }}
            </Button>
          </template>
          <div
            v-bkloading="{ loading: isOpenEnvConfigLoading, zIndex: 1000 }"
            class="flex-1 flex flex-col min-h-0 bg-[#fff] p-[16px]"
          >
            <FlexRow
              class="mb-[12px] items-center h-[32px] flex-shrink-0"
              lclass="flex items-center"
            >
              <template #left>
                <div
                  v-if="isEnableEnvConfig"
                  class="flex items-center gap-[6px] mr-[16px]"
                >
                  <!-- 环境配置下拉列表 -->
                  <EnvPerspectiveSelect
                    :env-list="envList"
                    :model-value="currentEnv.name || '__default__'"
                    :modified-env-names="modifiedEnvNames"
                    @change="handleEnvSelectChange"
                  />
                </div>
                <span class="text-[12px] text-[#4D4F56] mr-[6px]">{{ $t('开启按环境配置') }}：</span>
                <Tag
                  v-if="!isEditing"
                  :theme="isEnableEnvConfig ? 'success' : 'default'"
                >
                  {{ isEnableEnvConfig ? $t('已开启') : $t('未开启') }}
                </Tag>
                <PopConfirm
                  v-else
                  :confirm-text="$t('确认开启')"
                  :content="$t('开启后，将无法再切回默认模式，请谨慎操作！')"
                  :disabled="isEnableEnvConfig"
                  :title="$t('确认开启环境配置？')"
                  trigger="click"
                  width="288"
                  @confirm="handleEnableEnvConfig"
                >
                  <Switcher
                    v-bk-tooltips="{
                      content: $t('开启后，暂不支持关闭'),
                      placement: 'right',
                      disabled: !isEnableEnvConfig,
                      delay: 300,
                    }"
                    :disabled="isEnableEnvConfig"
                    size="small"
                    theme="primary"
                    :value="isEnableEnvConfig"
                  />
                </PopConfirm>
              </template>
              <template #right>
                <div class="flex items-center">
                  <template v-if="isEnableEnvConfig && !isDefaultConfig(currentEnv?.name ?? '')">
                    <IconTextButton
                      :active="showType === 'completeValues'"
                      icon="bkms-icon bkms-icon-yanjing-kejian"
                      :text="$t('完整配置')"
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
                    :active="showType === 'variables'"
                    class="text-[12px]"
                    icon="bkms-icon bkms-icon-variable"
                    :text="$t('环境变量')"
                    @click="toggleAsideShow('variables')"
                  />
                </div>
              </template>
            </FlexRow>

            <!-- 编辑器主体 -->
            <div class="flex-1 min-h-0">
              <ResizeLayout
                v-if="isEnableEnvConfig"
                :border="false"
                class="min-h-[0px] h-full editor-aside-layout"
                :initial-divide="asideInitialDivide"
                placement="right"
              >
                <template #aside>
                  <!-- 完整配置 -->
                  <MsHighlightjs
                    v-if="showType === 'completeValues'"
                    class="!h-full ml-[16px]"
                    :code="completeValues"
                    :loading="completeValuesLoading"
                  >
                    <template #title>
                      {{ $t('完整配置') }}
                    </template>
                  </MsHighlightjs>
                  <!-- 开启环境配置-查看环境变量 -->
                  <ViewDefaultEnvVars
                    v-else-if="showType === 'variables'"
                    ref="viewDefaultEnvVarsRef"
                    class="!h-full ml-[16px]"
                    :custom-request-fn="handleGetVarEnv"
                    :env-list="envList"
                    v-bind="viewDefaultEnvVarsProps"
                  >
                    <!-- trpc 提示信息 -->
                    <template
                      v-if="isTrpcApp"
                      #alert
                    >
                      <TrpcEnvVarTip />
                    </template>
                  </ViewDefaultEnvVars>
                </template>
                <template #main>
                  <div class="flex flex-col h-full">
                    <Alert
                      class="mb-[16px] flex-shrink-0"
                      :theme="isDefaultConfig(currentEnv?.name ?? '') ? 'warning' : 'info'"
                    >
                      <span v-if="isDefaultConfig(currentEnv?.name ?? '')">
                        {{ $t('所有环境的基准配置，修改后对所有环境生效。') }}
                      </span>
                      <span v-else>{{ $t('仅定义差异项，与默认配置合并后生成完整配置。') }}</span>
                    </Alert>
                    <ResizeLayout
                      ref="errorRef"
                      :auto-minimize="true"
                      class="flex-1 min-h-0"
                      :disabled="!editorErr.message?.length"
                      :max="300"
                      :min="100"
                      placement="bottom"
                    >
                      <template #aside>
                        <EditorStatus
                          v-show="!!editorErr.message?.length"
                          :message="editorErr.message"
                        />
                      </template>
                      <template #main>
                        <MsEditor
                          ref="msEditorRef"
                          v-bkloading="{ loading: isEditorLoading, zIndex: 1000 }"
                          :lang="editorLang"
                          :readonly="!isEditing"
                          :title="editorLang"
                          @change="handleEditorChange"
                          @error="handleEditorErr"
                        />
                      </template>
                    </ResizeLayout>
                    <Alert
                      v-if="adminIpWarning.message"
                      class="mt-[16px] flex-shrink-0"
                      theme="warning"
                    >
                      {{ adminIpWarning.message }}
                    </Alert>
                  </div>
                </template>
              </ResizeLayout>
              <!-- 未开启环境配置 -->
              <ResizeLayout
                v-else
                :border="false"
                class="min-h-[0px] h-full yaml-editor-layout"
                :initial-divide="asideInitialDivide"
                placement="right"
              >
                <template #aside>
                  <!-- 未开启环境配置-查看环境变量 -->
                  <ViewDefaultEnvVars
                    v-if="showType === 'variables'"
                    ref="viewDefaultEnvVarsRef"
                    class="ml-[16px]"
                    :custom-request-fn="handleGetVarEnv"
                    :env-list="envList"
                    v-bind="viewDefaultEnvVarsProps"
                  >
                    <template
                      v-if="isTrpcApp"
                      #alert
                    >
                      <TrpcEnvVarTip />
                    </template>
                  </ViewDefaultEnvVars>
                </template>
                <template #main>
                  <div class="flex flex-col h-full min-h-0">
                    <ResizeLayout
                      ref="errorRef"
                      :auto-minimize="true"
                      class="flex-1 min-h-0"
                      :disabled="!editorErr.message?.length"
                      :max="300"
                      :min="100"
                      placement="bottom"
                    >
                      <template #aside>
                        <EditorStatus
                          v-show="!!editorErr.message?.length"
                          :message="editorErr.message"
                        />
                      </template>
                      <template #main>
                        <MsEditor
                          ref="msEditorRef"
                          v-bkloading="{ loading: isEditorLoading, zIndex: 1000 }"
                          :lang="editorLang"
                          :readonly="!isEditing"
                          :title="editorLang"
                          @change="handleEditorChange"
                          @error="handleEditorErr"
                        />
                      </template>
                    </ResizeLayout>
                    <Alert
                      v-if="adminIpWarning.message"
                      class="mt-[16px] flex-shrink-0"
                      closable
                      theme="warning"
                    >
                      {{ adminIpWarning.message }}
                      <Button
                        class="ml-[6px]"
                        text
                        theme="primary"
                        @click="handleViewAdminDoc"
                      >
                        {{ $t('管理命令配置说明') }}
                      </Button>
                    </Alert>
                  </div>
                </template>
              </ResizeLayout>
            </div>
            <!-- 底部操作栏（编辑态显示） -->
            <div
              v-if="isEditing"
              class="flex items-center gap-[8px] pt-[16px] flex-shrink-0"
            >
              <Button
                :disabled="!!editorErr.message?.length"
                theme="primary"
                @click="handleBeforeSave"
              >
                {{ $t('保存') }}
              </Button>
              <Button @click="handleCancelEdit">
                {{ $t('取消') }}
              </Button>
            </div>
          </div>
        </BkmsContent>
      </div>
    </Skeleton>

    <!-- 版本列表侧边栏 -->
    <VersionListSideslider
      v-model:visible="showVersionListSideslider"
      :config-file-list="configFileList"
      :current-env-name="currentEnv.name"
      :enable-env-config="isEnableEnvConfig"
      :env-list="envList"
      @refresh="fetchConfigFileList"
      @rollback="handleRollbackRefresh"
    />

    <!-- 保存版本确认弹窗 -->
    <SaveVersionConfirmDialog
      ref="saveVersionDialogRef"
      v-model:is-show="showSaveVersionDialog"
      :next-version="nextVersion"
      @confirm="handleSaveVersionConfirm"
    />

    <!-- 清空文件内容处理方式选择弹窗 -->
    <ClearFileContentDialog
      v-model:is-show="showClearFileContentDialog"
      :loading="isSubmitLoading"
      @confirm="handleClearFileContentConfirm"
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, ref, watch } from 'vue';

  import {
    Alert,
    Button,
    Divider,
    Form,
    Input,
    Message,
    OverflowTitle,
    PopConfirm,
    ResizeLayout,
    Switcher,
    Tag,
  } from 'bkui-vue';
  import { cloneDeep, set } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { parse as parseYaml } from 'yaml';
  import { AppDetailOutputObj, AppModelSpecInput, TafSpecOutputObj, TrpcSpecOutputObj } from '~/@types/v1/app';
  import {
    AppConfigFileOutputObj,
    CreateAppConfigFileOutput,
    GetAppConfigFileDetailsOutput,
    ListAppConfigFilesOutput,
    UpdateAppConfigFileContentOutput,
  } from '~/@types/v1/app-config-files';
  import { EnvOutput } from '~/@types/v1/env';
  import { AppConfigFilesService, EnvService, EnvvarsService } from '~/api/modules/v1';
  import { convertToYaml, hasErrorCode } from '~/common/util';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import useSpecField from '~/composables/use-spec-field';
  import { useAppDetail } from '~/stores/app-detail';

  import ClearFileContentDialog from './components/clear-file-content-dialog.vue';
  import SaveVersionConfirmDialog from './components/save-version-confirm-dialog.vue';
  import TrpcEnvVarTip from './components/trpc-env-var-tip.vue';
  import EnvPerspectiveSelect from './env-perspective-select.vue';
  import VersionListSideslider from './version-list-sideslider.vue';

  import type { IMonacoEditorErrorMarkerItem } from '~/common/util';

  type ClearFileContentAction = 'deleteFile' | 'saveEmpty';

  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  const { confirmBox } = useLeaveConfirm();
  const { appType, specFieldName, updateSpecApi } = useSpecField();

  // 是否允许编辑文件名（仅 TAF 应用支持）
  const isFileNameEditable = computed(() => appType.value === 'taf');

  // TAF 应用框架配置为 XML 格式，其他为 YAML 格式
  const isTafApp = computed(() => appType.value === 'taf');
  const isTrpcApp = computed(() => appType.value === 'trpc');
  const editorLang = computed(() => (isTafApp.value ? 'xml' : 'yaml'));

  // ViewDefaultEnvVars 组件的动态 props（trpc 应用需要特殊格式）
  const viewDefaultEnvVarsProps = computed(() => {
    if (isTafApp.value) return {};
    return {
      'copy-format': (key: string) => `\${${key}}`,
      // trpc 应用需要多种格式复制
      ...(isTrpcApp.value
        ? {
            'copy-options': [
              {
                id: 'trpc-runtime',
                format: (key: string) => `\${${key}}`,
                description: t('tRPC 运行时解析，平台不渲染进配置文件'),
                recommended: true,
              },
              {
                id: 'platform-render',
                format: (key: string) => `\${{ env.${key} }}`,
                description: t('平台下发前渲染为实际值写入配置'),
              },
            ],
          }
        : {}),
      'express-template': '${var_key}',
    };
  });

  // 当前 spec 的文件信息
  const currentFileSpec = computed(() => appData.value?.appModelSpec?.[specFieldName.value]);

  // 应用数据
  const appData = ref<AppDetailOutputObj>();

  // admin.ip 警告信息
  const adminIpWarning = ref<{
    message: string;
    type: '' | 'invalid' | 'pod_ip';
  }>({
    message: '',
    type: '',
  });

  // ========== 文件信息统一编辑 ==========
  const isFileInfoEditing = ref(false);
  const isFileInfoSaving = ref(false);
  const fileInfoFormRef = ref<InstanceType<typeof Form> | null>(null);
  const fileInfoFormData = ref({ fileName: '', filePath: '' });
  const fileInfoFormRules = {
    fileName: [
      {
        required: true,
        message: t('请输入配置文件名称'),
        trigger: 'blur',
      },
    ],
    filePath: [
      {
        required: true,
        message: t('请输入配置文件路径'),
        trigger: 'blur',
      },
    ],
  };

  function handleFileInfoCancel() {
    isFileInfoEditing.value = false;
    fileInfoFormRef.value?.clearValidate();
  }

  function handleFileInfoEdit() {
    fileInfoFormData.value.fileName = currentFileSpec.value?.fileName || '';
    fileInfoFormData.value.filePath = currentFileSpec.value?.filePath || '';
    isFileInfoEditing.value = true;
  }

  // 文件信息保存
  async function handleFileInfoSave() {
    try {
      await fileInfoFormRef.value?.validate();
      isFileInfoSaving.value = true;

      const spec = currentFileSpec.value;
      const fileNameChanged = isFileNameEditable.value && fileInfoFormData.value.fileName !== (spec?.fileName || '');
      const filePathChanged = fileInfoFormData.value.filePath !== (spec?.filePath || '');

      // 构建更新数据
      const updatedAppModelSpec = cloneDeep(appData.value?.appModelSpec || {}) as AppModelSpecInput;
      const target = updatedAppModelSpec[specFieldName.value] as TafSpecOutputObj | TrpcSpecOutputObj;
      if (fileNameChanged) set(target, 'fileName', fileInfoFormData.value.fileName);
      if (filePathChanged) set(target, 'filePath', fileInfoFormData.value.filePath);

      const result = await updateSpecApi
        .value({
          appID: appDetailStore.appID,
          appModelSpec: updatedAppModelSpec,
        })
        .then(() => true)
        .catch(() => false);

      if (result) {
        Message({ message: t('操作成功'), theme: 'success' });
        await getData();
        isFileInfoEditing.value = false;
      }
    } finally {
      isFileInfoSaving.value = false;
    }
  }

  // ========== YAML 编辑器（原 EditYaml 逻辑） ==========
  // 编辑态控制
  const isEditing = ref(false);

  function handleCancelEdit() {
    isEditing.value = false;
    closeAside();
    // 恢复原始内容
    msEditorRef.value?.setValue(currentEnvOriginalContent.value);
  }

  function handleStartEdit() {
    isEditing.value = true;
  }

  const isEnableEnvConfig = ref(false);

  // 当前选中的环境
  const defaultEnv = {
    name: '',
    displayName: t('默认配置'),
    type: '',
  } as EnvOutput;
  const currentEnv = ref({ ...defaultEnv } as EnvOutput);

  // 当前环境的原始内容（用于判断是否修改）
  const currentEnvOriginalContent = ref('');

  // 配置文件列表
  const configFileList = ref<AppConfigFileOutputObj[]>([]);

  /** 已修改的环境名称列表（基于 configFileList，排除默认配置） */
  const modifiedEnvNames = computed((): string[] =>
    configFileList.value.filter(item => item.envName !== '').map(item => item.envName!),
  );

  // 编辑器引用
  const msEditorRef = ref<InstanceType<typeof MsEditor> | null>(null);

  // 侧边栏控制
  const showType = ref<'completeValues' | 'variables'>();
  const isEditorLoading = ref(false);
  const isOpenEnvConfigLoading = ref(false);
  const isSubmitLoading = ref(false);

  // 完整配置
  const completeValues = ref<string>('');
  const completeValuesLoading = ref<boolean>(false);

  const viewDefaultEnvVarsRef = ref();

  // 获取完整配置
  async function getCompleteValues() {
    completeValues.value = '';
    completeValuesLoading.value = true;
    try {
      const defaultConfigFile = configFileList.value.find(item => item.envName === '');
      const res = await AppConfigFilesService.previewOverlayMerge({
        appID: appDetailStore.appID,
        id: defaultConfigFile?.id || '',
        overlayContent: msEditorRef.value?.getValue() || '',
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

  function toggleAsideShow(type: 'completeValues' | 'variables') {
    if (showType.value === type) {
      closeAside();
    } else {
      showType.value = type;
      nextTick(() => {
        if (type === 'completeValues') {
          getCompleteValues();
        } else if (type === 'variables' && viewDefaultEnvVarsRef.value && currentEnv.value.name) {
          viewDefaultEnvVarsRef.value.setCurEnv(currentEnv.value.name);
        }
      });
    }
  }

  // 环境列表（包含默认配置）
  const envListWithDefault = computed(() => {
    return [defaultEnv, ...envList.value];
  });

  // 计算侧边栏的初始分割比例
  const asideInitialDivide = computed(() => {
    return showType.value ? '50%' : 0;
  });

  function closeAside() {
    showType.value = undefined;
  }

  // 下拉框选中环境
  function handleEnvSelectChange(envName: string) {
    const realName = envName === '__default__' ? '' : envName;
    const env = envListWithDefault.value.find(e => e.name === realName);
    if (env) {
      handleEnvChange(env);
    }
  }

  // 编辑器错误处理
  const editorErr = ref<{
    message: string[];
    type: string;
  }>({
    type: '',
    message: [],
  });
  const errorRef = ref<InstanceType<typeof ResizeLayout> | null>(null);

  /**
   * 检测 admin.ip 值并更新警告信息
   */
  function checkAdminIp(yamlContent: string) {
    if (isTafApp.value) return;
    const adminIp = getAdminIpFromYaml(yamlContent);
    const emptyWarning = { message: '', type: '' as const };

    if (adminIp == null || ['127.0.0.1', '0.0.0.0', '${BKMS_ADMIN_IP}'].includes(adminIp)) {
      adminIpWarning.value = emptyWarning;
      return;
    }

    if (adminIp === '${BKMS_POD_IP}') {
      adminIpWarning.value = {
        message: t(
          'admin.ip 配置为 {0} 时，将无法使用「部署管理 - 管理命令」功能。如需使用，请替换为 127.0.0.1（仅 Pod 内访问）或 0.0.0.0（IDC 内访问）。',
          ['${BKMS_POD_IP}'],
        ),
        type: 'pod_ip',
      };
      return;
    }

    adminIpWarning.value = {
      message: t('admin.ip 当前值不合法，仅支持 127.0.0.1、0.0.0.0 或 {0}', ['${BKMS_POD_IP}']),
      type: 'invalid',
    };
  }

  // 获取配置文件详情
  async function fetchConfigFileDetail(envName: string = ''): Promise<string> {
    const configFileId = findConfigFileByEnvName(envName)?.id || '';
    if (!configFileId) return '';
    try {
      isEditorLoading.value = true;
      const fileDetail: GetAppConfigFileDetailsOutput = await AppConfigFilesService.getAppConfigFileDetails(
        {
          appID: appDetailStore.appID,
          id: configFileId,
        },
        { needRes: true },
      );

      let content = '';
      if (fileDetail.editableContentField === 'content') {
        content = fileDetail.content || '';
      } else if (fileDetail.editableContentField === 'overlayContent') {
        content = fileDetail.overlayContent || '';
      }

      return isTafApp.value ? content : convertToYaml(content);
    } catch (error) {
      console.error(error);
      return '';
    } finally {
      isEditorLoading.value = false;
    }
  }

  // 获取配置文件列表
  async function fetchConfigFileList() {
    try {
      const ret: ListAppConfigFilesOutput = await AppConfigFilesService.listAppConfigFiles(
        {
          appID: appDetailStore.appID,
        },
        { needRes: true },
      );
      configFileList.value = ret.items || [];
      if (ret?.items?.length === 1 && ret.items[0].envName === '') {
        isEnableEnvConfig.value = false;
      } else {
        isEnableEnvConfig.value = true;
      }
    } catch (error) {
      console.error(error);
      configFileList.value = [];
      isEnableEnvConfig.value = false;
    }
  }

  function findConfigFileByEnvName(envName: string): AppConfigFileOutputObj | undefined {
    return configFileList.value.find(item => item.envName === envName);
  }

  function getAdminIpFromYaml(yamlContent: string): string | undefined {
    if (!yamlContent) return undefined;

    try {
      const parsed = parseYaml(yamlContent);
      if (!parsed || !parsed.server) return undefined;

      const server = parsed.server;
      const language = appDetailStore.appDetail?.appModelSpec?.trpcSpec?.language || '';

      switch (language.toLowerCase()) {
        case 'go':
        case 'python':
          return server.admin?.ip;
        case 'cpp':
        case 'nodejs':
        case 'node':
          return server.admin_ip;
        case 'java':
          return server.admin?.admin_ip;
        default:
          return server.admin?.ip || server.admin_ip || server.admin?.admin_ip;
      }
    } catch {
      return undefined;
    }
  }

  // 保存前根据清空场景选择弹窗
  function handleBeforeSave() {
    if (shouldShowClearFileContentDialog()) {
      showClearFileContentDialog.value = true;
      return;
    }
    showSaveVersionDialog.value = true;
  }

  function handleClearFileContentConfirm(action: ClearFileContentAction) {
    handleSave(action === 'saveEmpty' ? '保存为空文件' : '', action);
  }

  // 编辑器内容变化处理
  function handleEditorChange(content: string) {
    checkAdminIp(content);
  }

  function handleEditorErr(err: IMonacoEditorErrorMarkerItem[]) {
    editorErr.value.type = 'content';
    editorErr.value.message = err.map(item => item.message);
    hideOrShowError();
  }

  // 确认开启环境配置
  function handleEnableEnvConfig() {
    isEnableEnvConfig.value = true;
    nextTick(() => {
      isOpenEnvConfigLoading.value = true;
      setTimeout(async () => {
        closeAside();
        msEditorRef.value?.setValue(currentEnvOriginalContent.value);
        isOpenEnvConfigLoading.value = false;
      }, 300);
    });
  }

  // 环境切换
  async function handleEnvChange(env: EnvOutput) {
    const currentContent = msEditorRef.value?.getValue() || '';
    const hasUnsavedChanges = currentContent !== currentEnvOriginalContent.value;

    if (hasUnsavedChanges) {
      const shouldContinue = await confirmBox(false, {
        validates: [() => false],
      });

      if (!shouldContinue) {
        return;
      }
    }

    if (isDefaultConfig(env?.name ?? '') && showType.value === 'completeValues') {
      closeAside();
    }

    currentEnv.value = env;
    const content = await fetchConfigFileDetail(env.name);

    currentEnvOriginalContent.value = content;
    msEditorRef.value?.setValue(content);
    checkAdminIp(content);

    if (showType.value === 'completeValues') {
      await getCompleteValues();
    } else if (showType.value === 'variables' && viewDefaultEnvVarsRef.value && currentEnv.value.name) {
      viewDefaultEnvVarsRef.value.setCurEnv(env.name);
    }
  }

  // 获取应用环境变量
  function handleGetVarEnv(env: string) {
    return EnvvarsService.listAppEnvVars({
      appID: appDetailStore.appID,
      envName: env,
    });
  }

  async function handleInitEditor() {
    try {
      isEditorLoading.value = true;
      currentEnv.value = Object.assign({}, defaultEnv);
      // configFileList 已在 initPage 中获取，此处直接使用
      const initialContent = (await fetchConfigFileDetail()) || '';
      msEditorRef.value?.setValue(initialContent);
      currentEnvOriginalContent.value = initialContent;
      checkAdminIp(initialContent);
    } catch (err) {
      console.error(err);
    } finally {
      isEditorLoading.value = false;
    }
  }

  // 保存
  async function handleSave(description: string = '', emptyContentAction?: ClearFileContentAction) {
    let isSaveSuccess = false;
    try {
      isSubmitLoading.value = true;
      const content = msEditorRef.value?.getValue() || '';
      const isSaveEmptyContent = emptyContentAction === 'saveEmpty' && content.trim() === '';
      const submitContent = isSaveEmptyContent ? '' : content;
      let savedContent = submitContent;

      if (isEnvModified(currentEnv.value?.name ?? '')) {
        const configId = findConfigFileByEnvName(currentEnv.value?.name ?? '')?.id || '';
        if (currentEnv.value.name === '') {
          await AppConfigFilesService.updateAppConfigFileContent(
            {
              appID: appDetailStore.appID,
              id: configId,
              previewOnly: false,
              content: submitContent,
              description,
              currentVersion: findConfigFileByEnvName('')?.currentVersion,
            },
            { needRes: true, interceptorErr: false },
          );
        } else if (content.trim() === '' && emptyContentAction === 'deleteFile') {
          // 非默认环境下配置内容为空时，删除该环境的覆盖配置
          await AppConfigFilesService.deleteAppConfigFile({
            appID: appDetailStore.appID,
            id: configId,
          });
          await fetchConfigFileList();
          // 删除后如果关闭了环境配置（只剩默认配置），填充默认配置内容
          if (!isEnableEnvConfig.value) {
            savedContent = (await fetchConfigFileDetail('')) || '';
          } else {
            savedContent = '';
          }
        } else {
          await updateOverlayContent(
            configId,
            submitContent,
            false,
            description,
            findConfigFileByEnvName(currentEnv.value?.name ?? '')?.currentVersion,
          );
        }
      } else {
        const defaultConfigId = findConfigFileByEnvName('')?.id || '';
        const createResult = (await AppConfigFilesService.createAppConfigFile(
          {
            appID: appDetailStore.appID,
            name: currentEnv.value?.name ?? '',
            type: 'overlay',
            baseAppConfigFileID: defaultConfigId,
            contentSourceType: 'local',
            envName: currentEnv.value.name,
            fileFormat: appDetailStore.appType === 'taf' ? 'taf' : 'yaml',
            description,
          },
          { needRes: true },
        )) as CreateAppConfigFileOutput;
        await fetchConfigFileList();

        if (createResult?.item?.id) {
          await updateOverlayContent(
            createResult.item.id,
            submitContent,
            false,
            description,
            findConfigFileByEnvName(currentEnv.value?.name ?? '')?.currentVersion,
          );
        }
      }
      await fetchConfigFileList();
      Message({
        theme: 'success',
        message: t('操作成功'),
      });
      currentEnvOriginalContent.value = savedContent;
      msEditorRef.value?.setValue(savedContent);
      checkAdminIp(savedContent);
      isEditing.value = false;
      closeAside();
      showSaveVersionDialog.value = false;
      isSaveSuccess = true;
    } catch (err) {
      saveVersionDialogRef.value?.stopLoading();
      if (hasErrorCode(err, 'APP_CONFIG_FILE_VERSION_CONFLICT')) {
        Message({
          theme: 'error',
          message: t('当前配置已被他人更新。为避免数据被覆盖，请刷新页面获取最新版本后重新编辑。'),
        });
      }
    } finally {
      isSubmitLoading.value = false;
      if (isSaveSuccess) {
        showClearFileContentDialog.value = false;
      }
    }
  }

  /** 保存版本确认回调 */
  function handleSaveVersionConfirm(description: string) {
    handleSave(description);
  }

  // 查看管理命令配置说明
  function handleViewAdminDoc() {
    window.open(`${import.meta.env.BK_DOC_URL}/p/4016336887`, '_blank');
  }

  function hideOrShowError() {
    if (!editorErr.value.message?.length && errorRef.value) {
      errorRef.value.asideRef.hidden = true;
    } else if (editorErr.value.message?.length && errorRef.value) {
      errorRef.value.asideRef.hidden = false;
    }
  }

  function isDefaultConfig(envName: string): boolean {
    return envName === '';
  }

  function isEnvModified(envName: string): boolean {
    return !!findConfigFileByEnvName(envName);
  }

  function shouldShowClearFileContentDialog() {
    const content = msEditorRef.value?.getValue() || '';
    const envName = currentEnv.value?.name ?? '';
    return isEnableEnvConfig.value && !isDefaultConfig(envName) && content.trim() === '' && isEnvModified(envName);
  }

  async function updateOverlayContent(
    configFileId: string,
    overlayContent: string,
    previewOnly = false,
    description = '',
    currentVersion = 0,
  ) {
    return (await AppConfigFilesService.updateAppConfigFileOverlayContent(
      {
        appID: appDetailStore.appID,
        id: configFileId,
        overlayContent,
        previewOnly,
        description,
        currentVersion,
      },
      { needRes: true, interceptorErr: false },
    )) as UpdateAppConfigFileContentOutput;
  }

  // ========== 数据获取 ==========
  const isLoading = ref(false);
  async function getData() {
    isLoading.value = true;
    try {
      const detail = await appDetailStore.fetchAppDetail();
      appData.value = detail || ({} as AppDetailOutputObj);
    } finally {
      isLoading.value = false;
    }
  }

  // 获取环境列表
  const envList = ref<EnvOutput[]>([]);

  // 版本列表侧边栏显示状态
  const showVersionListSideslider = ref(false);

  // 保存版本确认弹窗
  const showSaveVersionDialog = ref(false);
  const saveVersionDialogRef = ref<InstanceType<typeof SaveVersionConfirmDialog> | null>(null);
  const showClearFileContentDialog = ref(false);

  /** 下一版本号：当前环境的 currentVersion + 1 */
  const nextVersion = computed(() => {
    const currentConfig = findConfigFileByEnvName(currentEnv.value?.name ?? '');
    return Number(currentConfig?.currentVersion || 0) + 1;
  });

  async function getEnvList() {
    if (!appDetailStore.appID) {
      envList.value = [];
      return;
    }
    envList.value = await EnvService.listAppEnvs({
      appID: appDetailStore.appID,
    }).catch(() => []);
  }

  /** 版本列表侧边栏回滚后刷新文件内容 */
  async function handleRollbackRefresh() {
    await fetchConfigFileList();
    const content = await fetchConfigFileDetail(currentEnv.value.name);
    currentEnvOriginalContent.value = content;
    msEditorRef.value?.setValue(content);
    checkAdminIp(content);
  }

  // 监听应用详情变化
  watch(
    () => appDetailStore.appDetail,
    newDetail => {
      appData.value = newDetail || ({} as AppDetailOutputObj);
    },
    { immediate: true },
  );

  // 初始化隐藏错误侧栏
  watch(
    [editorErr.value, errorRef],
    () => {
      hideOrShowError();
    },
    { immediate: true },
  );

  async function initPage() {
    if (!appDetailStore.appID) {
      resetAppScopedState();
      return;
    }
    // 先并行获取环境列表和配置文件列表，确保 modifiedEnvNames 在组件渲染时已就绪
    await Promise.all([getEnvList(), getData(), fetchConfigFileList()]);
    // 数据加载完成后初始化编辑器
    handleInitEditor();
  }

  function resetAppScopedState() {
    appData.value = {} as AppDetailOutputObj;
    envList.value = [];
    configFileList.value = [];
    currentEnv.value = { ...defaultEnv };
    currentEnvOriginalContent.value = '';
    isEnableEnvConfig.value = false;
    isEditing.value = false;
    closeAside();
    msEditorRef.value?.setValue('');
  }

  watch(
    () => appDetailStore.appID,
    () => {
      initPage();
    },
    { immediate: true },
  );
</script>

<style lang="postcss" scoped>
  .editor-aside-layout > :deep(.bk-resize-layout-main) {
    padding-right: 16px;
  }
  .editor-aside-layout :deep(.bk-resize-layout-aside-content) {
    padding-right: 16px;
  }
  .yaml-editor-layout :deep(.bk-resize-layout-aside-content) {
    display: flex;
  }
  .info-title :deep(.bkms-content-title) {
    background-color: #eaebf0;
  }
  .yaml-content {
    &:deep(.bkms-content) {
      display: flex;
      flex-direction: column;
    }
  }
</style>
