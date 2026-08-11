<template>
  <Sideslider
    v-model:is-show="visible"
    :before-close="handleBeforeClose"
    :width="640"
    @closed="handleClosed"
  >
    <template #header>
      <span class="text-[16px] font-[600]">
        {{ isEdit ? $t('编辑策略') : $t('新建策略') }}
      </span>
    </template>
    <Form
      ref="formRef"
      class="px-[24px] py-[16px]"
      form-type="vertical"
      :model="formModel"
      :rules="formRules"
    >
      <!-- 模块1：监控数据 -->
      <ToggleCard
        class="mb-[16px]"
        :name="$t('监控数据')"
        type="normal"
      >
        <Form.FormItem
          :label="$t('策略名称')"
          property="displayName"
          required
        >
          <Input
            v-model.trim="formModel.displayName"
            clearable
            :maxlength="32"
            :placeholder="$t('请输入 1-32 字符的策略名称')"
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('监控指标')"
          property="strategyCode"
          required
        >
          <Select
            v-model="formModel.strategyCode"
            :placeholder="$t('请选择监控指标')"
          >
            <Select.Option
              v-for="item in strategyCodeOptions"
              :key="item.strategyCode"
              :name="item.name"
              :value="item.strategyCode"
            />
          </Select>
        </Form.FormItem>
      </ToggleCard>

      <!-- 模块2：检测规则 -->
      <ToggleCard
        class="mb-[16px]"
        :name="$t('检测规则')"
        type="normal"
      >
        <Form.FormItem
          :label="$t('告警级别')"
          property="severity"
        >
          <template #label>
            <div class="flex items-center gap-[8px]">
              <span>
                {{ $t('告警级别') }}
                <span class="text-[#EA3636]">*</span>
              </span>
              <Select
                v-model="formModel.severity"
                class="flex-1"
              >
                <template #trigger="{ selected }">
                  <div
                    class="simplicity-trigger flex items-center gap-[8px] h-[32px] px-[8px] border-b border-[#C4C6CC] bg-transparent transition-colors cursor-pointer hover:bg-[#F5F7FA]"
                  >
                    <div class="flex-1 min-w-0 truncate">
                      <SeverityLabel
                        v-if="selected.length"
                        :severity="selected[0].value"
                      />
                      <span
                        v-else
                        class="text-[#c4c6cc] text-[12px]!"
                      >
                        {{ $t('请选择告警级别') }}
                      </span>
                    </div>
                    <AngleDown
                      class="simplicity-trigger-arrow shrink-0 text-[#979BA5] transition-transform"
                      :height="20"
                      :width="20"
                    />
                  </div>
                </template>
                <Select.Option
                  v-for="item in severityOptions"
                  :key="item.id"
                  :name="item.name"
                  :value="item.id"
                >
                  <SeverityLabel :severity="item.id" />
                </Select.Option>
              </Select>
            </div>
          </template>
        </Form.FormItem>
        <Form.FormItem
          :label="$t('告警条件')"
          property="thresholdValue"
        >
          <template #label>
            <div class="flex items-center gap-[8px]">
              <span>
                {{ $t('告警条件') }}
                <span class="text-[#EA3636]">*</span>
              </span>
              <div class="flex-1 flex items-center gap-[8px] text-[#63656E]">
                <i18n-t :keypath="thresholdKeypath">
                  <Select
                    v-model="formModel.thresholdMethod"
                    class="w-[150px]"
                    :clearable="false"
                  >
                    <Select.Option
                      v-for="item in methodOptions"
                      :key="item.id"
                      :name="item.name"
                      :value="item.id"
                    />
                  </Select>
                  <Input
                    v-model="formModel.thresholdValue"
                    :max="100"
                    :min="0"
                    placeholder="0-100"
                    style="width: 120px"
                    type="number"
                  />
                </i18n-t>
              </div>
            </div>
          </template>
        </Form.FormItem>
      </ToggleCard>

      <!-- 模块3：判断条件 -->
      <ToggleCard
        class="mb-[16px]"
        :name="$t('判断条件')"
        type="normal"
      >
        <!-- 触发条件 -->
        <Form.FormItem :label="$t('触发条件')">
          <template #label>
            <div class="flex items-center gap-[8px]">
              <span>
                {{ $t('触发条件') }}
                <span class="text-[#EA3636]">*</span>
              </span>
              <div class="flex-1 flex items-center gap-[8px] flex-wrap text-[#63656E]">
                <i18n-t keypath="在 {0} 个周期内累计满足 {1} 次检测条件时，触发告警通知">
                  <Input
                    v-model="formModel.triggerCheckWindow"
                    behavior="simplicity"
                    :min="1"
                    style="width: 80px"
                    type="number"
                  />
                  <Input
                    v-model="formModel.triggerCount"
                    behavior="simplicity"
                    :min="1"
                    style="width: 80px"
                    type="number"
                  />
                </i18n-t>
              </div>
            </div>
          </template>
        </Form.FormItem>
        <!-- 恢复条件 -->
        <Form.FormItem :label="$t('恢复条件')">
          <template #label>
            <div class="flex items-center gap-[8px]">
              <span>
                {{ $t('恢复条件') }}
                <span class="text-[#EA3636]">*</span>
              </span>
              <div class="flex-1 flex items-center gap-[8px] flex-wrap text-[#63656E]">
                <i18n-t keypath="连续 {0} 个周期内不满足触发条件">
                  <Input
                    v-model="formModel.recoverCheckWindow"
                    behavior="simplicity"
                    :min="1"
                    style="width: 80px"
                    type="number"
                  />
                </i18n-t>
              </div>
            </div>
          </template>
        </Form.FormItem>
        <!-- 生效时间段 -->
        <Form.FormItem :label="$t('生效时间段')">
          <template #label>
            <div class="flex items-center gap-[8px]">
              <span class="shrink-0">
                {{ $t('生效时间段') }}
                <span class="text-[#EA3636]">*</span>
              </span>
              <TimePicker
                v-model="formModel.effectiveTimeRange"
                append-to-body
                class="grow-0 w-auto!"
                clearable
                format="HH:mm:ss"
                :placeholder="$t('开始时间 - 结束时间')"
                style="width: 100%"
                :type="'timerange'"
              />
            </div>
          </template>
        </Form.FormItem>
      </ToggleCard>

      <!-- 模块4：生效环境 -->
      <ToggleCard
        class="mb-[16px]"
        :name="$t('生效环境')"
        type="normal"
      >
        <Form.FormItem
          :label="$t('可用环境')"
          property="scopeType"
          required
        >
          <Radio.Group
            v-model="formModel.scopeType"
            class="flex flex-col"
          >
            <Radio label="all"
              ><span class="text-[14px]">{{ $t('所有环境') }}</span>
              <span class="text-[#979BA5] text-[14px]">{{ $t('( 所有环境都生效，包括新增的环境也会生效 )') }}</span>
            </Radio>
            <Radio
              class="ml-0!"
              label="env_type"
              ><span class="text-[14px]">{{ $t('按环境类型') }}</span>
              <span class="text-[#979BA5] text-[14px]">{{
                $t('( 针对环境类型生效，包括新增的环境也会生效 )')
              }}</span></Radio
            >
            <!-- 按环境类型：多选 -->
            <div
              v-if="formModel.scopeType === 'env_type'"
              class="flex items-center px-[12px] bg-[#F5F7FA] rounded-[2px]"
            >
              <Checkbox.Group v-model="formModel.envTypes">
                <Checkbox
                  v-for="envType in envTypeOptions"
                  :key="envType.id"
                  :label="envType.id"
                >
                  <span class="text-[12px]">{{ envType.name }}</span>
                </Checkbox>
              </Checkbox.Group>
            </div>
            <Radio
              class="ml-0!"
              label="specific_envs"
              ><span class="text-[14px]">{{ $t('特定环境') }}</span>
              <span class="text-[#979BA5] text-[14px]">{{
                $t('( 针对指定的环境生效，新增同类环境不会自动生效 )')
              }}</span></Radio
            >
            <!-- 特定环境：按环境类型分列多选 -->
            <div
              v-show="formModel.scopeType === 'specific_envs'"
              class="pt-[12px]"
            >
              <EnvGroupSelect
                v-model="formModel.envIDs"
                :disabled-statuses="['NotReady']"
                :env-list="envList"
                show-deploy-icon
                value-key="id"
              />
            </div>
          </Radio.Group>
        </Form.FormItem>
      </ToggleCard>

      <!-- 模块5：通知设置 -->
      <ToggleCard
        :name="$t('通知设置')"
        type="normal"
      >
        <Form.FormItem
          :label="$t('告警组')"
          property="noticeGroupIDs"
        >
          <template #label>
            <div class="flex items-start gap-[8px]">
              <span>
                {{ $t('告警组') }}
                <span class="text-[#EA3636]">*</span>
              </span>
              <div class="flex-1 flex flex-col gap-[8px]">
                <div class="flex items-center gap-[8px] flex-wrap min-h-[20px]">
                  <Tag
                    v-for="id in formModel.noticeGroupIDs"
                    :key="id"
                    closable
                    @close="handleRemoveNoticeGroup(id)"
                  >
                    <i class="bkms-icon bkms-icon-usergroup text-[#979BA5] text-[14px] mr-[2px]"></i>
                    {{ getNoticeGroupName(id) }}
                  </Tag>
                  <Button
                    text
                    theme="primary"
                    @click="isNoticeDialogShow = true"
                  >
                    <i class="bkms-icon bkms-icon-jiahao mr-[2px] text-[16px]"></i>
                    <span>{{ $t('添加告警组') }}</span>
                  </Button>
                </div>
                <span class="text-[#979BA5] text-[12px]">{{
                  $t('通知渠道在告警组内配置，选择告警组即关联其通知方式。')
                }}</span>
              </div>
            </div>
          </template>
        </Form.FormItem>
      </ToggleCard>
    </Form>

    <template #footer>
      <div class="flex items-center gap-[8px]">
        <Button
          :loading="submitting"
          theme="primary"
          @click="handleSubmit"
        >
          {{ $t('保存') }}
        </Button>
        <Button @click="handleCancel">
          {{ $t('取消') }}
        </Button>
      </div>
    </template>

    <!-- 告警组选择弹窗 -->
    <AlertGroupSelectDialog
      v-model:is-show="isNoticeDialogShow"
      :selected-ids="formModel.noticeGroupIDs"
      :user-groups="userGroups"
      @confirm="handleNoticeGroupConfirm"
    />
  </Sideslider>
