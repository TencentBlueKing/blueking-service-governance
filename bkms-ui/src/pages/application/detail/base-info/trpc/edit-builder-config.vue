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
    v-model:is-show="isShow"
    :before-close="handleBeforeClose"
    :title="$t('编辑构建配置')"
    :width="960"
    @closed="handleClose"
  >
    <div class="py-[16px] px-[24px]">
      <Form
        ref="formRef"
        form-type="vertical"
        :model="builderData"
        :rules="rules"
      >
        <Form.FormItem
          class="inline-block"
          :label="$t('来源')"
          required
        >
          <Button.ButtonGroup class="flex items-center w-[420px]">
            <Button
              class="flex-1"
              :selected="currentBuilderType === 'codeRepository'"
              @click="toggleBuilderType('codeRepository')"
              >{{ $t('源码仓库') }}</Button
            >
            <Button
              class="flex-1"
              :selected="currentBuilderType === 'pipeline'"
              @click="toggleBuilderType('pipeline')"
              >{{ $t('流水线') }}</Button
            >
          </Button.ButtonGroup>
        </Form.FormItem>
        <div v-if="currentBuilderType === 'codeRepository'">
          <Form.FormItem
            :label="$t('代码库')"
            property="repoBuildConfig.repoURL"
            required
          >
            <GitSelector
              v-model="builderData!.repoBuildConfig!.repoURL"
              :workspace="spaceStore.currentSpace"
              @change="handleProjectChange"
            />
          </Form.FormItem>
          <Form.FormItem
            class="flex-none"
            :label="$t('默认分支')"
            property="repoBuildConfig.defaultBranch"
            required
          >
            <Input v-model.trim="builderData!.repoBuildConfig!.defaultBranch" />
          </Form.FormItem>
          <Form.FormItem
            :label="$t('构建目录')"
            property="repoBuildConfig.sourceDir"
          >
            <Input v-model.trim="builderData!.repoBuildConfig!.sourceDir" />
          </Form.FormItem>
          <!-- 构建方式 -->
          <Form.FormItem
            :label="$t('构建方式')"
            property="repoBuildConfig.imageBuildMode"
            required
          >
            <CardRadio
              v-model="builderData!.repoBuildConfig!.imageBuildMode"
              :options="buildTypeOptions"
            />
          </Form.FormItem>
          <!-- 使用仓库 Dockerfile -->
          <template v-if="builderData!.repoBuildConfig!.imageBuildMode === 'repositoryDockerfile'">
            <Form.FormItem
              :label="$t('Dockerfile 路径')"
              property="repoBuildConfig.dockerfile"
            >
              <Input v-model.trim="builderData!.repoBuildConfig!.dockerfile" />
            </Form.FormItem>
          </template>
          <!-- 平台通用构建 -->
          <template v-else-if="builderData!.repoBuildConfig!.imageBuildMode === 'platform'">
            <Form.FormItem
              :label="$t('构建镜像')"
              property="repoBuildConfig.platformBuildConfig.builderImage"
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
            <Form.FormItem
              :label="$t('运行镜像')"
              property="repoBuildConfig.platformBuildConfig.runnerImage"
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
                    <span class="text-[14px] text-[#313238]">{{ $t('编译前置命令') }}</span>
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
                    <span class="text-[14px] text-[#313238]">{{ $t('编译命令') }}</span>
                    <Tag theme="info">{{ $t('builder 阶段') }}</Tag>
                  </div>
                  <Input
                    v-model="buildText"
                    :placeholder="buildPlaceholder"
                    :rows="4"
                    type="textarea"
                  />
                  <BuildOutputHint
                    :app-name="props.appName"
                    class="mt-[16px]"
                  />
                </div>
                <div class="mb-[24px] last:mb-0">
                  <div class="flex items-center gap-[8px] mb-[8px]">
                    <span class="text-[14px] text-[#313238]">{{ $t('运行环境命令') }}</span>
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
                    <span class="text-[14px] text-[#313238]">{{ $t('镜像入口命令') }}</span>
                    <Tag theme="success">{{ $t('runner 阶段') }}</Tag>
                  </div>
                  <Input
                    v-model.trim="builderData!.repoBuildConfig!.platformBuildConfig!.commands!.start"
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
          <Form.FormItem
            :label="$t('构建参数')"
            property="repoBuildConfig.dockerBuildArgs"
          >
            <GoBuildArgs
              v-model="builderData!.repoBuildConfig!.dockerBuildArgs"
              :language="props.language"
            />
          </Form.FormItem>
          <TagConfigForm
            v-model="tagConfig"
            :label="$t('推荐版本号')"
            property="repoBuildConfig.tagConfig"
          />
        </div>
        <template v-else-if="currentBuilderType === 'pipeline'">
          <Form.FormItem
            :label="$t('流水线')"
            property="pipelineBuildConfig.pipelineID"
            required
          >
            <!-- 流水线下拉组件 -->
            <PipelineSelector
              v-model="builderData.pipelineBuildConfig!.pipelineID"
              :workspace="spaceStore.currentSpace"
              @update:model-value="getParamsData"
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

          <Form.FormItem
            v-if="isPipelineIdValid && params.length"
            :label="$t('流水线参数配置')"
          >
            <div
              v-bkloading="{ loading: paramsLoading }"
              class="bg-[#F5F7FA] px-[24px] pt-[16px]"
            >
              <PipelineParamsForm
                ref="pipelineParamsRef"
                :params="validParams"
              />
            </div>
          </Form.FormItem>
          <Exception
            v-else
            :description="builderData.pipelineBuildConfig?.pipelineID ? $t('流水线无参数配置') : $t('请先选择流水线')"
            scene="part"
            type="empty"
          >
          </Exception>
          <!-- 流水线-推荐版本号 -->
          <TagConfigForm
            v-model="tagConfig"
            :label="$t('推荐版本号')"
            property="pipelineBuildConfig.tagConfig"
          />
        </template>
      </Form>
    </div>
    <template #footer>
      <div class="flex items-center">
        <Button
          class="min-w-[88px]"
          theme="primary"
          @click="handleSave"
        >
          {{ $t('保存') }}
        </Button>
        <Button
          class="min-w-[88px] ml-[8px]"
          @click="handleClose"
        >
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue';

  import { Alert, Button, Exception, Form, Input, Select, Sideslider, Tag } from 'bkui-vue';
  import { Share } from 'bkui-vue/lib/icon';
  import { cloneDeep, debounce } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { BuildConfigOutputObj, TagConfigOutputObj } from '~/@types/v1/app';
  import { BkCIOAuthGitProjectOutput, BkCIPipelineVariableOutput } from '~/@types/v1/bkintegrations-bkci';
  import { BkintegrationsBkciService } from '~/api/modules/v1';
  import { DOC_LINKS } from '~/common/const';
  import CardRadio, { type CardRadioOption } from '~/components/card-radio.vue';
  import TagConfigForm from '~/components/tag-config-form.vue';
  import ToggleCard from '~/components/toggle-card.vue';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import BuildOutputHint from '~/pages/application/components/build-output-hint.vue';
  import GoBuildArgs from '~/pages/application/components/go-build-args.vue';
  import PipelineParamsForm from '~/pages/application/components/pipeline-params-form.vue';
  import { usePlatformBuildImages } from '~/pages/application/template/trpc/use-platform-build-images';
  import { useSpaceStore } from '~/stores/space';

  // BuildConfigOutputObj 已包含 tagConfig 字段，使用 Partial 使其可选
  // 同时使 repoBuildConfig 和 pipelineBuildConfig 可以为 null
  type BuilderData = Omit<BuildConfigOutputObj, 'pipelineBuildConfig' | 'repoBuildConfig' | 'tagConfig'> & {
    pipelineBuildConfig?: BuildConfigOutputObj['pipelineBuildConfig'] | null;
    repoBuildConfig?: BuildConfigOutputObj['repoBuildConfig'] | null;
    tagConfig?: null | TagConfigOutputObj;
  };

  type BuilderType = 'codeRepository' | 'pipeline';
  interface IProps {
    appName?: string;
    language?: string;
    type: BuilderType;
  }

  const isShow = defineModel<boolean>('isShow');
  const builderData = defineModel<BuilderData>('builderData', { required: true });
  const props = defineProps<IProps>();
  const emit = defineEmits(['confirm']);
  const spaceStore = useSpaceStore();
  // 使用 useLeaveConfirm hook 管理表单变化检测
  const { confirmBox, forceCleanDirtyTag } = useLeaveConfirm(builderData);
  /** 当前选中的来源 */
  const currentBuilderType = ref();

  /** 缓存各来源的表单数据，切换时不丢失已填写内容 */
  const repoBuildConfigCache = ref<BuildConfigOutputObj['repoBuildConfig'] | null>(null);
  const pipelineBuildConfigCache = ref<BuildConfigOutputObj['pipelineBuildConfig'] | null>(null);
  const tagConfigCache = ref<Record<BuilderType, null | TagConfigOutputObj>>({
    codeRepository: null,
    pipeline: null,
  });
  /** 缓存流水线参数列表 */
  const pipelineParamsCache_list = ref<BkCIPipelineVariableOutput[]>([]);

  const DEFAULT_REPO_CONFIG: BuildConfigOutputObj['repoBuildConfig'] = {
    type: 'TGit',
    defaultBranch: '',
    repoAlias: '',
    repoURL: '',
    sourceDir: '',
    dockerfile: '',
    dockerBuildArgs: {},
    imageBuildMode: 'repositoryDockerfile',
    platformBuildConfig: {
      builderImage: '',
      runnerImage: '',
      commands: {
        build: [],
        preBuild: [],
        runtimeEnv: [],
        start: '',
      },
    },
  };
  const DEFAULT_PIPELINE_CONFIG: BuildConfigOutputObj['pipelineBuildConfig'] = {
    pipelineID: '',
    params: {},
  };

  const { t } = useI18n();
  const buildPlaceholder = computed(
    () => `${t('留空则使用平台默认：')}\ngo build -o /out/${props.appName || '{{ appName }}'} .`,
  );

  // 镜像字段的 required 规则覆盖 FormItem 默认的 trigger: 'change' 规则，
  // 避免自动选中镜像时（selectedImageId 程序化变化）触发校验导致红色错误闪烁
  const rules = ref({
    'repoBuildConfig.platformBuildConfig.builderImage': [
      {
        required: true,
        message: t('构建镜像不能为空'),
        validator: () => !!selectedBuilderImageId.value,
        trigger: 'custom',
      },
    ],
    'repoBuildConfig.platformBuildConfig.runnerImage': [
      {
        required: true,
        message: t('运行镜像不能为空'),
        validator: () => !!selectedRunnerImageId.value,
        trigger: 'custom',
      },
    ],
    'repoBuildConfig.sourceDir': [
      {
        message: t('构建目录不能以 / 开头'),
        trigger: 'blur',
        validator: (value: string) => !value || !value.startsWith('/'),
      },
    ],
  });

  // ========== 构建方式选项 ==========
  const buildTypeOptions = ref<CardRadioOption[]>([
    {
      value: 'platform',
      label: t('平台通用构建'),
      description: t('无需编写和维护 Dockerfile，平台按标准框架自动完成构建。可按需在「高级命令」补充构建步骤。'),
    },
    {
      value: 'repositoryDockerfile',
      label: t('使用仓库 Dockerfile'),
      description: t(
        '完全使用代码仓库中你自己维护的 Dockerfile 构建，平台不介入生成，构建过程以仓库内的 Dockerfile 为准。',
      ),
    },
  ]);

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
      builderData.value?.repoBuildConfig?.platformBuildConfig?.[type === 'builder' ? 'builderImage' : 'runnerImage'],
    setImageValue: (type, value) => {
      const config = builderData.value?.repoBuildConfig?.platformBuildConfig;
      if (config) {
        config[type === 'builder' ? 'builderImage' : 'runnerImage'] = value;
      }
    },
    getLanguage: () => props.language,
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
  watch(
    () => builderData.value?.repoBuildConfig?.imageBuildMode,
    newVal => {
      if (!builderData.value?.repoBuildConfig) return;
      if (newVal === 'platform') {
        builderData.value.repoBuildConfig.dockerfile = '';
        // 获取平台通用构建镜像列表
        fetchPlatformBuildImages('builder');
        fetchPlatformBuildImages('runner');
      } else if (newVal === 'repositoryDockerfile') {
        builderData.value.repoBuildConfig.platformBuildConfig = {
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
    get: () => builderData.value?.repoBuildConfig?.platformBuildConfig?.commands?.preBuild?.join('\n') ?? '',
    set: val => {
      if (builderData.value?.repoBuildConfig?.platformBuildConfig?.commands) {
        builderData.value.repoBuildConfig.platformBuildConfig.commands.preBuild = val
          ? val
              .split('\n')
              .map(s => s.trim())
              .filter(Boolean)
          : [];
      }
    },
  });

  const buildText = computed({
    get: () => builderData.value?.repoBuildConfig?.platformBuildConfig?.commands?.build?.join('\n') ?? '',
    set: val => {
      if (builderData.value?.repoBuildConfig?.platformBuildConfig?.commands) {
        builderData.value.repoBuildConfig.platformBuildConfig.commands.build = val
          ? val
              .split('\n')
              .map(s => s.trim())
              .filter(Boolean)
          : [];
      }
    },
  });

  const runtimeEnvText = computed({
    get: () => builderData.value?.repoBuildConfig?.platformBuildConfig?.commands?.runtimeEnv?.join('\n') ?? '',
    set: val => {
      if (builderData.value?.repoBuildConfig?.platformBuildConfig?.commands) {
        builderData.value.repoBuildConfig.platformBuildConfig.commands.runtimeEnv = val
          ? val
              .split('\n')
              .map(s => s.trim())
              .filter(Boolean)
          : [];
      }
    },
  });

  /** 缓存当前来源的表单数据 */
  const cacheCurrentTypeData = (type: BuilderType) => {
    if (type === 'codeRepository') {
      repoBuildConfigCache.value = cloneDeep(builderData.value.repoBuildConfig);
    } else {
      pipelineBuildConfigCache.value = cloneDeep(builderData.value.pipelineBuildConfig);
      pipelineParamsCache_list.value = cloneDeep(params.value);
    }
    tagConfigCache.value[type] = cloneDeep(tagConfig.value);
  };

  /** 恢复目标来源的表单数据 */
  const restoreTargetTypeData = (type: BuilderType) => {
    if (type === 'pipeline') {
      builderData.value.pipelineBuildConfig = pipelineBuildConfigCache.value
        ? cloneDeep(pipelineBuildConfigCache.value)
        : cloneDeep(DEFAULT_PIPELINE_CONFIG);
      builderData.value.repoBuildConfig = null;
      params.value = pipelineBuildConfigCache.value ? cloneDeep(pipelineParamsCache_list.value) : params.value;
    } else {
      builderData.value.repoBuildConfig = repoBuildConfigCache.value
        ? cloneDeep(repoBuildConfigCache.value)
        : cloneDeep(DEFAULT_REPO_CONFIG);
      builderData.value.pipelineBuildConfig = null;
      if (!pipelineBuildConfigCache.value) params.value = [];
    }
    tagConfig.value = tagConfigCache.value[type] ?? null;
  };

  /** 切换来源，保留之前填写的表单数据 */
  const toggleBuilderType = (type: BuilderType) => {
    if (type === currentBuilderType.value) return;
    cacheCurrentTypeData(currentBuilderType.value as BuilderType);
    restoreTargetTypeData(type);
    currentBuilderType.value = type;
  };

  // 推荐版本号配置（使用 TagConfigForm 组件的 v-model）
  const tagConfig = ref<null | TagConfigOutputObj>(null);

  /** 保存 tagConfig 到 builderData */
  const saveTagConfig = () => {
    builderData.value.tagConfig = tagConfig.value;
  };

  /** 从 builderData 回显 tagConfig */
  const restoreTagConfig = () => {
    const existingTagConfig = builderData.value?.tagConfig;
    tagConfig.value = existingTagConfig || null;
  };

  /** 切换代码库：同步别名并清空旧默认分支，避免新仓库仍显示无效分支 */
  const handleProjectChange = (data: BkCIOAuthGitProjectOutput) => {
    if (!builderData.value?.repoBuildConfig) return;
    builderData.value.repoBuildConfig.repoAlias = data?.alias || '';
    builderData.value.repoBuildConfig.defaultBranch = '';
  };
  /** 流水线动态表单字段 */
  const params = ref<BkCIPipelineVariableOutput[]>([]);
  const validParams = computed(() =>
    params.value.filter((item): item is BkCIPipelineVariableOutput & { id: string } => !!item.id),
  );
  /** 请求动态表单字段时loading */
  const paramsLoading = ref(false);
  /** 请求的流水线ID是否能够有效查找到字段 */
  const isPipelineIdValid = ref(true);

  /** 使用上一次的参数，作为动态表单的props传入 */
  const lastParamsValue = ref<Record<string, number | string | undefined>>({});
  /** 记录初始化时pipelineId和它的上一次使用的参数 */
  const pipelineParamsCache: Map<string, Record<string, number | string>> = new Map();

  /** 切换 使用上一次的参数/默认参数 */
  const handleFeedbackParamsValue = () => {
    /** 获取pipelineId缓存的数据（上一次的参数） */
    const pipelineID = builderData.value.pipelineBuildConfig?.pipelineID;
    if (!pipelineID) return;

    const cacheValue = pipelineParamsCache.get(pipelineID);
    if (cacheValue) {
      // 触发使用上一次的参数
      for (const item of params.value) {
        lastParamsValue.value[item.id!] = cacheValue[item.id!];
      }
    } else {
      for (const item of params.value) {
        lastParamsValue.value[item.id!] = item.defaultValue;
      }
    }
    // 更新动态表单内的input value
    pipelineParamsRef.value?.setData(lastParamsValue.value);
  };

  /** 获取动态表单，维护loading状态及pipeline有效状态 */
  const getParamsData = async (pipelineId: string) => {
    try {
      paramsLoading.value = true;
      const res = await BkintegrationsBkciService.getBkCIPipelineVariables({
        workspaceID: spaceStore.currentSpace,
        pipelineID: pipelineId,
      });
      params.value = res || [];
      isPipelineIdValid.value = true;
      // 回显pipelineId对应的cacheValue
      nextTick(() => handleFeedbackParamsValue());
    } catch (err) {
      console.error(err);
      isPipelineIdValid.value = false;
    } finally {
      paramsLoading.value = false;
    }
  };

  const formRef = ref();
  const pipelineParamsRef = ref();
  /** 保存编辑 */
  const handleSave = async () => {
    // 表单校验，主要用于 源码仓库
    const formValid = await formRef.value.validate().catch(() => false);
    if (!formValid) return;

    if (currentBuilderType.value === 'pipeline') {
      // 获取并验证流水线参数
      let params = {};
      if (pipelineParamsRef.value) {
        const { valid, data } = await pipelineParamsRef.value.save();
        if (!valid) return;
        params = data;
      }

      forceCleanDirtyTag(() => {
        builderData.value.sourceType = 'pipeline';
        if (builderData.value.pipelineBuildConfig) {
          builderData.value.pipelineBuildConfig.params = params;
        }
        // 保存时清除非当前来源的配置
        builderData.value.repoBuildConfig = null;
        saveTagConfig();
        emit('confirm');
      });
    } else {
      forceCleanDirtyTag(() => {
        builderData.value.sourceType = 'codeRepository';
        // 保存时清除非当前来源的配置
        builderData.value.pipelineBuildConfig = null;
        // 根据构建方式清理不相关字段
        if (builderData.value.repoBuildConfig?.imageBuildMode === 'platform') {
          // platform 模式下清空 dockerfile
          builderData.value.repoBuildConfig.dockerfile = '';
        } else {
          // repositoryDockerfile 模式下移除 platformBuildConfig
          builderData.value.repoBuildConfig!.platformBuildConfig = undefined;
        }
        saveTagConfig();
        emit('confirm');
      });
    }
  };

  /** 初始化侧栏数据 */
  const init = async () => {
    // 回显来源
    currentBuilderType.value = props.type;

    // 重置缓存（每次打开侧边栏时重新开始）
    repoBuildConfigCache.value = null;
    pipelineBuildConfigCache.value = null;
    pipelineParamsCache_list.value = [];
    tagConfigCache.value = { codeRepository: null, pipeline: null };

    // 将当前来源的初始数据存入缓存，以便切换回来时恢复
    if (props.type === 'pipeline' && builderData.value.pipelineBuildConfig) {
      pipelineBuildConfigCache.value = cloneDeep(builderData.value.pipelineBuildConfig);
      // 初始化时便将上一次使用的参数存入Map
      const pipelineID = builderData.value.pipelineBuildConfig.pipelineID ?? '';
      pipelineParamsCache.set(pipelineID, cloneDeep(builderData.value.pipelineBuildConfig.params ?? {}));

      if (pipelineID) {
        getParamsData(pipelineID);
      }
    } else if (props.type === 'codeRepository') {
      repoBuildConfigCache.value = cloneDeep(builderData.value?.repoBuildConfig);
      // 初始化构建方式默认值
      if (!builderData.value?.repoBuildConfig?.imageBuildMode) {
        if (builderData.value.repoBuildConfig) {
          builderData.value.repoBuildConfig.imageBuildMode = 'repositoryDockerfile';
        }
      }
      // 初始化 platformBuildConfig
      if (!builderData.value?.repoBuildConfig?.platformBuildConfig) {
        if (builderData.value.repoBuildConfig) {
          builderData.value.repoBuildConfig.platformBuildConfig = {
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
      }
      // 如果当前是 platform 模式，获取镜像列表并回填
      if (builderData.value?.repoBuildConfig?.imageBuildMode === 'platform') {
        fetchPlatformBuildImages('builder');
        fetchPlatformBuildImages('runner');
      }
    }
    // 回显 tagConfig 配置
    restoreTagConfig();
    // 将初始 tagConfig 存入对应来源的缓存
    tagConfigCache.value[props.type] = cloneDeep(tagConfig.value);
  };

  function handleBeforeClose() {
    return confirmBox();
  }

  async function handleClose() {
    if (await handleBeforeClose()) {
      isShow.value = false;
    }
  }

  /**
   * 查看流水线构建操作指引
   */
  function handleViewGuide() {
    const docUrl = `${import.meta.env.BK_DOC_URL}${DOC_LINKS.PIPELINE_BUILD_GUIDE}`;
    window.open(docUrl, '_blank');
  }

  watch(isShow, val => {
    if (val) {
      init();
    }
    forceCleanDirtyTag();
  });
</script>
