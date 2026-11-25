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
    theme="gray"
  >
    <template #loading>
      <div class="flex flex-col gap-[16px]">
        <!-- 环境概览 -->
        <div class="flex items-start p-[24px] bg-[#fff] rounded-[2px] shadow-[0_2px_4px_0_#1919290d]">
          <Layout.shape
            :height="72"
            :width="72"
          />
          <div class="flex-1 min-w-0 ml-[20px]">
            <div class="flex items-center min-h-[24px]">
              <Layout.shape
                :height="20"
                :width="176"
              />
              <Layout.shape
                class="ml-[8px]"
                :height="22"
                :width="48"
              />
              <Layout.shape
                class="ml-auto"
                :height="20"
                :width="52"
              />
            </div>
            <div class="flex flex-wrap gap-x-[40px] gap-y-[12px] mt-[16px]">
              <Layout.formItem
                :item-height="22"
                :item-width="112"
                :label-height="22"
                :label-width="72"
              />
              <Layout.formItem
                :item-height="22"
                :item-width="96"
                :label-height="22"
                :label-width="72"
              />
              <Layout.formItem
                :item-height="22"
                :item-width="160"
                :label-height="22"
                :label-width="72"
              />
            </div>
          </div>
        </div>

        <!-- 集群资源 -->
        <div class="overflow-hidden rounded-[2px] shadow-[0_2px_4px_0_#1919290d]">
          <div class="h-[32px] bg-[#EAEBF0]"></div>
          <div class="overflow-x-auto p-[24px] bg-[#fff]">
            <div class="flex min-w-max items-center gap-[40px]">
              <div class="flex shrink-0 items-center gap-[40px]">
                <Layout.formItem
                  :item-height="22"
                  :item-width="120"
                  :label-height="22"
                  :label-width="60"
                />
                <Layout.formItem
                  :item-height="22"
                  :item-width="148"
                  :label-height="22"
                  :label-width="36"
                />
                <Layout.formItem
                  :item-height="22"
                  :item-width="128"
                  :label-height="22"
                  :label-width="60"
                />
              </div>
              <div
                v-if="featuresStore.enableKubeInsight"
                class="ml-auto flex shrink-0 items-center gap-[16px]"
              >
                <Layout.shape
                  :height="16"
                  :width="80"
                />
                <Layout.shape
                  :height="16"
                  :width="144"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- 集群组件 -->
        <div class="overflow-hidden rounded-[2px] shadow-[0_2px_4px_0_#1919290d]">
          <div class="h-[32px] bg-[#EAEBF0]"></div>
          <div class="min-h-[200px] p-[16px] bg-[#fff]">
            <Layout.shape
              v-for="index in 3"
              :key="index"
              class="!block"
              :class="{ 'mt-[8px]': index > 1 }"
              :height="38"
              width="80%"
            />
          </div>
        </div>

        <div class="flex items-center gap-[12px] mt-[8px]">
          <Layout.shape
            :height="32"
            :width="100"
          />
          <Layout.shape
            :height="22"
            :width="300"
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
                  @click="handleBasicInfoEdit"
                >
                  {{ $t('编辑') }}
                </Button>
              </div>
              <div class="flex flex-wrap items-center gap-x-[40px] gap-y-[12px] mt-[16px]">
                <FieldItem
                  class="!w-auto"
                  :container-height="22"
                  field-direction="left"
                  :field-value="$t('环境名称')"
                  field-width="auto"
                  :value="envData?.name"
                  value-color="#313238"
                />
                <FieldItem
                  class="!w-auto"
                  :container-height="22"
                  field-direction="left"
                  :field-value="$t('创建人')"
                  field-width="auto"
                >
                  <template #value>
                    <OverflowTitle
                      class="max-w-[220px] text-[12px] text-[#313238]"
                      type="tips"
                    >
                      <UserDisplay :value="envData.creator" />
                    </OverflowTitle>
                  </template>
                </FieldItem>
                <FieldItem
                  class="!w-auto"
                  :container-height="22"
                  field-direction="left"
                  :field-value="$t('创建时间')"
                  field-width="auto"
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
              <div class="flex items-center h-[32px] w-[400px] px-[8px] border border-[#c4c6cc] rounded-[2px]">
                <OverflowTitle type="tips">
                  <UserDisplay
                    empty-text=""
                    :value="envData.creator"
                  />
                </OverflowTitle>
              </div>
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
          class="overflow-x-auto p-[24px] bg-[#fff]"
        >
          <div class="flex min-w-max items-center gap-[40px]">
            <div class="flex shrink-0 items-center gap-[40px]">
              <FieldItem
                class="!w-auto shrink-0"
                :container-height="22"
                field-direction="left"
                :field-value="$t('容器项目')"
                field-width="auto"
                :value="envData?.cluster?.projectCode"
                value-color="#313238"
                :value-max-width="220"
              />
              <FieldItem
                class="!w-auto shrink-0"
                :container-height="22"
                field-direction="left"
                :field-value="$t('集群')"
                field-width="auto"
                :value="envData?.cluster?.clusterID"
                value-color="#313238"
                :value-max-width="220"
              />
              <FieldItem
                class="!w-auto shrink-0"
                :container-height="22"
                field-direction="left"
                :field-value="$t('命名空间')"
                field-width="auto"
                :value="envData?.cluster?.namespace"
                value-color="#313238"
                :value-max-width="220"
              />
            </div>
            <div
              v-if="featuresStore.enableKubeInsight && (clusterHealthLoading || clusterHealthCounts)"
              :aria-busy="clusterHealthLoading"
              :aria-label="$t('集群诊断')"
              class="ml-auto flex shrink-0 items-center gap-[16px] whitespace-nowrap text-[12px]"
              role="status"
            >
              <Layout.shape
                v-if="clusterHealthLoading"
                :height="16"
                :width="208"
              />
              <template v-else-if="clusterHealthCounts">
                <span class="text-[#979BA5]">{{ $t('集群诊断') }}</span>
                <template v-if="hasClusterAlerts">
                  <span class="inline-flex items-center gap-[6px] text-[#EA3636]">
                    <span class="w-[6px] h-[6px] rounded-full bg-[#EA3636]"></span>
                    {{ $t('致命') }} {{ clusterHealthCounts.RISK }}
                  </span>
                  <span class="inline-flex items-center gap-[6px] text-[#F59500]">
                    <span class="w-[6px] h-[6px] rounded-full bg-[#F59500]"></span>
                    {{ $t('预警') }} {{ clusterHealthCounts.WARN }}
                  </span>
                </template>
                <span
                  v-else
                  class="inline-flex items-center gap-[6px] text-[#2DCB56]"
                >
                  <span class="w-[6px] h-[6px] rounded-full bg-[#2DCB56]"></span>
                  {{ $t('未发现异常') }}
                </span>
                <Button
                  text
                  theme="primary"
                  @click="router.push({ name: 'clusterHealthDiagnosis', params: { envId, space: workspaceId } })"
                >
                  {{ hasClusterAlerts ? $t('查看详情') : $t('查看报告') }}
                </Button>
              </template>
            </div>
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
                @update:value="handleClusterIDChange"
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
      :dialog-title="$t('无法修改集群信息')"
      :tips-text="$t('该环境已部署应用，请先卸载后再修改集群信息')"
    />
    <DeleteEnvAction
      ref="deleteEnvActionRef"
      @deleted="router.push({ name: 'env', params: { space: workspaceId } })"
    />
  </Skeleton>
