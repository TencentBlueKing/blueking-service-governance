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
  <Skeleton
    class="bg-[#fff]"
    :loading="isLoading"
  >
    <template #loading>
      <Layout.shape
        :height="28"
        width="100%"
      />
      <Layout.formItem
        class="mt-[12px] ml-[50px]"
        :item-height="18"
        :label-height="18"
      />
      <Layout.formItem
        class="mt-[12px] ml-[50px]"
        :item-height="18"
        :label-height="18"
      />
      <Layout.formItem
        class="mt-[12px] ml-[50px]"
        :item-height="18"
        :label-height="18"
      />
      <Layout.formItem
        class="mt-[12px] ml-[50px]"
        :item-height="18"
        :label-height="18"
      />
      <Layout.formItem
        class="mt-[12px] ml-[50px]"
        :item-height="18"
        :label-height="18"
      />
      <Layout.shape
        class="mt-[24px]"
        :height="28"
        width="100%"
      />
      <div class="flex mt-[12px]">
        <Layout.shape
          class="!block"
          :height="136"
          :width="240"
        />
        <div>
          <Layout.formItem
            class="ml-[30px]"
            :item-height="18"
            :label-height="18"
          />
          <Layout.formItem
            class="mt-[12px] ml-[30px]"
            :item-height="18"
            :label-height="18"
          />
          <Layout.formItem
            class="mt-[12px] ml-[30px]"
            :item-height="18"
            :label-height="18"
          />
        </div>
      </div>
    </template>
    <!-- 基础信息 -->
    <BkmsContent
      collapsible
      :show-edit-icon="!isBasicInfoEditing"
      :title="$t('基本信息')"
      @edit="handleBasicInfoEdit"
    >
      <div class="mt-[16px] text-[12px]">
        <template v-if="!isBasicInfoEditing">
          <DetailItem
            class="items-center"
            :label="$t('环境名称')"
            :label-width="160"
            :value="envData?.name"
          >
          </DetailItem>
          <DetailItem
            class="items-center"
            :label="$t('环境展示名')"
            :label-width="160"
            :value="envData?.displayName"
          >
          </DetailItem>
          <DetailItem
            class="items-center"
            :label="$t('环境分类')"
            :label-width="160"
            :value="selectedEnvName"
          >
          </DetailItem>
          <DetailItem
            class="items-center"
            :label="$t('创建人')"
            :label-width="160"
            :value="envData.creator"
          >
          </DetailItem>
          <DetailItem
            class="items-center"
            :label="$t('创建时间')"
            :label-width="160"
            :value="formatTimeByTimezone(envData?.createdAt?.toString() || '')"
          >
          </DetailItem>
        </template>
        <!-- 编辑态 -->
        <div v-else>
          <Form
            ref="basicInfoFormRef"
            :model="formData"
            :rules="basicInfoRules"
          >
            <Form.FormItem
              :label="$t('环境名称')"
              property="name"
            >
              <Input
                v-model.trim="formData.name"
                class="w-[400px]"
                readonly
              />
            </Form.FormItem>
            <Form.FormItem
              :label="$t('环境展示名')"
              property="displayName"
              required
            >
              <Input
                v-model.trim="formData.displayName"
                class="w-[400px]"
                :placeholder="basicInfoRules.displayName[0].message"
              />
            </Form.FormItem>
            <Form.FormItem
              :label="$t('环境分类')"
              property="type"
              required
            >
              <Select
                v-model="formData.type"
                class="mr-[8px] w-[400px]"
                :filterable="false"
              >
                <Select.Option
                  v-for="item in envTypeList"
                  :key="item.id"
                  :name="item.name"
                  :value="item.id"
                ></Select.Option>
              </Select>
            </Form.FormItem>
            <Form.FormItem
              :label="$t('创建人')"
              property="creator"
            >
              <Input
                v-model.trim="envData.creator"
                class="w-[400px]"
                readonly
              />
            </Form.FormItem>
            <Form.FormItem
              :label="$t('创建时间')"
              property="createdAt"
            >
              <DatePicker
                class="w-[400px]"
                :model-value="formattedCreatedAt"
                readonly
                type="datetime"
              />
            </Form.FormItem>
            <Form.FormItem>
              <Button
                :loading="basicInfoLoading"
                theme="primary"
                @click="handleBasicInfoConfirm"
                >{{ $t('确定') }}</Button
              >
              <Button
                class="ml-[8px]"
                @click="isBasicInfoEditing = false"
                >{{ $t('取消') }}</Button
              >
            </Form.FormItem>
          </Form>
        </div>
      </div>
    </BkmsContent>
    <!-- 集群资源 -->
    <BkmsContent
      class="mt-[24px]"
      collapsible
      :show-edit-icon="!clusterResources.isEdit"
      :title="$t('集群资源')"
      @edit="handleClusterResourcesEdit"
    >
      <div class="mt-[16px] text-[12px] flex">
        <div
          v-show="!clusterResources.isEdit"
          v-bkloading="{
            loading: clusterScoreLoading || isLoading,
            opacity: 1,
            size: 'small',
          }"
        >
          <div class="p-[12px] bg-[#FAFBFD] w-[240px] flex flex-col items-center">
            <ClusterHealthScore
              :count="{
                RISK: clusterScore?.RISK,
                WARN: clusterScore?.WARN,
              }"
              :radius="40"
              :value-size="18"
            />
          </div>
          <div class="w-full bg-[#F0F5FF] flex justify-center items-center h-[32px]">
            <Button
              v-bk-tooltips="{
                content: curDisabledClusterTips,
                placement: 'bottom',
                disabled: clusterScore,
              }"
              :disabled="clusterScore === null"
              text
              theme="primary"
              @click="$router.push({ name: 'clusterHealthDiagnosis', params: { envId: env } })"
            >
              {{ $t('集群诊断') }}
              <i class="bkms-icon bkms-icon-arrows-right text-[24px]"></i>
            </Button>
          </div>
        </div>
        <div v-if="!clusterResources.isEdit">
          <DetailItem
            class="items-center"
            :label="$t('容器项目')"
            :label-width="80"
            :value="envData?.cluster?.projectCode || '--'"
          >
          </DetailItem>
          <DetailItem
            class="items-center"
            :label="$t('集群')"
            :label-width="80"
            :value="envData?.cluster?.clusterID || '--'"
          >
          </DetailItem>
          <DetailItem
            class="items-center"
            :label="$t('命名空间')"
            :label-width="80"
            :value="envData?.cluster?.namespace || '--'"
          >
          </DetailItem>
        </div>
        <!-- 编辑态 -->
        <div v-else>
          <Form
            ref="clusterResourcesFormRef"
            :model="formData"
          >
            <Form.FormItem
              :label="$t('容器项目')"
              property="cluster.projectCode"
              required
            >
              <ProjectSelector
                class="w-[400px]"
                disabled
                :project-code="formData.cluster?.projectCode || ''"
                @change="(val: string) => (formData.cluster!.projectCode = val)"
              />
            </Form.FormItem>
            <Form.FormItem
              :label="$t('集群')"
              property="cluster.clusterID"
              required
            >
              <ClusterSelector
                class="w-[400px]"
                :list="clusterData"
                :loading="loading"
                :project-code="formData.cluster?.projectCode"
                :value="formData.cluster?.clusterID || ''"
                @update:cluster-type="val => (selectedClusterType = val)"
                @update:value="val => (formData.cluster!.clusterID = val)"
              />
            </Form.FormItem>
            <Form.FormItem
              :label="$t('命名空间')"
              property="cluster.namespace"
              required
            >
              <!-- projectID 容器项目ID -->
              <NamespaceSelector
                class="flex-1 w-[400px]"
                :cluster-id="formData.cluster?.clusterID || ''"
                :project-i-d="projectID"
                :value="formData.cluster?.namespace || ''"
                @update:value="val => (formData.cluster!.namespace = val)"
              />
            </Form.FormItem>
            <Form.FormItem>
              <Button
                :loading="clusterResources.loading"
                theme="primary"
                @click="handleClusterConfirm"
                >{{ $t('确定') }}</Button
              >
              <Button
                class="ml-[8px]"
                @click="clusterResources.isEdit = false"
                >{{ $t('取消') }}</Button
              >
            </Form.FormItem>
          </Form>
        </div>
      </div>
    </BkmsContent>
    <!-- 集群组件 -->
    <BkmsContent
      class="mt-[24px]"
      collapsible
      :title="$t('集群组件')"
    >
      <ClusterComponents
        :auto-expand-app-type="focusedAppType"
        :env-id="env"
        :has-cluster-config="hasClusterConfig"
      />
    </BkmsContent>
    <DeployedAppsWarning
      v-model:is-show="showDeployedWarning"
      :deployed-apps="deployedApps"
      :dialog-title="$t('无法修改集群信息')"
      :tips-text="$t('该环境已部署应用，请先卸载后再修改集群信息')"
    />
  </Skeleton>
