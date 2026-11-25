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
    @closed="handleClosed"
  >
    <template #header>
      <DividerHeader>
        <template #title>{{ headerTitle }}</template>
        {{ t('框架配置模板') }}
      </DividerHeader>
    </template>

    <Form
      ref="formRef"
      class="flex h-[calc(100vh-100px)] flex-col px-[24px] pb-[16px] pt-[24px]"
      form-type="vertical"
      :model="formModel"
      :rules="rules"
    >
      <Form.FormItem
        v-if="scene === 'create'"
        :label="t('类型')"
        property="appType"
        required
      >
        <Radio.Group
          class="w-full"
          :disabled="formLoading"
          :model-value="formModel.appType"
          @update:model-value="handleAppTypeChange"
        >
          <Radio.Button
            class="w-full"
            :label="APP_FRAMEWORKS.TRPC"
          >
            tRPC
          </Radio.Button>
          <Radio.Button
            class="w-full"
            :label="APP_FRAMEWORKS.TAF"
          >
            TAF
          </Radio.Button>
        </Radio.Group>
      </Form.FormItem>
      <Form.FormItem
        :label="t('模板名称')"
        property="name"
        required
      >
        <Input
          v-model.trim="formModel.name"
          :disabled="readonly || formLoading"
          :maxlength="64"
          :placeholder="t('请输入模板名称')"
        />
      </Form.FormItem>
      <Form.FormItem
        v-if="activeAppType === APP_FRAMEWORKS.TRPC"
        :label="t('语言')"
        property="language"
        required
      >
        <Select
          v-model="formModel.language"
          :clearable="false"
          :disabled="scene !== 'create' || formLoading"
        >
          <Select.Option
            v-for="language in TEMPLATE_LANGUAGES"
            :key="language"
            :label="TEMPLATE_LANGUAGE_LABELS[language]"
            :value="language"
          />
        </Select>
      </Form.FormItem>
      <Form.FormItem :label="t('配置文件名称')">
        <Input
          v-model.trim="formModel.fileName"
          disabled
        />
      </Form.FormItem>
      <Checkbox
        v-model="formModel.isDefault"
        :disabled="readonly || formLoading"
      >
        {{ t('创建应用时默认使用该模板') }}
      </Checkbox>
      <div class="mt-[16px] min-h-0 flex-1">
        <MsEditor
          v-model="formModel.fileContent"
          class="h-full"
          :lang="editorLang"
          :readonly="readonly || formLoading"
          :title="formModel.fileName || t('框架配置')"
        />
      </div>
      <p
        v-if="contentError"
        class="mt-[8px] text-[12px] text-[#EA3636]"
      >
        {{ t('请输入配置文件内容') }}
      </p>
    </Form>

    <template #footer>
      <div class="flex items-center gap-[8px]">
        <Button
          class="!w-[60px] !min-w-[60px]"
          :disabled="formLoading || detailLoadFailed"
          :loading="saving"
          theme="primary"
          @click="handleFooterConfirm"
        >
          {{ t('确定') }}
        </Button>
        <Button
          class="!w-[60px] !min-w-[60px]"
          @click="visible = false"
        >
          {{ t('取消') }}
        </Button>
      </div>
    </template>
  </Sideslider>
</template>