</template>

<script lang="ts">
  import { i18n } from '~/modules/i18n';

  const t = i18n.global.t.bind(i18n.global);

  /**
   * 监控指标固定枚举
   * - strategyCode：策略码，作为 Select 唯一值（同一 MonitorMetric 可能对应多个策略码）
   * - monitorMetric：实际 Prometheus 指标，对应后端 MonitorMetric 字段
   * - name：展示名
   */
  export const strategyCodeOptions = [
    {
      strategyCode: 'cpu_request_usage_high',
      monitorMetric: 'container_cpu_usage_seconds_total',
      name: t('CPU Requests 利用率'),
    },
    {
      strategyCode: 'cpu_limit_usage_high',
      monitorMetric: 'container_cpu_usage_seconds_total',
      name: t('CPU Limits 利用率'),
    },
    {
      strategyCode: 'memory_request_usage_high',
      monitorMetric: 'container_memory_working_set_bytes',
      name: t('内存 Requests 利用率'),
    },
    {
      strategyCode: 'memory_limit_usage_high',
      monitorMetric: 'container_memory_working_set_bytes',
      name: t('内存 Limits 利用率'),
    },
    {
      strategyCode: 'pod_restart_frequent',
      monitorMetric: 'kube_pod_container_status_restarts_total',
      name: t('容器重启次数'),
    },
  ];
