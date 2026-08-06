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
    :quick-close="false"
    :width="curView === 'config' ? 640 : 1300"
    @closed="handleClose"
  >
    <template #header>
      <DividerHeader :title="textMap[deployType].title">
        <span v-if="appDetailStore.app">{{ appDetailStore.app }}</span>
        <span v-if="envItem?.name">{{ `${$t('环境')}: ${envItem?.displayName}` }}</span>
      </DividerHeader>
    </template>
    <template v-if="curView === 'config'">
      <Form
        ref="formRef"
        v-bkloading="{
          loading: previewLoading,
          opacity: 0.5,
          mode: 'spin',
          theme: 'primary',
        }"
        class="p-[24px] pb-0"
        form-type="vertical"
        :model="formModel"
      >
        <Form.FormItem
          :label="$t('Chart 版本')"
          property="chartVersion"
          required
        >
          <Select
            v-model="formModel.chartVersion"
            :clearable="false"
            display-key="name"
            filterable
            id-key="name"
            :list="chartList"
          >
          </Select>
        </Form.FormItem>
        <Form.FormItem
          :label="$t('Values 文件')"
          property="valuesFileID"
          required
        >
          <Select
            v-model="formModel.valuesFileID"
            :clearable="false"
            display-key="name"
            filterable
            id-key="id"
            :list="valuesFileState.list"
            :loading="valuesFileState.loading"
          >
          </Select>
        </Form.FormItem>
        <Form.FormItem
          :label="$t('镜像 Tag')"
          property="imageTag"
          required
        >
          <Select
            v-model="formModel.imageTag"
            :clearable="false"
            display-key="tag"
            filterable
            id-key="tag"
            :list="imageTagState.list"
            :no-data-text="$t('暂无可用镜像')"
            :remote-method="handleImageSearch"
            :scroll-loading="imageTagState.loading"
            @scroll-end="handleScrollEnd"
          >
            <template #extension>
              <div class="w-full flex items-center justify-center">
                <template v-if="isProductionEnv">
                  <span class="text-[#63656e]">{{ $t('生产类型环境仅支持部署已晋级的 Tag') }}</span>
                  <Button
                    class="ml-[8px]"
                    text
                    theme="primary"
                    @click="handleGotoArtifact"
                  >
                    <Share class="mr-[4px]" />
                    {{ $t('去晋级') }}
                  </Button>
                </template>
                <Button
                  v-else
                  text
                  theme="primary"
                  @click="handleGotoBuild"
                >
                  <Share class="mr-[4px]" />
                  {{ $t('去构建') }}
                </Button>
              </div>
            </template>
          </Select>
        </Form.FormItem>
      </Form>
    </template>
    <div
      v-else
      class="h-[calc(100vh_-_100px)]"
    >
      <MsEditor
        :is-diff="true"
        lang="yaml"
        :model-value="previewData.target"
        :original="previewData.current"
        :readonly="true"
        :target-title="$t('新版本')"
        :title="$t('当前版本')"
      />
    </div>
    <!-- 环境变量未定义弹窗 -->
    <EnvUndefinedTips
      v-model:is-show="isShowEnvUndefinedTips"
      :missing-env-vars="missingEnvVars"
      :missing-vars="missingVars"
      @cancel="handleCancelEnvUndefinedTips"
      @go-modify="handleGoEnvVars"
      @still-deploy="handleStillDeploy"
    />

    <template #footer>
      <Button
        v-if="curView === 'config'"
        class="mr-[10px]"
        :loading="previewLoading"
        theme="primary"
        @click="handleChangeStep('next')"
      >
        {{ textMap[deployType].nextBtn }}
      </Button>
      <Button
        v-if="curView === 'preview'"
        class="mr-[10px]"
        @click="handleChangeStep('prev')"
      >
        {{ $t('上一步') }}
      </Button>
      <Button
        v-if="curView === 'preview'"
        class="mr-[10px]"
        :loading="confirmLoading"
        theme="primary"
        @click="handleConfirm"
      >
        {{ textMap[deployType].deployBtn }}
      </Button>
      <Button @click="handleClose">
        {{ $t('取消') }}
      </Button>
    </template>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, nextTick, ref, watch } from 'vue';

  import { Button, Form, Message, Select, Sideslider } from 'bkui-vue';
  import { Share } from 'bkui-vue/lib/icon';
  import { cloneDeep, debounce } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { AppConfigFileOutputObj, ListAppConfigFilesOutput } from '~/@types/v1/app-config-files';
  import { CreateHelmDeployRequest, PreviewHelmDeployOutput, PreviewHelmDeployRequest } from '~/@types/v1/deploy';
  import { EnvOutput } from '~/@types/v1/env';
  import { DeployableImageTagOutputObj } from '~/@types/v1/images';
  import { AppConfigFilesService, DeployService, ImagesService } from '~/api/modules/v1';
  import { useErrorHandler } from '~/composables/use-error-handler';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useAppDetail } from '~/stores/app-detail';

  import EnvUndefinedTips from './env-undefined-tips.vue';
  import { useHelmDeploy } from './use-helm-deploy';
  type HelmDeployType = 'Recreate' | 'RollingUpdate';
  interface IProps {
    deployType: HelmDeployType;
    envItem?: EnvOutput;
    laneName?: string;
  }

  const isShow = defineModel<boolean>('isShow');
  const props = defineProps<IProps>();
  const emit = defineEmits(['close', 'deploy']);

  const appDetailStore = useAppDetail();
  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const { chartList, deployHistoryList } = useHelmDeploy();

  interface ApiErrorResponse {
    error: Record<string, unknown>;
    status: number;
  }

  const textMap: Record<
    HelmDeployType,
    {
      deployBtn: string;
      nextBtn: string;
      title: string;
    }
  > = {
    RollingUpdate: {
      title: t('部署应用'),
      deployBtn: t('确定部署'),
      nextBtn: t('下一步：{0}', [t('预览部署')]),
    },
    Recreate: {
      title: t('更新应用'),
      deployBtn: t('确定更新'),
      nextBtn: t('下一步：{0}', [t('预览更新')]),
    },
  };

  const curView = ref<'config' | 'preview'>('config');
  const previewLoading = ref(false);
  const confirmLoading = ref(false);

  /** 环境变量未定义弹窗状态 */
  const isShowEnvUndefinedTips = ref(false);
  /** env 命名空间未定义变量 */
  const missingEnvVars = ref<string[]>([]);
  /** 非 env 命名空间未定义变量 */
  const missingVars = ref<string[]>([]);
  /** 保存预览数据，在"仍然部署"时使用 */
  let pendingPreviewData: null | PreviewHelmDeployOutput = null;

  const formRef = ref();
  /** 首次部署 form */
  const formModel = ref<{
    chartVersion: string;
    envName: string;
    imageTag: string;
    valuesFileID: string;
  }>({
    chartVersion: '',
    envName: '',
    imageTag: '',
    valuesFileID: '',
  });
  // 使用 useLeaveConfirm hook 管理表单变化检测
  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formModel);

  /** 预览数据 */
  const previewData = ref<PreviewHelmDeployOutput>({
    current: '',
    target: '',
  });
  const valuesFileState = ref({
    list: [] as AppConfigFileOutputObj[],
    loading: false,
  });

  const imageTagState = ref({
    list: [] as DeployableImageTagOutputObj[],
    loading: false,
    page: 1,
    hasMore: true,
    imageKeyword: '',
  });

  // 预览参数
  const deployOrPreviewParams = computed(() => {
    const params = {
      appID: appDetailStore.appID,
      envName: props.envItem?.name,
      chartVersion: formModel.value.chartVersion,
      imageTag: formModel.value.imageTag,
      trafficLaneName: props.laneName,
      valuesFileID: formModel.value.valuesFileID,
    };
    return params;
  });

  // 获取可部署镜像 Tag 列表
  async function getListAppImages(page = 1) {
    imageTagState.value.loading = true;
    const res = await ImagesService.listDeployableImageTags({
      appID: appDetailStore.appID,
      envName: props.envItem?.name ?? '',
      page,
      pageSize: 10,
      keyword: imageTagState.value.imageKeyword,
    }).finally(() => (imageTagState.value.loading = false));
    const { results = [] } = res;
    imageTagState.value.list = page === 1 ? results : [...imageTagState.value.list, ...results];
    imageTagState.value.hasMore = results.length === 10;
    imageTagState.value.page = page;
  }

  // 获取 Values 文件列表
  async function getValuesFileList() {
    try {
      valuesFileState.value.loading = true;
      const ret: ListAppConfigFilesOutput = await AppConfigFilesService.listAppConfigFiles(
        { appID: appDetailStore.appID },
        { needRes: true },
      );
      valuesFileState.value.list = ret?.items || [];
    } finally {
      valuesFileState.value.loading = false;
    }
  }

  function handleBeforeClose() {
    return confirmBox();
  }

  // 取消环境变量未定义弹窗
  function handleCancelEnvUndefinedTips() {
    missingEnvVars.value = [];
    missingVars.value = [];
    pendingPreviewData = null;
  }

  async function handleChangeStep(step: 'next' | 'prev') {
    if (step === 'next') {
      await handleGetPreview();
    } else {
      curView.value = 'config';
    }
  }

  async function handleClose() {
    if (await handleBeforeClose()) {
      isShow.value = false;
      emit('close');
    }
  }

  async function handleConfirm() {
    try {
      confirmLoading.value = true;
      const params = cloneDeep(deployOrPreviewParams.value) as CreateHelmDeployRequest;
      await DeployService.createHelmDeploy(params, { interceptorErr: false, originalResponse: true });
      forceCleanDirtyTag(() => {
        emit('deploy');
        handleResetForm();
        Message({
          theme: 'success',
          message: t('部署成功'),
        });
        isShow.value = false;
      });
    } catch (err: unknown) {
      const errorResponse = err as ApiErrorResponse;
      const { handleError } = useErrorHandler();
      // 自定义409错误处理
      handleError(errorResponse.error ?? errorResponse, 409, {
        theme: 'error',
        message: t('部署失败，已有应用部署任务进行中'),
      });
    } finally {
      confirmLoading.value = false;
    }
  }

  // 预览
  async function handleGetPreview() {
    try {
      const valid = await formRef.value.validate().catch(() => false);
      if (!valid) return;
      previewLoading.value = true;
      const params = cloneDeep(deployOrPreviewParams.value) as PreviewHelmDeployRequest;
      const res: PreviewHelmDeployOutput = await DeployService.previewHelmDeploy(params, { needRes: true });
      const envVars = res.missingEnvVars ?? [];
      const otherVars = res.missingVars ?? [];
      // 存在未定义变量时展示预检弹窗
      if (envVars.length > 0 || otherVars.length > 0) {
        missingEnvVars.value = envVars;
        missingVars.value = otherVars;
        pendingPreviewData = res;
        isShowEnvUndefinedTips.value = true;
      } else {
        Object.assign(previewData.value, res);
        curView.value = 'preview';
      }
    } catch (err) {
      console.error(err);
    } finally {
      previewLoading.value = false;
    }
  }

  // 前往环境管理-指定环境的环境配置页面 或者 应用编排页面
  function handleGoEnvVars(source: string = 'env') {
    let resolved = null;
    if (source === 'env') {
      resolved = router.resolve({
        name: 'env',
        params: { space: route.params.space },
        query: {
          active: props.envItem?.name,
          activeTab: 'setting',
        },
      });
    } else {
      resolved = router.resolve({
        name: 'detail',
        params: { ...route.params, menuName: 'orchestrate' },
      });
    }

    window.open(resolved.href, '_blank');
  }

  // 仍然部署：使用保存的预览数据进入预览页面
  function handleStillDeploy() {
    if (pendingPreviewData) {
      Object.assign(previewData.value, pendingPreviewData);
      pendingPreviewData = null;
      curView.value = 'preview';
    }
  }

  const isProductionEnv = computed(() => props.envItem?.type === 'production');

  // 跳转到制品管理页面
  function handleGotoArtifact() {
    router.push({
      name: 'detail',
      params: {
        ...route.params,
        menuName: 'artifact',
      },
    });
  }

  // 跳转到构建管理页面
  function handleGotoBuild() {
    router.push({
      name: 'detail',
      params: {
        ...route.params,
        menuName: 'build',
      },
    });
  }

  // 镜像搜索
  const handleImageSearch = debounce((keyword: string) => {
    imageTagState.value.imageKeyword = keyword;
    imageTagState.value.hasMore = false;
    getListAppImages();
  }, 300);

  // 重置表单
  async function handleResetForm() {
    formModel.value = {
      chartVersion: '',
      envName: '',
      imageTag: '',
      valuesFileID: '',
    };
    await nextTick();
    formRef.value?.clearValidate?.();
  }

  // 滚动底部加载更多
  function handleScrollEnd() {
    if (imageTagState.value.hasMore && !imageTagState.value.loading) {
      getListAppImages(imageTagState.value.page + 1);
    }
  }

  watch(isShow, async newVal => {
    if (newVal) {
      await Promise.all([getValuesFileList(), getListAppImages()]);
      // 记忆最近一次部署的参数
      const latestDeployHistory = deployHistoryList.value?.[0];
      formModel.value.valuesFileID = latestDeployHistory?.valuesFileID ?? '';
      formModel.value.chartVersion = latestDeployHistory?.chartVersion ?? '';
    } else {
      withPausedWatch(() => {
        handleResetForm();
        curView.value = 'config';
      });
    }
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-sideslider-header .bk-sideslider-close) {
    height: inherit;
  }
</style>
