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
        :disabled="replicasLocked"
        :min="1"
        :precision="0"
        type="number"
      />
      <!-- 分批更新与修改实例数互斥：已选择分批时锁定实例数，并说明如何调整 -->
      <div
        v-if="replicasLockedHint"
        class="mt-[4px] text-[12px] leading-[20px] text-[#979BA5]"
      >
        {{ replicasLockedHint }}
      </div>
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

    <!-- 发布方式 / 发布策略：与实例列表「全量更新」共用同一份准入规则与交互 -->
    <ReleaseStrategyField
      v-model:strategy-id="formModel.releaseStrategyID"
      :disabled-reason="canaryDisabledReason"
      :existing-names="existingStrategyNames"
      :hint-text="strategyHintText"
      :loading="strategyLoading"
      :options="strategyOptions"
      :release-mode="formModel.releaseMode"
      :selected-strategy="selectedStrategy"
      @cloned="refreshReleaseStrategies"
      @created="handleStrategyCreated"
      @update:release-mode="handleChangeReleaseMode"
    />
  </Form>

  <!-- 发布预览：摘要栏 + 批次时间线 -->
  <ReleasePreviewPanel
    v-if="releasePreview"
    class="mb-[24px]"
    :preview="releasePreview"
  />

  <!-- “资源规格” 和 “环境变量” 统一预检查弹窗 -->
  <EnvVarPrecheckDialog
    v-if="precheckBeforeSubmit"
    v-model:is-show="isShowPrecheckDialog"
    :env-id="precheckEnvId"
    :env-name="precheckEnvName"
    :mismatches="federationMismatches"
    :missing-required-cluster-addons="missingRequiredClusterAddons"
    :undefined-vars="undefinedVars"
    :unsupported-components="unsupportedComponents"
    @cancel="cancelDeploy"
    @go-modify="cancelDeploy"
    @still-deploy="continueDeploy"
  />
</template>

