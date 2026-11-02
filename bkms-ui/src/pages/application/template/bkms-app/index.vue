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
  <ScrollLayout
    class="bg-[#F5F7FA]"
    content-class="w-full relative !flex-1"
  >
    <Alert
      v-if="error"
      class="m-[24px]"
      theme="warning"
    >
      <template #title>
        {{ $t('应用运行时目录加载失败，请重试') }}
        <Button
          text
          theme="primary"
          @click="load"
          >{{ $t('重试') }}</Button
        >
      </template>
    </Alert>
    <div
      v-else-if="loading"
      class="flex justify-center p-[40px]"
    >
      <Loading
        mode="spin"
        theme="primary"
      />
    </div>
    <template v-else-if="runtime && form">
      <div
        v-if="step === 2"
        class="w-full max-w-[1400px] p-[20px]"
      >
        <ToggleCard
          class="bg-[#fff] rounded-[2px] shadow-[0_2px_4px_0_#1919290d]"
          :name="$t('应用模板')"
          type="normal"
        >
          <div class="px-[24px] pb-[20px]">
            <div class="flex items-center rounded-[2px] bg-[#F5F7FA] px-[16px] py-[12px]">
              <span class="template-summary-icon">
                <component
                  :is="selectedLogo"
                  v-if="selectedLogo"
                  class="size-[32px] rounded-[4px]"
                />
                <i
                  v-else
                  :class="[
                    'bkms-icon text-[#1B44C8]',
                    framework === 'trpc' ? 'bkms-icon-trpc text-[14px]' : 'bkms-icon-taf text-[32px]',
                  ]"
                ></i>
              </span>
              <div class="ml-[12px] min-w-0 flex-1">
                <div class="text-[14px] font-bold leading-[22px] text-[#313238]">{{ selectedLabel }}</div>
                <div class="mt-[2px] text-[12px] leading-[18px] text-[#979BA5]">
                  {{ $t(selectedDescription) }}
                </div>
              </div>
              <Button
                class="ml-[16px] shrink-0"
                text
                theme="primary"
                @click="changeTemplate"
                >{{ $t('更换') }}</Button
              >
            </div>
          </div>
        </ToggleCard>
        <ParamConfig
          :key="`${framework}:${language}`"
          ref="paramRef"
          class="mt-[16px]"
          :form="form"
          :platform-build-supported="isRuntimeModSupported(runtime, 'platformBuild')"
        />
      </div>
      <TrpcAppConfig
        v-else-if="step === 3 && form.framework === 'trpc'"
        :key="`${framework}:${language}`"
        ref="trpcRef"
        :content="form.frameworkConfigFile.content"
        :editor-lang="editorLang"
        :editor-title="form.frameworkConfigFile.name"
        :value="{ appModelSpec: form.appModelSpec }"
      />
      <TafAppConfig
        v-else-if="step === 3 && form.framework === 'taf'"
        :key="`${framework}:${language}`"
        ref="tafRef"
        :editor-lang="editorLang"
        :value="tafValue"
      />
      <Result
        v-else-if="step === 4"
        :app-type="createdType"
        :data="result"
        @recreate="recreate"
      />
    </template>
    <template #footer>
      <div
        v-if="runtime && form && step !== 4"
        class="flex justify-center gap-[8px] bg-[#fff] py-[10px] shadow-[0_-3px_4px_0_#0000000a]"
      >
        <Button
          class="min-w-[88px]"
          :disabled="submitting"
          @click="previous"
          >{{ $t('上一步') }}</Button
        >
        <Button
          v-if="step === 2 && hasFrameworkFile"
          class="min-w-[88px]"
          theme="primary"
          @click="next"
          >{{ $t('下一步') }}</Button
        >
        <Button
          v-else
          class="min-w-[88px]"
          :loading="submitting"
          theme="primary"
          @click="createApp"
          >{{ $t('创建') }}</Button
        >
        <Button
          class="min-w-[88px]"
          :disabled="submitting"
          @click="cancel"
          >{{ $t('取消') }}</Button
        >
      </div>
    </template>
  </ScrollLayout>
