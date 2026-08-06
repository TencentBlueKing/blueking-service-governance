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
  <Button
    class="ml-[6px] bg-[#fff]"
    :disabled="!!loading"
    :loading="isInitLoading || isSubmitLoading || !!loading"
    @click="handleOpen"
  >
    {{ $t('扩缩容') }}
    <!-- 自动扩缩容 Tag 提示 -->
    <AutoScaleTag
      class="ml-[6px] cursor-pointer"
      :enabled="initialEnabled"
      :status="gpaStatus"
      @click.stop="handleScaleTagClick"
    />
  </Button>

  <Sideslider
    v-model:is-show="isShow"
    :before-close="handleBeforeClose"
    render-directive="if"
    width="1040"
    @closed="handleClosed"
  >
    <template #header>
      <DividerHeader>
        <template #title>
          <span class="text-[16px]">{{ $t('扩缩容配置') }}</span>
        </template>
        <div>
          <span v-if="trpcDeployStore.curEnvItem?.displayName">
            {{ `${$t('环境')}: ${trpcDeployStore.curEnvItem.displayName}` }}
          </span>
          （{{ `${$t('当前实例数')}: ${currentReplicasText}` }}）
        </div>
      </DividerHeader>
    </template>

    <Loading :loading="isInitLoading">
      <Form
        ref="formRef"
        class="px-[24px] pt-[18px]"
        form-type="vertical"
        :model="formModel"
      >
        <Form.FormItem
          :label="$t('扩缩容方式')"
          required
        >
          <Button.ButtonGroup class="flex items-center">
            <Button
              class="flex-1"
              :selected="formModel.mode === 'manual'"
              @click="handleChangeMode('manual')"
            >
              {{ $t('手动调节') }}
            </Button>
            <Button
              class="flex-1"
              :selected="formModel.mode === 'auto'"
              @click="handleChangeMode('auto')"
            >
              {{ $t('自动调节') }}
            </Button>
          </Button.ButtonGroup>
        </Form.FormItem>

        <template v-if="formModel.mode === 'manual'">
          <Form.FormItem
            :label="$t('实例数')"
            property="replicas"
            required
            :rules="replicasRules"
          >
            <Input
              v-model.number="formModel.replicas"
              class="w-[50%]"
              :min="0"
              :precision="0"
              type="number"
            />
          </Form.FormItem>
        </template>

        <template v-else>
          <div class="grid grid-cols-2 gap-x-[24px]">
            <Form.FormItem
              :label="$t('最小实例数')"
              property="minReplicas"
              required
              :rules="minReplicasRules"
            >
              <Input
                v-model.number="formModel.minReplicas"
                :min="1"
                :precision="0"
                type="number"
              />
            </Form.FormItem>
            <Form.FormItem
              :label="$t('最大实例数')"
              property="maxReplicas"
              required
              :rules="maxReplicasRules"
            >
              <Input
                v-model.number="formModel.maxReplicas"
                :min="1"
                :precision="0"
                type="number"
              />
            </Form.FormItem>
          </div>

          <Form.FormItem
            class="!mb-0"
            :label="$t('触发条件')"
            required
          >
            <div class="mb-[12px]">
              <Radio.Group
                v-model="formModel.computeByLimits"
                class="flex items-center"
              >
                <Radio :label="false">{{ $t('按 Requests 计算') }}</Radio>
                <Radio :label="true">{{ $t('按 Limits 计算') }}</Radio>
              </Radio.Group>
              <div class="text-[12px] leading-[20px] text-[#979BA5]">
                {{ computeByLimitsTip }}
              </div>
            </div>
            <div class="flex flex-col gap-[12px]">
              <div
                v-for="(metric, index) in formModel.metrics"
                :key="metric.id"
                class="grid grid-cols-[1fr_32px_1fr_52px] items-center"
              >
                <Form.FormItem
                  class="mb-0"
                  error-display-type="tooltips"
                  label=""
                  :property="`metrics.${index}.resource`"
                  :rules="metricResourceRules"
                >
                  <Select
                    v-model="metric.resource"
                    class="w-full"
                    :clearable="false"
                  >
                    <Select.Option
                      v-for="item in metricOptions"
                      :key="item.value"
                      :disabled="isMetricSelected(item.value, metric.id)"
                      :label="item.label"
                      :value="item.value"
                    />
                  </Select>
                </Form.FormItem>
                <div class="text-align-center text-[#63656E]">=</div>
                <Form.FormItem
                  class="mb-0"
                  error-display-type="tooltips"
                  label=""
                  :property="`metrics.${index}.averageUtilization`"
                  :rules="metricValueRules"
                >
                  <Input
                    v-model.number="metric.averageUtilization"
                    :max="100"
                    :min="1"
                    :precision="0"
                    suffix="%"
                    type="number"
                  />
                </Form.FormItem>
                <div class="ml-[12px] flex items-center gap-[8px] h-[32px]">
                  <i
                    v-if="canAddMetric"
                    class="bkms-icon bkms-icon-plus-circle-shape cursor-pointer text-[#979BA5] hover:text-[#4D4F56]"
                    @click="handleAddMetric"
                  ></i>
                  <i
                    v-else
                    v-bk-tooltips="$t('最多配置 2 项')"
                    class="bkms-icon bkms-icon-plus-circle-shape cursor-not-allowed text-[#DCDEE5]"
                  ></i>
                  <i
                    v-if="formModel.metrics.length > 1"
                    class="bkms-icon bkms-icon-minus-circle-shape cursor-pointer text-[#979BA5] hover:text-[#4D4F56]"
                    @click="handleRemoveMetric(metric.id)"
                  ></i>
                  <i
                    v-else
                    v-bk-tooltips="$t('至少保留一个')"
                    class="bkms-icon bkms-icon-minus-circle-shape cursor-not-allowed text-[#DCDEE5]"
                  ></i>
                </div>
              </div>
            </div>
          </Form.FormItem>

          <Alert
            class="mt-[12px]"
            closable
          >
            {{ $t('容忍度机制：使用率在 0.9~1.1 之间时不改变实例数，避免指标微小波动造成频繁伸缩') }}
            <Button
              class="ml-[6px]"
              text
              theme="primary"
              @click="handleViewDoc"
              >{{ $t('查看详情') }}
            </Button>
          </Alert>

          <!-- 高级设置 -->
          <Form.FormItem
            class="!mb-0"
            property="timeRanges"
            :rules="timeRangesRules"
          >
            <ScaleAdvancedSetting
              v-model="formModel.timeRanges"
              v-model:enabled="formModel.timeRangeEnabled"
              :max-replicas="Number(formModel.maxReplicas)"
              :min-replicas="Number(formModel.minReplicas)"
            />
          </Form.FormItem>
        </template>
      </Form>
    </Loading>

    <template #footer>
      <Button
        class="mr-[10px]"
        :loading="isSubmitLoading"
        theme="primary"
        @click="handleSubmit"
      >
        {{ $t('确定') }}
      </Button>
      <Button
        :loading="isSubmitLoading"
        @click="handleClose"
      >
        {{ $t('取消') }}
      </Button>
    </template>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, h, nextTick, onMounted, reactive, ref, watch } from 'vue';

  import { Alert, Button, Form, InfoBox, Input, Loading, Message, Radio, Select, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { GpaService, InstanceService } from '~/api/modules/v1';
  import { DOC_LINKS } from '~/common/const';
  import { hasErrorCode, showApiErrorMessage } from '~/common/util';
  import DividerHeader from '~/components/divider-header.vue';
  import { useGPAConfigPolling } from '~/composables/use-gpa-config-polling';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import AutoScaleTag from '~/pages/application/detail/components/auto-scale-tag.vue';
  import { useAppDetail } from '~/stores/app-detail';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import ScaleAdvancedSetting from './components/scale-advanced-setting.vue';

  import type { GPAConfigOutputObj, GPAMetricInput, GPATimeRangeInput } from '~/@types/v1/gpa';

  interface MetricFormItem {
    averageUtilization: number;
    id: string;
    resource: MetricResource;
  }
  type MetricResource = GPAMetricInput['resource'];

  type ScaleMode = 'auto' | 'manual';

  const GPA_STATUS_POLL_INTERVAL = 5000;
  const MAX_METRIC_COUNT = 2;
  const DEFAULT_METRIC_VALUE = 80;

  const emit = defineEmits(['update']);
  const props = defineProps<{
    effectiveReplicas?: number;
    loading?: boolean;
  }>();

  const trpcDeployStore = useTrpcDeployStore();
  const appDetailStore = useAppDetail();
  const { t } = useI18n();
  const router = useRouter();

  const isShow = ref(false);
  const isInitLoading = ref(false);
  const isSubmitLoading = ref(false);
  const formRef = ref<InstanceType<typeof Form>>();
  const hasGPAConfig = ref(false);
  const initialEnabled = ref(false);
  const initialMode = ref<ScaleMode>('manual');
  const headerCurrentReplicas = ref<number | string>('--');

  const formModel = reactive<{
    computeByLimits: boolean;
    maxReplicas: number;
    metrics: MetricFormItem[];
    minReplicas: number;
    mode: ScaleMode;
    replicas: number;
    timeRangeEnabled: boolean;
    timeRanges: GPATimeRangeInput[];
  }>({
    mode: 'manual',
    replicas: 0,
    computeByLimits: false,
    minReplicas: 1,
    maxReplicas: 1,
    metrics: [],
    timeRangeEnabled: false,
    timeRanges: [],
  });

  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formModel);
  const {
    config: gpaConfig,
    refresh: requestGPAConfig,
    startPolling: startGPAStatusPolling,
    status: gpaStatus,
    stopPolling: stopGPAStatusPolling,
  } = useGPAConfigPolling({
    appID: () => appDetailStore.appID,
    envName: () => trpcDeployStore.curEnvItem?.name,
    pollInterval: GPA_STATUS_POLL_INTERVAL,
  });
  const metricOptions: Array<{ label: string; value: MetricResource }> = [
    { label: 'CPU 使用率', value: 'cpu' },
    { label: '内存使用率', value: 'memory' },
  ];

  const canAddMetric = computed(() => formModel.metrics.length < MAX_METRIC_COUNT);

  const computeByLimitsTip = computed(() =>
    formModel.computeByLimits
      ? t('使用率 = 实际用量 ÷ Limits。仅在明确需要按上限衡量负载时选用。')
      : t('使用率 = 实际用量 ÷ Requests。默认算法，适合大多数场景'),
  );

  // Header 展示打开侧边栏时的当前实例数快照，避免表单模式切换导致展示值跳变。
  const currentReplicasText = computed(() => {
    if (isInitLoading.value) return '--';
    return headerCurrentReplicas.value;
  });

  const replicasRules = [
    {
      validator: (value: number | string) => isValidInteger(value) && Number(value) >= 0,
      message: t('请输入大于或等于 0 的整数'),
      trigger: 'blur',
    },
  ];

  const minReplicasRules = [
    {
      validator: (value: number | string) => isValidInteger(value) && Number(value) >= 1,
      message: t('请输入大于或等于 1 的整数'),
      trigger: 'blur',
    },
  ];

  const maxReplicasRules = [
    {
      validator: (value: number | string) =>
        isValidInteger(value) && Number(value) >= 1 && Number(value) >= Number(formModel.minReplicas),
      message: t('最大实例数必须大于或等于最小实例数'),
      trigger: 'blur',
    },
  ];

  const metricResourceRules = [
    {
      validator: (value: MetricResource) => !!value,
      message: t('请选择指标'),
      trigger: 'change',
    },
    {
      validator: (value: MetricResource) => formModel.metrics.filter(metric => metric.resource === value).length <= 1,
      message: t('指标不能重复'),
      trigger: 'change',
    },
  ];

  const metricValueRules = [
    {
      validator: (value: number | string) => isValidInteger(value) && Number(value) >= 1 && Number(value) <= 100,
      message: t('请输入 1-100 的整数'),
      trigger: 'blur',
    },
  ];

  const timeRangesRules = [
    {
      validator: (value: GPATimeRangeInput[]) => !formModel.timeRangeEnabled || value.length > 0,
      message: t('请至少新增一条定时扩缩容策略'),
      trigger: 'change',
    },
    {
      validator: (value: GPATimeRangeInput[]) =>
        !formModel.timeRangeEnabled ||
        value.every(
          item =>
            Number(item.desiredReplicas) >= Number(formModel.minReplicas) &&
            Number(item.desiredReplicas) <= Number(formModel.maxReplicas),
        ),
      message: t('定时扩缩容策略的期望实例数必须在最小实例数和最大实例数之间'),
      trigger: 'change',
    },
  ];

  // 创建一条触发指标表单行，未指定指标时选择当前未使用的指标。
  function createMetric(resource?: MetricResource, averageUtilization = DEFAULT_METRIC_VALUE): MetricFormItem {
    return {
      id: generateId(),
      resource: resource ?? getAvailableMetricResource(),
      averageUtilization,
    };
  }

  // 拉取当前环境 GPA 配置，并根据结果初始化侧边栏表单。
  async function fetchGPAConfig() {
    isInitLoading.value = true;
    try {
      const res = await requestGPAConfig();
      initForm(res);
      updateGPAStatusPolling(res);
    } finally {
      isInitLoading.value = false;
      forceCleanDirtyTag(() => formRef.value?.clearValidate?.());
    }
  }

  // 生成指标行唯一 ID，用于 Vue 列表渲染和删除定位。
  function generateId() {
    return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
  }

  // 获取当前尚未被选择的指标类型，作为新增指标行的默认值。
  function getAvailableMetricResource(): MetricResource {
    return (
      metricOptions.find(option => !formModel.metrics.some(metric => metric.resource === option.value))?.value ?? 'cpu'
    );
  }

  // 获取当前生效实例数，优先使用环境生效配置，其次使用部署规格。
  function getCurrentReplicas() {
    return props.effectiveReplicas ?? trpcDeployStore.deploySpec?.replicas ?? 1;
  }

  // 获取自动调节默认实例数，保证符合 GPA 最小实例数必须 >= 1 的约束。
  function getDefaultAutoReplicas() {
    return Math.max(1, getCurrentReplicas());
  }

  // 新增一条触发条件指标，最多允许 cpu / memory 两条。
  function handleAddMetric() {
    if (!canAddMetric.value) return;
    formModel.metrics.push(createMetric());
  }

  // 保存自动调节配置，提交 GPA 最小/最大实例数和指标阈值。
  async function handleAutoSubmit() {
    const shouldConfirmSwitch = initialMode.value !== 'auto';
    const shouldEnableGPA = hasGPAConfig.value && !initialEnabled.value;
    if (shouldConfirmSwitch && !(await showToggleConfirm('auto'))) {
      return false;
    }

    try {
      await GpaService.upsertAppEnvGPAConfig(
        {
          appID: appDetailStore.appID,
          envName: trpcDeployStore.curEnvItem?.name ?? '',
          computeByLimits: formModel.computeByLimits,
          minReplicas: Number(formModel.minReplicas),
          maxReplicas: Number(formModel.maxReplicas),
          metrics: formModel.metrics.map(metric => ({
            resource: metric.resource,
            averageUtilization: Number(metric.averageUtilization),
          })),
          timeRanges: formModel.timeRangeEnabled ? formModel.timeRanges : [],
        },
        { interceptorErr: false, needStatus: true },
      );
      if (shouldEnableGPA) {
        await toggleGPAConfig(true);
      }
      hasGPAConfig.value = true;
      initialEnabled.value = true;
      initialMode.value = 'auto';
      startGPAStatusPolling();
      return true;
    } catch (error) {
      if (hasErrorCode(error, 'COMPONENT_NOT_INSTALLED')) {
        showGPAComponentMissingTips();
        return false;
      }
      showApiErrorMessage(error);
      return false;
    }
  }

  // 侧边栏关闭前检查表单是否有未保存修改。
  function handleBeforeClose() {
    return confirmBox();
  }

  // 切换扩缩容方式，并在进入自动调节时补齐默认指标行。
  function handleChangeMode(mode: ScaleMode) {
    formModel.mode = mode;
    if (mode === 'auto' && formModel.metrics.length === 0) {
      formModel.metrics.push(createMetric());
    }
    updateGPAStatusPolling();
    nextTick(() => formRef.value?.clearValidate?.());
  }

  // 处理取消按钮关闭，关闭前复用离开确认。
  async function handleClose() {
    if (await handleBeforeClose()) {
      isShow.value = false;
    }
  }

  // 侧边栏完全关闭后清理表单校验状态。
  function handleClosed() {
    updateGPAStatusPolling();
    formRef.value?.clearValidate?.();
  }

  // 保存手动调节实例数，沿用原有实例扩缩容接口。
  async function handleManualSubmit() {
    const shouldDisableGPA = hasGPAConfig.value && initialEnabled.value;
    if (shouldDisableGPA && !(await showToggleConfirm('manual'))) {
      return false;
    }

    try {
      await InstanceService.scaleAppInstances({
        appID: appDetailStore.appID,
        envName: trpcDeployStore.curEnvItem?.name ?? '',
        targetReplicas: Number(formModel.replicas),
      });
      if (shouldDisableGPA) {
        await toggleGPAConfig(false).catch(error => {
          showApiErrorMessage(error);
          throw error;
        });
        initialEnabled.value = false;
        initialMode.value = 'manual';
        stopGPAStatusPolling();
      }
      return true;
    } catch (error) {
      console.warn(error);
      return false;
    }
  }

  // 打开侧边栏并加载当前环境扩缩容配置。
  async function handleOpen() {
    isShow.value = true;
    await fetchGPAConfig();
  }

  // 删除指定触发条件指标，至少保留一条。
  function handleRemoveMetric(id: string) {
    if (formModel.metrics.length <= 1) return;
    const index = formModel.metrics.findIndex(metric => metric.id === id);
    if (index > -1) {
      formModel.metrics.splice(index, 1);
    }
  }

  // 处理按钮内 Tag 点击，Tag 组件不会稳定冒泡到外层 Button，需要显式复用打开逻辑。
  function handleScaleTagClick() {
    if (isInitLoading.value || isSubmitLoading.value || props.loading) return;
    handleOpen();
  }

  // 提交当前模式表单，并在成功后关闭侧边栏、通知父组件刷新。
  async function handleSubmit() {
    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid) return;

    isSubmitLoading.value = true;
    const result = formModel.mode === 'manual' ? await handleManualSubmit() : await handleAutoSubmit();
    isSubmitLoading.value = false;
    if (!result) return;

    forceCleanDirtyTag(() => {
      Message({
        theme: 'success',
        message: t('操作成功'),
      });
      isShow.value = false;
      emit('update');
    });
  }

  // 查看扩缩容稳定性文档
  function handleViewDoc() {
    window.open(window.BK_DOC_URL + DOC_LINKS.SCALE_STABILITY, '_blank');
  }

  // 根据接口返回配置初始化手动或自动调节表单。
  function initForm(config: GPAConfigOutputObj | null) {
    withPausedWatch(() => {
      const currentReplicas = getCurrentReplicas();
      // 后端当前只支持 cpu / memory，且 averageUtilization 必须是 number 类型，过滤掉未知指标，避免提交生成类型以外的数据。
      const validMetrics = (config?.metrics || []).filter(
        (metric): metric is Required<Pick<GPAMetricInput, 'averageUtilization' | 'resource'>> =>
          (metric.resource === 'cpu' || metric.resource === 'memory') && typeof metric.averageUtilization === 'number',
      );

      syncGPAStatus(config);
      headerCurrentReplicas.value = initialEnabled.value
        ? (config?.status?.currentReplicas ?? currentReplicas)
        : currentReplicas;
      formModel.replicas = currentReplicas;
      formModel.computeByLimits = config?.computeByLimits ?? false;

      if (hasGPAConfig.value) {
        formModel.mode = initialMode.value;
        formModel.minReplicas = config?.minReplicas ?? getDefaultAutoReplicas();
        formModel.maxReplicas = config?.maxReplicas ?? formModel.minReplicas;
        formModel.metrics =
          validMetrics.length > 0
            ? validMetrics
                .slice(0, MAX_METRIC_COUNT)
                .map(metric => createMetric(metric.resource, metric.averageUtilization))
            : [createMetric('cpu')];
        formModel.timeRanges = normalizeTimeRanges(config?.timeRanges);
        formModel.timeRangeEnabled = formModel.timeRanges.length > 0;
      } else {
        formModel.mode = 'manual';
        initialEnabled.value = false;
        initialMode.value = 'manual';
        formModel.minReplicas = getDefaultAutoReplicas();
        formModel.maxReplicas = getDefaultAutoReplicas();
        formModel.metrics = [createMetric('cpu')];
        formModel.timeRangeEnabled = false;
        formModel.timeRanges = [];
      }
    });
  }

  // 判断某指标是否已被其他指标行选择，用于禁用重复选项。
  function isMetricSelected(resource: MetricResource, currentId: string) {
    return formModel.metrics.some(metric => metric.id !== currentId && metric.resource === resource);
  }

  function isTimeRangesValid() {
    if (!formModel.timeRangeEnabled) return true;
    return (
      formModel.timeRanges.length > 0 &&
      formModel.timeRanges.every(
        item =>
          Number(item.desiredReplicas) >= Number(formModel.minReplicas) &&
          Number(item.desiredReplicas) <= Number(formModel.maxReplicas),
      )
    );
  }

  // 判断输入值是否为整数，兼容 number input 返回的字符串或数字。
  function isValidInteger(value: number | string | undefined) {
    if (value === '' || value === undefined || value === null) return false;
    return Number.isInteger(Number(value));
  }

  // 过滤接口返回的无效定时策略，保证提交时只保留本版本支持的字段。
  function normalizeTimeRanges(timeRanges: GPAConfigOutputObj['timeRanges']): GPATimeRangeInput[] {
    return (timeRanges || []).filter(
      (item): item is GPATimeRangeInput =>
        typeof item.desiredReplicas === 'number' && typeof item.schedule === 'string' && !!item.schedule,
    );
  }

  // 刷新按钮展示所需的 GPA 启用状态；失败时只记录日志，避免阻断部署页渲染。
  async function refreshGPAStatus() {
    try {
      const config = await requestGPAConfig();
      initForm(config);
      updateGPAStatusPolling(config);
      forceCleanDirtyTag(() => formRef.value?.clearValidate?.());
    } catch (error) {
      console.warn(error);
    }
  }

  // GPA 组件缺失提示
  function showGPAComponentMissingTips() {
    InfoBox({
      type: 'warning',
      title: t('无法启用自动扩缩容'),
      content: h('div', { class: 'bg-[#F5F7FA] px-[16px] py-[12px] text-left leading-[24px] text-[#4D4F56]' }, [
        t('当前环境对应集群未安装 General-Pod-Autoscaler 组件。请先在'),
        h('span', { class: 'px-[4px] text-[#3A84FF]' }, t('环境管理 / 集群组件')),
        t('页面安装组件后，再配置自动扩缩容。'),
      ]),
      headerAlign: 'center',
      contentAlign: 'left',
      footerAlign: 'center',
      confirmText: t('前往安装组件'),
      cancelText: t('关闭'),
      onConfirm: () => {
        isShow.value = false;
        router.push({
          name: 'env',
          query: {
            active: trpcDeployStore.curEnvItem?.name,
            activeTab: 'basicInfo',
            appID: appDetailStore.appID,
            appType: appDetailStore.appType,
          },
        });
      },
    });
  }

  // 弹出 GPA 启用状态切换二次确认，避免用户误关或误开自动扩缩容。
  function showToggleConfirm(targetMode: ScaleMode) {
    const isManual = targetMode === 'manual';
    return new Promise<boolean>(resolve => {
      InfoBox({
        title: isManual ? t('确认切换为手动调节') : t('确认切换为自动调节'),
        content: h('div', { class: 'bg-[#F5F7FA] px-[16px] py-[12px] text-left leading-[22px]' }, [
          h('div', { class: 'mb-[4px]' }, [isManual ? t('切换为手动调节后：') : t('切换为自动调节后：')]),
          h('ul', { class: 'list-disc pl-[18px]' }, [
            h('li', [
              isManual
                ? t('系统将停止自动扩缩容，不再根据负载自动增减实例')
                : t('系统将开启自动扩缩容，根据触发条件自动增减实例'),
            ]),
            h('li', [isManual ? t('当前自动调节配置会保留，后续可随时重新开启') : t('实例数将由自动调节策略接管')]),
          ]),
        ]),
        headerAlign: 'center',
        footerAlign: 'center',
        contentAlign: 'left',
        confirmText: isManual ? t('确认切换为手动') : t('确认切换为自动'),
        cancelText: t('取消'),
        onCancel: () => resolve(false),
        onConfirm: () => resolve(true),
      });
    });
  }

  // 同步 GPA 启用状态和运行状态；轮询场景不能重置表单配置项。
  function syncGPAStatus(config: GPAConfigOutputObj | null) {
    hasGPAConfig.value = !!config;
    initialEnabled.value = !!config?.enabled;
    initialMode.value = initialEnabled.value ? 'auto' : 'manual';
  }

  // 调用 GPA 开关接口，只改变已有 scaler 的启用状态，不删除持久化配置。
  function toggleGPAConfig(enabled: boolean) {
    return GpaService.toggleAppEnvGPAConfig(
      {
        appID: appDetailStore.appID,
        envName: trpcDeployStore.curEnvItem?.name ?? '',
        enabled,
      },
      { interceptorErr: false, needStatus: true },
    );
  }

  // 当前环境处于已启用的自动调节模式时，保持 5s 静默刷新 GPA 状态。
  function updateGPAStatusPolling(config?: GPAConfigOutputObj | null) {
    const enabled = config === undefined ? initialEnabled.value : !!config?.enabled;
    if (enabled && formModel.mode === 'auto') {
      startGPAStatusPolling();
      return;
    }
    stopGPAStatusPolling();
  }

  // 当前环境变化时重新获取 GPA 状态，避免跨环境残留自动扩缩容提示。
  watch(
    () => trpcDeployStore.curEnvItem?.name,
    () => {
      stopGPAStatusPolling();
      if (!isShow.value) {
        refreshGPAStatus();
      }
    },
  );

  watch(gpaConfig, config => {
    syncGPAStatus(config);
    updateGPAStatusPolling(config);
  });

  watch(
    () => [formModel.timeRangeEnabled, formModel.timeRanges, formModel.minReplicas, formModel.maxReplicas],
    () => {
      if (isTimeRangesValid()) {
        nextTick(() => formRef.value?.clearValidate?.('timeRanges'));
      }
    },
    { deep: true },
  );

  // 组件挂载后静默刷新 GPA 状态，用于在按钮上展示自动扩缩容提示。
  onMounted(() => {
    refreshGPAStatus();
  });
</script>
