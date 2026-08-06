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
  <Form.FormItem
    :label="$t('镜像来源')"
    :property="`${prefix}sourceType`"
    required
  >
    <Radio.Group
      v-model="buildConfigData.sourceType"
      class="w-full"
      @change="handleSourceTypeChange"
    >
      <Radio.Button
        v-bk-tooltips="{
          content: $t('Agones 应用不支持该功能'),
          disabled: !forceDisableCodeRepo,
        }"
        class="w-full"
        :disabled="forceDisableCodeRepo"
        label="codeRepository"
      >
        {{ $t('代码仓库') }}
      </Radio.Button>
      <Radio.Button
        class="w-full"
        label="imageRegistry"
        >{{ $t('镜像仓库') }}</Radio.Button
      >
    </Radio.Group>
  </Form.FormItem>
  <!-- 代码仓库 -->
  <template v-if="buildConfigData.sourceType === 'codeRepository'">
    <Form.FormItem
      :label="$t('代码库')"
      :property="`${prefix}repoBuildConfig.repoURL`"
      required
    >
      <GitSelector
        v-model="buildConfigData.repoBuildConfig!.repoURL"
        :workspace="spaceStore.currentSpace"
        @change="handleRepoURLChange"
      />
    </Form.FormItem>
    <Form.FormItem
      :label="$t('默认分支')"
      :property="`${prefix}repoBuildConfig.defaultBranch`"
      required
    >
      <Input
        v-model.trim="buildConfigData.repoBuildConfig!.defaultBranch"
        clearable
      />
    </Form.FormItem>
    <Form.FormItem
      :label="$t('构建目录')"
      :property="`${prefix}repoBuildConfig.sourceDir`"
    >
      <Input
        v-model.trim="buildConfigData.repoBuildConfig!.sourceDir"
        clearable
        :placeholder="$t('请输入应用所在子目录，不填则默认为根目录')"
      />
    </Form.FormItem>
    <Form.FormItem
      :label="`Dockerfile ${$t('路径')}`"
      :property="`${prefix}repoBuildConfig.dockerfile`"
    >
      <Input
        v-model.trim="buildConfigData.repoBuildConfig!.dockerfile"
        clearable
        :placeholder="$t('相对于构建目录的路径，若留空，默认为构建目录下名为 Dockerfile 的文件')"
      />
    </Form.FormItem>
    <Form.FormItem
      :label="$t('构建参数')"
      :property="`${prefix}repoBuildConfig.dockerBuildArgs`"
    >
      <KeyValue
        :model-value="dockerBuildArgs"
        @update:model-value="handleDockerBuildArgsChange"
      />
    </Form.FormItem>
    <TagConfigForm
      v-if="showTagConfig"
      v-model="tagConfig"
      :label="$t('推荐版本号')"
      :property="`${prefix}tagConfig`"
    />
  </template>
  <template v-else-if="buildConfigData.sourceType === 'imageRegistry'">
    <Form.FormItem
      class="col-span-2"
      :label="$t('镜像仓库')"
      :property="`${prefix}imageBuildConfig.name`"
      required
    >
      <Input
        v-model.trim="buildConfigData.imageBuildConfig!.name"
        clearable
        :placeholder="$t('请输入镜像仓库，如：{0}', ['mirrors.tencent.com/blueking/helloworld'])"
      />
      <div class="text-[#979BA5] text-[12px] leading-[20px]">{{ $t('镜像仓库地址，无需包含 tag') }}</div>
    </Form.FormItem>
    <Form.FormItem
      :label="$t('镜像凭证')"
      :property="`${prefix}imageBuildConfig.username`"
    >
      <div class="flex items-center gap-[8px]">
        <Input
          v-model.trim="buildConfigData.imageBuildConfig!.username"
          class="flex-1"
          clearable
          :placeholder="$t('请输入账号')"
        />
        <!-- 构建配置 -->
        <PasswordInput
          v-model="buildConfigData.imageBuildConfig!.password"
          v-model:modified="passwordModified"
          class="flex-1"
          :has-credential="hasInitialCredential"
          :is-edit-mode="isEditMode"
        />
      </div>
      <div class="text-[#979BA5] text-[12px] leading-[20px]">{{ $t('私有镜像需要填写镜像凭证才能拉去镜像') }}</div>
    </Form.FormItem>
  </template>