</script>

<script lang="ts" setup>
  import { computed, reactive, ref, watch } from 'vue';

  import { Button, Checkbox, Form, Input, Message, Radio, Select, Sideslider, Tag, TimePicker } from 'bkui-vue';
  import { AngleDown } from 'bkui-vue/lib/icon';
  import dayjs from 'dayjs';
  import { useI18n } from 'vue-i18n';
  import { BkintegrationsBkmonitorService } from '~/api/modules/v1/bkintegrations-bkmonitor';
  import { BKMS_REGEX, COUNT_UNIT_STRATEGY_CODES } from '~/common/const';
  import ToggleCard from '~/components/toggle-card.vue';
  import useEnvManager from '~/composables/use-env-manager';
  import { useFocusOnErrorField } from '~/composables/use-focus-on-error-field';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';

  import EnvGroupSelect from '~/components/env-group-selector.vue';
  import AlertGroupSelectDialog from './alert-group-select-dialog.vue';
  import SeverityLabel from './severity-label.vue';

  import type { AlertStrategyOutput, EffectiveScopeInput, UserGroup } from '~/@types/v1/bkintegrations-bkmonitor';

  interface Emits {
    (e: 'update:isShow', value: boolean): void;
    (e: 'success'): void;
  }

  interface Props {
    data?: AlertStrategyOutput;
    isShow: boolean;
    /** 表单模式：create 新建 / edit 编辑 */
    mode?: 'create' | 'edit';
    userGroups: UserGroup[];
  }

  const props = defineProps<Props>();
  const emit = defineEmits<Emits>();
  const { t } = useI18n();

  const appDetailStore = useAppDetail();
  const spaceStore = useSpaceStore();
  const { envList, getAppEnvList } = useEnvManager();
  const { focusOnErrorField } = useFocusOnErrorField();

  const visible = computed({
    get: () => props.isShow,
    set: (val: boolean) => emit('update:isShow', val),
  });

  const isEdit = computed(() => !!props.data?.id);

  /** 告警级别枚举 */
  const severityOptions = [
    { id: 1, name: t('致命') },
    { id: 2, name: t('预警') },
    { id: 3, name: t('提醒') },
  ];

  /** 阈值比较方法枚举 */
  const methodOptions = [
    { id: 'gte', name: t('>=') },
    { id: 'gt', name: t('>') },
    { id: 'lte', name: t('<= ') },
    { id: 'lt', name: t('<') },
    { id: 'eq', name: t('=') },
    { id: 'neq', name: t('≠') },
  ];

  /** 环境类型枚举 */
  const envTypeOptions = [
    { id: 'development', name: t('开发') },
    { id: 'test', name: t('测试') },
    { id: 'staging', name: t('预发布') },
    { id: 'production', name: t('生产') },
  ];

  interface FormModel {
    displayName: string;
    effectiveTimeRange?: [string, string];
    envIDs: string[];
    envTypes: string[];
    monitorMetric: string;
    noticeGroupIDs: number[];
    recoverCheckWindow: '' | number;
    scopeType: 'all' | 'env_type' | 'specific_envs';
    severity: 1 | 2 | 3 | '';
    strategyCode: string;
    thresholdMethod: string;
    thresholdValue: '' | number;
    triggerCheckWindow: '' | number;
    triggerCount: '' | number;
  }

  function createDefaultForm(): FormModel {
    return {
      displayName: '',
      strategyCode: '',
      monitorMetric: '',
      severity: '',
      thresholdMethod: 'gte',
      thresholdValue: 85,
      triggerCheckWindow: 1,
      triggerCount: 1,
      recoverCheckWindow: 1,
      effectiveTimeRange: ['00:00:00', '23:59:59'],
      scopeType: 'all',
      envTypes: [],
      envIDs: [],
      noticeGroupIDs: [],
    };
  }

  const formModel = reactive<FormModel>(createDefaultForm());

  /** 阈值描述文案 key：计数类指标（如 Pod 重启）单位用「次」，其余用「%」 */
  const thresholdKeypath = computed(() =>
    COUNT_UNIT_STRATEGY_CODES.has(formModel.strategyCode)
      ? '当前值 {0} {1}次 时触发告警'
      : '当前值 {0} {1}% 时触发告警',
  );

  const formRef = ref<InstanceType<typeof Form>>();
  const submitting = ref(false);
  const isNoticeDialogShow = ref(false);

  const { confirmBox, withPausedWatch, forceCleanDirtyTag } = useLeaveConfirm(formModel);

  /** 表单校验规则 */
  const formRules = computed(() => ({
    displayName: [
      { required: true, message: t('请输入 1-32 字符的策略名称'), trigger: 'blur' },
      {
        validator: () => BKMS_REGEX.envDisplayNameRegex.test(formModel.displayName || ''),
        message: t('请输入 1-32 字符的策略名称'),
        trigger: 'blur',
      },
    ],
    strategyCodeOptions: [{ required: true, message: t('请选择监控指标'), trigger: 'change' }],
    severity: [{ required: true, message: t('请选择告警级别'), trigger: 'change' }],
    thresholdValue: [
      {
        validator: () => formModel.thresholdValue !== '' && formModel.thresholdValue != null,
        message: t('请输入阈值'),
        trigger: 'change',
      },
    ],
    noticeGroupIDs: [
      {
        validator: () => formModel.noticeGroupIDs.length > 0,
        message: t('请选择告警组'),
        trigger: 'change',
      },
    ],
  }));

  /** 告警组 ID -> 名称映射 */
  const noticeGroupNameMap = computed(() => {
    const map = new Map<number, string>();
    props.userGroups.forEach(g => {
      if (g.id != null) map.set(g.id, g.name || String(g.id));
    });
    return map;
  });

  /** 组装生效范围 */
  function assembleScope(): EffectiveScopeInput {
    if (formModel.scopeType === 'all') {
      return { type: 'all' };
    }
    if (formModel.scopeType === 'env_type') {
      return { type: 'env_type', envTypes: formModel.envTypes };
    }
    return { type: 'specific_envs', envIDs: formModel.envIDs };
  }

  /** 编辑态回填表单 */
  function fillForm(data: AlertStrategyOutput) {
    withPausedWatch(() => {
      Object.assign(formModel, createDefaultForm());
      formModel.displayName = data.displayName || '';
      formModel.strategyCode = data.strategyCode || '';
      formModel.monitorMetric = data.monitorMetric || '';
      formModel.severity = (data.severity as 1 | 2 | 3) || '';
      formModel.thresholdMethod = data.threshold?.method || 'gte';
      formModel.thresholdValue = data.threshold?.value ?? '';
      formModel.triggerCheckWindow = data.triggerCondition?.checkWindow ?? 1;
      formModel.triggerCount = data.triggerCondition?.count ?? 1;
      formModel.recoverCheckWindow = data.recoverCondition?.checkWindow ?? 1;
      const tr = data.effectiveTimeRange;
      formModel.effectiveTimeRange =
        tr?.startTime && tr?.endTime ? [tr.startTime, tr.endTime] : ['00:00:00', '23:59:59'];
      formModel.scopeType = data.effectiveScope?.type || 'all';
      formModel.envTypes = data.effectiveScope?.envTypes || [];
      formModel.envIDs = data.effectiveScope?.envIDs || [];
      formModel.noticeGroupIDs = data.noticeGroupIDs || [];
    });
    forceCleanDirtyTag();
    formRef.value?.clearValidate();
  }

  /** 格式化时间段为字符串 */
  function formatTimeRange(): undefined | { endTime: string; startTime: string } {
    const range = formModel.effectiveTimeRange;
    if (!range || range.length !== 2 || !range[0] || !range[1]) return undefined;
    const formatVal = (v: unknown) => {
      if (!v) return '';
      if (typeof v === 'string') return v;
      return dayjs(v as Date).format('HH:mm:ss');
    };
    return { startTime: formatVal(range[0]), endTime: formatVal(range[1]) };
  }

  function getNoticeGroupName(id: number) {
    return noticeGroupNameMap.value.get(id) || String(id);
  }

  async function handleBeforeClose() {
    return confirmBox();
  }

  async function handleCancel() {
    if (await handleBeforeClose()) {
      visible.value = false;
    }
  }

  function handleClosed() {
    Object.assign(formModel, createDefaultForm());
    formRef.value?.clearValidate();
  }

  function handleNoticeGroupConfirm(ids: number[]) {
    formModel.noticeGroupIDs = ids;
  }

  function handleRemoveNoticeGroup(id: number) {
    formModel.noticeGroupIDs = formModel.noticeGroupIDs.filter(i => i !== id);
  }

  /** 提交表单 */
  async function handleSubmit() {
    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid) {
      focusOnErrorField();
      return;
    }

    if (!appDetailStore.appID || !spaceStore.currentSpace) return;

    submitting.value = true;
    try {
      const effectiveTimeRange = formatTimeRange();
      const triggerCondition =
        formModel.triggerCheckWindow !== '' && formModel.triggerCount !== ''
          ? { checkWindow: Number(formModel.triggerCheckWindow), count: Number(formModel.triggerCount) }
          : undefined;
      const recoverCondition =
        formModel.recoverCheckWindow !== '' ? { checkWindow: Number(formModel.recoverCheckWindow) } : undefined;

      const threshold = {
        method: formModel.thresholdMethod as 'eq' | 'gt' | 'gte' | 'lt' | 'lte' | 'neq',
        value: Number(formModel.thresholdValue),
      };
      const effectiveScope = assembleScope();

      if (isEdit.value && props.data?.id) {
        await BkintegrationsBkmonitorService.updateAlertStrategy({
          workspaceID: spaceStore.currentSpace,
          appID: appDetailStore.appID,
          strategyID: props.data.id,
          displayName: formModel.displayName,
          effectiveScope,
          effectiveTimeRange,
          noticeGroupIDs: formModel.noticeGroupIDs,
          recoverCondition,
          severity: formModel.severity as 1 | 2 | 3,
          threshold,
          triggerCondition,
        });
        Message({ theme: 'success', message: t('保存成功') });
      } else {
        await BkintegrationsBkmonitorService.createAlertStrategy({
          workspaceID: spaceStore.currentSpace,
          appID: appDetailStore.appID,
          displayName: formModel.displayName,
          strategyCode: formModel.strategyCode,
          monitorMetric: formModel.monitorMetric,
          severity: formModel.severity as 1 | 2 | 3,
          threshold,
          effectiveScope,
          effectiveTimeRange,
          triggerCondition,
          recoverCondition,
          noticeGroupIDs: formModel.noticeGroupIDs,
          enabled: true,
        });
        Message({ theme: 'success', message: t('创建成功') });
      }
      forceCleanDirtyTag();
      emit('success');
      visible.value = false;
    } finally {
      submitting.value = false;
    }
  }

  // 侧滑打开时：加载环境列表 + 回填编辑数据
  watch(
    () => props.isShow,
    async val => {
      if (val) {
        // 立即清除上一次残留的校验错误，避免渲染时闪现红色提示
        formRef.value?.clearValidate();
        if (appDetailStore.appID) {
          await getAppEnvList(appDetailStore.appID);
        }
        if (props.data) {
          fillForm(props.data);
        } else {
          withPausedWatch(() => {
            Object.assign(formModel, createDefaultForm());
          });
          forceCleanDirtyTag();
        }
      }
    },
  );
</script>

<style lang="postcss" scoped>
  /* trigger 插槽模拟 behavior="simplicity" 的展开/聚焦态：底边框高亮为主色 */
  :deep(.bk-select.popover-show) .simplicity-trigger,
  :deep(.bk-select.is-focus) .simplicity-trigger {
    border-bottom-color: #3a84ff;
  }

  /* 下拉展开时右侧箭头旋转 180° */
  :deep(.bk-select.popover-show) .simplicity-trigger-arrow {
    transform: rotate(180deg);
  }
</style>
