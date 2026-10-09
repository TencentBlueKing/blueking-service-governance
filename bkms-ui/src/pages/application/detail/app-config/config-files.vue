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

<!-- 应用配置 - 配置文件管理：左侧文件列表 + 右侧文件信息与内容编辑 -->
<template>
  <div class="h-full min-h-0">
    <ResizeLayout
      :border="false"
      class="h-full min-h-0"
      collapsible
      :initial-divide="260"
      :is-collapsed="isFileListCollapsed"
      :max="360"
      :min="220"
      @collapse-change="isFileListCollapsed = $event"
    >
      <template #aside>
        <ConfigFileList
          :active-id="activeDefID"
          :create-disabled="envListLoadFailed"
          :files="fileListItems"
          @create="handleOpenCreateFile"
          @delete="handleDeleteFile"
          @select="handleSelectFile"
        />
      </template>

      <template #main>
        <div class="flex h-full min-w-0 flex-col overflow-auto px-[24px] pb-[16px] pt-[20px]">
          <Skeleton
            :full-height="false"
            :loading="isLoading || listLoading || appDetailStore.loading"
          >
            <template #loading>
              <Layout.shape
                :height="28"
                width="100%"
              />
              <div class="my-[16px] grid grid-cols-2 gap-4 gap-y-2 pl-[60px]">
                <Layout.formItem />
                <Layout.formItem />
              </div>
              <Layout.shape
                :height="360"
                width="100%"
              />
            </template>

            <Alert
              v-if="envListLoadFailed"
              class="mb-[16px]"
              theme="error"
            >
              {{ $t('环境列表加载失败，挂载范围相关操作暂不可用。') }}
              <Button
                class="ml-[6px]"
                :loading="envListLoading"
                text
                theme="primary"
                @click="getEnvList"
              >
                {{ $t('重试') }}
              </Button>
            </Alert>

            <Exception
              v-if="!activeDef"
              class="h-full"
              scene="part"
              type="empty"
            />
            <template v-else>
              <div class="mb-[16px] shrink-0">
                <BkmsContent
                  class="info-title shadow-[0_2px_4px_0_#1919290d]"
                  :show-edit-icon="!isFileInfoEditing && (isFrameworkFile || !envListLoadFailed)"
                  :title="$t('文件信息')"
                  @edit="handleFileInfoEdit"
                >
                  <div class="bg-[#fff] p-[16px]">
                    <div
                      v-if="!isFileInfoEditing"
                      class="grid grid-cols-2 gap-[12px] gap-y-2 pl-[44px]"
                    >
                      <FieldItem
                        class="min-h-[30px]"
                        :container-height="20"
                        :field-value="isFrameworkFile ? $t('框架配置文件') : $t('文件名')"
                      >
                        <template #value
                          ><span class="text-[12px]">{{ activeDisplayName || '--' }}</span></template
                        >
                      </FieldItem>
                      <FieldItem
                        class="min-h-[30px]"
                        :container-height="20"
                        :field-value="$t('挂载路径')"
                      >
                        <template #value>
                          <OverflowTitle
                            class="max-w-[300px] text-[12px]"
                            type="tips"
                          >
                            {{ activeDisplayPath || '--' }}
                          </OverflowTitle>
                        </template>
                      </FieldItem>
                      <FieldItem
                        class="min-h-[30px] !items-start"
                        container-height="auto"
                        :field-value="$t('挂载环境')"
                      >
                        <template #value>
                          <span
                            v-if="isFrameworkFile || activeMountScope === 'all'"
                            class="text-[12px]"
                          >
                            {{ $t('全部环境') }}
                          </span>
                          <div
                            v-else
                            class="flex flex-wrap gap-[4px]"
                          >
                            <Tag
                              v-for="env in mountedEnvList"
                              :key="env.name"
                              size="large"
                            >
                              {{ env.displayName || env.name }}
                            </Tag>
                          </div>
                        </template>
                      </FieldItem>
                      <FieldItem
                        v-if="!isFrameworkFile"
                        class="min-h-[30px]"
                        :container-height="20"
                        :field-value="$t('渲染环境变量')"
                      >
                        <template #value>
                          <Tag :theme="activeDef.enableEnvVarRender ? 'success' : 'default'">
                            {{ activeDef.enableEnvVarRender ? $t('已开启') : $t('未开启') }}
                          </Tag>
                        </template>
                      </FieldItem>
                    </div>

                    <Form
                      v-else
                      ref="fileInfoFormRef"
                      form-type="vertical"
                      :model="fileInfoFormData"
                      :rules="fileInfoFormRules"
                    >
                      <Form.FormItem
                        :label="isFrameworkFile ? $t('框架配置文件') : $t('文件名')"
                        property="name"
                        required
                      >
                        <Input
                          v-model.trim="fileInfoFormData.name"
                          :disabled="isFrameworkFile && !isFileNameEditable"
                          :maxlength="isFrameworkFile ? 100 : 64"
                        />
                      </Form.FormItem>
                      <Form.FormItem
                        :label="$t('挂载路径')"
                        property="mountDir"
                        required
                      >
                        <Input
                          v-model.trim="fileInfoFormData.mountDir"
                          clearable
                        />
                        <div class="mt-[4px] text-[12px] text-[#979BA5]">
                          {{
                            $t('文件最终挂载到 {0}', [
                              `${fileInfoFormData.mountDir || '/[路径]'}/${fileInfoFormData.name || '[文件名]'}`,
                            ])
                          }}
                        </div>
                      </Form.FormItem>
                      <Form.FormItem
                        class="mount-env-form-item"
                        :label="$t('挂载环境')"
                        property="mountedEnvNames"
                        :required="!isFrameworkFile && fileInfoFormData.scope === 'envs'"
                      >
                        <MountScopeField
                          v-model:env-names="fileInfoFormData.mountedEnvNames"
                          v-model:scope="fileInfoFormData.scope"
                          :disabled="isFrameworkFile"
                          :env-list="envList"
                          @update:scope="handleFileInfoScopeChange"
                        />
                      </Form.FormItem>
                      <Form.FormItem
                        v-if="!isFrameworkFile"
                        :label="$t('渲染环境变量')"
                      >
                        <div class="flex items-center">
                          <Switcher
                            v-model="fileInfoFormData.enableEnvVarRender"
                            size="small"
                            theme="primary"
                          />
                          <span class="ml-[10px] text-[12px] text-[#979BA5]">
                            {{ $t('开启后，文件内容中的 {0} 会在下发前被渲染为实际值。', [envVarPlaceholder]) }}
                          </span>
                        </div>
                      </Form.FormItem>
                      <Form.FormItem class="!mb-0">
                        <Button
                          :loading="isFileInfoSaving"
                          theme="primary"
                          @click="handleFileInfoSave"
                        >
                          {{ $t('保存') }}
                        </Button>
                        <Button
                          class="ml-[8px]"
                          @click="handleFileInfoCancel"
                          >{{ $t('取消') }}</Button
                        >
                      </Form.FormItem>
                    </Form>
                  </div>
                </BkmsContent>
              </div>

              <div class="min-h-[500px] flex-1">
                <BkmsContent
                  class="yaml-content info-title flex h-full flex-col shadow-[0_2px_4px_0_#1919290d]"
                  :show-edit-icon="!isEditing && !isFileInfoEditing && effectiveEditableContentField !== 'none'"
                  @edit="handleStartEdit"
                >
                  <template #title>{{ $t('文件内容') }}</template>
                  <template #action>
                    <div class="flex items-center">
                      <IconTextButton
                        :active="showVariables"
                        icon="bkms-icon bkms-icon-variable"
                        :text="$t('环境变量')"
                        @click="toggleVariables"
                      />
                      <Divider
                        class="h-[12px]"
                        color="#C4C6CC"
                        direction="vertical"
                        type="solid"
                      />
                      <Button
                        :disabled="!currentInstanceFileID"
                        text
                        theme="primary"
                        @click="showVersionListSideslider = true"
                      >
                        <i class="bkms-icon bkms-icon-time-2 mr-[4px] mt-[-1px] text-[14px]"></i>
                        {{ $t('版本列表') }}
                      </Button>
                    </div>
                  </template>

                  <div
                    v-bkloading="{ loading: detailLoading || isModeSaving, zIndex: 1000 }"
                    class="flex min-h-0 flex-1 flex-col bg-[#fff] p-[16px]"
                  >
                    <div class="mb-[12px] flex h-[32px] shrink-0 items-center">
                      <span class="mr-[6px] text-[12px] text-[#4D4F56]">{{ $t('按环境配置') }}：</span>
                      <Tag
                        v-if="!isEditing"
                        :theme="isIndependentConfig ? 'success' : 'default'"
                      >
                        {{ isIndependentConfig ? $t('已开启') : $t('未开启') }}
                      </Tag>
                      <Switcher
                        v-else
                        v-model="envConfigSwitcher"
                        size="small"
                        theme="primary"
                      />
                    </div>

                    <ResizeLayout
                      :border="false"
                      class="editor-aside-layout min-h-0 flex-1"
                      :initial-divide="showVariables ? '50%' : 0"
                      placement="right"
                    >
                      <template #aside>
                        <ViewDefaultEnvVars
                          v-if="showVariables"
                          ref="viewDefaultEnvVarsRef"
                          class="ml-[16px] !h-full"
                          :custom-request-fn="handleGetVarEnv"
                          :env-list="envList"
                          v-bind="viewDefaultEnvVarsProps"
                        >
                          <template
                            v-if="isTrpcApp && isFrameworkFile"
                            #alert
                            ><TrpcEnvVarTip
                          /></template>
                        </ViewDefaultEnvVars>
                      </template>
                      <template #main>
                        <div class="flex h-full min-h-0 flex-col">
                          <ResizeLayout
                            :auto-minimize="true"
                            class="min-h-0 flex-1"
                            :collapsible="true"
                            :disabled="!hasEditorError"
                            :is-collapsed="!hasEditorError"
                            :max="300"
                            :min="100"
                            placement="bottom"
                          >
                            <template #collapse-trigger />
                            <template #aside>
                              <EditorStatus
                                v-if="hasEditorError"
                                :message="editorErrMessages"
                              />
                            </template>
                            <template #main>
                              <MsEditor
                                ref="msEditorRef"
                                v-bkloading="{ loading: isEditorLoading || fullContentLoading, zIndex: 1000 }"
                                :lang="editorLang"
                                :model-value="editableDraft"
                                :readonly="isEditorReadonly"
                                :title="editorLang"
                                @change="handleEditorChange"
                                @error="handleEditorErr"
                              >
                                <template #title>
                                  <div class="flex h-full min-w-0 items-center">
                                    <span class="shrink-0">{{ editorLang }}</span>
                                    <template v-if="isIndependentConfig">
                                      <div class="mx-[12px] h-[14px] w-px shrink-0 bg-[#63656E]"></div>
                                      <EnvPerspectiveSelect
                                        class="!h-[26px] !min-w-[240px] shrink-0"
                                        :env-list="perspectiveEnvList"
                                        :label="$t('环境')"
                                        :model-value="currentEnv.name || '__default__'"
                                        :modified-env-names="modifiedEnvNames"
                                        theme="dark"
                                        @change="handleEnvSelectChange"
                                      />
                                      <OverflowTitle
                                        v-if="editorContentHint"
                                        :key="editorContentHint"
                                        class="ml-[8px] min-w-0 text-[12px] text-[#979ba5]"
                                        resizeable
                                        type="tips"
                                      >
                                        {{ editorContentHint }}
                                      </OverflowTitle>
                                    </template>
                                    <Button
                                      v-if="canResetCurrentEnv"
                                      class="ml-[8px] shrink-0 !text-[12px]"
                                      text
                                      theme="primary"
                                      @click="handleResetCurrentEnv"
                                    >
                                      {{ $t('恢复为默认配置') }}
                                    </Button>
                                  </div>
                                </template>
                                <template
                                  v-if="canShowFullContent"
                                  #tools
                                >
                                  <div
                                    class="mr-[8px] inline-flex h-[24px] items-stretch rounded-[2px] border border-[#63656e] border-solid bg-transparent"
                                  >
                                    <button
                                      v-for="item in contentViewOptions"
                                      :key="item.value"
                                      :aria-pressed="contentView === item.value"
                                      class="inline-flex h-full appearance-none items-center justify-center gap-[4px] rounded-none border-0 border-solid bg-transparent px-[8px] text-[12px] leading-[22px] text-[#979ba5] outline-offset-[-1px] cursor-pointer focus-visible:outline focus-visible:outline-[1px] focus-visible:outline-[#3a84ff]"
                                      :class="[
                                        item.divider ? 'border-r border-r-[#63656e]' : '',
                                        contentView === item.value
                                          ? '!bg-[#4d4d4d] text-[#c4c6cc]'
                                          : 'hover:bg-[#3a3a3a] hover:text-[#c4c6cc]',
                                      ]"
                                      type="button"
                                      @click="contentView = item.value"
                                    >
                                      <span>{{ $t(item.label) }}</span>
                                    </button>
                                  </div>
                                </template>
                              </MsEditor>
                            </template>
                          </ResizeLayout>
                          <Alert
                            v-if="adminIpWarning.message"
                            class="mt-[12px] shrink-0"
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

                    <div
                      v-if="isEditing"
                      class="flex shrink-0 items-center gap-[8px] pt-[16px]"
                    >
                      <Button
                        :disabled="isEditorReadonly || hasEditorError"
                        theme="primary"
                        @click="handleBeforeSave"
                      >
                        {{ $t('保存') }}
                      </Button>
                      <Button @click="handleCancelEdit">{{ $t('取消') }}</Button>
                    </div>
                  </div>
                </BkmsContent>
              </div>
            </template>
          </Skeleton>
        </div>
      </template>
    </ResizeLayout>

    <CreateConfigFileSideslider
      v-model:is-show="showCreateSideslider"
      :env-list="envList"
      :loading="isCreateSubmitting"
      @submit="handleCreateFile"
    />
    <MountPreviewSideslider
      v-model:is-show="mountPreviewVisible"
      :app-id="appDetailStore.appID"
      :def-order="mountPreviewDefOrder"
      :env-list="envList"
      :framework-file-name="currentFileSpec?.fileName || ''"
      :framework-file-path="currentFileSpec?.filePath || ''"
    />
    <VersionListSideslider
      v-model:visible="showVersionListSideslider"
      :config-file-list="[]"
      :current-file-id="currentInstanceFileID"
      :current-file-name="activeDisplayName"
      :current-version="currentInstanceVersion"
      @refresh="handleRollbackRefresh"
      @rollback="handleRollbackRefresh"
    />
    <SaveVersionConfirmDialog
      ref="saveVersionDialogRef"
      v-model:is-show="showSaveVersionDialog"
      :next-version="nextVersion"
      @confirm="handleSave"
    />
    <ClearFileContentDialog
      v-model:is-show="showClearFileContentDialog"
      :loading="isSubmitLoading"
      @confirm="handleClearFileContentConfirm"
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, inject, nextTick, reactive, ref, watch } from 'vue';

  import {
    Alert,
    Button,
    Divider,
    Exception,
    Form,
    InfoBox,
    Input,
    Message,
    OverflowTitle,
    ResizeLayout,
    Switcher,
    Tag,
  } from 'bkui-vue';
  import { cloneDeep, set } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { parse as parseYaml } from 'yaml';
  import { AppConfigFilesService, EnvService, EnvvarsService } from '~/api/modules/v1';
  import { BKMS_REGEX } from '~/common/const';
  import { hasErrorCode } from '~/common/util';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import useSpecField from '~/composables/use-spec-field';
  import ClearFileContentDialog from '~/pages/application/detail/app-config/components/clear-file-content-dialog.vue';
  import ConfigFileList from '~/pages/application/detail/app-config/components/config-file-list.vue';
  import CreateConfigFileSideslider from '~/pages/application/detail/app-config/components/create-config-file-sideslider.vue';
  import MountPreviewSideslider from '~/pages/application/detail/app-config/components/mount-preview-sideslider.vue';
  import MountScopeField from '~/pages/application/detail/app-config/components/mount-scope-field.vue';
  import SaveVersionConfirmDialog from '~/pages/application/detail/app-config/components/save-version-confirm-dialog.vue';
  import TrpcEnvVarTip from '~/pages/application/detail/app-config/components/trpc-env-var-tip.vue';
  import EnvPerspectiveSelect from '~/pages/application/detail/app-config/env-perspective-select.vue';
  import {
    MOUNT_PREVIEW_VISIBLE_KEY,
    useConfigFileDefs,
  } from '~/pages/application/detail/app-config/use-config-file-defs';
  import VersionListSideslider from '~/pages/application/detail/app-config/version-list-sideslider.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import type { AppDetailOutputObj, AppModelSpecInput, TafSpecOutputObj, TrpcSpecOutputObj } from '~/@types/v1/app';
  import type { UpdateDefInput } from '~/@types/v1/app-config-file-defs';
  import type { EnvOutput } from '~/@types/v1/env';
  import type { IMonacoEditorErrorMarkerItem } from '~/common/util';
  import type { MountScope } from '~/pages/application/detail/app-config/components/mount-scope-field.vue';
  import type {
    ConfigFileListItem,
    PlainFileCreateInput,
  } from '~/pages/application/detail/app-config/use-config-file-defs';

  type ClearFileContentAction = 'deleteFile' | 'saveEmpty';
  type ContentView = 'editable' | 'full';

  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  const { confirmBox } = useLeaveConfirm();
  const { appType, specFieldName, updateSpecApi } = useSpecField();
  const mountPreviewVisible = inject(MOUNT_PREVIEW_VISIBLE_KEY, ref(false));
  const appID = computed(() => appDetailStore.appID);
  const {
    activeDef,
    activeDefID,
    createPlain,
    defs,
    deleteDef,
    detail,
    detailLoading,
    fetchDefs,
    listLoading,
    modifiedEnvNames,
    refreshActive,
    reset,
    resetEnv,
    saveContent,
    updateDef,
  } = useConfigFileDefs(appID);

  const appData = ref<AppDetailOutputObj>(); // 应用详情（appModelSpec 为框架配置文件名/路径来源）
  const envList = ref<EnvOutput[]>([]); // 环境列表
  const envListLoading = ref(false);
  const envListLoadFailed = ref(false);
  const isLoading = ref(false);
  const isFileListCollapsed = ref(false); // 左侧文件列表是否折叠
  const isFileInfoEditing = ref(false); // 「文件信息」是否处于编辑态
  const isFileInfoSaving = ref(false);
  const isCreateSubmitting = ref(false);
  const isModeSaving = ref(false); // 统一/按环境配置切换保存中
  const isEditing = ref(false); // 文件内容是否处于编辑态
  const isEditorLoading = ref(false);
  const isSubmitLoading = ref(false);
  const fullContentLoading = ref(false); // 合并完整内容预览加载中
  let detailLoadRequestID = 0;
  let envListRequestID = 0;
  let fullContentRequestID = 0;

  // ---- 弹层与面板显隐 ----
  const showCreateSideslider = ref(false); // 新建配置文件侧滑
  const showVersionListSideslider = ref(false); // 版本列表侧滑
  const showSaveVersionDialog = ref(false);
  const showClearFileContentDialog = ref(false); // 保存空内容确认弹窗
  const showVariables = ref(false);
  const showFullContent = ref(false); // 是否查看合并后的完整内容

  // ---- 编辑器内容 ----
  const editableDraft = ref('');
  const originalContent = ref(''); // 进入编辑时的原始内容，用于判断有无改动
  const msEditorRef = ref<InstanceType<typeof MsEditor> | null>(null);
  const saveVersionDialogRef = ref<InstanceType<typeof SaveVersionConfirmDialog> | null>(null);
  const viewDefaultEnvVarsRef = ref();
  const defaultEnv = computed(() => ({ name: '', displayName: t('默认配置'), type: '' }) as EnvOutput);
  const currentEnv = ref<EnvOutput>({ name: '', displayName: '', type: '' } as EnvOutput); // 当前查看/编辑的环境
  const allEnvNames = computed(() => envList.value.map(env => env.name).filter((name): name is string => !!name));
  const currentFileSpec = computed(() => appData.value?.appModelSpec?.[specFieldName.value]); // 当前应用框架（taf/trpc）的 spec
  const isFrameworkFile = computed(() => activeDef.value?.configKind === 'framework');
  const isTafApp = computed(() => appType.value === 'taf');
  const isTrpcApp = computed(() => appType.value === 'trpc');
  const isFileNameEditable = computed(() => !isFrameworkFile.value || isTafApp.value); // 框架文件名仅 taf 应用可编辑
  // 是否开启按环境配置（false 为统一配置），优先取环境详情，回退到 def
  const isIndependentConfig = computed(
    () => !(detail.value?.isUnifiedConfig ?? activeDef.value?.isUnifiedConfig ?? true),
  );
  // 编辑器语言：框架文件 taf 用 xml、trpc 用 yaml，普通文件用 plaintext
  const editorLang = computed(() => (isFrameworkFile.value ? (isTafApp.value ? 'xml' : 'yaml') : 'plaintext'));
  // 展示用文件名/路径：框架文件取 spec 中的值，普通文件取 def 本身
  const activeDisplayName = computed(() =>
    isFrameworkFile.value
      ? currentFileSpec.value?.fileName || activeDef.value?.name || ''
      : activeDef.value?.name || '',
  );
  const activeDisplayPath = computed(() =>
    isFrameworkFile.value ? currentFileSpec.value?.filePath || '' : activeDef.value?.mountDir || '',
  );
  // 挂载范围：框架文件或已挂载全部环境时视为「全部环境」
  const activeMountScope = computed<MountScope>(() => {
    if (isFrameworkFile.value || activeDef.value?.mountScope === 'all') return 'all';
    const allNames = allEnvNames.value;
    return allNames.length && allNames.every(name => activeDef.value?.mountedEnvNames.includes(name)) ? 'all' : 'envs';
  });
  // 已挂载环境列表
  const mountedEnvList = computed(() =>
    envList.value.filter(env => !!env.name && activeDef.value?.mountedEnvNames.includes(env.name)),
  );
  // 环境视角下拉可选项：统一配置或全部环境挂载时展示所有环境
  const perspectiveEnvList = computed(() => {
    if (isFrameworkFile.value || activeMountScope.value === 'all') return envList.value;
    return mountedEnvList.value;
  });
  // 文件列表展示项：框架文件用 spec 的文件名/路径
  const fileListItems = computed<ConfigFileListItem[]>(() =>
    defs.value.map(file => ({
      ...file,
      displayName: file.configKind === 'framework' ? currentFileSpec.value?.fileName || file.name : file.name,
      displayPath: file.configKind === 'framework' ? currentFileSpec.value?.filePath || file.mountDir : file.mountDir,
      mountScope:
        file.configKind === 'framework' ||
        (file.mountScope === 'envs' &&
          allEnvNames.value.length > 0 &&
          sameStringSet(file.mountedEnvNames, allEnvNames.value))
          ? 'all'
          : file.mountScope,
    })),
  );
  // 挂载预览表格排序依据：与左侧文件列表保持一致，框架配置文件置顶
  const mountPreviewDefOrder = computed(() =>
    [
      ...defs.value.filter(file => file.configKind === 'framework'),
      ...defs.value.filter(file => file.configKind !== 'framework'),
    ]
      .map(file => file.id)
      .filter(Boolean),
  );
  // 框架文件 + 按环境配置 + 已选环境时才支持「差异内容/完整内容」切换
  const canShowFullContent = computed(
    () => isFrameworkFile.value && isIndependentConfig.value && !!currentEnv.value.name,
  );
  // 按环境配置且当前环境存在独立实例时，才可「恢复为默认配置」
  const canResetCurrentEnv = computed(
    () => isIndependentConfig.value && !!currentEnv.value.name && !!detail.value?.hasEnvInstance,
  );
  // 编辑器顶部提示文案
  const editorContentHint = computed(() => {
    if (!isFrameworkFile.value || !isIndependentConfig.value) return '';
    if (!currentEnv.value.name) return t('所有环境的基准配置，修改后对所有环境生效。');
    if (showFullContent.value) return t('合并后的完整内容，仅供查看');
    return t('仅定义差异项，与默认配置合并后生成完整配置。');
  });
  // 框架文件 + 按环境配置 + 已选环境且无独立实例时，编辑的是 overlayContent（差异内容）
  const isFrameworkEnvWithoutInstance = computed(
    () =>
      isFrameworkFile.value &&
      isIndependentConfig.value &&
      !!currentEnv.value.name &&
      detail.value?.hasEnvInstance === false,
  );
  // 实际可编辑的内容字段：overlayContent / content / none（不可编辑）
  const effectiveEditableContentField = computed(() =>
    isFrameworkEnvWithoutInstance.value ? 'overlayContent' : detail.value?.editableContentField || 'none',
  );
  // 当前环境的实例文件 ID；环境无独立实例时为空（版本列表不可用）
  const currentInstanceFileID = computed(() =>
    currentEnv.value.name && !detail.value?.hasEnvInstance ? '' : detail.value?.fileId || '',
  );
  const currentInstanceVersion = computed(() =>
    currentInstanceFileID.value ? Number(detail.value?.currentVersion || 0) : 0,
  );
  const isEditorReadonly = computed(
    () => !isEditing.value || showFullContent.value || effectiveEditableContentField.value === 'none',
  );
  const nextVersion = computed(() => currentInstanceVersion.value + 1);
  // 「差异内容/完整内容」切换项，divider 控制与前一项的分隔线
  const contentViewOptions: { divider: boolean; label: string; value: ContentView }[] = [
    { divider: true, label: '差异内容', value: 'editable' },
    { divider: false, label: '完整内容', value: 'full' },
  ];
  // 「差异内容/完整内容」切换
  const contentView = computed<ContentView>({
    get: () => (showFullContent.value ? 'full' : 'editable'),
    set: value => void handleContentViewChange(value),
  });
  // 「按环境配置」开关，写操作转 handleEnvConfigChange
  const envConfigSwitcher = computed({
    get: () => isIndependentConfig.value,
    set: value => void handleEnvConfigChange(value),
  });

  const envVarPlaceholder = '${{ env.<Key> }}';

  const fileInfoFormData = reactive({
    enableEnvVarRender: false,
    mountDir: '',
    mountedEnvNames: [] as string[],
    name: '',
    scope: 'all' as MountScope,
  });
  const fileInfoFormRef = ref<InstanceType<typeof Form> | null>(null);
  const fileInfoFormRules = {
    name: [
      {
        message: t('文件名仅支持数字、字母、下划线(_)、中划线(-)、点(.)'),
        trigger: 'blur',
        validator: (value: string) =>
          isFrameworkFile.value ||
          !activeDef.value ||
          value === activeDef.value.name ||
          BKMS_REGEX.appConfigFileNameRegex.test(value),
      },
    ],
    mountDir: [
      {
        message: t('配置文件路径必须以 / 开头，且不能为 / 或以 / 结尾'),
        trigger: 'blur',
        validator: (value: string) => isFrameworkFile.value || BKMS_REGEX.appConfigMountDirRegex.test(value),
      },
    ],
    mountedEnvNames: [
      {
        message: t('请至少选择一个挂载环境'),
        trigger: 'change',
        validator: () => fileInfoFormData.scope !== 'envs' || fileInfoFormData.mountedEnvNames.length > 0,
      },
    ],
  };
  const fileInfoOriginalScope = ref<MountScope>('all'); // 进入编辑时的挂载范围，保存时用于保留原语义
  const editorErrMessages = ref<string[]>([]); // monaco 语法错误
  const hasEditorError = computed(() => editorErrMessages.value.length > 0);
  const adminIpWarning = ref<{ message: string; type: '' | 'invalid' | 'pod_ip' }>({ message: '', type: '' }); // admin.ip 校验告警（仅 trpc 框架文件）

  const viewDefaultEnvVarsProps = computed(() => {
    if (!isFrameworkFile.value || isTafApp.value) return {};
    return {
      'copy-format': (key: string) => `\${${key}}`,
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

  // 将详情中可编辑的内容写入编辑器，并重置错误与 admin.ip 校验
  async function applyDetailToEditor() {
    await nextTick();
    const content = getEditableContent();
    editorErrMessages.value = [];
    originalContent.value = content;
    editableDraft.value = content;
    msEditorRef.value?.setValue(content);
    checkAdminIp(content);
  }

  // 校验 trpc 框架配置中的 admin.ip，为 POD_IP 或不合法时生成告警
  function checkAdminIp(content: string) {
    if (!isFrameworkFile.value || isTafApp.value) {
      adminIpWarning.value = { message: '', type: '' };
      return;
    }
    const adminIp = getAdminIpFromYaml(content);
    if (adminIp == null || ['127.0.0.1', '0.0.0.0', '${BKMS_ADMIN_IP}'].includes(adminIp)) {
      adminIpWarning.value = { message: '', type: '' };
    } else if (adminIp === '${BKMS_POD_IP}') {
      adminIpWarning.value = {
        message: t(
          'admin.ip 配置为 {0} 时，将无法使用「部署管理 - 管理命令」功能。如需使用，请替换为 127.0.0.1（仅 Pod 内访问）或 0.0.0.0（IDC 内访问）。',
          ['${BKMS_POD_IP}'],
        ),
        type: 'pod_ip',
      };
    } else {
      adminIpWarning.value = {
        message: t('admin.ip 当前值不合法，仅支持 127.0.0.1、0.0.0.0 或 {0}', ['${BKMS_POD_IP}']),
        type: 'invalid',
      };
    }
  }

  // 内容或文件信息存在未保存改动时弹出离开确认，返回是否可继续后续操作
  async function confirmDiscardChanges() {
    const draft = showFullContent.value ? editableDraft.value : msEditorRef.value?.getValue() || '';
    const hasContentChanges = isEditing.value && draft !== originalContent.value;
    if (!hasContentChanges && !hasFileInfoChanges()) return true;
    return await confirmBox(false, { validates: [() => false] });
  }

  function discardCurrentEdits() {
    if (isEditing.value) handleCancelEdit();
    isFileInfoEditing.value = false;
  }

  // 从 YAML 解析 admin.ip，按应用语言兼容不同的字段结构
  function getAdminIpFromYaml(content: string): string | undefined {
    if (!content) return undefined;
    try {
      const server = parseYaml(content)?.server;
      if (!server) return undefined;
      const language = appDetailStore.appDetail?.appModelSpec?.trpcSpec?.language?.toLowerCase() || '';
      if (['cpp', 'nodejs', 'node'].includes(language)) return server.admin_ip;
      if (language === 'java') return server.admin?.admin_ip;
      return server.admin?.ip || server.admin_ip || server.admin?.admin_ip;
    } catch {
      return undefined;
    }
  }

  // 拉取应用详情
  async function getAppData() {
    const result = await appDetailStore.fetchAppDetail();
    appData.value = result || ({} as AppDetailOutputObj);
  }

  // 按实际可编辑字段取编辑器应展示的内容
  function getEditableContent() {
    if (!detail.value) return '';
    return effectiveEditableContentField.value === 'overlayContent'
      ? detail.value.overlayContent
      : detail.value.content;
  }

  // 拉取应用环境列表；失败时保留显式错误态，避免把接口异常误判为应用没有环境。
  async function getEnvList() {
    const appID = appDetailStore.appID;
    const requestID = ++envListRequestID;
    if (!appID) {
      envList.value = [];
      envListLoadFailed.value = false;
      return;
    }
    envListLoading.value = true;
    try {
      const result = await EnvService.listAppEnvs({ appID });
      if (requestID !== envListRequestID || appID !== appDetailStore.appID) return;
      envList.value = result;
      envListLoadFailed.value = false;
    } catch {
      if (requestID !== envListRequestID || appID !== appDetailStore.appID) return;
      envList.value = [];
      envListLoadFailed.value = true;
    } finally {
      if (requestID === envListRequestID) envListLoading.value = false;
    }
  }

  // 保存前置处理：框架文件按环境配置下清空内容需先弹确认，否则弹版本说明
  function handleBeforeSave() {
    if (shouldShowClearFileContentDialog()) {
      showClearFileContentDialog.value = true;
      return;
    }
    showSaveVersionDialog.value = true;
  }

  // 取消编辑，还原编辑器内容并收起面板
  function handleCancelEdit() {
    fullContentRequestID += 1;
    fullContentLoading.value = false;
    isEditing.value = false;
    showFullContent.value = false;
    showVariables.value = false;
    msEditorRef.value?.setValue(originalContent.value);
    checkAdminIp(originalContent.value);
  }

  function handleClearFileContentConfirm(action: ClearFileContentAction) {
    handleSave(action === 'saveEmpty' ? t('保存为空文件') : '', action);
  }

  // 切换差异/完整内容视图；切到完整视图时调用合并预览接口生成完整配置
  async function handleContentViewChange(view: ContentView) {
    if (view === 'editable') {
      fullContentRequestID += 1;
      fullContentLoading.value = false;
      showFullContent.value = false;
      await nextTick();
      msEditorRef.value?.setValue(editableDraft.value);
      return;
    }
    editableDraft.value = msEditorRef.value?.getValue() || '';
    showFullContent.value = true;
    const requestID = ++fullContentRequestID;
    const requestedDefID = activeDefID.value;
    const requestedEnvName = currentEnv.value.name;
    fullContentLoading.value = true;
    try {
      // Framework 环境实例只保存 overlayContent，详情中的 content 不是可复用的完整内容。
      // 因此无论差异是否修改，都通过服务端合并生成当前完整内容。
      const defaultFileID = detail.value?.baseAppConfigFileId || detail.value?.fileId || activeDef.value?.fileId || '';
      const content = await AppConfigFilesService.previewOverlayMerge({
        appID: appDetailStore.appID,
        id: defaultFileID,
        overlayContent: editableDraft.value,
      });
      if (
        requestID !== fullContentRequestID ||
        !showFullContent.value ||
        requestedDefID !== activeDefID.value ||
        requestedEnvName !== currentEnv.value.name
      ) {
        return;
      }
      await nextTick();
      msEditorRef.value?.setValue(content);
    } catch {
      if (requestID !== fullContentRequestID) return;
      showFullContent.value = false;
      await nextTick();
      msEditorRef.value?.setValue(editableDraft.value);
    } finally {
      if (requestID === fullContentRequestID) fullContentLoading.value = false;
    }
  }

  // 创建普通配置文件，成功后自动选中新文件并加载详情
  async function handleCreateFile(input: PlainFileCreateInput) {
    if (!(await confirmDiscardChanges())) return;
    discardCurrentEdits();
    isCreateSubmitting.value = true;
    try {
      await createPlain(input);
      showCreateSideslider.value = false;
      currentEnv.value = { ...defaultEnv.value };
      await loadCurrentDetail('');
      Message({ theme: 'success', message: t('已创建，请填写文件内容') });
    } catch (error) {
      currentEnv.value = { ...defaultEnv.value };
      if (activeDefID.value) await loadCurrentDetail('').catch(() => undefined);
      throw error;
    } finally {
      isCreateSubmitting.value = false;
    }
  }

  // 删除配置文件；删除当前选中文件时重置环境并加载新选中项
  async function handleDeleteFile(file: ConfigFileListItem) {
    const isDeletingActive = file.id === activeDefID.value;
    if (isDeletingActive && !(await confirmDiscardChanges())) return;
    if (isDeletingActive) discardCurrentEdits();
    await deleteDef(file.id);
    if (isDeletingActive) {
      currentEnv.value = { ...defaultEnv.value };
      if (activeDefID.value) await loadCurrentDetail('');
    }
    Message({ theme: 'success', message: t('操作成功') });
  }

  // 编辑器内容变更：非完整内容视图下记录草稿，并重新校验 admin.ip
  function handleEditorChange(content: string) {
    if (!showFullContent.value) editableDraft.value = content;
    checkAdminIp(content);
  }

  // 收集 monaco 语法错误用于底部状态栏展示
  function handleEditorErr(errors: IMonacoEditorErrorMarkerItem[]) {
    editorErrMessages.value = errors.map(item => item.message);
  }

  // 切换统一/按环境配置；关闭按环境配置会删除各环境单独配置，需二次确认
  async function handleEnvConfigChange(enabled: boolean) {
    if (enabled === isIndependentConfig.value || isModeSaving.value) return;
    if (!(await confirmDiscardChanges())) return;
    // 退出编辑态放到确认之后：关闭按环境配置的二次确认弹窗被取消时，需保留当前编辑态与草稿
    const applyChange = async () => {
      discardCurrentEdits();
      isModeSaving.value = true;
      try {
        await updateDef({ isUnifiedConfig: !enabled });
        if (!enabled) currentEnv.value = { ...defaultEnv.value };
        await fetchDefs(activeDefID.value);
        await loadCurrentDetail('');
        if (!enabled) Message({ theme: 'success', message: t('操作成功') });
      } finally {
        isModeSaving.value = false;
      }
    };
    if (enabled) {
      await applyChange();
      return;
    }
    InfoBox({
      title: t('确认关闭按环境配置？'),
      content: t('关闭后，各环境单独修改的配置及历史版本将被删除，所有环境使用默认配置。'),
      confirmText: t('确认关闭'),
      cancelText: t('取消'),
      onConfirm: applyChange,
    });
  }

  // 切换环境视角（'__default__' 表示默认/基准配置）并重新加载详情
  async function handleEnvSelectChange(envName: string) {
    const realName = envName === '__default__' ? '' : envName;
    if (realName === currentEnv.value.name || !(await confirmDiscardChanges())) return;
    discardCurrentEdits();
    currentEnv.value = realName
      ? perspectiveEnvList.value.find(env => env.name === realName) || { ...defaultEnv.value }
      : { ...defaultEnv.value };
    await loadCurrentDetail(realName);
    if (showVariables.value && realName) viewDefaultEnvVarsRef.value?.setCurEnv(realName);
  }

  function handleFileInfoCancel() {
    isFileInfoEditing.value = false;
    fileInfoFormRef.value?.clearValidate();
  }

  // 进入「文件信息」编辑态并回填表单
  async function handleFileInfoEdit() {
    if (!activeDef.value) return;
    if (!(await confirmDiscardChanges())) return;
    discardCurrentEdits();
    Object.assign(fileInfoFormData, {
      enableEnvVarRender: activeDef.value.enableEnvVarRender,
      mountDir: activeDisplayPath.value,
      mountedEnvNames: [...activeDef.value.mountedEnvNames],
      name: activeDisplayName.value,
      scope: activeMountScope.value,
    });
    fileInfoOriginalScope.value = activeMountScope.value;
    isFileInfoEditing.value = true;
  }

  // 保存「文件信息」：框架文件更新 appModelSpec，普通文件更新 def（仅提交变更字段）
  async function handleFileInfoSave() {
    if (!activeDef.value) return;
    const isValid = await fileInfoFormRef.value
      ?.validate()
      .then(() => true)
      .catch(() => false);
    if (!isValid) return;
    isFileInfoSaving.value = true;
    try {
      if (isFrameworkFile.value) {
        const updatedSpec = cloneDeep(appData.value?.appModelSpec || {}) as AppModelSpecInput;
        const target = updatedSpec[specFieldName.value] as TafSpecOutputObj | TrpcSpecOutputObj;
        if (isFileNameEditable.value) set(target, 'fileName', fileInfoFormData.name);
        set(target, 'filePath', fileInfoFormData.mountDir);
        await updateSpecApi.value({ appID: appDetailStore.appID, appModelSpec: updatedSpec });
        await getAppData();
      } else {
        const changes: UpdateDefInput = {};
        if (fileInfoFormData.name !== activeDef.value.name) changes.name = fileInfoFormData.name;
        if (fileInfoFormData.mountDir !== activeDef.value.mountDir) changes.mountDir = fileInfoFormData.mountDir;
        if (fileInfoFormData.enableEnvVarRender !== activeDef.value.enableEnvVarRender) {
          changes.enableEnvVarRender = fileInfoFormData.enableEnvVarRender;
        }
        // 后端当前无法通过更新接口把指定范围恢复成 nil；切换到全部时提交当前全部环境作为兼容值。
        // 原本就是全部环境时保留原值，避免无修改保存把 nil 语义改成环境快照。
        const nextEnvNames =
          fileInfoFormData.scope === 'all'
            ? fileInfoOriginalScope.value === 'all'
              ? activeDef.value.mountedEnvNames
              : allEnvNames.value
            : fileInfoFormData.mountedEnvNames;
        if (!sameStringSet(nextEnvNames, activeDef.value.mountedEnvNames)) changes.mountedEnvNames = nextEnvNames;
        if (Object.keys(changes).length) await updateDef(changes);
      }
      const shouldResetEnv =
        !isFrameworkFile.value &&
        !!currentEnv.value.name &&
        fileInfoFormData.scope === 'envs' &&
        !fileInfoFormData.mountedEnvNames.includes(currentEnv.value.name);
      await fetchDefs(activeDefID.value);
      if (shouldResetEnv) currentEnv.value = { ...defaultEnv.value };
      await loadCurrentDetail(currentEnv.value.name || '');
      isFileInfoEditing.value = false;
      Message({ theme: 'success', message: t('操作成功') });
    } finally {
      isFileInfoSaving.value = false;
    }
  }

  function handleFileInfoScopeChange(scope: MountScope) {
    if (scope === 'all') fileInfoFormRef.value?.clearValidate('mountedEnvNames');
  }

  // 环境变量面板按环境查询变量列表
  function handleGetVarEnv(envName: string) {
    return EnvvarsService.listAppEnvVars({ appID: appDetailStore.appID, envName });
  }

  // 打开新建配置文件侧滑（先处理未保存改动）
  async function handleOpenCreateFile() {
    if (!(await confirmDiscardChanges())) return;
    discardCurrentEdits();
    showCreateSideslider.value = true;
  }

  // 将当前环境恢复为默认配置（删除该环境的单独配置及历史版本）
  async function handleResetCurrentEnv() {
    const envName = currentEnv.value.name;
    if (!envName) return;
    if (!(await confirmDiscardChanges())) return;
    InfoBox({
      title: t('恢复为默认配置？'),
      content: t('该环境的单独配置及历史版本将被删除，之后使用默认配置。'),
      confirmText: t('确定'),
      cancelText: t('取消'),
      // 确认后才退出编辑态，取消时保留当前编辑态与草稿
      onConfirm: async () => {
        discardCurrentEdits();
        await resetEnv(envName);
        await loadCurrentDetail(envName);
        Message({ theme: 'success', message: t('操作成功') });
      },
    });
  }

  // 版本回滚/刷新后重拉文件列表与当前详情
  async function handleRollbackRefresh() {
    await fetchDefs(activeDefID.value);
    await loadCurrentDetail(currentEnv.value.name || '');
  }

  // 保存文件内容：deleteFile 动作调 resetEnv 删除环境单独配置，否则保存内容并生成新版本；
  // 版本冲突（APP_CONFIG_FILE_VERSION_CONFLICT）时提示刷新后重新编辑
  async function handleSave(description = '', emptyAction?: ClearFileContentAction) {
    let success = false;
    isSubmitLoading.value = true;
    try {
      if (emptyAction === 'deleteFile' && currentEnv.value.name) {
        await resetEnv(currentEnv.value.name);
      } else {
        await saveContent(currentEnv.value.name || '', msEditorRef.value?.getValue() || '', description);
      }
      await fetchDefs(activeDefID.value);
      await loadCurrentDetail(currentEnv.value.name || '');
      isEditing.value = false;
      showSaveVersionDialog.value = false;
      showClearFileContentDialog.value = false;
      showVariables.value = false;
      Message({ theme: 'success', message: t('操作成功') });
      success = true;
    } catch (error) {
      saveVersionDialogRef.value?.stopLoading();
      if (hasErrorCode(error, 'APP_CONFIG_FILE_VERSION_CONFLICT')) {
        Message({
          theme: 'error',
          message: t('当前配置已被他人更新。为避免数据被覆盖，请刷新页面获取最新版本后重新编辑。'),
        });
      } else {
        Message({ theme: 'error', message: t('保存失败，请稍后重试') });
      }
    } finally {
      isSubmitLoading.value = false;
      if (!success && emptyAction === 'saveEmpty') showClearFileContentDialog.value = true;
    }
  }

  // 切换左侧选中的文件并加载其详情
  async function handleSelectFile(id: string) {
    if (id === activeDefID.value || !(await confirmDiscardChanges())) return;
    discardCurrentEdits();
    activeDefID.value = id;
    currentEnv.value = { ...defaultEnv.value };
    showVariables.value = false;
    await loadCurrentDetail('');
  }

  function handleStartEdit() {
    if (effectiveEditableContentField.value === 'none') return;
    isEditing.value = true;
  }

  function handleViewAdminDoc() {
    window.open(`${import.meta.env.BK_DOC_URL}/p/4016336887`, '_blank');
  }

  function hasFileInfoChanges() {
    if (!isFileInfoEditing.value || !activeDef.value) return false;
    if (fileInfoFormData.name !== activeDisplayName.value || fileInfoFormData.mountDir !== activeDisplayPath.value) {
      return true;
    }
    if (isFrameworkFile.value) return false;
    return (
      fileInfoFormData.enableEnvVarRender !== activeDef.value.enableEnvVarRender ||
      fileInfoFormData.scope !== fileInfoOriginalScope.value ||
      (fileInfoFormData.scope === 'envs' &&
        !sameStringSet(fileInfoFormData.mountedEnvNames, activeDef.value.mountedEnvNames))
    );
  }

  // 页面初始化：并行拉取应用详情、环境列表、文件定义列表，再加载当前详情
  async function initPage() {
    if (!appDetailStore.appID) {
      resetPage();
      return;
    }
    isLoading.value = true;
    try {
      await Promise.all([getAppData(), getEnvList(), fetchDefs()]);
      currentEnv.value = { ...defaultEnv.value };
      if (activeDefID.value) await loadCurrentDetail('');
    } finally {
      isLoading.value = false;
    }
  }

  // 加载当前文件在指定环境的详情并写入编辑器
  async function loadCurrentDetail(envName = currentEnv.value.name || '') {
    const requestID = ++detailLoadRequestID;
    const requestedAppID = appDetailStore.appID;
    const requestedDefID = activeDefID.value;
    fullContentRequestID += 1;
    fullContentLoading.value = false;
    showFullContent.value = false;
    isEditorLoading.value = true;
    try {
      await refreshActive(envName);
      if (
        requestID !== detailLoadRequestID ||
        requestedAppID !== appDetailStore.appID ||
        requestedDefID !== activeDefID.value ||
        envName !== currentEnv.value.name
      ) {
        return;
      }
      await applyDetailToEditor();
    } catch (error) {
      if (requestID !== detailLoadRequestID) return;
      originalContent.value = '';
      editableDraft.value = '';
      msEditorRef.value?.setValue('');
      throw error;
    } finally {
      if (requestID === detailLoadRequestID) isEditorLoading.value = false;
    }
  }

  // 清空页面全部状态（无 appID 时使用）
  function resetPage() {
    detailLoadRequestID += 1;
    envListRequestID += 1;
    fullContentRequestID += 1;
    reset();
    appData.value = {} as AppDetailOutputObj;
    envList.value = [];
    envListLoading.value = false;
    envListLoadFailed.value = false;
    currentEnv.value = { ...defaultEnv.value };
    originalContent.value = '';
    editableDraft.value = '';
    isEditing.value = false;
    isFileInfoEditing.value = false;
    showVariables.value = false;
    showFullContent.value = false;
    fullContentLoading.value = false;
    isEditorLoading.value = false;
    msEditorRef.value?.setValue('');
  }

  // 忽略顺序比较两个字符串数组是否相同
  function sameStringSet(left: string[], right: string[]) {
    return left.length === right.length && left.every(item => right.includes(item));
  }

  // 框架文件按环境配置下，当前环境实例内容被清空时需二次确认
  function shouldShowClearFileContentDialog() {
    return (
      isFrameworkFile.value &&
      isIndependentConfig.value &&
      !!currentEnv.value.name &&
      !!detail.value?.hasEnvInstance &&
      !(msEditorRef.value?.getValue() || '').trim()
    );
  }

  function toggleVariables() {
    showVariables.value = !showVariables.value;
    if (showVariables.value && currentEnv.value.name) {
      nextTick(() => viewDefaultEnvVarsRef.value?.setCurEnv(currentEnv.value.name));
    }
  }

  // 同步 store 中的应用详情；appID 变化时重新初始化页面
  watch(
    () => appDetailStore.appDetail,
    value => {
      appData.value = value || ({} as AppDetailOutputObj);
    },
    { immediate: true },
  );
  watch(() => appDetailStore.appID, initPage, { immediate: true });
</script>

<style lang="postcss" scoped>
  .mount-env-form-item :deep(.bk-form-content) {
    display: flex;
    flex-direction: column;
  }

  .mount-env-form-item :deep(.bk-radio-group) {
    display: contents;
  }

  .mount-env-form-item :deep(.bk-form-error) {
    position: static;
    order: 1;
  }

  .mount-env-form-item :deep(.bk-radio-group > :last-child) {
    order: 2;
  }

  .editor-aside-layout > :deep(.bk-resize-layout-main) {
    padding-right: 16px;
  }

  .editor-aside-layout :deep(.bk-resize-layout-aside-content) {
    padding-right: 16px;
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