<script setup lang="ts">
  import { computed, nextTick, ref, watch } from 'vue';

  import { Button, Checkbox, Form, Input, Message, Radio, Select, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { FrameworkConfigTemplatesService } from '~/api/modules/v1';
  import { APP_FRAMEWORKS } from '~/common/app-types';
  import DividerHeader from '~/components/divider-header.vue';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';
  import { useSpaceStore } from '~/stores/space';

  import {
    type FrameworkAppType,
    type FrameworkTemplateRecord as FrameworkTemplate,
    defaultTemplateFileName,
    defaultTemplateLanguage,
    normalizeTemplate,
    TEMPLATE_LANGUAGE_LABELS,
    TEMPLATE_LANGUAGES,
  } from './types';

  import type { TemplateOutputObj } from '~/@types/v1/framework-config-templates';

  type Scene = 'create' | 'edit' | 'view';

  const props = defineProps<{
    appType: FrameworkAppType;
    isShow: boolean;
    scene: Scene;
    template: FrameworkTemplate | null;
    templates: FrameworkTemplate[];
  }>();

  const emit = defineEmits<{
    success: [appType: FrameworkAppType];
    'update:isShow': [value: boolean];
  }>();

  const { t } = useI18n();
  const spaceStore = useSpaceStore();
  const formRef = ref<InstanceType<typeof Form> | null>(null);
  const saving = ref(false);
  const contentError = ref(false);
  const initialDefault = ref(false);
  // 初始化数据（模板详情/平台模板）加载中：期间禁用编辑与保存
  const formLoading = ref(false);
  // 编辑/查看场景模板详情加载失败：保持空表单并阻止保存，避免把空内容写入模板
  const detailLoadFailed = ref(false);
  let syncingForm = false;
  let templateSyncSeq = 0;
  type FormLanguage = '' | 'cpp' | 'go';

  interface FormModel {
    appType: FrameworkAppType;
    fileContent: string;
    fileName: string;
    isDefault: boolean;
    language: FormLanguage;
    name: string;
  }

  const formModel = ref<FormModel>({
    appType: APP_FRAMEWORKS.TRPC,
    fileContent: '',
    fileName: '',
    isDefault: false,
    language: 'go',
    name: '',
  });

  const visible = computed({
    get: () => props.isShow,
    set: value => emit('update:isShow', value),
  });
  const readonly = computed(() => props.scene === 'view');
  const activeAppType = computed(() => (props.scene === 'create' ? formModel.value.appType : props.appType));
  const editorLang = computed(() => (activeAppType.value === APP_FRAMEWORKS.TAF ? 'xml' : 'yaml'));
  const headerTitle = computed(() => {
    if (props.scene === 'edit') return t('编辑模板');
    if (props.scene === 'view') return t('查看模板');
    return t('新建模板');
  });
  const rules = {
    appType: [{ required: true, message: t('请选择'), trigger: 'change' }],
    name: [{ required: true, message: t('请输入模板名称'), trigger: 'blur' }],
    language: [{ required: true, message: t('请选择'), trigger: 'change' }],
  };

  function handleAppTypeChange(appType: FrameworkAppType) {
    if (formLoading.value || formModel.value.appType === appType) return;
    formModel.value.appType = appType;
  }

  function handleClosed() {
    contentError.value = false;
    formRef.value?.clearValidate();
  }

  async function handleConfirm() {
    // 初始化未完成或详情加载失败时禁止保存，防止把上一模板/空表单的数据写入当前模板
    if (formLoading.value || detailLoadFailed.value) return;
    contentError.value = !formModel.value.fileContent.trim();
    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid || contentError.value || !spaceStore.currentSpace) return;

    saving.value = true;
    const payload = {
      workspaceID: spaceStore.currentSpace,
      name: formModel.value.name.trim(),
      fileName: formModel.value.fileName.trim(),
      content: formModel.value.fileContent,
    };
    const defaultPayload =
      props.scene === 'create' || formModel.value.isDefault !== initialDefault.value
        ? { isDefault: formModel.value.isDefault }
        : {};
    try {
      if (props.scene === 'edit' && props.template) {
        await FrameworkConfigTemplatesService.updateWorkspaceFrameworkConfigTemplate({
          ...payload,
          ...defaultPayload,
          templateID: props.template.id,
        });
      } else {
        await FrameworkConfigTemplatesService.createWorkspaceFrameworkConfigTemplate({
          ...payload,
          ...defaultPayload,
          framework: activeAppType.value,
          language: formModel.value.language,
        });
      }
      Message({ theme: 'success', message: t('保存成功') });
      visible.value = false;
      emit('success', activeAppType.value);
    } finally {
      saving.value = false;
    }
  }

  function handleFooterConfirm() {
    if (readonly.value) {
      visible.value = false;
      return;
    }
    void handleConfirm();
  }

  async function loadPlatformTemplate(appType: FrameworkAppType, language: FormLanguage) {
    const cached = platformOf(appType, language);
    if (cached || appType === APP_FRAMEWORKS.TAF || !spaceStore.currentSpace) return cached;
    let result: TemplateOutputObj[] = [];
    try {
      const response = await FrameworkConfigTemplatesService.listWorkspaceFrameworkConfigTemplates({
        workspaceID: spaceStore.currentSpace,
        framework: appType,
        ...(language ? { language } : {}),
      });
      result = Array.isArray(response) ? response : [];
    } catch {
      return;
    }
    const platform = result.find(
      item => item.builtin && item.framework === appType && (!language || item.language === language),
    );
    if (!platform) return;
    return normalizeTemplate(platform, language);
  }

  async function loadTemplateDetail(template: FrameworkTemplate | null | undefined) {
    if (!template || !spaceStore.currentSpace || template.fileContent) return template;
    return FrameworkConfigTemplatesService.getWorkspaceFrameworkConfigTemplate({
      workspaceID: spaceStore.currentSpace,
      templateID: template.id,
    })
      .then(detail => ({
        ...template,
        fileContent: detail?.content || '',
        fileName: detail?.fileName || template.fileName,
      }))
      .catch(() => null);
  }

  function platformOf(appType: FrameworkAppType, language: string) {
    return props.templates.find(
      item => item.source === 'platform' && item.appType === appType && item.language === language,
    );
  }

  async function resetForm() {
    const seq = ++templateSyncSeq;
    // 打开侧栏时立即清空旧表单并进入加载态：
    // 详情请求完成前不残留上一模板的数据，也无法保存，防止用当前模板 ID 提交上一模板的内容
    syncingForm = true;
    formLoading.value = true;
    detailLoadFailed.value = false;
    formModel.value = {
      appType: props.appType,
      fileContent: '',
      fileName: '',
      isDefault: false,
      language: defaultTemplateLanguage(props.appType),
      name: '',
    };
    contentError.value = false;
    formRef.value?.clearValidate();
    try {
      const current = await loadTemplateDetail(props.template);
      if (seq !== templateSyncSeq || !visible.value) return;
      if (props.scene !== 'create' && props.template && !current) {
        // 模板详情加载失败：弹窗提示并保持空表单、禁用保存，而非把失败当作空内容提交
        detailLoadFailed.value = true;
        Message({ theme: 'error', message: t('模板详情加载失败，请重试') });
        return;
      }
      const displayName = current?.source === 'platform' ? t('平台默认') : current?.name || '';
      const appType = props.appType;
      const language = current?.language || defaultTemplateLanguage(appType);
      const starter =
        props.scene === 'create' ? await loadTemplateDetail(await loadPlatformTemplate(appType, language)) : null;
      if (seq !== templateSyncSeq || !visible.value) return;
      const starterFileName = starter?.fileName || defaultTemplateFileName(appType, language);
      const starterFileContent = starter?.fileContent || '';
      initialDefault.value = props.scene === 'create' ? false : Boolean(current?.isDefault);
      formModel.value = {
        appType,
        name: props.scene === 'create' ? '' : displayName,
        language,
        fileName:
          props.scene === 'create' ? starterFileName : current?.fileName || defaultTemplateFileName(appType, language),
        fileContent: props.scene === 'create' ? starterFileContent : current?.fileContent || '',
        isDefault: props.scene === 'create' ? false : Boolean(current?.isDefault),
      };
      contentError.value = false;
      formRef.value?.clearValidate();
    } finally {
      if (seq === templateSyncSeq) {
        formLoading.value = false;
        await nextTick();
        if (seq === templateSyncSeq) syncingForm = false;
      }
    }
  }

  async function syncPlatformTemplate(appType: FrameworkAppType, language: FormLanguage) {
    const seq = ++templateSyncSeq;
    const platform = await loadTemplateDetail(await loadPlatformTemplate(appType, language));
    if (
      seq !== templateSyncSeq ||
      !visible.value ||
      formModel.value.appType !== appType ||
      formModel.value.language !== language
    ) {
      return;
    }
    formModel.value.fileName = platform?.fileName || defaultTemplateFileName(appType, language);
    formModel.value.fileContent = platform?.fileContent || '';
  }

  watch(
    () => formModel.value.appType,
    (next, previous) => {
      if (syncingForm || !visible.value || props.scene !== 'create' || !previous || previous === next) return;
      formModel.value.language = defaultTemplateLanguage(next);
      void syncPlatformTemplate(next, formModel.value.language);
    },
  );

  watch(
    () => formModel.value.language,
    (language, previous) => {
      if (
        syncingForm ||
        !visible.value ||
        props.scene !== 'create' ||
        formModel.value.appType !== APP_FRAMEWORKS.TRPC ||
        !previous ||
        previous === language
      ) {
        return;
      }
      void syncPlatformTemplate(APP_FRAMEWORKS.TRPC, language);
    },
  );

  watch(
    () => props.isShow,
    shown => {
      if (shown) void resetForm();
    },
    { immediate: true },
  );
</script>
