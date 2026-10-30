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
          :model-value="formModel.appType"
          @update:model-value="handleAppTypeChange"
        >
          <Radio.Button
            class="w-full"
            label="trpc"
            @click="handleAppTypeChange('trpc')"
          >
            tRPC
          </Radio.Button>
          <Radio.Button
            class="w-full"
            label="taf"
            @click="handleAppTypeChange('taf')"
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
          :disabled="readonly"
          :maxlength="64"
          :placeholder="t('请输入模板名称')"
        />
      </Form.FormItem>
      <Form.FormItem
        v-if="activeAppType === 'trpc'"
        :label="t('语言')"
        property="language"
        required
      >
        <Select
          v-model="formModel.language"
          :clearable="false"
          :disabled="scene !== 'create'"
        >
          <Select.Option
            label="Go"
            value="go"
          />
          <Select.Option
            label="C++"
            value="cpp"
          />
        </Select>
      </Form.FormItem>
      <Form.FormItem
        :label="t('配置文件名称')"
        property="fileName"
        required
      >
        <Input
          v-model.trim="formModel.fileName"
          disabled
          :maxlength="128"
          :placeholder="t('请输入配置文件名称')"
        />
      </Form.FormItem>
      <Checkbox
        v-model="formModel.isDefault"
        :disabled="readonly"
      >
        {{ t('创建应用时默认使用该模板') }}
      </Checkbox>
      <div class="mt-[16px] min-h-0 flex-1">
        <MsEditor
          v-model="formModel.fileContent"
          class="h-full"
          :lang="editorLang"
          :readonly="readonly"
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
  import DividerHeader from '~/components/divider-header.vue';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';
  import { useSpaceStore } from '~/stores/space';

  import type { FrameworkAppType, FrameworkTemplateRecord as FrameworkTemplate } from './types';
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
    appType: 'trpc',
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
  const editorLang = computed(() => (activeAppType.value === 'taf' ? 'xml' : 'yaml'));
  const headerTitle = computed(() => {
    if (props.scene === 'edit') return t('编辑模板');
    if (props.scene === 'view') return t('查看模板');
    return t('新建模板');
  });
  const rules = {
    appType: [{ required: true, message: t('请选择'), trigger: 'change' }],
    name: [{ required: true, message: t('请输入模板名称'), trigger: 'blur' }],
    language: [{ required: true, message: t('请选择'), trigger: 'change' }],
    fileName: [{ required: true, message: t('请输入配置文件名称'), trigger: 'blur' }],
  };

  function defaultFileName(appType: FrameworkAppType, language: string) {
    if (appType === 'taf') return 'config.conf';
    return `trpc_${language || 'go'}.yaml`;
  }

  function handleAppTypeChange(appType: Event | FrameworkAppType) {
    if (typeof appType !== 'string' || formModel.value.appType === appType) return;
    formModel.value.appType = appType;
  }

  function handleClosed() {
    contentError.value = false;
    formRef.value?.clearValidate();
  }

  async function handleConfirm() {
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
    if (cached || appType === 'taf' || !spaceStore.currentSpace) return cached;
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
    const templateLanguage = platform.language === 'go' || platform.language === 'cpp' ? platform.language : language;
    return {
      appType,
      createdAt: platform.createdAt || '',
      fileContent: platform.content || '',
      fileName: platform.fileName || '',
      id: platform.id || '',
      isDefault: Boolean(platform.isDefault),
      language: templateLanguage,
      name: platform.name || '',
      source: 'platform' as const,
      updatedAt: platform.updatedAt,
    } satisfies FrameworkTemplate;
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
      .catch(() => template);
  }

  function platformOf(appType: FrameworkAppType, language: string) {
    return props.templates.find(
      item => item.source === 'platform' && item.appType === appType && item.language === language,
    );
  }

  async function resetForm() {
    const seq = ++templateSyncSeq;
    const current = await loadTemplateDetail(props.template);
    const displayName = current?.source === 'platform' ? t('平台默认') : current?.name || '';
    const appType = props.appType;
    const language = appType === 'trpc' ? current?.language || 'go' : 'cpp';
    const starter =
      props.scene === 'create' ? await loadTemplateDetail(await loadPlatformTemplate(appType, language)) : null;
    if (seq !== templateSyncSeq || !visible.value) return;
    syncingForm = true;
    const starterFileName = starter?.fileName || defaultFileName(appType, language);
    const starterFileContent = starter?.fileContent || '';
    initialDefault.value = props.scene === 'create' ? false : Boolean(current?.isDefault);
    formModel.value = {
      appType,
      name: props.scene === 'create' ? '' : displayName,
      language,
      fileName: props.scene === 'create' ? starterFileName : current?.fileName || defaultFileName(appType, language),
      fileContent: props.scene === 'create' ? starterFileContent : current?.fileContent || '',
      isDefault: props.scene === 'create' ? false : Boolean(current?.isDefault),
    };
    contentError.value = false;
    formRef.value?.clearValidate();
    await nextTick();
    if (seq === templateSyncSeq) syncingForm = false;
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
    formModel.value.fileName = platform?.fileName || defaultFileName(appType, language);
    formModel.value.fileContent = platform?.fileContent || '';
  }

  watch(
    () => formModel.value.appType,
    (next, previous) => {
      if (syncingForm || !visible.value || props.scene !== 'create' || !previous || previous === next) return;
      const nextLanguage = next === 'trpc' ? 'go' : 'cpp';
      formModel.value.language = nextLanguage;
      void syncPlatformTemplate(next, nextLanguage);
    },
  );

  watch(
    () => formModel.value.language,
    (language, previous) => {
      if (
        syncingForm ||
        !visible.value ||
        props.scene !== 'create' ||
        formModel.value.appType !== 'trpc' ||
        !previous ||
        previous === language
      ) {
        return;
      }
      void syncPlatformTemplate('trpc', language);
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
