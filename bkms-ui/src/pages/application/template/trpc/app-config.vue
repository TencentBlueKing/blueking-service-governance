<!--
 - TencentBlueKing is pleased to support the open source community by making
 - 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
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
  <div class="flex flex-col p-[20px] max-w-[1400px] min-w-0 w-full">
    <ToggleCard
      class="bg-[#fff] rounded-[2px] shadow-[0_2px_4px_0_#1919290d]"
      :name="$t('启动配置')"
      type="normal"
    >
      <Form
        ref="formRef"
        class="config-fields pb-[24px]"
        :label-width="135"
        :model="formData"
      >
        <Form.FormItem :label="$t('启动命令')">
          <StartupCommandInput
            ref="startupCommandRef"
            default-mode="shell"
            exec-background
            :value="formData"
          />
        </Form.FormItem>
        <Form.FormItem
          class="!mb-0"
          :label="$t('环境变量')"
        >
          <KeyValue
            v-model="formData.env"
            :key-placeholder="$t('请输入变量名')"
            textarea
            :value-placeholder="$t('请输入变量值')"
          />
        </Form.FormItem>
      </Form>
    </ToggleCard>
    <ToggleCard
      :class="[
        'bg-[#fff] rounded-[2px] overflow-hidden shadow-[0_2px_4px_0_#1919290d] mt-[16px]',
        { 'h-full': isYamlCardExpanded },
      ]"
      content-class="!mt-0 h-full p-[24px]"
      type="normal"
      @change="handleYamlCardChange"
    >
      <template #title>
        <span class="ml-[10px] select-none font-bold text-[14px]">{{ editorTitle }}</span>
        <span
          v-show="showTemplateSelect"
          class="mx-[12px] inline-block h-[14px] w-px bg-[#DCDEE5]"
        ></span>
        <span
          v-show="showTemplateSelect"
          class="inline-flex"
          @click.stop
        >
          <FrameworkTemplateSelect
            app-type="trpc"
            :language="language"
            variant="text"
            :workspace-id="workspaceId"
            @availability="handleTemplateAvailability"
            @loading="handleTemplateLoading"
            @select="handleTemplateSelect"
          />
        </span>
      </template>
      <template #header-right>
        <IconTextButton
          v-show="isYamlCardExpanded"
          :active="showEnvVar"
          class="text-[12px] !pr-0"
          icon="bkms-icon bkms-icon-variable"
          :text="$t('环境变量')"
          @click="toggleEnvVarShow()"
        />
      </template>
      <ResizeLayout
        ref="resizeLayoutRef"
        :border="false"
        class="h-[60vh] min-h-0 yaml-sideslider-layout"
        initial-divide="50%"
        placement="right"
      >
        <template #aside>
          <!-- 默认环境变量 -->
          <ViewDefaultEnvVars
            class="h-full ml-[16px]"
            :copy-format="key => `\${${key}}`"
            :custom-request-fn="handleGetVarEnv"
            :env-list="envList"
            :express-template="'${var_key}'"
          />
        </template>
        <template #main>
          <!-- 代码编辑器 -->
          <ResizeLayout
            ref="errorRef"
            :auto-minimize="true"
            :border="false"
            class="h-full"
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
                v-model="editorContent"
                class="h-full"
                :lang="editorLang"
                :readonly="templateLoading"
                :title="editorTitle"
                @error="handleEditorErr"
              />
            </template>
          </ResizeLayout>
        </template>
      </ResizeLayout>
    </ToggleCard>
  </div>
</template>
<script setup lang="ts">
  import { ref } from 'vue';

  import { Form, ResizeLayout } from 'bkui-vue';

  import FrameworkTemplateSelect from '../components/framework-template-select.vue';
  import { useAppConfigCommon } from '../composables/use-app-config-common';

  const props = defineProps<{
    content?: string;
    editorLang?: string;
    editorTitle?: string;
    language: string;
    value: { appModelSpec: AppModelSpecInput };
    workspaceId: string;
  }>();
  import KeyValue from '~/components/key-value.vue';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';
  import StartupCommandInput from '~/components/startup-command-input.vue';

  import type { AppModelSpecInput } from '~/@types/v1/app';
  import type { TemplateOutputObj } from '~/@types/v1/framework-config-templates';

  const {
    editorErr,
    errorRef,
    envList,
    handleEditorErr,
    handleGetVarEnv,
    handleTemplateAvailability,
    handleTemplateLoading,
    handleYamlCardChange,
    isYamlCardExpanded,
    resetStatus,
    resizeLayoutRef,
    showEnvVar,
    showTemplateSelect,
    templateLoading,
    toggleEnvVarShow,
  } = useAppConfigCommon();

  const editorContent = ref(props.content || '');

  const formData = ref<{
    args: string[];
    command: string[];
    env: Record<string, string>[];
    shCommand: string;
  }>({
    command: props.value.appModelSpec.command || [],
    args: props.value.appModelSpec.args || [],
    env: (props.value.appModelSpec.envVars || []).map(item => ({ key: item.key, value: item.value ?? '' })),
    shCommand: props.value.appModelSpec.shCommand || '',
  });

  const startupCommandRef = ref<InstanceType<typeof StartupCommandInput> | null>(null);
  const msEditorRef = ref<InstanceType<typeof MsEditor> | null>(null);

  function getValue() {
    const env = formData.value.env.map(item => ({
      ...props.value.appModelSpec.envVars?.find(variable => variable.key === item.key),
      key: item.key,
      value: item.value,
    }));
    return {
      ...formData.value,
      env,
      ...startupCommandRef.value?.getValue(),
      content: msEditorRef.value?.getValue() || '',
    };
  }

  function handleTemplateSelect(template: TemplateOutputObj) {
    if (template.content === undefined) return;
    editorContent.value = template.content;
    resetStatus();
  }

  async function validate() {
    const commandValid = await startupCommandRef.value?.validate();
    if (!commandValid || editorErr.value?.message?.length) return false;
    return true;
  }

  defineExpose({
    getValue,
    validate,
    resetStatus,
  });
</script>

<style lang="postcss" scoped>
  .config-fields {
    padding-inline: clamp(24px, calc((100vw - 910px) / 2), 245px);
  }

  .yaml-sideslider-layout > :deep(.bk-resize-layout-main) {
    padding-right: 16px;
  }

  .yaml-sideslider-layout :deep(.bk-resize-layout-aside-content) {
    padding-right: 16px;
  }
</style>
