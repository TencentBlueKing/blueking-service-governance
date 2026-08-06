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
      <ToggleCard
        class="bg-[#fff] rounded-[2px]"
        :name="$t('基础信息')"
        type="normal"
      >
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
          <Form.FormItem
            :label="$t('语言')"
            property="appModelSpec.trpcSpec.language"
            required
          >
            <Select v-model="formData.appModelSpec.trpcSpec.language">
              <Select.Option
                v-for="item in languageOptions"
                :key="item"
                :value="item"
                >{{ item }}</Select.Option
              >
            </Select>
          </Form.FormItem>
        </div>
      </ToggleCard>
      <!-- 构建配置 -->
      <ToggleCard
        class="mt-[16px] bg-[#fff] rounded-[2px]"
        :name="$t('构建配置')"
        type="normal"
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
            <!-- 构建方式 -->
            <Form.FormItem
              :label="$t('构建方式')"
              property="buildConfig.repoBuildConfig.imageBuildMode"
              required
            >
              <CardRadio
                v-model="formData.buildConfig.repoBuildConfig.imageBuildMode"
                :options="buildTypeOptions"
              />
            </Form.FormItem>
            <Form.FormItem
              v-if="formData.buildConfig.repoBuildConfig.imageBuildMode === 'repositoryDockerfile'"
              :label="$t('Dockerfile 路径')"
              property="buildConfig.repoBuildConfig.dockerfile"
            >
              <Input
                v-model.trim="formData.buildConfig.repoBuildConfig.dockerfile"
                :placeholder="$t('相对于构建目录的路径，若留空，默认为构建目录下名为 “Dockerfile” 的文件')"
              />
            </Form.FormItem>
            <template v-else-if="formData.buildConfig.repoBuildConfig.imageBuildMode === 'platform'">
              <!-- 构建镜像 -->
              <Form.FormItem
                :label="$t('构建镜像')"
                property="buildConfig.repoBuildConfig.platformBuildConfig.builderImage"
                required
              >
                <div class="flex gap-[8px] items-start">
                  <Select
                    v-model="selectedBuilderImageId"
                    class="flex-grow-2"
                    display-key="name"
                    filterable
                    id-key="id"
                    :input-search="false"
                    :list="builderImages"
                    :placeholder="$t('请选择镜像')"
                    @change="handleBuilderImageChange"
                  />
                  <Select
                    v-model="selectedBuilderTag"
                    class="flex-grow-1"
                    :disabled="!selectedBuilderImageId"
                    display-key="tag"
                    filterable
                    id-key="tag"
                    :list="builderImageTags"
                    :loading="builderTagLoading"
                    :placeholder="$t('请选择标签')"
                    :remote-method="searchBuilderTag"
                  />
                </div>
                <div
                  v-if="staleBuilderWarning"
                  class="text-[12px] text-[#fe9c00] mt-[4px]"
                >
                  {{ $t('该镜像已不在当前平台镜像列表中，请重新选择') }}
                </div>
                <div class="text-[12px] text-[#979ba5]">
                  {{ $t('用于编译源码的镜像（含 Go 工具链），对应 builder 阶段基础镜像。') }}
                </div>
              </Form.FormItem>
              <!-- 运行镜像 -->
              <Form.FormItem
                :label="$t('运行镜像')"
                property="buildConfig.repoBuildConfig.platformBuildConfig.runnerImage"
                required
              >
                <div class="flex gap-[8px] items-start">
                  <Select
                    v-model="selectedRunnerImageId"
                    class="flex-grow-2"
                    display-key="name"
                    filterable
                    id-key="id"
                    :input-search="false"
                    :list="runnerImages"
                    :placeholder="$t('请选择镜像')"
                    @change="handleRunnerImageChange"
                  />
                  <Select
                    v-model="selectedRunnerTag"
                    class="flex-grow-1"
                    :disabled="!selectedRunnerImageId"
                    display-key="tag"
                    filterable
                    id-key="tag"
                    :list="runnerImageTags"
                    :loading="runnerTagLoading"
                    :placeholder="$t('请选择标签')"
                    :remote-method="searchRunnerTag"
                  />
                </div>
                <div
                  v-if="staleRunnerWarning"
                  class="text-[12px] text-[#fe9c00] mt-[4px]"
                >
                  {{ $t('该镜像已不在当前平台镜像列表中，请重新选择') }}
                </div>
                <div class="text-[12px] text-[#979ba5]">
                  {{ $t('运行服务的基础镜像（精简、无编译器），对应 runner 阶段基础镜像。') }}
                </div>
                <Alert
                  v-if="imageVersionMismatchWarning"
                  class="mt-[4px]"
                  theme="warning"
                >
                  <template #title>
                    <div class="leading-[22px]">
                      {{
                        $t(
                          '构建镜像的版本（{0}）与运行镜像（{1}）不一致，可能导致 glibc/musl 等基础库差异，引发运行时兼容问题。建议保持两者一致，如已确认可忽略。',
                          [imageVersionMismatchWarning.builderVersion, imageVersionMismatchWarning.runnerVersion],
                        )
                      }}
                    </div>
                  </template>
                </Alert>
              </Form.FormItem>
              <!-- 高级命令 -->
              <Form.FormItem
                class="!block"
                :label="$t('高级命令')"
              >
                <ToggleCard
                  class="bg-[#fff] rounded-[2px] border-[1px] border-[#e5e5e5]"
                  content-class="p-[16px]"
                  :model-value="false"
                  type="normal"
                >
                  <template #title>
                    <span class="font-bold mx-[8px]">{{ $t('高级构建命令') }}</span>
                    <span>{{ $t('可选，补充或覆盖平台默认的构建步骤') }}</span>
                  </template>
                  <div class="mb-[24px] last:mb-0">
                    <div class="flex items-center gap-[8px] mb-[8px]">
                      <span class="text-[14px]">{{ $t('编译前置命令') }}</span>
                      <Tag theme="info">{{ $t('builder 阶段') }}</Tag>
                    </div>
                    <Input
                      v-model="preBuildText"
                      :placeholder="$t('在 go build 之前执行，如代码生成、安装工具等')"
                      :rows="4"
                      type="textarea"
                    />
                  </div>
                  <div class="mb-[24px] last:mb-0">
                    <div class="flex items-center gap-[8px] mb-[8px]">
                      <span class="text-[14px]">{{ $t('编译命令') }}</span>
                      <Tag theme="info">{{ $t('builder 阶段') }}</Tag>
                    </div>
                    <Input
                      v-model="buildText"
                      :placeholder="$t('留空则使用平台默认：{0}', ['CGO_ENABLED=0 go build -trimpath -o <app> .'])"
                      :rows="4"
                      type="textarea"
                    />
                  </div>
                  <div class="mb-[24px] last:mb-0">
                    <div class="flex items-center gap-[8px] mb-[8px]">
                      <span class="text-[14px]">{{ $t('运行环境命令') }}</span>
                      <Tag theme="success">{{ $t('runner 阶段') }}</Tag>
                    </div>
                    <Input
                      v-model="runtimeEnvText"
                      :placeholder="$t('配置运行镜像环境，如 apk add --no-cache ca-certificates')"
                      :rows="4"
                      type="textarea"
                    />
                  </div>
                  <div class="mb-[24px] last:mb-0">
                    <div class="flex items-center gap-[8px] mb-[8px]">
                      <span class="text-[14px]">{{ $t('镜像入口命令') }}</span>
                      <Tag theme="success">{{ $t('runner 阶段') }}</Tag>
                    </div>
                    <Input
                      v-model.trim="formData.buildConfig.repoBuildConfig.platformBuildConfig.commands.start"
                      :placeholder="$t('留空则使用平台默认：运行编译产物二进制')"
                      :rows="4"
                      type="textarea"
                    />
                    <span class="text-[#c4c6cc]">{{
                      $t('写入镜像的默认启动方式（ENTRYPOINT），构建时固化到镜像中。部署时可被「运行命令」覆盖')
                    }}</span>
                  </div>
                </ToggleCard>
              </Form.FormItem>
            </template>
            <!-- 构建参数 -->
            <Form.FormItem
              :label="$t('构建参数')"
              property="buildConfig.repoBuildConfig.dockerBuildArgs"
            >
              <KeyValue
                :model-value="dockerBuildArgs"
                @update:model-value="handleDockerBuildArgsChange"
              />
              <!-- 等价构建命令（示意） -->
              <div class="mb-[24px]">
                <span class="inline-block mb-[2px] mt-[10px] font-bold">{{ $t('等价构建命令（示意）') }}</span>
                <div
                  class="bg-[#f5f7fa] rounded-[4px] p-[12px] font-mono leading-[22px] text-[#63656e] overflow-x-auto"
                >
                  <pre
                    v-bk-xss-html="highlightedBuildCommand"
                    class="m-0 whitespace-pre-wrap break-all bg-transparent!"
                  ></pre>
                </div>
              </div>
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
  import { computed, onBeforeMount, onBeforeUnmount, ref, watch } from 'vue';

  import { Alert, Button, Form, Input, Select, Tag } from 'bkui-vue';
  import { Share } from 'bkui-vue/lib/icon';
  import hljs from 'highlight.js/lib/core';
  import dockerfile from 'highlight.js/lib/languages/dockerfile';
  import { cloneDeep, debounce } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import { DOC_LINKS } from '~/common/const';
  import { BKMS_REGEX } from '~/common/const';
  import CardRadio, { type CardRadioOption } from '~/components/card-radio.vue';

  import PipelineConfig from '../components/pipeline-config.vue';
  import { usePlatformBuildImages } from './use-platform-build-images';

  import type { CreateAppRequest } from '~/@types/app';
  import type { GetAppIDAutoSuffixResponse } from '~/@types/app';
  import type {
    AppModelSpecInput,
    BuildConfigInput,
    PipelineBuildConfigInput,
    RepoBuildConfigInput,
    TrpcSpecInput,
  } from '~/@types/v1/app';
  import type { BkCIOAuthGitProjectOutput } from '~/@types/v1/bkintegrations-bkci';

  import 'highlight.js/styles/github.css'; // 代码块高亮样式

  // 注册 dockerfile 用于等价构建命令的语法高亮（docker build 命令的 --build-arg、镜像名等元素能正确着色）
  hljs.registerLanguage('dockerfile', dockerfile);

  // 表单场景：repoBuildConfig 和 pipelineBuildConfig 在初始化时即赋值，始终存在
  // dockerBuildArgs 在表单场景下始终为 {} 初始化，不会是 undefined
  type TrpcFormRequest = Omit<CreateAppRequest, 'appModelSpec' | 'buildConfig'> & {
    appModelSpec: Omit<AppModelSpecInput, 'trpcSpec'> & {
      trpcSpec: TrpcSpecInput;
    };
    buildConfig: Omit<BuildConfigInput, 'pipelineBuildConfig' | 'repoBuildConfig'> & {
      pipelineBuildConfig: PipelineBuildConfigInput;
      repoBuildConfig: Omit<RepoBuildConfigInput, 'dockerBuildArgs'> & {
        dockerBuildArgs: Record<string, string>;
        imageBuildMode: 'platform' | 'repositoryDockerfile';
        platformBuildConfig: {
          builderImage: string;
          commands: {
            build: string[];
            preBuild: string[];
            runtimeEnv: string[];
            start: string;
          };
          runnerImage: string;
        };
      };
    };
  };

  const props = defineProps({
    form: {
      type: Object as PropType<CreateAppRequest>,
      default: () => ({}),
    },
  });

  const { t } = useI18n();

  const formData = ref<TrpcFormRequest>(props.form as TrpcFormRequest);
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
    'buildConfig.repoBuildConfig.platformBuildConfig.builderImage': [
      {
        required: true,
        message: t('构建镜像不能为空'),
        validator: () => !!selectedBuilderImageId.value,
        trigger: 'custom',
      },
    ],
    'buildConfig.repoBuildConfig.platformBuildConfig.runnerImage': [
      {
        required: true,
        message: t('运行镜像不能为空'),
        validator: () => !!selectedRunnerImageId.value,
        trigger: 'custom',
      },
    ],
    'buildConfig.repoBuildConfig.sourceDir': [
      {
        message: t('构建目录不能以 / 开头'),
        trigger: 'blur',
        validator: (value: string) => !value || !value.startsWith('/'),
      },
    ],
  });

  const languageOptions = ref(['go', 'cpp']);

  // cpp 语言暂不支持平台通用构建，options 中禁用 platform 项并 hover 提示「暂不支持」
  const buildTypeOptions = computed<CardRadioOption[]>(() => {
    const isCpp = formData.value.appModelSpec?.trpcSpec?.language === 'cpp';
    return [
      {
        value: 'platform',
        label: t('平台通用构建'),
        description: t('无需编写和维护 Dockerfile，平台按标准框架自动完成构建。可按需在「高级命令」补充构建步骤。'),
        disabled: isCpp,
        disabledTip: t('暂不支持'),
      },
      {
        value: 'repositoryDockerfile',
        label: t('使用仓库 Dockerfile'),
        description: t(
          '完全使用代码仓库中你自己维护的 Dockerfile 构建，平台不介入生成，构建过程以仓库内的 Dockerfile 为准。',
        ),
      },
    ];
  });

  // 切换为 cpp 时，若当前为平台通用构建则自动切回「使用仓库 Dockerfile」
  watch(
    () => formData.value.appModelSpec?.trpcSpec?.language,
    newLang => {
      if (newLang === 'cpp' && formData.value.buildConfig.repoBuildConfig.imageBuildMode === 'platform') {
        formData.value.buildConfig.repoBuildConfig.imageBuildMode = 'repositoryDockerfile';
      }
    },
  );

  // ========== 镜像选择 ==========
  const {
    builderImages,
    runnerImages,
    selectedBuilderImageId,
    selectedRunnerImageId,
    builderImageTags,
    runnerImageTags,
    selectedBuilderTag,
    selectedRunnerTag,
    builderTagLoading,
    runnerTagLoading,
    staleBuilderWarning,
    staleRunnerWarning,
    imageVersionMismatchWarning,
    fetchImageTags,
    fetchPlatformBuildImages,
    handleBuilderImageChange,
    handleRunnerImageChange,
    resetState,
  } = usePlatformBuildImages({
    getStoredImage: type =>
      formData.value.buildConfig.repoBuildConfig.platformBuildConfig?.[
        type === 'builder' ? 'builderImage' : 'runnerImage'
      ],
    setImageValue: (type, value) => {
      formData.value.buildConfig.repoBuildConfig.platformBuildConfig[
        type === 'builder' ? 'builderImage' : 'runnerImage'
      ] = value;
    },
    getLanguage: () => formData.value.appModelSpec?.trpcSpec?.language,
  });

  // ========== 镜像 tag 远程搜索 ==========
  const searchBuilderTag = debounce((keyword: string) => {
    if (selectedBuilderImageId.value) {
      fetchImageTags('builder', selectedBuilderImageId.value, keyword);
    }
  }, 300);

  const searchRunnerTag = debounce((keyword: string) => {
    if (selectedRunnerImageId.value) {
      fetchImageTags('runner', selectedRunnerImageId.value, keyword);
    }
  }, 300);

  onBeforeUnmount(() => {
    searchBuilderTag.cancel();
    searchRunnerTag.cancel();
  });

  // ========== 构建方式互斥 ==========
  // 切到 platform → 清空 dockerfile；切到 repositoryDockerfile → 重置 platformBuildConfig
  watch(
    () => formData.value.buildConfig.repoBuildConfig.imageBuildMode,
    newVal => {
      if (newVal === 'platform') {
        formData.value.buildConfig.repoBuildConfig.dockerfile = '';
      } else if (newVal === 'repositoryDockerfile') {
        formData.value.buildConfig.repoBuildConfig.platformBuildConfig = {
          builderImage: '',
          runnerImage: '',
          commands: {
            build: [],
            preBuild: [],
            runtimeEnv: [],
            start: '',
          },
        };
        resetState();
      }
    },
  );

  // ========== 数组命令 ↔ textarea 文本转换（写入时每行自动去除首尾空格并过滤空行）==========
  const preBuildText = computed({
    get: () => formData.value.buildConfig.repoBuildConfig.platformBuildConfig?.commands?.preBuild?.join('\n') ?? '',
    set: val => {
      formData.value.buildConfig.repoBuildConfig.platformBuildConfig.commands.preBuild = val
        ? val
            .split('\n')
            .map(s => s.trim())
            .filter(Boolean)
        : [];
    },
  });

  const buildText = computed({
    get: () => formData.value.buildConfig.repoBuildConfig.platformBuildConfig?.commands?.build?.join('\n') ?? '',
    set: val => {
      formData.value.buildConfig.repoBuildConfig.platformBuildConfig.commands.build = val
        ? val
            .split('\n')
            .map(s => s.trim())
            .filter(Boolean)
        : [];
    },
  });

  const runtimeEnvText = computed({
    get: () => formData.value.buildConfig.repoBuildConfig.platformBuildConfig?.commands?.runtimeEnv?.join('\n') ?? '',
    set: val => {
      formData.value.buildConfig.repoBuildConfig.platformBuildConfig.commands.runtimeEnv = val
        ? val
            .split('\n')
            .map(s => s.trim())
            .filter(Boolean)
        : [];
    },
  });

  function getValue() {
    sanitizeCommands();
    const data = cloneDeep(formData.value);
    // platform 模式下强制清空 dockerfile
    if (data.buildConfig.repoBuildConfig.imageBuildMode === 'platform') {
      data.buildConfig.repoBuildConfig.dockerfile = '';
    }
    // repositoryDockerfile 模式下移除 platformBuildConfig
    if (data.buildConfig.repoBuildConfig.imageBuildMode === 'repositoryDockerfile') {
      (data.buildConfig.repoBuildConfig as Record<string, unknown>).platformBuildConfig = undefined;
    }
    return data;
  }

  function sanitizeCommands() {
    const commands = formData.value.buildConfig.repoBuildConfig.platformBuildConfig?.commands;
    if (!commands) return;
    // 过滤空行、trim、限制每行长度不超过 4096
    const sanitizeArray = (arr: string[]): string[] => {
      return arr
        .map(s => s.trim())
        .filter(s => s.length > 0 && !s.includes('\n') && s.length <= 4096)
        .slice(0, 32);
    };
    commands.preBuild = sanitizeArray(commands.preBuild);
    commands.build = sanitizeArray(commands.build);
    commands.runtimeEnv = sanitizeArray(commands.runtimeEnv);
    // start 校验：非空且不含换行符，长度不超过 4096
    if (commands.start) {
      commands.start = commands.start.trim().replace(/\n/g, '').slice(0, 4096);
    }
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
    }))) as GetAppIDAutoSuffixResponse;
    appIdSuffix.value = ret.suffix ?? '';
  }

  const dockerBuildArgs = computed(() =>
    Object.entries(formData.value.buildConfig.repoBuildConfig.dockerBuildArgs).map(([key, value]) => ({
      key,
      value,
    })),
  );

  // 等价构建命令（示意）
  const equivalentBuildCommand = computed(() => {
    const sourceDir = formData.value.buildConfig.repoBuildConfig.sourceDir?.trim();
    const buildArgs = formData.value.buildConfig.repoBuildConfig.dockerBuildArgs;
    const appName = formData.value.id || formData.value.name || '<应用名>';
    const lines: string[] = [];
    if (sourceDir) {
      lines.push(`# 进入 ${sourceDir} 目录执行构建，仅该目录下的文件参与镜像构建`);
    } else {
      lines.push('# 进入仓库根目录执行构建，仅该目录下的文件参与镜像构建');
    }
    lines.push('# Dockerfile 由平台根据标准框架自动生成');
    lines.push('docker build \\');
    const argEntries = Object.entries(buildArgs || {});
    argEntries.forEach(([key, value]) => {
      lines.push(`  --build-arg ${key}=${value} \\`);
    });
    lines.push('  -f Dockerfile \\');
    lines.push(`  -t ${appName}:<tag> .`);
    return lines.join('\n');
  });

  // 对等价构建命令做语法高亮（bash 语法）
  const highlightedBuildCommand = computed(
    () => hljs.highlight(equivalentBuildCommand.value, { language: 'dockerfile' }).value,
  );

  // 构建参数变化
  function handleDockerBuildArgsChange(newArgs: Record<string, string> | Record<string, string>[]) {
    const argsObject: Record<string, string> = {};
    if (Array.isArray(newArgs)) {
      newArgs.forEach(arg => {
        if (arg.key && arg.value) {
          argsObject[arg.key] = arg.value;
        }
      });
    } else {
      Object.assign(argsObject, newArgs);
    }
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
    setSearchPipelineId();
    getAppIDAutoSuffix();
    formData.value.buildConfig.sourceType = 'codeRepository';
    // 初始化构建方式默认值：空值按 repositoryDockerfile 兼容
    if (!formData.value.buildConfig.repoBuildConfig.imageBuildMode) {
      formData.value.buildConfig.repoBuildConfig.imageBuildMode = 'repositoryDockerfile';
    }
    // 初始化 platformBuildConfig
    if (!formData.value.buildConfig.repoBuildConfig.platformBuildConfig) {
      formData.value.buildConfig.repoBuildConfig.platformBuildConfig = {
        builderImage: '',
        runnerImage: '',
        commands: {
          build: [],
          preBuild: [],
          runtimeEnv: [],
          start: '',
        },
      };
    }
    // 获取构建镜像和运行镜像列表
    fetchPlatformBuildImages('builder');
    fetchPlatformBuildImages('runner');
  });

  // language 变化时重新拉取构建镜像列表
  watch(
    () => formData.value.appModelSpec?.trpcSpec?.language,
    () => {
      fetchPlatformBuildImages('builder');
      fetchPlatformBuildImages('runner');
    },
  );

  defineExpose({
    validate,
    getValue,
    getAppIDAutoSuffix,
  });
</script>
