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
    :full-height="false"
    :loading="isLoading"
  >
    <template #loading>
      <Layout.shape
        :height="120"
        width="100%"
      />
      <Layout.shape
        class="mt-[16px]"
        :height="28"
        width="100%"
      />
      <div class="flex mt-[12px]">
        <Layout.shape
          class="!block"
          :height="136"
          :width="280"
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
    <div class="flex flex-col gap-[16px]">
      <!-- 环境概览 -->
      <div class="bg-[#fff] shadow-[0_2px_4px_0_#1919290d] rounded-[2px]">
        <template v-if="!isBasicInfoEditing">
          <div class="flex items-start p-[24px]">
            <div
              class="relative flex shrink-0 items-center justify-center w-[72px] h-[72px] text-[32px] font-bold text-white rounded-[8px]"
              :style="{ backgroundColor: envAvatarColor }"
            >
              {{ envInitial }}
            </div>
            <div class="min-w-0 flex-1 ml-[20px]">
              <div class="flex items-center min-h-[24px]">
                <span class="text-[16px] leading-[24px] font-bold text-[#313238] truncate">
                  {{ envData?.displayName || envData?.name || '--' }}
                </span>
                <Tag
                  v-if="selectedEnvName"
                  class="ml-[8px] shrink-0"
                  :class="envTypeTagClassMap[envData?.type || '']"
                >
                  {{ selectedEnvName }}
                </Tag>
                <Button
                  class="ml-auto shrink-0"
                  text
                  theme="primary"
                  @click="handleBasicInfoEdit"
                >
                  <EditLine class="mr-[4px]" />
                  {{ $t('编辑') }}
                </Button>
              </div>
              <div class="flex flex-wrap items-center gap-x-[40px] gap-y-[12px] mt-[16px]">
                <FieldItem
                  class="!w-auto"
                  :container-height="22"
                  field-direction="left"
                  :field-value="$t('环境名称')"
                  :field-width="72"
                  :value="envData?.name"
                  value-color="#313238"
                />
                <FieldItem
                  class="!w-auto"
                  :container-height="22"
                  field-direction="left"
                  :field-value="$t('创建人')"
                  :field-width="72"
                  :value="envData.creator"
                  value-color="#313238"
                />
                <FieldItem
                  class="!w-auto"
                  :container-height="22"
                  field-direction="left"
                  :field-value="$t('创建时间')"
                  :field-width="72"
                  :value="formatTimeByTimezone(envData?.createdAt?.toString() || '')"
                  value-color="#313238"
                />
              </div>
            </div>
          </div>
        </template>
        <div
          v-else
          class="p-[24px]"
        >
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
              <template #label>
                <span class="inline-flex items-center">
                  <span class="mr-[4px]">{{ $t('环境分类') }}</span>
                  <EnvCategoryDescription />
                </span>
              </template>
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
            <Form.FormItem :label="$t('创建人')">
              <Input
                class="w-[400px]"
                :model-value="envData.creator"
                readonly
              />
            </Form.FormItem>
            <Form.FormItem :label="$t('创建时间')">
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

      <!-- 集群资源 -->
      <BkmsContent
        class="info-title shadow-[0_2px_4px_0_#1919290d]"
        collapsible
        :editing="clusterResources.isEdit"
        :show-edit-icon="!clusterResources.isEdit"
        :title="$t('集群资源')"
        @edit="handleClusterResourcesEdit"
      >
        <div
          v-show="!clusterResources.isEdit"
          v-bkloading="{
            loading: clusterScoreLoading || isLoading,
            opacity: 1,
            size: 'small',
          }"
          class="flex flex-wrap gap-[32px] items-stretch p-[24px] bg-[#fff]"
        >
          <div class="w-[240px] shrink-0 flex flex-col rounded-[2px] overflow-hidden">
            <div class="flex-1 flex flex-col items-center justify-center p-[12px] bg-[#FAFBFD]">
              <ClusterHealthScore
                :count="{
                  RISK: clusterScore?.RISK,
                  WARN: clusterScore?.WARN,
                }"
                :radius="40"
                :value-size="18"
              />
            </div>
            <div
              class="w-full bg-[#F0F5FF] hover:bg-[#E1ECFF] flex justify-center items-center h-[32px] cursor-pointer transition-colors"
            >
              <Button
                v-bk-tooltips="{
                  content: curDisabledClusterTips,
                  placement: 'bottom',
                  disabled: !!clusterScore,
                }"
                :disabled="clusterScore === null"
                text
                theme="primary"
                @click="$router.push({ name: 'clusterHealthDiagnosis', params: { envId, space: $route.params.space } })"
              >
                {{ $t('集群诊断') }}
                <i class="bkms-icon bkms-icon-arrows-right text-[24px]"></i>
              </Button>
            </div>
          </div>
          <div class="flex-1 grid grid-cols-1 gap-[12px] content-center min-w-0">
            <FieldItem
              :container-height="22"
              field-direction="left"
              :field-value="$t('容器项目')"
              :field-width="88"
              :value="envData?.cluster?.projectCode"
              value-color="#313238"
              :value-max-width="'100%'"
            />
            <FieldItem
              :container-height="22"
              field-direction="left"
              :field-value="$t('集群')"
              :field-width="88"
              :value="envData?.cluster?.clusterID"
              value-color="#313238"
              :value-max-width="'100%'"
            />
            <FieldItem
              :container-height="22"
              field-direction="left"
              :field-value="$t('命名空间')"
              :field-width="88"
              :value="envData?.cluster?.namespace"
              value-color="#313238"
              :value-max-width="'100%'"
            />
          </div>
        </div>
        <div
          v-if="clusterResources.isEdit"
          class="p-[24px] bg-[#fff]"
        >
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
      </BkmsContent>

      <!-- 集群组件 -->
      <BkmsContent
        class="info-title shadow-[0_2px_4px_0_#1919290d]"
        collapsible
        :title="$t('集群组件')"
      >
        <div class="px-[16px] py-[16px] bg-[#fff]">
          <ClusterComponents
            :auto-expand-app-type="focusedAppType"
            :env-id="envId"
            :has-cluster-config="hasClusterConfig"
          />
        </div>
      </BkmsContent>

      <div class="flex items-center gap-[12px] mt-[8px]">
        <Button
          class="shrink-0"
          theme="danger"
          @click="deleteEnvActionRef?.show(envData)"
        >
          {{ $t('删除环境') }}
        </Button>
        <div class="flex items-center gap-[4px] text-[12px] min-w-0">
          <ExclamationCircleShape class="text-[14px] text-[#F59500] shrink-0" />
          <span class="text-[#4D4F56]">
            {{ $t('删除环境后，所有配置、环境变量和操作记录将永久删除') }}
          </span>
        </div>
      </div>
    </div>
    <DeployedAppsWarning
      v-model:is-show="showDeployedWarning"
      :deployed-apps="warningDeployedApps"
      :dialog-title="deployedWarningTitle"
      :tips-text="deployedWarningTips"
    />
    <DeleteEnvAction
      ref="deleteEnvActionRef"
      @deleted="router.push({ name: 'env', params: { space: workspaceId } })"
    />
  </Skeleton>