<script lang="ts" setup>
  import { computed, reactive, ref } from 'vue';

  import { Button, Form, Input, Message, Select } from 'bkui-vue';
  import { RELEASE_MODE } from '~/common/enums/deploy';
  import { type RevisionType, revisionRequestFields } from '~/components/code-revision/revision';
  import { isAppModelAppType } from '~/composables/app-type';
  import { useAppRepoRefSelect } from '~/composables/use-app-repo-ref-select';
  import { useAppVisibleEnvs } from '~/composables/use-app-visible-envs';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useRecommendTag } from '~/composables/use-recommend-tag';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import ImageSelect from '../../components/image-select.vue';
  import EnvVarPrecheckDialog from './env-var-precheck-dialog.vue';
  import ReleasePreviewPanel from './release-preview-panel.vue';
  import ReleaseStrategyField from './release-strategy-field.vue';
  import { type DeployableAppType, type DeployParams, useDeployAPIs } from './use-deploy';
  import { useDeployPrecheck } from './use-deploy-precheck';
  import { useReleaseStrategy } from './use-release-strategy';

  import type { EnvOutput } from '~/@types/v1/env';

  // 镜像来源类型：从源码构建 / 已构建镜像
  type ImageSourceType = 'code' | 'image';

  const props = withDefaults(
    defineProps<{
      effectiveReplicas?: number; // 当前生效的实例数，用于重置表单时回填
      envName?: string;
      envType?: string;
      /** 目标环境是否为联邦集群环境；联邦集群暂不支持按策略分批发布。 */
      isFederationEnv?: boolean;
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
    unsupportedComponents,
    precheck,
    precheckEnvName,
    precheckEnvId,
    undefinedVars,
  } = useDeployPrecheck();

  // branchSelectRef 必须解构出来：模板里的 ref="branchSelectRef" 只在存在同名顶层绑定时才会写回，
  // 否则 prepare / reset / dismissIfOutside 全部因 ?. 静默失效（默认分支预选、下拉清理都不生效）。
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
    releaseMode: 'all' | 'canary';
    releaseStrategyID: string;
    replicas: number;
    revision: string;
    revisionType: RevisionType;
  }>({
    replicas: 1,
    imageTag: '',
    revision: '',
    revisionType: 'branch',
    releaseMode: 'all',
    releaseStrategyID: '',
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

  // ---------- 发布方式 / 发布策略：与实例列表「全量更新」共用 useReleaseStrategy ----------

  const {
    canaryDisabledReason,
    existingStrategyNames,
    handleChangeReleaseMode,
    handleStrategyCreated,
    releasePreview,
    // 查看抽屉里克隆为空间策略成功后刷新列表，成功提示由查看抽屉展示
    refresh: refreshReleaseStrategies,
    replicasLocked,
    replicasLockedHint,
    resetToAllInstances,
    selectedStrategy,
    strategyHintText,
    strategyLoading,
    strategyOptions,
  } = useReleaseStrategy({
    currentReplicas: () => props.effectiveReplicas,
    envName,
    isAppModelApp: () => isAppModelAppType(appDetailStore.appType),
    isFederationEnv: () => Boolean(props.isFederationEnv),
    isBuildFromSource: () => imageSource.value === 'code',
    model: formModel,
  });
  // 离开确认：表单有变更时提示用户
  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formModel);

  /**
   * 提交结果：
   * - submitted=false 表示未真正发起部署（表单校验 / 环境预检查 / 环境资格拦截），调用方应停留在侧栏；
   * - ok 表示部署接口是否成功返回，releaseMode 供调用方判断是否需要跳转「发布详情」。
   */
  interface DeploySubmitResult {
    ok: boolean;
    releaseMode: 'all' | 'canary';
    submitted: boolean;
  }

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
    // 提交前按最新资格兜底校验：分批发布已失效（如主工作负载异步判定为 Deployment）时拦截，
    // 不把后端必拒的 releaseStrategyID 透传出去。
    if (formModel.releaseMode === RELEASE_MODE.CANARY && canaryDisabledReason.value) {
      Message({ theme: 'error', message: canaryDisabledReason.value });
      return false;
    }
    const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
    const params: DeployParams = {
      appID: appDetailStore.appID,
      envName: targetEnv.name,
      imageTag: formModel.imageTag,
      replicas: Number(formModel.replicas),
    };
    if (imageSource.value === 'image') {
      // 仅镜像直连部署 + 分批发布 + 已选策略时透传发布策略；构建部署接口无该字段。
      if (formModel.releaseMode === RELEASE_MODE.CANARY && formModel.releaseStrategyID) {
        params.releaseStrategyID = formModel.releaseStrategyID;
      }
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
      // 构建部署不支持分批发布，切到源码构建时回落为一次更新全部
      resetToAllInstances();
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
      formModel.releaseMode = 'all';
      formModel.releaseStrategyID = '';
      imageSource.value = 'image';
    });
    forceCleanDirtyTag(() => {
      clearValidate();
      closeDropdown();
    });
  }

  // 提交部署：内部完成校验、部署和脏标记清理
  async function submit(targetEnv: EnvOutput): Promise<DeploySubmitResult> {
    const releaseMode = formModel.releaseMode;
    const valid = await validate();
    if (!valid) return { ok: false, releaseMode, submitted: false };

    if (props.precheckBeforeSubmit) {
      const precheckPassed = await precheck(targetEnv);
      if (!precheckPassed) return { ok: false, releaseMode, submitted: false };
    }

    try {
      if (!(await deploy(targetEnv))) return { ok: false, releaseMode, submitted: false };
    } catch (error) {
      // 请求异常已由拦截器提示，这里只记录日志：灰度发布即使失败也要进入发布详情。
      console.warn(error);
      return { ok: false, releaseMode, submitted: true };
    }
    forceCleanDirtyTag();
    return { ok: true, releaseMode, submitted: true };
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
