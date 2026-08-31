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
        <Form.FormItem
          :label="$t('更新内容')"
          required
        >
          <Radio.Group
            v-model="formModel.updateContent"
            @change="handleUpdateContentChange"
          >
            <Radio label="both">{{ $t('镜像+配置') }}</Radio>
            <Radio label="config">{{ $t('仅配置') }}</Radio>
          </Radio.Group>
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
          <Form.FormItem
            v-if="imageSource === 'code'"
            :label="$t('代码分支')"
            property="branch"
            required
          >
            <RepoRefSelect
              ref="branchSelectRef"
              v-model="formModel.branch"
              :repository-id="repoAlias"
              :workspace-id="workspaceId"
              @branch-commit="handleBranchSelect"
            />
          </Form.FormItem>
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
        <Alert
          v-if="formModel.updateContent !== 'both'"
          class="mb-[24px]"
          closable
          theme="warning"
        >
          {{ alertContent }}
        </Alert>
        <!-- 目前只能选滚动更新，因此先隐藏更新类型 -->
        <!-- <Form.FormItem
        :label="$t('更新类型')"
        required
      >
        <Radio.Group
          v-model="formModel.deployType"
          class="flex flex-col"
        >
          <Radio label="RollingUpdate">{{ $t('滚动更新') }} ( RollingUpdate )</Radio>
          <Radio
            class="!ml-[0px]"
            :disabled="formModel.updateContent !== 'image'"
            label="InplaceUpdate"
          >
            <span
              v-bk-tooltips="{
                content: $t('更新内容包含配置时，不支持原地更新'),
                placement: 'top',
                disabled: formModel.updateContent === 'image',
              }"
              >{{ $t('原地更新') }} ( InplaceUpdate )</span
            >
            <span class="text-[#bbbdc3]">
              <i class="bkms-icon bkms-icon-circle-info"></i>
              {{ $t('仅更新容器镜像，不重建 Pod，更新速度更快') }}
            </span>
          </Radio>
        </Radio.Group>
      </Form.FormItem> -->
      </Form>
    </template>
    <template v-else>
      <BuildLogPanel
        :active="showBuildLog"
        :build-info="buildInfo"
      />
    </template>

    <div
      v-if="!showBuildLog"
      class="mt-[32px] px-[24px]"
    >
      <Button
        :loading="loading"
        theme="primary"
        @click="handleDeploy"
      >
        {{ $t('部署') }}
      </Button>
      <Button
        class="ml-[8px]"
        :loading="loading"
        @click="handleClose"
      >
        {{ $t('取消') }}
      </Button>
    </div>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, reactive, ref, watch } from 'vue';

  import { Alert, Button, Form, Input, Message, Radio, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { InstanceService } from '~/api/modules/v1';
  import { useAppRepoRefSelect } from '~/composables/use-app-repo-ref-select';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useRecommendTag } from '~/composables/use-recommend-tag';
  import ImageSelect from '~/pages/application/components/image-select.vue';
  import BuildLogPanel from '~/pages/application/detail/components/view-build-log/build-log-panel.vue';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import { type DeployableAppType, type DeployParams, useDeployAPIs } from '../use-deploy';

  import type { AppModelDeployRecordOutputObj } from '~/@types/v1/deploy';
  import type { BuildInfo, BuildStatus } from '~/pages/application/detail/components/view-build-log/type';

  type ImageSourceType = 'code' | 'image';

  const isShow = defineModel<boolean>('isShow');
  const emit = defineEmits<{
    update: [keepOpen: boolean];
  }>();
  const props = defineProps<{
    effectiveReplicas?: number;
    isProdEnv?: boolean;
    latestBuildId?: string;
    latestBuildStatus?: BuildStatus;
  }>();

  const { t } = useI18n();
  const trpcDeployStore = useTrpcDeployStore();
  const appDetailStore = useAppDetail();

  const { workspaceId, repoAlias, branchSelectRef, prepareBranchAfterMount } = useAppRepoRefSelect(
    () => appDetailStore.appDetail?.buildConfig?.repoBuildConfig?.repoAlias || '',
  );

  const imageSource = ref<ImageSourceType>('image');
  const formRef = ref();
  const imageSelectRef = ref();
  const formModel = reactive<{
    branch: string;
    deployType: 'InplaceUpdate' | 'RollingUpdate';
    imageTag: string;
    updateContent: 'both' | 'config' | 'image';
  }>({
    branch: '',
    deployType: 'RollingUpdate',
    imageTag: '',
    updateContent: 'both',
  });
  // 切换为“仅配置”时会保留之前的镜像来源，因此只有“镜像+配置”选择源码时才触发构建。
  const shouldBuildFromSource = computed(() => formModel.updateContent === 'both' && imageSource.value === 'code');
  const { getDefaultBranch, fetchRecommendTag } = useRecommendTag(() => formModel.branch, {
    manualFetchOnly: computed(() => !repoAlias.value),
    onRecommend: tag => {
      if (shouldBuildFromSource.value) {
        formModel.imageTag = tag;
      }
    },
  });
  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formModel);

  const alertContent = computed(() => {
    if (formModel.updateContent === 'config') {
      return t('本次更新仅变更应用的配置信息（包括环境变量等），镜像 Tag 保持不变');
    }
    return t('本次更新仅变更镜像 Tag，应用的配置信息（包括环境变量等）保持不变');
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

  /** 分支变化立即拉取推荐镜像 Tag */
  function handleBranchSelect(value: string) {
    if (value) {
      fetchRecommendTag(value);
    }
  }

  /** 切换镜像来源；源码模式 prepare 默认分支并预拉列表 */
  async function handleChangeImageSource(source: ImageSourceType) {
    imageSource.value = source;
    if (source === 'code') {
      // 推荐 Tag 由 useRecommendTag 监听 formModel.branch 变化自动触发
      await prepareBranchAfterMount(getDefaultBranch());
    } else {
      formModel.branch = '';
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
    try {
      // 根据应用类型获取对应的部署 API
      const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
      // 根据镜像来源选择部署方式
      const params: DeployParams = {
        appID: appDetailStore.appID,
        envName: trpcDeployStore.curEnvItem!.name ?? '',
        imageTag: formModel.imageTag,
        replicas: props.effectiveReplicas ?? trpcDeployStore.deploySpec?.replicas ?? 0,
      };
      if (shouldBuildFromSource.value) {
        params.branch = formModel.branch;
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
          revision: buildRes.revision || formModel.branch,
          status,
        });
      } else {
        // 已构建镜像 和 “仅配置”均直接部署；“仅配置”的镜像 Tag 已回填为当前部署版本。
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

    loading.value = true;
    try {
      let result = false;
      if (formModel.updateContent !== 'image') {
        result = await handleConfigAndImage();
      } else {
        result = await handleImage();
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
          message: t('操作成功'),
        });
      });
    } catch (error) {
      console.warn(error);
    } finally {
      loading.value = false;
    }
  }

  /**
   * 仅镜像
   */
  async function handleImage() {
    try {
      await InstanceService.updateAppInstances({
        appID: appDetailStore.appID,
        envName: trpcDeployStore.curEnvItem?.name ?? '',
        imageTag: formModel.imageTag,
        updateStrategy: formModel.deployType,
        instanceIDs: [],
      });
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

    // 非仅镜像时,重置为滚动更新(原地更新仅支持镜像更新)
    if (val !== 'image') {
      formModel.deployType = 'RollingUpdate';
    }
  }

  watch(isShow, async val => {
    if (!val) {
      withPausedWatch(() => {
        formModel.branch = '';
        formModel.deployType = 'RollingUpdate';
        formModel.updateContent = 'both';
        formModel.imageTag = '';
      });
      imageSource.value = 'image';
      showBuildLog.value = false;
      formRef.value?.clearValidate();
    } else {
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