</template>
<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Form, Input, Radio } from 'bkui-vue';
  import { cloneDeep, isEqual } from 'lodash-es';
  import { useSpaceStore } from '~/stores/space';

  import type { BuildConfigInput, RepoBuildConfigInput, TagConfigOutputObj } from '~/@types/v1/app';
  import type { BkCIOAuthGitProjectOutput } from '~/@types/v1/bkintegrations-bkci';

  // BuildConfig 扩展类型，包含 tagConfig
  interface BuildConfigWithTagConfig extends BuildConfigInput {
    tagConfig?: null | TagConfigOutputObj;
  }

  interface IEmit {
    (e: 'update:modelValue', data: BuildConfigWithTagConfig): void;
  }
  interface IProps {
    forceDisableCodeRepo?: boolean;
    isEditMode?: boolean;
    modelValue: BuildConfigWithTagConfig;
    showTagConfig?: boolean;
    validatePrefix?: string;
  }
  const props = withDefaults(defineProps<IProps>(), {
    showTagConfig: true,
    forceDisableCodeRepo: false,
    isEditMode: false,
  });
  const emits = defineEmits<IEmit>();

  const spaceStore = useSpaceStore();

  const buildConfigData = ref(props.modelValue);
  const passwordModified = defineModel<boolean>('passwordModified', { default: false });
  const prefix = computed(() => (props.validatePrefix ? `${props.validatePrefix}.` : ''));

  // 推荐版本号配置
  const tagConfig = ref<null | TagConfigOutputObj>(props.modelValue?.tagConfig || null);
  // 保存初始 tagConfig 值，用于切换镜像来源时重置
  let initialTagConfig: null | TagConfigOutputObj = cloneDeep(props.modelValue?.tagConfig) || null;
  /** 记录是否有凭证（username 有值），用于控制 PasswordInput 占位符显示 */
  const hasInitialCredential = ref(false);

  watch(
    () => props.modelValue,
    newVal => {
      const clonedTagConfig = cloneDeep(newVal?.tagConfig) || null;
      tagConfig.value = clonedTagConfig;
      initialTagConfig = clonedTagConfig;

      // 先判断数据是否需要更新
      const needUpdate = !isEqual(newVal, buildConfigData.value);
      if (needUpdate) {
        buildConfigData.value = cloneDeep(newVal);
      }

      // 根据 API 返回的 username 判断是否有凭证（username 有值才显示小圆点占位）
      const username = buildConfigData.value?.imageBuildConfig?.username;
      hasInitialCredential.value = typeof username === 'string' && username.length > 0;
      passwordModified.value = false;
    },
    { immediate: true },
  );

  watch(buildConfigData, () => {
    emits('update:modelValue', buildConfigData.value);
  });

  // 监听 tagConfig 变化，同步到 buildConfigData
  watch(tagConfig, val => {
    buildConfigData.value.tagConfig = val;
    emits('update:modelValue', buildConfigData.value);
  });

  // 镜像来源
  function handleSourceTypeChange() {
    if (buildConfigData.value.sourceType === 'codeRepository') {
      (buildConfigData.value as Partial<BuildConfigInput>).imageBuildConfig = undefined;
      (buildConfigData.value.repoBuildConfig as Partial<RepoBuildConfigInput>) = {
        type: 'TGit', // todo 目前GitSelector组件只支持TGit
        dockerBuildArgs: {},
      };
    } else if (buildConfigData.value.sourceType === 'imageRegistry') {
      (buildConfigData.value as Partial<BuildConfigInput>).repoBuildConfig = undefined;
      buildConfigData.value.imageBuildConfig = {
        name: '',
        username: '',
        password: '',
      };
      passwordModified.value = false;
    }
    tagConfig.value = cloneDeep(initialTagConfig);
  }

  // 构建参数
  const dockerBuildArgs = computed(() =>
    Object.entries(buildConfigData.value.repoBuildConfig?.dockerBuildArgs || {}).map(([key, value]) => ({
      key,
      value: String(value),
    })),
  );
  function handleDockerBuildArgsChange(value: Record<string, string> | Record<string, string>[]) {
    if (Array.isArray(value)) {
      // 处理数组格式
      const argsObject: Record<string, string> = {};
      value.forEach(arg => {
        if (arg.key && arg.value) {
          argsObject[arg.key] = arg.value;
        }
      });
      // todo 目前GitSelector组件只支持TGit
      buildConfigData.value.repoBuildConfig!.type = 'TGit';
      buildConfigData.value.repoBuildConfig!.dockerBuildArgs = argsObject;
    }
  }

  // 仓库地址变更
  function handleRepoURLChange(project: BkCIOAuthGitProjectOutput) {
    buildConfigData.value.repoBuildConfig!.repoAlias = project.alias ?? '';
  }
</script>
