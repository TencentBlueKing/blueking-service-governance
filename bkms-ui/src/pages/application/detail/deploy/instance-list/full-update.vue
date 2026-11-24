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
    :class="{ 'full-update-sideslider': showBuildLog }"
    render-directive="if"
    :width="showBuildLog ? 1200 : 640"
  >
    <template #header>
      <DividerHeader>
        <template #title>
          <span class="text-[16px]">{{ $t('部署') }}</span>
        </template>
        <span v-if="appDetailStore.app">
          {{ appDetailStore.app }}
        </span>
        <span v-if="trpcDeployStore?.curEnvItem?.displayName">
          {{ `${$t('环境')}: ${trpcDeployStore.curEnvItem.displayName}` }}
        </span>
      </DividerHeader>
    </template>
    <template v-if="!showBuildLog">
      <Form
        ref="formRef"
        class="pt-[18px] px-[24px]"
        form-type="vertical"
        :model="formModel"
      >
        <!-- 灰度进行中：部署会结束本次灰度并按本次配置更新实例，提交前需二次确认 -->
        <Alert
          v-if="canaryInProgress"
          class="!mb-[16px]"
          theme="warning"
        >
          {{ $t('当前环境正在灰度发布，重新部署将结束本次灰度，并按本次配置更新实例') }}
        </Alert>

        <Form.FormItem
          class="!mb-[14px]"
          :label="$t('更新内容')"
          required
        >
          <Radio.Group
            v-model="formModel.updateContent"
            @change="handleUpdateContentChange"
          >
            <Radio
              v-bk-tooltips="{ content: deployDisabledReason, disabled: !deployDisabledReason }"
              :disabled="!!deployDisabledReason"
              label="both"
              >{{ $t('镜像+配置') }}</Radio
            >
            <Radio
              v-bk-tooltips="{ content: deployDisabledReason, disabled: !deployDisabledReason }"
              :disabled="!!deployDisabledReason"
              label="config"
              >{{ $t('仅配置') }}</Radio
            >
            <Radio
              :disabled="!!inplaceRestartDisabledReason"
              label="image"
            >
              <span
                v-bk-tooltips="{
                  content: inplaceRestartDisabledReason,
                  disabled: !inplaceRestartDisabledReason,
                }"
              >
                {{ $t('仅镜像（原地重启）') }}
              </span>
            </Radio>
          </Radio.Group>
        </Form.FormItem>

        <Alert
          v-if="formModel.updateContent !== 'both'"
          class="mb-[24px]"
          closable
          theme="info"
        >
          {{ alertContent }}
        </Alert>

        <Form.FormItem
          v-if="formModel.updateContent !== 'image'"
          :label="$t('实例数')"
          property="replicas"
          required
          :rules="replicasRules"
        >
          <Input
            v-model.number="formModel.replicas"
            :disabled="replicasLocked"
            :min="1"
            :precision="0"
            type="number"
          />
          <div
            v-if="hasInvalidEffectiveReplicas"
            class="mt-[4px] text-[12px] leading-[20px] text-[#979BA5]"
          >
            {{ $t('实例删除后副本数已变为 0，请填写期望实例数后重新部署') }}
          </div>
          <!-- 分批更新与修改实例数互斥：已选择分批时锁定实例数，并说明如何调整 -->
          <div
            v-if="replicasLockedHint"
            class="mt-[4px] text-[12px] leading-[20px] text-[#979BA5]"
          >
            {{ replicasLockedHint }}
          </div>
        </Form.FormItem>

        <Form.FormItem
          v-if="formModel.updateContent === 'image'"
          :label="$t('镜像 Tag')"
          property="imageTag"
          required
        >
          <ImageSelect
            ref="imageSelectRef"
            v-model:value="formModel.imageTag"
          />
        </Form.FormItem>

        <template v-if="formModel.updateContent === 'both'">
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
          <CodeRevisionField
            v-if="imageSource === 'code'"
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
              v-else
              ref="imageSelectRef"
              v-model:value="formModel.imageTag"
            />
          </Form.FormItem>
        </template>

        <!-- 发布方式 / 发布策略：与「快速部署」共用同一份准入规则与交互 -->
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
        class="mx-[24px] mb-[24px]"
        :preview="releasePreview"
      />
    </template>
    <template v-else>
      <BuildLogPanel
        :active="showBuildLog"
        :build-info="buildInfo"
      />
    </template>
    <template
      v-if="!showBuildLog"
      #footer
    >
      <Button
        class="mr-[10px]"
        :loading="loading"
        theme="primary"
        @click="handleDeploy"
      >
        {{ $t('部署') }}
      </Button>
      <Button
        :loading="loading"
        @click="handleClose"
      >
        {{ $t('取消') }}
      </Button>
    </template>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, h, reactive, ref, watch } from 'vue';

  import { Alert, Button, Form, InfoBox, Input, Message, Radio, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { InstanceService } from '~/api/modules/v1';
  import { RELEASE_MODE } from '~/common/enums/deploy';
  import { type RevisionType, revisionRequestFields } from '~/components/code-revision/revision';
  import { isAppModelAppType } from '~/composables/app-type';
  import { useAppRepoRefSelect } from '~/composables/use-app-repo-ref-select';
  import { useAppVisibleEnvs } from '~/composables/use-app-visible-envs';
  import useIsFederationEnv from '~/composables/use-is-federation-env';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useRecommendTag } from '~/composables/use-recommend-tag';
  import ImageSelect from '~/pages/application/components/image-select.vue';
  import BuildLogPanel from '~/pages/application/detail/components/view-build-log/build-log-panel.vue';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import ReleasePreviewPanel from '../release-preview-panel.vue';
  import ReleaseStrategyField from '../release-strategy-field.vue';
  import { type DeployableAppType, type DeployParams, useDeployAPIs } from '../use-deploy';
  import { useReleaseStrategy } from '../use-release-strategy';

  import type { AppModelDeployRecordOutputObj } from '~/@types/v1/deploy';
  import type { UpdateAppInstancesRequest } from '~/@types/v1/instance';
  import type { BuildInfo, BuildStatus } from '~/pages/application/detail/components/view-build-log/type';

  type ImageSourceType = 'code' | 'image';

  const isShow = defineModel<boolean>('isShow');
  const emit = defineEmits<{
    update: [keepOpen: boolean];
    /** 灰度发布提交后：由父组件切换到发布详情（无论发布结果成功或失败都跳转） */
    viewReleaseDetail: [];
  }>();
  const props = defineProps<{
    /** 当前环境是否正在灰度发布：提交会结束本次灰度，需提示 + 二次确认，且不支持原地重启 */
    canaryInProgress?: boolean;
    effectiveReplicas?: number;
    isProdEnv?: boolean;
    latestBuildId?: string;
    latestBuildStatus?: BuildStatus;
  }>();

  const { t } = useI18n();
  const trpcDeployStore = useTrpcDeployStore();
  const appDetailStore = useAppDetail();
  const { checkDeployEnv, getEnvDeployDisabledReason } = useAppVisibleEnvs();
  const deployDisabledReason = computed(() => getEnvDeployDisabledReason(trpcDeployStore.curEnvItem));
  const isFederationEnv = useIsFederationEnv(() => trpcDeployStore.curEnvItem);
  /** 「仅镜像（原地重启）」不可用的原因：联邦集群不支持该操作，灰度进行中原位重启会打断本次灰度。 */
  const inplaceRestartDisabledReason = computed(() => {
    if (isFederationEnv.value) return t('联邦集群不支持原地重启');
    if (props.canaryInProgress) return t('灰度发布中不支持原地重启');
    return '';
  });

  const { workspaceId, repoAlias, branchSelectRef, prepareBranchAfterMount } = useAppRepoRefSelect(
    () => appDetailStore.appDetail?.buildConfig?.repoBuildConfig?.repoAlias || '',
  );

  const imageSource = ref<ImageSourceType>('image');
  const formRef = ref();
  const imageSelectRef = ref();
  const showCodeSource = computed(() => appDetailStore.appDetail?.buildConfig?.sourceType !== 'pipeline');
  const defaultRevision = ref('');
  const formModel = reactive<{
    imageTag: string;
    releaseMode: 'all' | 'canary';
    releaseStrategyID: string;
    replicas?: number;
    revision: string;
    revisionType: RevisionType;
    updateContent: 'both' | 'config' | 'image';
  }>({
    revision: '',
    revisionType: 'branch',
    imageTag: '',
    updateContent: 'both',
    releaseMode: 'all',
    releaseStrategyID: '',
  });
  // 切换为“仅配置”时会保留之前的镜像来源，因此只有“镜像+配置”选择源码时才触发构建。
  const shouldBuildFromSource = computed(() => formModel.updateContent === 'both' && imageSource.value === 'code');
  const { getDefaultBranch, fetchRecommendTag } = useRecommendTag(
    () => ({ revisionType: formModel.revisionType, revision: formModel.revision }),
    {
      manualFetchOnly: computed(() => !repoAlias.value),
      onRecommend: tag => {
        if (shouldBuildFromSource.value) {
          formModel.imageTag = tag;
        }
      },
    },
  );

  /** 来源或修订确认后重算推荐版本号，包括用户改过的版本号 */
  function handleRevisionConfirm() {
    fetchRecommendTag();
  }
  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formModel);

  // ---------- 发布方式 / 发布策略：与「快速部署」共用 useReleaseStrategy ----------

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
    envName: () => trpcDeployStore.curEnvItem?.name ?? '',
    isAppModelApp: () => isAppModelAppType(appDetailStore.appType),
    isFederationEnv,
    // 切换为「仅配置」时会保留之前的镜像来源，因此只有「镜像+配置」选择源码时才算构建部署
    isBuildFromSource: shouldBuildFromSource,
    // 「仅镜像（原地重启）」走实例更新接口，同样不支持分批发布
    isInplaceRestart: () => formModel.updateContent === 'image',
    model: formModel,
  });

  function isValidReplicas(value: number | string | undefined) {
    return Number.isInteger(Number(value)) && Number(value) >= 1;
  }

  const replicasRules = [
    {
      validator: isValidReplicas,
      message: t('请输入大于或等于 1 的整数'),
      trigger: 'blur',
    },
  ];

  // 仅在明确取到 0（实例被删光）时提示；undefined 表示规格未知，不能断言为 0。
  const hasInvalidEffectiveReplicas = computed(() => props.effectiveReplicas === 0);

  const alertContent = computed(() => {
    if (formModel.updateContent === 'config') {
      return t('本次更新仅变更应用的配置信息（包括环境变量等），镜像 Tag 保持不变');
    }
    return t(
      '本次更新仅变更镜像 Tag，应用的配置信息（包括环境变量等）保持不变。仅更新容器镜像，不重建 Pod，更新速度更快',
    );
  });
  const loading = ref(false);
  const curImageTag = ref('');

  // 构建日志相关
  const showBuildLog = ref(false);
  /** buildAndCreateDeploy 返回的构建信息，统一供提示、日志请求和流水线跳转使用。 */
  const buildInfo = reactive<BuildInfo>({
    buildID: '',
    imageTag: '',
    operator: '',
    pipelineID: '',
    revision: '',
    status: 'running',
  });
  /**
   * 部署提交成功的顶部提示文案：
   * - 一次更新全部：正在更新全部 N 个实例（实例数缺失时退回通用文案）；
   * - 按策略分批：开始第 1 批灰度发布。
   */
  function buildSubmitSuccessMessage(): string {
    if (formModel.releaseMode === RELEASE_MODE.CANARY) {
      return t('部署任务已提交，开始第 1 批灰度发布');
    }
    const replicas = Number(formModel.replicas) || props.effectiveReplicas || 0;
    return replicas > 0 ? t('部署任务已提交，正在更新全部 {0} 个实例', [replicas]) : t('操作成功');
  }

  /** 灰度进行中的提交确认：说明本次灰度将结束、实例按当前侧栏配置更新。 */
  function confirmEndCanaryAndDeploy() {
    InfoBox({
      title: t('确认结束灰度并重新部署？'),
      headerAlign: 'center',
      footerAlign: 'center',
      contentAlign: 'center',
      content: h(
        'div',
        {
          class: 'rounded-[2px] bg-[#F5F7FA] px-[12px] py-[16px] text-[14px] leading-[22px] text-[#4D4F56]',
        },
        t('本次灰度发布将结束，实例按本次配置更新'),
      ),
      confirmText: t('重新部署'),
      cancelText: t('取消'),
      onConfirm: () => {
        void submitDeploy();
      },
    });
  }

  // 通过接口获取当前镜像 Tag
  async function fetchCurrentImageTag() {
    const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
    const res = await deployAPIs
      .listDeployRecords({
        appID: appDetailStore.appID,
        envName: trpcDeployStore.curEnvItem?.name ?? '',
        page: 1,
        pageSize: 1,
      })
      .catch(() => ({ count: 0, results: [] as AppModelDeployRecordOutputObj[] }));
    curImageTag.value = res!.results[0]?.imageTag?.split(':')?.pop() || '';
  }

  // 侧边栏关闭前确认
  function handleBeforeClose(): Promise<boolean> {
    return confirmBox();
  }

  /** 切换镜像来源；源码模式 prepare 默认分支并预拉列表 */
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

  // 关闭弹窗
  async function handleClose() {
    if (await handleBeforeClose()) {
      // 关闭 ImageSelect 的下拉框
      imageSelectRef.value?.closeDropdown?.();
      isShow.value = false;
    }
  }

  /**
   * 配置+镜像、仅配置
   */
  async function handleConfigAndImage() {
    const env = trpcDeployStore.curEnvItem;
    if (!env?.name || !checkDeployEnv(env)) return false;
    try {
      // 根据应用类型获取对应的部署 API
      const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
      // 根据镜像来源选择部署方式
      const params: DeployParams = {
        appID: appDetailStore.appID,
        envName: env.name,
        imageTag: formModel.imageTag,
        replicas: Number(formModel.replicas),
      };
      if (shouldBuildFromSource.value) {
        Object.assign(params, revisionRequestFields(showCodeSource.value, formModel.revisionType, formModel.revision));
        const buildRes = await deployAPIs.buildAndCreateDeploy!(params);
        // 后端状态类型为 string，这里收窄为 BuildTips 支持的状态。
        const validStatuses: BuildStatus[] = ['failed', 'pollingBroken', 'running', 'success', 'warning'];
        const status = validStatuses.includes(buildRes.status as BuildStatus)
          ? (buildRes.status as BuildStatus)
          : 'warning';
        // 将后续展示、日志请求和流水线跳转所需字段统一保存到 buildInfo。
        Object.assign(buildInfo, {
          buildID: buildRes.buildID || '',
          imageTag: buildRes.params?.BKMS_IMAGE_TAG || formModel.imageTag,
          operator: buildRes.operator || '',
          pipelineID: buildRes.pipelineID || '',
          revision: buildRes.revision || formModel.revision,
          status,
        });
      } else {
        // 已构建镜像 和 “仅配置”均直接部署；“仅配置”的镜像 Tag 已回填为当前部署版本。
        // 已构建镜像 + 分批发布 + 已选策略时透传发布策略；构建部署接口无该字段。
        if (formModel.releaseMode === RELEASE_MODE.CANARY && formModel.releaseStrategyID) {
          params.releaseStrategyID = formModel.releaseStrategyID;
        }
        await deployAPIs.createDeployDirectly!(params);
      }
      return true;
    } catch (error) {
      console.warn(error);
      return false;
    }
  }

  // 部署
  async function handleDeploy() {
    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid) return;

    // 提交前按最新资格兜底校验：分批发布已失效（如主工作负载异步判定为 Deployment）时拦截，
    // 不把后端必拒的 releaseStrategyID 透传出去。
    if (formModel.releaseMode === RELEASE_MODE.CANARY && canaryDisabledReason.value) {
      Message({ theme: 'error', message: canaryDisabledReason.value });
      return;
    }

    // 灰度进行中提交部署会结束本次灰度：先让用户确认，确认后才真正提交。
    if (props.canaryInProgress) {
      confirmEndCanaryAndDeploy();
      return;
    }

    await submitDeploy();
  }

  /**
   * 仅镜像
   */
  async function handleImage() {
    if (isFederationEnv.value) return false;
    try {
      const params: UpdateAppInstancesRequest = {
        appID: appDetailStore.appID,
        envName: trpcDeployStore.curEnvItem?.name ?? '',
        imageTag: formModel.imageTag,
        updateStrategy: 'InplaceUpdate',
        // 全量更新传空数组
        instanceIDs: [],
      };
      await InstanceService.updateAppInstances(params);
      return true;
    } catch (error) {
      console.warn(error);
      return false;
    }
  }

  /**
   * 切换更新内容
   * @param val 'both' | 'config' | 'image'
   */
  function handleUpdateContentChange(val: 'both' | 'config' | 'image') {
    // 仅配置时,使用当前镜像 Tag;其他情况清空让用户选择
    formModel.imageTag = val === 'config' ? curImageTag.value : '';
    // 仅镜像（原地重启）不支持分批发布，切回时静默重置，避免触发禁用重置提示
    if (val === 'image') resetToAllInstances();
  }

  /** 真正提交部署：灰度发布提交后进入发布详情，其它方式成功后按构建模式决定是否关闭侧栏。 */
  async function submitDeploy() {
    // 灰度发布：提交请求结束后无论成败都进入发布详情，只有表单 / 资格校验不通过时留在侧栏。
    const isCanaryRelease = formModel.releaseMode === RELEASE_MODE.CANARY;

    loading.value = true;
    try {
      let result = false;
      if (formModel.updateContent !== 'image') {
        result = await handleConfigAndImage();
      } else {
        result = await handleImage();
      }

      if (isCanaryRelease) {
        forceCleanDirtyTag(() => {
          if (result) {
            Message({
              theme: 'success',
              message: buildSubmitSuccessMessage(),
            });
          }
          emit('update', false);
          emit('viewReleaseDetail');
        });
        return;
      }

      if (!result) return;

      if (shouldBuildFromSource.value) {
        showBuildLog.value = true;
      }

      forceCleanDirtyTag(() => {
        // 源码构建后保留当前侧滑，以便持续展示流式日志；其他部署方式沿用成功后关闭侧滑的交互。
        emit('update', shouldBuildFromSource.value);
        Message({
          theme: 'success',
          message: buildSubmitSuccessMessage(),
        });
      });
    } catch (error) {
      console.warn(error);
    } finally {
      loading.value = false;
    }
  }

  watch(isShow, async val => {
    if (!val) {
      withPausedWatch(() => {
        formModel.revision = '';
        formModel.revisionType = 'branch';
        formModel.updateContent = 'both';
        formModel.imageTag = '';
        formModel.replicas = undefined;
        formModel.releaseMode = 'all';
        formModel.releaseStrategyID = '';
      });
      imageSource.value = 'image';
      showBuildLog.value = false;
      formRef.value?.clearValidate();
    } else {
      // 打开弹窗时回填当前生效实例数；withPausedWatch 避免初始化赋值被标记为未保存修改。
      withPausedWatch(() => {
        formModel.updateContent = deployDisabledReason.value ? 'image' : 'both';
        formModel.replicas = isValidReplicas(props.effectiveReplicas) ? props.effectiveReplicas : undefined;
      });
      // 打开弹窗时获取当前镜像 Tag
      await fetchCurrentImageTag();
    }
  });

  // 关闭构建日志内容区时重置本次构建信息。
  watch(
    showBuildLog,
    val => {
      if (!val) {
        Object.assign(buildInfo, {
          buildID: '',
          imageTag: '',
          operator: '',
          pipelineID: '',
          revision: '',
          status: 'running',
        });
      }
    },
    { immediate: false },
  );

  // 部署页轮询到同一构建的新状态后，同步更新当前日志提示。
  watch([() => props.latestBuildId, () => props.latestBuildStatus], ([latestBuildId, latestBuildStatus]) => {
    if (showBuildLog.value && latestBuildId === buildInfo.buildID && latestBuildStatus) {
      buildInfo.status = latestBuildStatus;
    }
  });
</script>

<style lang="postcss" scoped>
  .full-update-sideslider :deep(.bk-modal-content) {
    height: calc(100% - 52px) !important;
    overflow: hidden !important;
    scrollbar-gutter: auto !important;
    > div {
      height: 100%;
      .bk-sideslider-content {
        height: 100%;
      }
    }
  }
</style>