</template>
<script setup lang="ts">
  import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
  import type { Component } from 'vue';

  import { Alert, Button, Loading } from 'bkui-vue';
  import { useRoute, useRouter } from 'vue-router';
  import { AppService } from '~/api/modules/v1/app';
  import GoLogo from '~/assets/lang-icons/go.svg';
  import NodejsLogo from '~/assets/lang-icons/nodejs.svg';
  import PythonLogo from '~/assets/lang-icons/python.svg';
  import { type IAppType, APP_TYPES } from '~/composables/app-type';
  import { getFrameworkFileFormat, isRuntimeModSupported } from '~/composables/use-app-runtime';
  import { useFocusOnErrorField } from '~/composables/use-focus-on-error-field';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useAppDetail } from '~/stores/app-detail';

  import Result, { type IResultData } from '../../result.vue';
  import TafAppConfig from '../taf/app-config.vue';
  import TrpcAppConfig from '../trpc/app-config.vue';
  import trpcTemplate from '../trpc/trpc-go-template.yaml?raw';
  import ParamConfig from './param-config.vue';
  import {
    type BKMSAppForm,
    buildBKMSAppPayload,
    createBKMSAppForm,
    findRuntime,
    useRuntimeDefinitions,
  } from './use-create-app';

  const props = defineProps({ step: { type: Number, default: 2 } });
  const emit = defineEmits(['update-step', 'update-text', 'next', 'pre']);
  const route = useRoute();
  const router = useRouter();
  const appDetailStore = useAppDetail();
  const { focusOnErrorField } = useFocusOnErrorField();
  const { definitions, loading, error, load } = useRuntimeDefinitions();
  const form = ref<BKMSAppForm | null>(null);
  const { confirmBox, forceCleanDirtyTag } = useLeaveConfirm(computed(() => form.value || {}));
  const paramRef = ref<InstanceType<typeof ParamConfig> | null>(null);
  const trpcRef = ref<InstanceType<typeof TrpcAppConfig> | null>(null);
  const tafRef = ref<InstanceType<typeof TafAppConfig> | null>(null);
  const submitting = ref(false);
  const createdType = ref<IAppType>(APP_TYPES.BKMS_APP);
  const result = ref<IResultData>({ status: 'CREATING', name: '', msg: '' });
  const workspaceID = computed(() => String(route.params.space || ''));
  const framework = computed(() => (typeof route.query.framework === 'string' ? route.query.framework : ''));
  const language = computed(() => (typeof route.query.language === 'string' ? route.query.language : ''));
  // 离开页面后，迟到的校验和创建响应不再推进步骤或写入全局应用上下文。
  let disposed = false;
  onBeforeUnmount(() => {
    disposed = true;
  });
  const runtime = computed(() => findRuntime(definitions.value, framework.value, language.value));
  const hasFrameworkFile = computed(() => isRuntimeModSupported(runtime.value, 'frameworkFile'));
  const editorLang = computed(() =>
    getFrameworkFileFormat(runtime.value) === 'taf' ? 'xml' : getFrameworkFileFormat(runtime.value),
  );
  const selectedLabel = computed(() => {
    const name =
      ({ go: 'Go', cpp: 'C++', python: 'Python', nodejs: 'Node.js' } as Record<string, string>)[language.value] ||
      language.value;
    return framework.value === 'blank' ? name : `${framework.value === 'trpc' ? 'tRPC' : 'TAF'} · ${name}`;
  });
  const languageLogos: Record<string, Component> = { go: GoLogo, python: PythonLogo, nodejs: NodejsLogo };
  const selectedLogo = computed(() => (framework.value === 'blank' ? languageLogos[language.value] : undefined));
  const descriptions: Record<string, string> = {
    trpc: '适用于使用 tRPC 协议的服务',
    taf: '采用 TAF 框架实现的服务端应用',
    go: '高并发 Web / 微服务，如 Gin、Echo',
    python: '数据服务、脚本任务与 AI 应用常见选择',
    nodejs: '实时接口、前端聚合层与前后端一体服务',
  };
  const selectedDescription = computed(
    () => descriptions[framework.value === 'blank' ? language.value : framework.value] || '',
  );
  // 适配已有 TAF 编辑器的数据结构，提交时仍转换为新创建接口的字段。
  const tafValue = computed(() => ({
    command: form.value?.appModelSpec.command || [],
    args: form.value?.appModelSpec.args || [],
    shCommand: form.value?.appModelSpec.shCommand || '',
    envVars: (form.value?.appModelSpec.envVars || []).map(item => ({
      key: item.key,
      value: item.value ?? '',
      description: item.description || '',
    })),
    tafSpec: {
      fileName: form.value?.frameworkConfigFile.name || '',
      filePath: form.value?.frameworkConfigFile.mountDir || '',
      fileContent: form.value?.frameworkConfigFile.content || '',
    },
  }));

  watch([definitions, framework, language], () => {
    if (!definitions.value) return;
    if (!runtime.value) {
      router.replace({ name: 'createApplication' });
      return;
    }
    if (form.value?.framework === framework.value && form.value.language === language.value) return;
    form.value = createBKMSAppForm(workspaceID.value, framework.value, language.value);
    if (framework.value === 'trpc') form.value.frameworkConfigFile.content = trpcTemplate;
    emit('update-text', { type: framework.value === 'blank' ? 'blank' : 'bkmsApp' });
    emit('update-step', 2);
    forceCleanDirtyTag();
  });

  async function cancel() {
    if (await confirmBox()) router.push({ name: 'app' });
  }
  async function changeTemplate() {
    if (await confirmBox()) router.push({ name: 'createApplication' });
  }
  function collectAppConfig() {
    if (!form.value) return;
    if (trpcRef.value) {
      const data = trpcRef.value.getValue();
      form.value.appModelSpec = {
        command: data.command,
        args: data.args,
        shCommand: data.shCommand,
        envVars: data.env,
      };
      form.value.frameworkConfigFile.content = data.content;
    } else if (tafRef.value) {
      const data = tafRef.value.getValue();
      form.value.appModelSpec = {
        command: data.command,
        args: data.args,
        shCommand: data.shCommand || '',
        envVars: data.envVars.map(item => {
          const original = form.value?.appModelSpec.envVars?.find(variable => variable.key === item.key);
          return {
            key: item.key,
            value: item.value ?? '',
            description: original?.description ?? item.description ?? '',
            isSensitive: original?.isSensitive ?? false,
          };
        }),
      };
      form.value.frameworkConfigFile = {
        name: data.tafSpec.fileName,
        mountDir: data.tafSpec.filePath,
        content: data.tafSpec.fileContent || '',
      };
    }
  }
  async function collectParameters() {
    if (!paramRef.value) return false;
    const valid = await paramRef.value.validate().catch(() => false);
    if (disposed || !paramRef.value) return false;
    if (!valid) {
      focusOnErrorField();
      return false;
    }
    form.value = paramRef.value.getValue();
    return true;
  }
  async function createApp() {
    if (submitting.value || !form.value || !runtime.value) return;
    // 校验阶段即加锁，避免异步校验期间重复点击发出多次创建请求
    submitting.value = true;
    try {
      if (!(await validateCurrentStep()) || !form.value || !runtime.value) return;
      await submit(buildBKMSAppPayload(form.value, runtime.value));
    } finally {
      submitting.value = false;
    }
  }
  async function next() {
    if (await collectParameters()) emit('update-step', 3);
  }
  async function previous() {
    if (props.step === 3) {
      collectAppConfig();
      emit('update-step', 2);
    } else {
      await changeTemplate();
    }
  }
  function recreate() {
    if (result.value.status === 'SUCCESS') {
      form.value = createBKMSAppForm(workspaceID.value, framework.value, language.value);
      if (framework.value === 'trpc') form.value.frameworkConfigFile.content = trpcTemplate;
      forceCleanDirtyTag();
    }
    emit('update-step', 2);
  }
  async function submit(params: ReturnType<typeof buildBKMSAppPayload>) {
    result.value = { status: 'CREATING', name: params.name, msg: '' };
    emit('update-step', 4);
    try {
      const app = await AppService.createBKMSApp(params, { interceptorErr: false });
      if (disposed) return;
      appDetailStore.reset();
      appDetailStore.updateAppID(app.id || params.id);
      appDetailStore.updateAppName(app.name || params.name);
      createdType.value = (app.type || APP_TYPES.BKMS_APP) as IAppType;
      appDetailStore.updateAppType(createdType.value);
      result.value = { status: 'SUCCESS', name: app.name || params.name, msg: '' };
      forceCleanDirtyTag();
    } catch (error) {
      if (disposed) return;
      const failure = error as { error?: { message?: string }; message?: string; msg?: string };
      result.value = {
        status: 'FAILED',
        name: params.name,
        msg: failure.msg || failure.message || failure.error?.message || '',
      };
    }
  }
  async function validateCurrentStep() {
    if (props.step === 2) return collectParameters();
    const appConfig = form.value?.framework === 'trpc' ? trpcRef.value : tafRef.value;
    const valid = await appConfig?.validate().catch(() => false);
    if (disposed || !valid) return false;
    collectAppConfig();
    return true;
  }
  onMounted(() => {
    emit('update-step', 2);
    load();
  });
</script>

<style scoped>
  .template-summary-icon {
    @apply inline-flex size-[40px] shrink-0 items-center justify-center overflow-hidden leading-none;
  }
</style>
