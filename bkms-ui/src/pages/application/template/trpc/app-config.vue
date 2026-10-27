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
  <div class="flex flex-col p-[20px] !max-w-[1400px] w-full">
    <ToggleCard
      class="bg-[#fff] shadow-sm px-[16px]"
      content-class="overflow-x-auto"
      :name="$t('应用配置')"
    >
      <Form
        ref="formRef"
        class="mx-auto w-[calc(100%-490px)] min-w-[640px]"
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
      :class="['bg-[#fff] shadow-sm p-[16px] mt-[16px]', { 'h-full': isYamlCardExpanded }]"
      content-class="h-full"
      :name="editorTitle"
      @change="handleYamlCardChange"
    >
      <template #header-right>
        <IconTextButton
          v-show="isYamlCardExpanded"
          :active="showEnvVar"
          class="text-[12px] p-[10px]"
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
                class="h-full"
                :model-value="builderTemplate"
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
  import { onMounted, ref, watch } from 'vue';

  import { Form, ResizeLayout } from 'bkui-vue';
  import { ApiServerService } from '~/api/modules/bkmsserver';

  import builderTemplate from './trpc-go-template.yaml?raw';

  const props = defineProps<{
    editorTitle?: string;
    value: { appModelSpec: AppModelSpecInput };
  }>();

  import KeyValue from '~/components/key-value.vue';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';
  import StartupCommandInput from '~/components/startup-command-input.vue';
  import useEnvManager from '~/composables/use-env-manager';

  import type { AppModelSpecInput } from '~/@types/v1/app';
  import type { IMonacoEditorErrorMarkerItem } from '~/common/util';

  const { envList, handleGetEnvList } = useEnvManager();

  const formData = ref<{
    args: string[];
    command: string[];
    env: Record<string, string>[];
    shCommand: string;
  }>({
    command: props.value.appModelSpec.command || [],
    args: props.value.appModelSpec.args || [],
    env: [],
    shCommand: props.value.appModelSpec.shCommand || '',
  });

  const startupCommandRef = ref<InstanceType<typeof StartupCommandInput> | null>(null);
  const msEditorRef = ref<InstanceType<typeof MsEditor> | null>(null);
  // yaml异常
  const editorErr = ref<{
    message: string[];
    type: string;
  }>({
    type: '',
    message: [],
  });
  const errorRef = ref<InstanceType<typeof ResizeLayout> | null>(null);
  function getValue() {
    formData.value.env = formData.value.env.map(item => ({ key: item.key, value: item.value }));
    return {
      ...formData.value,
      ...startupCommandRef.value?.getValue(),
      content: msEditorRef.value?.getValue() || '',
    };
  }
  function handleEditorErr(err: IMonacoEditorErrorMarkerItem[]) {
    // 捕获编辑器错误提示
    editorErr.value.type = 'content'; // 编辑内容错误
    editorErr.value.message = err.map(item => item.message);
    hideOrShowError();
  }

  function hideOrShowError() {
    if (!editorErr.value?.message?.length && errorRef.value) {
      errorRef.value.asideRef.hidden = true;
    } else if (editorErr.value?.message?.length && errorRef.value) {
      errorRef.value.asideRef.hidden = false;
    }
  }

  function resetStatus() {
    editorErr.value = {
      type: '',
      message: [],
    };
    hideOrShowError();
  }
  async function validate() {
    const commandValid = await startupCommandRef.value?.validate();
    if (!commandValid || editorErr.value?.message?.length) return false;
    return true;
  }

  const showEnvVar = ref(false);
  const isYamlCardExpanded = ref(true);
  const resizeLayoutRef = ref<InstanceType<typeof ResizeLayout> | null>(null);

  // 获取应用环境变量
  function handleGetVarEnv(env: string) {
    const envID = envList.value.find(item => item.name === env)?.id;
    if (!envID) return Promise.resolve([]);
    return ApiServerService.ListEnvAvailableEnvVars({ envID });
  }

  // 处理yaml卡片展开/收起状态变化
  function handleYamlCardChange(expanded: boolean) {
    isYamlCardExpanded.value = expanded;
  }

  function setEnvVarAsideVisible(visible: boolean) {
    if (resizeLayoutRef.value?.asideRef) {
      resizeLayoutRef.value.asideRef.hidden = !visible;
    }
  }

  function toggleEnvVarShow() {
    showEnvVar.value = !showEnvVar.value;
    setEnvVarAsideVisible(showEnvVar.value);
  }

  // 初始化隐藏错误侧栏
  watch(
    [editorErr.value, errorRef],
    () => {
      hideOrShowError();
    },
    { immediate: true },
  );

  onMounted(async () => {
    await handleGetEnvList();
    setEnvVarAsideVisible(showEnvVar.value);
  });

  defineExpose({
    getValue,
    validate,
    resetStatus,
  });
</script>

<style lang="postcss" scoped>
  .yaml-sideslider-layout > :deep(.bk-resize-layout-main) {
    padding-right: 16px;
  }

  .yaml-sideslider-layout :deep(.bk-resize-layout-aside-content) {
    padding-right: 16px;
  }
</style>
