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
  <div class="flex justify-center max-w-[1400px] w-full">
    <Form
      ref="formRef"
      class="w-full p-[20px]"
      :label-width="135"
      :model="formData"
      :rules="rules"
    >
      <ToggleCard :name="$t('基础信息')">
        <div class="px-[245px]">
          <Form.FormItem
            class="row-start-2"
            :label="$t('应用名称')"
            property="name"
            required
          >
            <Input
              v-model.trim="formData.name"
              :placeholder="$t('请输入 1-20 个字符的小写字母、数字、中划线，以小写字母开头，提交后不可修改')"
            />
          </Form.FormItem>
          <Form.FormItem
            :label="$t('应用 ID')"
            property="id"
            required
          >
            <Input
              v-model="formData.id"
              disabled
              :placeholder="$t('自动生成')"
            />
          </Form.FormItem>
        </div>
      </ToggleCard>
      <!-- 构建配置 -->
      <ToggleCard
        class="mt-[16px]"
        :name="$t('构建配置')"
      >
        <div class="px-[245px]">
          <Form.FormItem
            :label="$t('来源')"
            required
          >
            <Button.ButtonGroup class="flex items-center">
              <Button
                class="flex-1"
                :selected="builderType === 'codeRepository'"
                @click="handleChangeBuilderType('codeRepository')"
                >{{ $t('代码仓库') }}</Button
              >
              <Button
                class="flex-1"
                :selected="builderType === 'pipeline'"
                @click="handleChangeBuilderType('pipeline')"
                >{{ $t('流水线') }}</Button
              >
            </Button.ButtonGroup>
          </Form.FormItem>
          <template v-if="builderType === 'codeRepository'">
            <Form.FormItem
              :label="$t('代码库')"
              property="buildConfig.repoBuildConfig.repoURL"
              required
            >
              <GitSelector
                v-model="formData.buildConfig.repoBuildConfig.repoURL"
                :workspace="formData.workspaceID"
                @change="handleProjectChange"
              />
            </Form.FormItem>
            <Form.FormItem
              class="flex-none"
              :label="$t('默认分支')"
              property="buildConfig.repoBuildConfig.defaultBranch"
              required
            >
              <Input v-model.trim="formData.buildConfig.repoBuildConfig.defaultBranch" />
            </Form.FormItem>
            <Form.FormItem
              :label="$t('构建目录')"
              property="buildConfig.repoBuildConfig.sourceDir"
            >
              <Input
                v-model.trim="formData.buildConfig.repoBuildConfig.sourceDir"
                clearable
                :placeholder="$t('请输入应用所在子目录，不填则默认为根目录')"
              />
            </Form.FormItem>
            <Form.FormItem
              :label="$t('Dockerfile 路径')"
              property="buildConfig.repoBuildConfig.dockerfile"
            >
              <Input
                v-model.trim="formData.buildConfig.repoBuildConfig.dockerfile"
                :placeholder="$t('相对于构建目录的路径，若留空，默认为构建目录下名为 “Dockerfile” 的文件')"
              />
            </Form.FormItem>
            <!-- 构建参数 -->
            <Form.FormItem
              :label="$t('构建参数')"
              property="buildConfig.repoBuildConfig.dockerBuildArgs"
            >
              <KeyValue
                :model-value="dockerBuildArgs"
                @update:model-value="handleDockerBuildArgsChange"
              />
            </Form.FormItem>
          </template>
          <div v-else-if="builderType === 'pipeline'">
            <Form.FormItem
              :label="$t('流水线')"
              property="buildConfig.pipelineBuildConfig.pipelineID"
              required
            >
              <PipelineSelector
                v-model="formData.buildConfig.pipelineBuildConfig.pipelineID"
                :workspace="formData.workspaceID"
              />
              <div class="text-[12px] text-[#979ba5]">
                {{ $t('需要保证流水线会将构建的镜像推送到当前空间的镜像仓库下') }}
                <Button
                  text
                  theme="primary"
                  @click="handleViewGuide"
                >
                  {{ $t('查看操作指引') }}
                  <Share class="ml-[6px]" />
                </Button>
              </div>
            </Form.FormItem>

            <PipelineConfig
              ref="pipelineConfigRef"
              :pipeline-id="formData.buildConfig.pipelineBuildConfig.pipelineID"
              @save="handleSavePipelineParams"
              @validate="val => (isPipelineEnabled = val)"
            />
          </div>
        </div>
      </ToggleCard>
    </Form>
  </div>
