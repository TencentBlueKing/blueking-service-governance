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
    :label="$t('来源')"
    :property="`${prefix}helmSource.repoType`"
    required
    :rules="rules.repoType"
  >
    <Radio.Group
      v-model="repoType"
      class="w-full"
    >
      <Radio.Button
        class="w-full"
        :label="REPO_TYPE_GIT"
        >{{ $t('源码仓库') }}</Radio.Button
      >
      <Radio.Button
        class="w-full"
        :label="REPO_TYPE_HELM"
        >Helm Repo</Radio.Button
      >
    </Radio.Group>
  </Form.FormItem>
  <!-- 源码 -->
  <template v-if="repoType === REPO_TYPE_GIT">
    <Form.FormItem
      :label="$t('代码库')"
      :property="`${prefix}helmSource.gitRepoConfig.repoURL`"
      required
      :rules="rules.gitRepo.repoURL"
    >
      <GitSelector
        v-model="gitRepoConfig.repoURL"
        :workspace="spaceStore.currentSpace"
        @change="handleRepoURLChange"
      />
    </Form.FormItem>
    <Form.FormItem
      :label="$t('分支')"
      :property="`${prefix}helmSource.gitRepoConfig.revision`"
      required
      :rules="rules.gitRepo.revision"
    >
      <Input
        v-model.trim="gitRepoConfig.revision"
        clearable
      />
    </Form.FormItem>
    <Form.FormItem
      :label="$t('路径')"
      :property="`${prefix}helmSource.gitRepoConfig.sourceDir`"
      required
      :rules="rules.gitRepo.sourceDir"
    >
      <Input
        v-model.trim="gitRepoConfig.sourceDir"
        :placeholder="$t('Helm Chart 在仓库内的相对路径（根目录请填 .）')"
        required
      />
    </Form.FormItem>
  </template>
  <!-- helm repo -->
  <template v-else-if="repoType === REPO_TYPE_HELM">
    <Form.FormItem
      :label="$t('仓库地址')"
      :property="`${prefix}helmSource.helmRepoConfig.repoURL`"
      required
      :rules="rules.helmRepo.repoURL"
    >
      <Input
        v-model.trim="helmRepoConfig.repoURL"
        clearable
      />
    </Form.FormItem>
    <Form.FormItem
      class="flex-none"
      label="Chart"
      :property="`${prefix}helmSource.helmRepoConfig.chartName`"
      required
      :rules="rules.helmRepo.chartName"
    >
      <Input
        v-model.trim="helmRepoConfig.chartName"
        clearable
      />
    </Form.FormItem>
    <Form.FormItem
      :label="$t('Values 文件')"
      :property="`${prefix}helmSource.valueFiles`"
    >
      <Input
        clearable
        :model-value="valueFiles"
        :placeholder="$t('请输入，若留空则默认使用 values.yaml')"
        @change="handleValuesFileChange"
      />
    </Form.FormItem>
    <Form.FormItem :label="$t('凭证')">
      <div class="flex gap-[12px]">
        <Form.FormItem
          class="flex-1 mb-[0px]"
          label=""
          :property="`${prefix}helmSource.helmRepoConfig.username`"
        >
          <Input
            v-model.trim="helmRepoConfig.username"
            clearable
            :placeholder="$t('请输入用户名')"
          />
        </Form.FormItem>
        <Form.FormItem
          class="flex-1 mb-[0px]"
          label=""
          :property="`${prefix}helmSource.helmRepoConfig.password`"
        >
          <PasswordInput
            v-model="helmRepoConfig.password"
            v-model:modified="passwordModified"
            :has-credential="hasInitialCredential"
            :is-edit-mode="isEditMode"
          />
        </Form.FormItem>
      </div>
      <div class="text-[#979BA5] text-[12px] leading-[20px]">{{ $t('私有仓库需要填写凭证才能拉取 chart') }}</div>
    </Form.FormItem>
  </template>
