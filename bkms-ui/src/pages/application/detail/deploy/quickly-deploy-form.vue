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
  <Form
    ref="formRef"
    form-type="vertical"
    :model="formModel"
  >
    <Form.FormItem
      :label="$t('实例数')"
      property="replicas"
      required
    >
      <Input
        v-model.number="formModel.replicas"
        :min="1"
        :precision="0"
        type="number"
      />
    </Form.FormItem>

    <Form.FormItem
      :label="$t('镜像来源')"
      required
    >
      <Button.ButtonGroup class="flex items-center">
        <Button
          class="flex-1"
          :selected="imageSource === 'image'"
          @click="handleChangeImageSource('image')"
        >
          {{ $t('已构建镜像') }}
        </Button>
        <Button
          v-bk-tooltips="{
            content: $t('生产类型环境只能部署已经晋级的镜像 Tag'),
            disabled: !isProdEnv,
          }"
          class="flex-1"
          :disabled="isProdEnv"
          :selected="imageSource === 'code'"
          @click="handleChangeImageSource('code')"
        >
          {{ $t('从源码构建') }}
        </Button>
      </Button.ButtonGroup>
    </Form.FormItem>
    <template v-if="imageSource === 'code'">
      <CodeRevisionField
        ref="branchSelectRef"
        v-model="formModel.revision"
        v-model:revision-type="formModel.revisionType"
        :default-revision="defaultRevision"
        :repository-id="repoAlias"
        required
        :show-source="showCodeSource"
        :workspace-id="workspaceId"
        @confirm="handleRevisionConfirm"
      />
    </template>
    <Form.FormItem
      :label="$t('镜像 Tag')"
      property="imageTag"
      required
    >
      <Input
        v-if="imageSource === 'code'"
        v-model.trim="formModel.imageTag"
      />
      <ImageSelect
        v-else-if="envName"
        ref="imageSelectRef"
        v-model:value="formModel.imageTag"
        :env-name="envName"
        :env-type="envType"
      />
      <!-- 占位 -->
      <Select
        v-else
        v-bk-tooltips="$t('请选择环境')"
        disabled
      />
    </Form.FormItem>
  </Form>

  <!-- “资源规格” 和 “环境变量” 统一预检查弹窗 -->
  <EnvVarPrecheckDialog
    v-if="precheckBeforeSubmit"
    v-model:is-show="isShowPrecheckDialog"
    :env-id="precheckEnvId"
    :env-name="precheckEnvName"
    :mismatches="federationMismatches"
    :missing-required-cluster-addons="missingRequiredClusterAddons"
    :undefined-vars="undefinedVars"
    @cancel="cancelDeploy"
    @go-modify="cancelDeploy"
    @still-deploy="continueDeploy"
  />
</template>

