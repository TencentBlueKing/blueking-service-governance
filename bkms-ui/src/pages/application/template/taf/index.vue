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
  <ScrollLayout content-class="w-full relative">
    <!-- 表单 -->
    <ParamConfig
      v-if="step === 2"
      ref="formRef"
      :form="appConfig"
    />
    <!-- 应用配置 -->
    <AppConfig
      v-else-if="step === 3"
      ref="appRef"
      :editor-title="appConfig.appModelSpec.tafSpec.fileName"
      :value="appConfig.appModelSpec"
    />
    <!-- 创建结果 -->
    <Result
      v-else-if="step === 4"
      :data="result"
      @recreate="handleRecreate"
    />
    <!-- 操作按钮栏 -->
    <template #footer="{ hasScroll }">
      <div
        v-if="step !== 4"
        :class="['flex justify-center py-[10px] transition w-full', hasScroll ? 'bg-[#fff]' : '']"
      >
        <div class="">
          <Button
            class="mr-[10px]"
            @click="preStep"
            >{{ $t('上一步') }}</Button
          >
          <Button
            v-if="step === 2"
            class="mr-[10px]"
            theme="primary"
            @click="nextStep"
          >
            {{ $t('下一步') }}
          </Button>
          <Button
            v-if="step === 3"
            class="mr-[10px]"
            theme="primary"
            @click="createApp"
          >
            {{ $t('创建') }}
          </Button>
          <Button
            class="mr-[10px]"
            @click="cancel"
            >{{ $t('取消') }}</Button
          >
        </div>
      </div>
    </template>
  </ScrollLayout>
</template>
<script lang="ts" setup>
  import { computed, onMounted, ref } from 'vue';

  import { Button } from 'bkui-vue';
  import { cloneDeep } from 'lodash-es';
  import { useRoute, useRouter } from 'vue-router';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import { useFocusOnErrorField } from '~/composables/use-focus-on-error-field';
  import useLeaveConfirm from '~/composables/use-leave-confirm';

  import Result from '../../result.vue';
  import AppConfig from './app-config.vue';
  import ParamConfig from './param-config.vue';

  import type { IResultData } from '../../result.vue';
  import type { TafCreateAppRequest } from './param-config.vue';
  import type { CreateAppRequest } from '~/@types/v1/app';

  const props = defineProps({
    step: { type: Number, default: 2 },
  });

  const emits = defineEmits(['next', 'pre', 'cancel', 'update-step']);

  const route = useRoute(); // 获取路由信息

  const { confirmBox, forceCleanDirtyTag } = useLeaveConfirm();
  const router = useRouter();

  const space = computed(() => (route.params?.space as string) || '');

  // 表单数据
  const appConfig = ref<TafCreateAppRequest>({
    workspaceID: space.value,
    type: 'taf',
    name: '',
    id: '',
    buildConfig: {
      sourceType: 'codeRepository',
      // 代码仓库配置
      repoBuildConfig: {
        // 代码库类型：TGit、GitHub
        type: 'TGit',
        defaultBranch: 'master',
        repoAlias: '',
        repoURL: '',
        sourceDir: '',
        dockerfile: './Dockerfile',
        dockerBuildArgs: {},
      },
      // 流水线配置
      pipelineBuildConfig: {
        pipelineID: '',
        params: {},
      },
    },
    appModelSpec: {
      command: [],
      args: [],
      envVars: [],
      tafSpec: {
        filePath: '/usr/local/taf/conf',
        fileName: 'config.conf',
        fileContent: '',
      },
    },
  });

  // 底部按钮操作
  // 下一步
  const { focusOnErrorField } = useFocusOnErrorField();
  const formRef = ref<InstanceType<typeof ParamConfig> | null>(null);
  const nextStep = async () => {
    const validate = await formRef.value?.validate().catch(() => false);
    if (!validate) {
      focusOnErrorField();
      return;
    }
    if (formRef.value) {
      appConfig.value = formRef.value?.getValue() as TafCreateAppRequest;
    }
    emits('next');
  };
  // 上一步
  const preStep = async () => {
    emits('pre');
    // 重置错误提示
    appRef.value?.resetStatus?.();

    if (props.step === 2) {
      router.back();
    }
  };
  // 取消
  const cancel = async () => {
    const enableLeave = await confirmBox();
    if (enableLeave) {
      emits('update-step', 1);
      router.back();
    }
  };

  // 创建应用
  const appRef = ref<InstanceType<typeof AppConfig> | null>(null);
  const result = ref<IResultData>({
    status: 'CREATING',
    name: '',
    msg: '',
  });

  async function createApp() {
    const valid = await appRef.value?.validate();
    if (!valid || !appRef.value) return;

    await prepareAppConfigData();

    result.value.name = appConfig.value.name;
    emits('next');
    const paramsData = cloneDeep(appConfig.value) as DeepPartial<CreateAppRequest>;
    if (paramsData?.buildConfig?.sourceType && paramsData?.buildConfig?.sourceType === 'codeRepository') {
      paramsData.buildConfig.pipelineBuildConfig = undefined;
    } else if (paramsData?.buildConfig?.sourceType && paramsData.buildConfig.sourceType === 'pipeline') {
      paramsData.buildConfig.repoBuildConfig = undefined;
    }
    // needRes 获取接口返回的 message
    await ApiServerService.CreateApp(
      {
        ...paramsData,
        workspaceID: space.value,
      } as CreateAppRequest,
      { needRes: true },
    )
      .then(() => {
        forceCleanDirtyTag(() => {
          result.value.status = 'SUCCESS';
        });
      })
      .catch(err => {
        forceCleanDirtyTag(() => {
          result.value.status = 'FAILED';
          result.value.msg = err.msg;
        });
      });
  }

  // 重新创建
  function handleRecreate() {
    emits('update-step', 2);
    result.value = {
      status: 'CREATING',
      name: '',
      msg: '',
    };
    // 重新获取应用 ID 后缀
    formRef.value?.getAppIDAutoSuffix?.();
  }
  // 应用配置数据
  async function prepareAppConfigData() {
    if (!appRef.value) return;

    const data = await appRef.value.getValue();
    appConfig.value.appModelSpec = data as TafCreateAppRequest['appModelSpec'];
  }

  onMounted(() => {
    if (props.step === 1) {
      emits('next');
    }
  });
</script>