</template>
<script lang="ts" setup>
  import { computed, onBeforeUnmount, ref, watch } from 'vue';

  import { Button, DatePicker, Form, Input, Message, OverflowTitle, Select, Tag } from 'bkui-vue';
  import { ExclamationCircleShape } from 'bkui-vue/lib/icon';
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
  import { useEnvDetailStore } from '~/stores/env-detail';
  import { useFrontendFeaturesStore } from '~/stores/frontend-features';
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
  const featuresStore = useFrontendFeaturesStore();
  const envDetailStore = useEnvDetailStore();
  let disposed = false;
  onBeforeUnmount(() => {
    disposed = true;
  });

  const envId = computed(() => props.env || String(route.params.envId || '') || envDetailStore.currentEnv?.id || '');
  const workspaceId = computed(
    () => props.workspace || String(route.params.space || '') || spaceStore.currentSpace || '',
  );

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
  let clusterEditSequence = 0;

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

  function handleClusterIDChange(clusterID: string) {
    const cluster = formData.value.cluster!;
    if (cluster.clusterID === clusterID) return;
    cluster.clusterID = clusterID;
    cluster.namespace = '';
  }

  // 修改集群前重新检查已部署应用，避免使用页面加载时的旧详情放行。
  async function handleClusterResourcesEdit() {
    if (!envId.value) return;
    const current = ++clusterEditSequence;
    const requestedEnvId = envId.value;
    // 环境或空间切换时序号会先递增，卸载时 disposed 为真，迟到响应据此丢弃。
    const isCurrent = () => !disposed && current === clusterEditSequence;
    showDeployedWarning.value = false;
    try {
      const detail = await EnvService.getEnv({ envID: requestedEnvId });
      if (!isCurrent()) return;
      warningDeployedApps.value = detail.appDeployStatuses || [];
      if (warningDeployedApps.value.length) {
        showDeployedWarning.value = true;
        return;
      }

      formData.value.cluster = {
        projectCode: detail.cluster?.projectCode || '',
        clusterID: detail.cluster?.clusterID || '',
        clusterType: detail.cluster?.clusterType || '',
        namespace: detail.cluster?.namespace || '',
      };
      selectedClusterType.value = '';
      clusterResources.value.isEdit = true;
    } catch {
      if (isCurrent()) Message({ message: t('获取环境详情失败，无法验证部署状态'), theme: 'error' });
    }
  }

  function resetClusterResources() {
    Object.assign(clusterResources.value, { isEdit: false, loading: false });
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
  let envDetailSequence = 0;
  async function getEnvDetail() {
    const sequence = ++envDetailSequence;
    const requestedWorkspace = workspaceId.value;
    const requestedEnvId = envId.value;
    if (!requestedWorkspace || !requestedEnvId) {
      isLoading.value = false;
      return;
    }
    isLoading.value = true;
    const newData =
      envDetailStore.currentEnv?.id === requestedEnvId
        ? envDetailStore.currentEnv
        : await envDetailStore.waitForCurrentEnv(requestedEnvId);
    if (
      disposed ||
      sequence !== envDetailSequence ||
      requestedWorkspace !== workspaceId.value ||
      requestedEnvId !== envId.value
    ) {
      return;
    }
    if (!newData) {
      isLoading.value = false;
      return;
    }
    updateViewData(newData);
    isLoading.value = false;

    initCLusterHealth();
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
      if (disposed || savedEnvId !== envId.value || savedWorkspace !== workspaceId.value) return result;
      const savedEnv = envDetailStore.currentEnv;
      if (savedEnv?.id === savedEnvId) updateViewData(savedEnv);
      Message({
        message: t('操作成功'),
        theme: 'success',
        delay: 1500,
      });

      emits('update', envData.value);
      initCLusterHealth();
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
  const clusterHealthLoading = ref(false);
  const clusterHealthCounts = ref<null | { RISK: number; WARN: number }>(null);
  const hasClusterAlerts = computed(
    () => !!clusterHealthCounts.value && clusterHealthCounts.value.RISK + clusterHealthCounts.value.WARN > 0,
  );
  let clusterHealthSequence = 0;

  const { loading, clusterData, getClusterList } = useClusterSelector(projectID.value || '', 'all');

  async function initCLusterHealth() {
    resetClusterHealth();
    if (!featuresStore.enableKubeInsight || !envId.value || !hasClusterConfig.value) return;
    const sequence = clusterHealthSequence;
    const requestedEnvId = envId.value;
    const requestedWorkspace = workspaceId.value;
    const requestedClusterId = envData.value.cluster?.clusterID;
    const isCurrentRequest = () =>
      !disposed &&
      sequence === clusterHealthSequence &&
      requestedEnvId === envId.value &&
      requestedWorkspace === workspaceId.value &&
      requestedClusterId === envData.value.cluster?.clusterID &&
      featuresStore.enableKubeInsight;
    clusterHealthLoading.value = true;
    const report = await BkintegrationsKubeinsightService.getLatestEnvReport(
      { envID: requestedEnvId },
      { interceptorErr: false },
    ).catch(() => null);
    if (!disposed && sequence === clusterHealthSequence) clusterHealthLoading.value = false;
    if (!isCurrentRequest() || !report) return;
    const activeItems = (report.abnormalItems || []).filter(item => !item.recovered && item.level !== 'RECOVERED');
    const counts = countBy(activeItems, 'level');
    clusterHealthCounts.value = { RISK: counts.RISK || 0, WARN: counts.WARN || 0 };
  }

  // 初始化集群列表
  async function initClusters() {
    projectID.value = spaceStore.workspaceDetail?.bkSystems?.bkBCSProjectID || '';
    await getClusterList(projectID.value);
  }

  function resetClusterHealth() {
    ++clusterHealthSequence;
    clusterHealthLoading.value = false;
    clusterHealthCounts.value = null;
  }

  // 初始化
  watch(
    [workspaceId, envId],
    async () => {
      resetClusterHealth();
      ++clusterEditSequence;
      showDeployedWarning.value = false;
      resetClusterResources();
      deleteEnvActionRef.value?.reset();
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