<script lang="ts" setup>
  import { computed, reactive, ref } from 'vue';

  import { Button, Form, Input, Select } from 'bkui-vue';
  import { type RevisionType, revisionRequestFields } from '~/components/code-revision/revision';
  import { useAppRepoRefSelect } from '~/composables/use-app-repo-ref-select';
  import { useAppVisibleEnvs } from '~/composables/use-app-visible-envs';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useRecommendTag } from '~/composables/use-recommend-tag';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import ImageSelect from '../../components/image-select.vue';
  import EnvVarPrecheckDialog from './env-var-precheck-dialog.vue';
  import { type DeployableAppType, type DeployParams, useDeployAPIs } from './use-deploy';
  import { useDeployPrecheck } from './use-deploy-precheck';

  import type { EnvOutput } from '~/@types/v1/env';

  // 镜像来源类型：从源码构建 / 已构建镜像
  type ImageSourceType = 'code' | 'image';

  const props = withDefaults(
    defineProps<{
      effectiveReplicas?: number; // 当前生效的实例数，用于重置表单时回填
      envName?: string;
      envType?: string;
      isProdEnv?: boolean;
      /** 是否在提交表单时执行预检查；普通部署已在打开侧栏前完成检查。 */
      precheckBeforeSubmit?: boolean;
      /** 总览入口强制使用显式传入的目标环境，防止未选中时误回退到 store 中的旧环境。 */
      useProvidedEnv?: boolean;
    }>(),
    {
      precheckBeforeSubmit: true,
    },
  );

  const trpcDeployStore = useTrpcDeployStore();
  const appDetailStore = useAppDetail();
  const { checkDeployEnv } = useAppVisibleEnvs();
  const {
    cancelDeploy,
    continueDeploy,
    federationMismatches,
    isShowPrecheckDialog,
    missingRequiredClusterAddons,
    precheck,
    precheckEnvName,
    precheckEnvId,
    undefinedVars,
  } = useDeployPrecheck();

  const { workspaceId, repoAlias, branchSelectRef, prepareBranchAfterMount } = useAppRepoRefSelect(
    () => appDetailStore.appDetail?.buildConfig?.repoBuildConfig?.repoAlias || '',
  );

  // 当前选择的镜像来源
  const imageSource = ref<ImageSourceType>('image');
  const formRef = ref();
  const imageSelectRef = ref();
  // 表单数据模型
  const showCodeSource = computed(() => appDetailStore.appDetail?.buildConfig?.sourceType !== 'pipeline');
  const defaultRevision = ref('');
  const formModel = reactive<{
    imageTag: string;
    replicas: number;
    revision: string;
    revisionType: RevisionType;
  }>({
    replicas: 1,
    imageTag: '',
    revision: '',
    revisionType: 'branch',
  });

  // 实例列表入口沿用 store 当前环境；总览入口只认侧栏选中的目标环境，即使暂时为空也不回退。
  const envName = computed(() =>
    props.useProvidedEnv ? (props.envName ?? '') : props.envName || trpcDeployStore.curEnvItem?.name || '',
  );
  const envType = computed(() =>
    props.useProvidedEnv ? (props.envType ?? '') : props.envType || trpcDeployStore.curEnvItem?.type || '',
  );
  // 是否为生产环境：props 优先，否则按 envType 判断
  const isProdEnv = computed(() => props.isProdEnv ?? envType.value === 'production');

  // 推荐镜像 Tag：源码构建模式下自动推荐
  const { getDefaultBranch, fetchRecommendTag } = useRecommendTag(
    () => ({ revisionType: formModel.revisionType, revision: formModel.revision }),
    {
      manualFetchOnly: computed(() => !repoAlias.value),
      onRecommend: tag => {
        if (imageSource.value === 'code') {
          formModel.imageTag = tag;
        }
      },
    },
  );

  /** 来源或修订确认后重算推荐版本号，包括用户改过的版本号 */
  function handleRevisionConfirm() {
    fetchRecommendTag();
  }
  // 离开确认：表单有变更时提示用户
  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formModel);

  // 清除表单校验结果
  function clearValidate() {
    formRef.value?.clearValidate?.();
  }

  // 关闭镜像选择下拉框
  function closeDropdown() {
    imageSelectRef.value?.closeDropdown?.();
  }

  function confirmLeave() {
    return confirmBox();
  }

  // 执行部署：根据镜像来源调用直接部署或构建+部署接口
  async function deploy(targetEnv: EnvOutput) {
    if (!targetEnv.name || !checkDeployEnv(targetEnv)) return false;
    const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
    const params: DeployParams = {
      appID: appDetailStore.appID,
      envName: targetEnv.name,
      imageTag: formModel.imageTag,
      replicas: Number(formModel.replicas),
    };
    if (imageSource.value === 'image') {
      await deployAPIs.createDeployDirectly!(params);
    } else {
      Object.assign(params, revisionRequestFields(showCodeSource.value, formModel.revisionType, formModel.revision));
      await deployAPIs.buildAndCreateDeploy!(params);
    }
    return true;
  }

  /** 切换镜像来源；源码模式 prepare 默认分支并预拉列表，镜像模式清空源码相关字段。 */
  async function handleChangeImageSource(source: ImageSourceType) {
    imageSource.value = source;
    if (source === 'code') {
      defaultRevision.value = getDefaultBranch();
      formModel.revisionType = 'branch';
      await prepareBranchAfterMount(defaultRevision.value);
      // 无仓库别名时 prepare 不会发出确认，需要主动拉一次推荐版本号
      if (!repoAlias.value) {
        await fetchRecommendTag();
      }
    } else {
      formModel.revision = '';
      formModel.revisionType = 'branch';
      formModel.imageTag = '';
    }
  }

  // 重置表单：回填实例数，其余字段清空，并统一清理内部 UI 状态
  function reset(replicas = props.effectiveReplicas || 1) {
    withPausedWatch(() => {
      formModel.replicas = replicas;
      formModel.imageTag = '';
      formModel.revision = '';
      formModel.revisionType = 'branch';
      imageSource.value = 'image';
    });
    forceCleanDirtyTag(() => {
      clearValidate();
      closeDropdown();
    });
  }

  // 提交部署：内部完成校验、部署和脏标记清理
  async function submit(targetEnv: EnvOutput) {
    const valid = await validate();
    if (!valid) return false;

    if (props.precheckBeforeSubmit) {
      const precheckPassed = await precheck(targetEnv);
      if (!precheckPassed) return false;
    }

    if (!(await deploy(targetEnv))) return false;
    forceCleanDirtyTag();
    return true;
  }

  // 校验表单
  async function validate() {
    return await formRef.value?.validate?.().catch(() => false);
  }

  // 暴露业务级方法供父组件调用
  defineExpose({
    confirmLeave,
    reset,
    submit,
    validate,
  });
</script>