</template>
<script lang="ts" setup>
  import { computed, onBeforeUnmount, ref, watch } from 'vue';

  import { Button, DatePicker, Form, Input, Message, Select, Tag } from 'bkui-vue';
  import { EditLine, ExclamationCircleShape } from 'bkui-vue/lib/icon';
  import { cloneDeep, countBy } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { EnvAppDeployStatusOutput, EnvDetailOutput, UpdateEnvBasicInfoInput } from '~/@types/v1/env';
  import { BkintegrationsKubeinsightService, EnvService } from '~/api/modules/v1';
  import { BKMS_REGEX } from '~/common/const';
  import { formatTimeByTimezone } from '~/common/util';
  import useClusterSelector from '~/components/cluster-selector/use-cluster-selector';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { envTypeAvatarColorMap, envTypeTagClassMap } from '~/composables/use-env-manager';
  import { useErrorHandler } from '~/composables/use-error-handler';
  import { useEnvDetailStore } from '~/stores/env-detail';
  import { useSpaceStore } from '~/stores/space';

  import ClusterComponents from './cluster-components/cluster-components.vue';
  import DeleteEnvAction from './components/delete-env-action.vue';
  import EnvCategoryDescription from './components/env-category-description.vue';
  import DeployedAppsWarning from './components/project-selector/deployed-apps-warning.vue';
  import ProjectSelector from './components/project-selector/project-selector.vue';

  const props = defineProps<{
    env?: string;
    workspace?: string;
  }>();
  const emits = defineEmits(['update']);
  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const spaceStore = useSpaceStore();
  const envDetailStore = useEnvDetailStore();
  const { handleError } = useErrorHandler();
  let disposed = false;
  onBeforeUnmount(() => {
    disposed = true;
  });

  const routeEnvId = String(route.params.envId || '');
  const routeSpace = String(route.params.space || '');
  const envId = computed(() => props.env || routeEnvId || envDetailStore.currentEnv?.id || '');
  const workspaceId = computed(() => props.workspace || routeSpace || spaceStore.currentSpace || '');

  const envData = ref<EnvDetailOutput>({} as EnvDetailOutput);
  const formattedCreatedAt = computed(() => (envData.value.createdAt ? new Date(envData.value.createdAt) : ''));
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
    { id: 'staging', name: t('预发布') },
    { id: 'production', name: t('生产') },
  ];

  // 环境中文名称
  const selectedEnvName = computed(() => envTypeList.find(item => item.id === envData.value?.type)?.name || '');
  const envAvatarColor = computed(() => envTypeAvatarColorMap[envData.value?.type || ''] || '#3A84FF');
  const envInitial = computed(() => {
    const label = envData.value?.displayName || envData.value?.name || '';
    return label.slice(0, 1) || '-';
  });

  // 基础信息确认
  async function handleBasicInfoConfirm() {
    try {
      await basicInfoFormRef.value?.validate();
      basicInfoLoading.value = true;
      // 更新基础信息
      if (await handleUpdateEnvBasicInfo()) isBasicInfoEditing.value = false;
      basicInfoLoading.value = false;
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
  const deleteEnvActionRef = ref<InstanceType<typeof DeleteEnvAction>>();
  const warningDeployedApps = ref<EnvAppDeployStatusOutput[]>([]);
  const deployedWarningTitle = ref('');
  const deployedWarningTips = ref('');

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
      showDeployedAppsWarning(t('无法修改集群信息'), t('该环境已部署应用，请先卸载后再修改集群信息'));
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
   * 弹出已部署应用警告 Dialog（环境有已部署应用时阻止修改集群信息 / 删除环境）
   */
  function showDeployedAppsWarning(title: string, tips: string) {
    warningDeployedApps.value = deployedApps.value;
    deployedWarningTitle.value = title;
    deployedWarningTips.value = tips;
    showDeployedWarning.value = true;
  }

  const clusterResourcesFormRef = ref<InstanceType<typeof Form>>();
  // 修改集群资源数据
  async function handleClusterConfirm() {
    try {
      await clusterResourcesFormRef.value?.validate();
      clusterResources.value.loading = true;
      if (await handleUpdateEnvCluster()) resetClusterResources();
      clusterResources.value.loading = false;
    } catch {
      clusterResources.value.loading = false;
    }
  }

  // 获取环境详情
  const isLoading = ref<boolean>(false);
  async function getEnvDetail() {
    if (!workspaceId.value || !envId.value) {
      return;
    }
    isLoading.value = true;
    const newData =
      envDetailStore.currentEnv?.id === envId.value
        ? envDetailStore.currentEnv
        : await EnvService.getEnv({ envID: envId.value }).catch(() => null);
    if (disposed || !newData) {
      isLoading.value = false;
      return;
    }
    updateViewData(newData);
    isLoading.value = false;

    if (newData.cluster?.clusterID) {
      initCLusterHealth();
    }
  }

  // 更新环境信息的通用处理函数
  async function handleUpdate(apiCall: () => Promise<unknown>, changes: Partial<EnvDetailOutput>) {
    const savedEnvId = envId.value;
    const savedWorkspace = workspaceId.value;
    const result = await apiCall()
      .then(() => true)
      .catch(() => false);
    if (result) {
      await envDetailStore.syncSavedEnv(savedWorkspace, savedEnvId, changes);
      if (disposed) return result;
      const savedEnv = envDetailStore.currentEnv;
      if (savedEnv?.id === savedEnvId) updateViewData(savedEnv);
      Message({
        message: t('操作成功'),
        theme: 'success',
        delay: 1500,
      });

      emits('update', envData.value);
      if (envData.value.cluster?.clusterID) initCLusterHealth();
    }
    return result;
  }

  // 更新环境基础信息
  async function handleUpdateEnvBasicInfo() {
    const changes = {
      displayName: formData.value.displayName || '',
      type: formData.value.type as UpdateEnvBasicInfoInput['type'],
    };
    return handleUpdate(
      () => EnvService.updateEnvBasicInfo({ envID: envId.value, ...changes }, { irrevocable: true }),
      changes,
    );
  }

  // 更新环境集群资源
  async function handleUpdateEnvCluster() {
    const cluster = {
      ...formData.value.cluster,
      clusterID: formData.value.cluster?.clusterID || '',
      clusterType: selectedClusterType.value,
      namespace: formData.value.cluster?.namespace || '',
    };
    return handleUpdate(
      () =>
        EnvService.updateEnvCluster(
          {
            envID: envId.value,
            clusterID: cluster.clusterID,
            clusterType: cluster.clusterType,
            namespace: cluster.namespace,
          },
          { irrevocable: true },
        ),
      { cluster },
    );
  }

  // 更新视图数据
  function updateViewData(row: EnvDetailOutput) {
    envData.value = row;
    formData.value = cloneDeep(row);
  }

  // 保存期间切走再返回时，接收 Store 的最终结果，同时保留当前尚未提交的编辑内容。
  watch(
    () => envDetailStore.currentEnv,
    data => {
      if (!disposed && data?.id === envId.value) {
        envData.value = data;
        if (!isBasicInfoEditing.value && !clusterResources.value.isEdit) formData.value = cloneDeep(data);
      }
    },
  );

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
        { envID: envId.value },
        { interceptorErr: false, originalResponse: true, needStatus: true },
      );
      const res = await response.json();
      if (disposed) return;
      clusterScore.value = countBy(res.data.abnormalItems, 'level');
    } catch (err: unknown) {
      if (disposed) return;
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
    [workspaceId, envId],
    async () => {
      await getEnvDetail();
      if (!disposed) await initClusters();
    },
    { immediate: true },
  );
</script>

<style lang="postcss" scoped>
  .info-title :deep(.bkms-content-title) {
    background-color: #eaebf0;
  }
</style>