</template>
<script lang="ts" setup>
  import type { PropType } from 'vue';
  import { computed, onBeforeMount, ref, watch } from 'vue';

  import { Button, Form, Input } from 'bkui-vue';
  import { cloneDeep } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import { BKMS_REGEX, DOC_LINKS } from '~/common/const';

  import PipelineConfig from '../components/pipeline-config.vue';

  import type { CreateAppRequest } from '~/@types/v1/app';
  import type { GetAppIDAutoSuffixOutput } from '~/@types/v1/app';
  import type {
    AppModelSpecInput,
    PipelineBuildConfigInput,
    RepoBuildConfigInput,
    TafSpecInput,
    VariableInput,
  } from '~/@types/v1/app';
  import type { BkCIOAuthGitProjectOutput } from '~/@types/v1/bkintegrations-bkci';

  // taf 表单场景：appModelSpec、tafSpec、repoBuildConfig、pipelineBuildConfig 始终存在
  // dockerBuildArgs 在表单场景下始终为 {} 初始化，不会是 undefined
  export type TafCreateAppRequest = Omit<CreateAppRequest, 'appModelSpec' | 'buildConfig'> & {
    appModelSpec: Omit<AppModelSpecInput, 'args' | 'command' | 'envVars' | 'tafSpec'> & {
      args: string[];
      command: string[];
      envVars: VariableInput[];
      tafSpec: TafSpecInput;
    };
    buildConfig: Omit<CreateAppRequest['buildConfig'], 'pipelineBuildConfig' | 'repoBuildConfig'> & {
      pipelineBuildConfig: PipelineBuildConfigInput;
      repoBuildConfig: Omit<RepoBuildConfigInput, 'dockerBuildArgs'> & {
        dockerBuildArgs: Record<string, string>;
      };
    };
  };

  const props = defineProps({
    form: {
      type: Object as PropType<TafCreateAppRequest>,
      default: () => ({}),
    },
  });

  const { t } = useI18n();

  const formData = ref<TafCreateAppRequest>(props.form);
  const formRef = ref<InstanceType<typeof Form> | null>(null);
  async function validate() {
    try {
      // 校验基本表单
      const formValid = await formRef.value?.validate();
      // 流水线类型，还需要校验 PipelineConfig 组件
      if (builderType.value === 'pipeline' && pipelineConfigRef.value) {
        const pipelineValid = await pipelineConfigRef.value.validate();
        const configValid = await pipelineConfigRef.value.configValidate();
        return formValid && pipelineValid && configValid.valid;
      }
      return formValid;
    } catch {
      return false;
    }
  }
  const rules = ref({
    name: [
      {
        required: true,
        message: t('必填项'),
        trigger: 'blur',
      },
      {
        message: t('请输入 1-20 个字符的小写字母、数字、中划线，以小写字母开头'),
        trigger: 'blur',
        validator: () => BKMS_REGEX.appNameRegex.test(formData.value.name),
      },
    ],
    'builder.pipeline': [
      {
        message: t('无该流水线信息'),
        validator: () => !!isPipelineEnabled.value,
      },
    ],
  });

  function getValue() {
    const data = cloneDeep(formData.value);
    return data;
  }

  // 构建配置
  const builderType = computed<'codeRepository' | 'pipeline'>(
    () => formData.value.buildConfig.sourceType as 'codeRepository' | 'pipeline',
  );
  function handleChangeBuilderType(type: typeof builderType.value) {
    formData.value.buildConfig.sourceType = type;
  }

  // 设置别名
  function handleProjectChange(data: BkCIOAuthGitProjectOutput) {
    formData.value.buildConfig.repoBuildConfig.repoAlias = data?.alias ?? '';
  }

  /** input失焦时赋值，配合 PipelineParams 组件内watch逻辑，触发params查询 */
  const isPipelineEnabled = ref<boolean>(false); // 流水线是否可用
  const setSearchPipelineId = () => {
    if (formData.value.buildConfig.pipelineBuildConfig) {
      formData.value.buildConfig.pipelineBuildConfig.params = {}; // 重置流水线配置
    }
  };

  const pipelineConfigRef = ref<InstanceType<typeof PipelineConfig> | null>(null);
  const handleSavePipelineParams = (params: Record<string, string>) => {
    formData.value.buildConfig.pipelineBuildConfig.params = params;
    pipelineConfigRef.value?.validate();
  };

  const appIdSuffix = ref('');
  // 获取应用 ID 后缀
  async function getAppIDAutoSuffix() {
    const ret = (await ApiServerService.GetAppIDAutoSuffix({}, { needRes: true }).catch(() => ({
      suffix: '',
    }))) as GetAppIDAutoSuffixOutput;
    appIdSuffix.value = ret.suffix || '';
  }

  const dockerBuildArgs = computed(() =>
    Object.entries(formData.value.buildConfig.repoBuildConfig.dockerBuildArgs).map(([key, value]) => ({
      key,
      value,
    })),
  );
  // 构建参数变化
  function handleDockerBuildArgsChange(newArgs: Record<string, string> | Record<string, string>[]) {
    const argsObject: Record<string, string> = {};

    // 类型守卫：处理数组类型
    if (Array.isArray(newArgs)) {
      newArgs.forEach(arg => {
        if (arg.key && arg.value) {
          argsObject[arg.key] = arg.value;
        }
      });
    } else {
      // 处理对象类型（直接赋值）
      Object.assign(argsObject, newArgs);
    }

    // 目前GitSelector组件只支持TGit
    formData.value.buildConfig.repoBuildConfig.type = 'TGit';
    formData.value.buildConfig.repoBuildConfig.dockerBuildArgs = argsObject;
  }

  // 查看流水线构建操作指引
  function handleViewGuide() {
    const docUrl = `${import.meta.env.BK_DOC_URL}${DOC_LINKS.PIPELINE_BUILD_GUIDE}`;
    window.open(docUrl, '_blank');
  }

  watch(isPipelineEnabled, () => {
    // 校验流水线是否可用
    pipelineConfigRef.value?.validate();
  });

  watch(
    () => formData.value.name,
    newVal => {
      if (!newVal) {
        formData.value.id = '';
        return;
      }
      formData.value.id = `${newVal}${appIdSuffix.value}`;
    },
    { immediate: true },
  );

  onBeforeMount(() => {
    if (formData.value.buildConfig.pipelineBuildConfig.pipelineID) return;
    setSearchPipelineId();
    getAppIDAutoSuffix();
    formData.value.buildConfig.sourceType = 'codeRepository';
  });

  defineExpose({
    validate,
    getValue,
    getAppIDAutoSuffix,
  });
</script>