</template>
<script setup lang="ts">
  import { computed, onMounted, ref, watch } from 'vue';

  import { Form, Input, Radio } from 'bkui-vue';
  import { cloneDeep, isEqual } from 'lodash-es';
  import { useSpaceStore } from '~/stores/space';

  import type { HelmGitRepoConfigInput, HelmRepoConfigInput, HelmSourceInput, HelmSpecInput } from '~/@types/v1/app';
  import type { BkCIOAuthGitProjectOutput } from '~/@types/v1/bkintegrations-bkci';

  // repoType 常量
  const REPO_TYPE_GIT = 'GitRepo';
  const REPO_TYPE_HELM = 'HelmRepo';
  // const REPO_TYPE_BCS = 'BCSRepo';

  interface IProps {
    initialData?: HelmSpecInput;
    isEditMode?: boolean;
    validatePrefix?: string;
  }
  const props = withDefaults(defineProps<IProps>(), {
    isEditMode: false,
  });

  type RepoType = typeof REPO_TYPE_GIT | typeof REPO_TYPE_HELM;

  /** 记录是否有凭证（username 有值），用于控制 PasswordInput 占位符显示 */
  const hasInitialCredential = ref(false);
  const passwordModified = ref(false);
  const prefix = computed(() => (props.validatePrefix ? `${props.validatePrefix}.` : ''));

  // repoType 计算属性，简化模板中使用
  const repoType = computed<RepoType>({
    get: () => helmSpecData.value?.helmSource?.repoType as RepoType,
    set: (val: RepoType) => {
      helmSpecData.value.helmSource.repoType = val;
    },
  });

  const spaceStore = useSpaceStore();

  // 表单校验规则
  const rules = computed(() => ({
    repoType: [
      {
        required: true,
        trigger: 'change',
        validator: () => !!helmSpecData.value.helmSource.repoType,
      },
    ],
    gitRepo: {
      repoURL: [
        {
          required: true,
          trigger: 'blur',
          validator: () => !!helmSpecData.value.helmSource.gitRepoConfig?.repoURL,
        },
      ],
      revision: [
        {
          required: true,
          trigger: 'blur',
          validator: () => !!helmSpecData.value.helmSource.gitRepoConfig?.revision,
        },
      ],
      sourceDir: [
        {
          required: true,
          trigger: 'blur',
          validator: () => !!helmSpecData.value.helmSource.gitRepoConfig?.sourceDir,
        },
      ],
    },
    helmRepo: {
      repoURL: [
        {
          required: true,
          trigger: 'blur',
          validator: () => !!helmSpecData.value.helmSource.helmRepoConfig?.repoURL,
        },
      ],
      chartName: [
        {
          required: true,
          trigger: 'blur',
          validator: () => !!helmSpecData.value.helmSource.helmRepoConfig?.chartName,
        },
      ],
    },
  }));
  // 默认配置
  const GIT_REPO_CONFIG_DEFAULT: HelmGitRepoConfigInput = {
    type: 'TGit',
    repoAlias: '',
    repoURL: '',
    revision: '',
    sourceDir: '',
  };

  const HELM_REPO_CONFIG_DEFAULT: HelmRepoConfigInput = {
    chartName: '',
    password: '',
    repoURL: '',
    username: '',
  };

  // 缓存
  const gitRepoCache = ref<HelmGitRepoConfigInput>(cloneDeep(GIT_REPO_CONFIG_DEFAULT));
  const helmRepoCache = ref<HelmRepoConfigInput>(cloneDeep(HELM_REPO_CONFIG_DEFAULT));

  // 创建默认 HelmSpec
  const getDefaultHelmSpec = (): HelmSpecInput => ({
    helmSource: {
      repoType: REPO_TYPE_GIT,
      gitRepoConfig: cloneDeep(GIT_REPO_CONFIG_DEFAULT),
      helmRepoConfig: cloneDeep(HELM_REPO_CONFIG_DEFAULT),
      bcsRepoConfig: { projectCode: '', repoName: '', chartName: '' },
      valueFiles: ['values.yaml'],
    },
  });

  const helmSpecData = ref<HelmSpecInput>(props.initialData ? cloneDeep(props.initialData) : getDefaultHelmSpec());

  // 初始化时重新赋值 helmSource，确保只有一个 repoType 对应的配置
  const initHelmSpecData = () => {
    const currentRepoType = repoType.value;
    const valueFiles = helmSpecData.value?.helmSource?.valueFiles;
    if (currentRepoType === REPO_TYPE_GIT) {
      helmSpecData.value.helmSource = {
        repoType: REPO_TYPE_GIT,
        gitRepoConfig: helmSpecData.value.helmSource.gitRepoConfig || cloneDeep(GIT_REPO_CONFIG_DEFAULT),
        valueFiles: valueFiles || ['values.yaml'],
      } as HelmSourceInput;
    } else if (currentRepoType === REPO_TYPE_HELM) {
      helmSpecData.value.helmSource = {
        repoType: REPO_TYPE_HELM,
        helmRepoConfig: helmSpecData.value.helmSource.helmRepoConfig || cloneDeep(HELM_REPO_CONFIG_DEFAULT),
        valueFiles: valueFiles || ['values.yaml'],
      } as HelmSourceInput;
    }
    // 根据 API 返回的 username 判断是否有凭证（username 有值才显示小圆点占位）
    const username = helmSpecData.value?.helmSource?.helmRepoConfig?.username;
    hasInitialCredential.value = typeof username === 'string' && username.length > 0;
    passwordModified.value = false;
  };

  // 仓库地址变更
  function handleRepoURLChange(project: BkCIOAuthGitProjectOutput) {
    helmSpecData.value.helmSource.gitRepoConfig!.repoAlias = project.alias!;
  }

  // 桥接计算属性，解决模板中深层可选属性的类型安全问题
  const gitRepoConfig = computed({
    get: () => helmSpecData.value.helmSource.gitRepoConfig!,
    set: (val: HelmGitRepoConfigInput) => {
      helmSpecData.value.helmSource.gitRepoConfig = val;
    },
  });
  const helmRepoConfig = computed({
    get: () => helmSpecData.value.helmSource.helmRepoConfig!,
    set: (val: HelmRepoConfigInput) => {
      helmSpecData.value.helmSource.helmRepoConfig = val;
    },
  });

  const valueFiles = computed(() => helmSpecData.value?.helmSource.valueFiles?.join(','));

  function getPasswordModified(): boolean {
    return passwordModified.value;
  }

  // 暴露 getValue 方法供父组件调用
  function getValue(): HelmSpecInput | null {
    return cloneDeep(helmSpecData.value);
  }

  function handleValuesFileChange(v: string) {
    helmSpecData.value.helmSource.valueFiles = v.split(',');
  }

  // 监听 repoType 变化，缓存/恢复数据
  watch(repoType, (newRepoType, oldRepoType) => {
    if (!oldRepoType) return;

    const valueFiles = helmSpecData.value.helmSource.valueFiles || ['values.yaml'];

    // 1. 缓存旧配置
    if (oldRepoType === REPO_TYPE_GIT) {
      gitRepoCache.value = cloneDeep(helmSpecData.value.helmSource.gitRepoConfig!);
    } else if (oldRepoType === REPO_TYPE_HELM) {
      helmRepoCache.value = cloneDeep(helmSpecData.value.helmSource.helmRepoConfig!);
    }

    // 2. 从缓存恢复新配置，重新构建 helmSource
    if (newRepoType === REPO_TYPE_GIT) {
      helmSpecData.value.helmSource = {
        repoType: REPO_TYPE_GIT,
        gitRepoConfig: cloneDeep(gitRepoCache.value),
        valueFiles,
      } as HelmSourceInput;
    } else if (newRepoType === REPO_TYPE_HELM) {
      helmSpecData.value.helmSource = {
        repoType: REPO_TYPE_HELM,
        helmRepoConfig: cloneDeep(helmRepoCache.value),
        valueFiles,
      } as HelmSourceInput;
    }
    passwordModified.value = false;
  });

  // 监听 initialData 变化，同步到内部
  watch(
    () => props.initialData,
    newVal => {
      if (newVal === undefined) return;
      if (isEqual(newVal, helmSpecData.value)) return;
      helmSpecData.value = cloneDeep(newVal);
      initHelmSpecData();
    },
    { immediate: true },
  );

  onMounted(() => {
    initHelmSpecData();
  });

  defineExpose({ getPasswordModified, getValue });
</script>