</template>
<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Button, DatePicker, Form, Input, Message, Select } from 'bkui-vue';
  import { cloneDeep, countBy } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { useRoute } from 'vue-router';
  import { EnvDetailOutput, UpdateEnvBasicInfoInput } from '~/@types/v1/env';
  import { BkintegrationsKubeinsightService, EnvService } from '~/api/modules/v1';
  import { BKMS_REGEX } from '~/common/const';
  import { formatTimeByTimezone } from '~/common/util';
  import useClusterSelector from '~/components/cluster-selector/use-cluster-selector';
  import Layout from '~/components/skeleton/skeleton-layout';
  import useEnvManager from '~/composables/use-env-manager';
  import { useErrorHandler } from '~/composables/use-error-handler';
  import { useSpaceStore } from '~/stores/space';

  import ClusterComponents from './cluster-components/cluster-components.vue';
  import DeployedAppsWarning from './components/project-selector/deployed-apps-warning.vue';
  import ProjectSelector from './components/project-selector/project-selector.vue';

  const props = defineProps<{
    env: string;
    workspace: string;
  }>();
  const emits = defineEmits(['update']);
  const { t } = useI18n();
  const route = useRoute();
  const spaceStore = useSpaceStore();
  const { handleError } = useErrorHandler();

  const envData = ref<EnvDetailOutput>({} as EnvDetailOutput);
  const formData = ref<EnvDetailOutput>({} as EnvDetailOutput);
  const hasClusterConfig = computed(() => !!envData.value?.cluster?.clusterID);

  // 基础信息-相关逻辑
  const basicInfoFormRef = ref<InstanceType<typeof Form> | null>(null);
  // 编辑状态与加载状态
  const isBasicInfoEditing = ref(false);
  const basicInfoLoading = ref(false);
  const basicInfoRules = {
    displayName: [
      {
        validator: () => BKMS_REGEX.envDisplayNameRegex.test(formData.value.displayName || ''),
        message: t('请输入1-32字符的环境名称'),
        trigger: 'blur',
      },
    ],
    type: [{ required: true, message: t('环境分类不能为空'), trigger: 'change' }],
  };

  // 环境分类列表
  const envTypeList = [
    { id: 'development', name: t('开发') },
    { id: 'test', name: t('测试') },
    { id: 'production', name: t('生产') },
  ];

  // 环境中文名称
  const selectedEnvName = computed(() => envTypeList.find(item => item.id === envData.value?.type)?.name || '');

  // 处理创建时间的时区转换，用于 DatePicker 显示
  const formattedCreatedAt = computed(() => {
    if (!envData.value?.createdAt) return '';
    // 直接返回原始 Date 对象，DatePicker 会自动按本地时区显示
    return new Date(envData.value.createdAt);
  });

  // 基础信息确认
  async function handleBasicInfoConfirm() {
    try {
      await basicInfoFormRef.value?.validate();
      basicInfoLoading.value = true;
      // 更新基础信息
      await handleUpdateEnvBasicInfo();
      basicInfoLoading.value = false;
      isBasicInfoEditing.value = false;
    } catch {
      basicInfoLoading.value = false;
    }
  }

  // 基础信息编辑模式
  function handleBasicInfoEdit() {
    formData.value.displayName = envData.value.displayName;
    formData.value.type = envData.value.type;
    isBasicInfoEditing.value = true;
  }

  // 集群资源
  const clusterResources = ref({
    isEdit: false,
    loading: false,
  });
  const showDeployedWarning = ref(false);

  // clusterType 不从详情取，完全来自选择器根据集群列表查出
  const selectedClusterType = ref('');

  // 已部署应用列表（来自环境详情 appDeployStatuses）
  const deployedApps = computed(() => envData.value?.appDeployStatuses || []);
  const focusedAppType = computed(() => {
    const appID = (route.query.appID || '') as string;
    const appType = (route.query.appType || '') as string;
    const targetApp = appID ? deployedApps.value.find(app => app.appID === appID) : null;
    return targetApp?.appType || appType;
  });

  // 集群资源编辑
  function handleClusterResourcesEdit() {
    if (deployedApps.value.length > 0) {
      showDeployedAppsWarning();
      return;
    }

    formData.value.cluster = {
      projectCode: envData.value.cluster?.projectCode || '',
      clusterID: envData.value.cluster?.clusterID || '',
      clusterType: envData.value.cluster?.clusterType || '',
      namespace: envData.value.cluster?.namespace || '',
    };
    selectedClusterType.value = '';
    clusterResources.value.isEdit = true;
  }

  function resetClusterResources() {
    Object.assign(clusterResources.value, { isEdit: false, loading: false });
  }

  /**
   * 弹出已部署应用警告 Dialog（环境有已部署应用时阻止修改集群信息）
   */
  function showDeployedAppsWarning() {
    showDeployedWarning.value = true;
  }

  const clusterResourcesFormRef = ref<InstanceType<typeof Form>>();
  // 修改集群资源数据
  async function handleClusterConfirm() {
    try {
      await clusterResourcesFormRef.value?.validate();
      clusterResources.value.loading = true;
      await handleUpdateEnvCluster();
      resetClusterResources();
    } catch {
      clusterResources.value.loading = false;
    }
  }

  // 获取环境详情
  const isLoading = ref<boolean>(false);
  const { handleGetEnvDetail } = useEnvManager();
  async function getEnvDetail(showLoading = true) {
    if (!props.workspace || !props.env) {
      return;
    }
    if (showLoading) {
      isLoading.value = true;
    }
    const newData = (await handleGetEnvDetail(props.env)) || ({} as EnvDetailOutput);
    updateViewData(newData);
    isLoading.value = false;

    if (newData.cluster?.clusterID) {
      initCLusterHealth();
    }
  }

  // 更新环境信息的通用处理函数
  async function handleUpdate(apiCall: () => Promise<unknown>) {
    const result = await apiCall()
      .then(() => true)
      .catch(() => false);
    if (result) {
      Message({
        message: t('操作成功'),
        theme: 'success',
        delay: 1500,
      });

      await getEnvDetail(false);
      emits('update', envData.value);
    }
    return result;
  }

  // 更新环境基础信息
  async function handleUpdateEnvBasicInfo() {
    return handleUpdate(() =>
      EnvService.updateEnvBasicInfo({
        envID: props.env,
        displayName: formData.value.displayName || '',
        type: formData.value.type as UpdateEnvBasicInfoInput['type'],
      }),
    );
  }

  // 更新环境集群资源
  async function handleUpdateEnvCluster() {
    return handleUpdate(() =>
      EnvService.updateEnvCluster({
        envID: props.env,
        clusterID: formData.value.cluster?.clusterID || '',
        clusterType: selectedClusterType.value,
        namespace: formData.value.cluster?.namespace || '',
      }),
    );
  }

  // 更新视图数据
  function updateViewData(row: EnvDetailOutput) {
    envData.value = row;
    formData.value = cloneDeep(row);
  }

  const projectID = ref<string>('');
  const clusterScoreLoading = ref(false);
  const clusterScore = ref<null | {
    RISK?: number;
    WARN?: number;
  }>(null);
  const clusterHealthTipsMap = {
    notFount: t('集群未正确安装 kubeinsight 组件，无法查看健康诊断结果'),
    notCluster: t('无集群资源信息'),
  };
  const curDisabledClusterTips = computed(() => {
    if (clusterScore.value === null) {
      return envData?.value?.cluster?.clusterID ? clusterHealthTipsMap.notFount : clusterHealthTipsMap.notCluster;
    }
    return '';
  });

  const { loading, clusterData, getClusterList } = useClusterSelector(projectID.value || '', 'all');

  async function initCLusterHealth() {
    clusterScoreLoading.value = true;
    try {
      const response: Response = await BkintegrationsKubeinsightService.getLatestEnvReport(
        { envID: props.env },
        { interceptorErr: false, originalResponse: true, needStatus: true },
      );
      const res = await response.json();
      clusterScore.value = countBy(res.data.abnormalItems, 'level');
    } catch (err: unknown) {
      const errorInfo = err as { error?: { message?: string }; status: number };
      const { error, status } = errorInfo;
      if (status !== 404) {
        handleError(error ?? {}, 500, {
          theme: 'error',
          message: {
            code: status,
            overview: error?.message || t('请求异常'),
            suggestion: '',
            type: 'json',
            details: `${JSON.stringify(error || {}, null, 2)}`,
          },
        });
      }
    } finally {
      clusterScoreLoading.value = false;
    }
  }

  // 初始化集群列表
  async function initClusters() {
    projectID.value = spaceStore.workspaceDetail?.bkSystems?.bkBCSProjectID || '';
    await getClusterList(projectID.value);
  }

  // 初始化
  watch(
    [() => props.workspace, () => props.env],
    async () => {
      await getEnvDetail();
      await initClusters();
    },
    { immediate: true },
  );
</script>
