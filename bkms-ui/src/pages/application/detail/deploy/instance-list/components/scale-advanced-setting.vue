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
  <div class="mt-[24px]">
    <div class="mb-[6px] leading-[22px] text-[14px] text-[#4D4F56]">{{ $t('高级设置') }}</div>
    <div class="flex items-center gap-[8px] h-[22px]">
      <Checkbox v-model="enabledValue">
        <span class="text-[#3A84FF]">{{ $t('定时扩缩容') }}</span>
      </Checkbox>
      <span class="text-[#979BA5] text-[12px]">
        {{
          $t(
            '定时策略与触发条件同时生效时，取两者期望实例数的较大值执行；多条定时策略执行时间重叠时，同样取较大的期望实例数。',
          )
        }}
      </span>
    </div>

    <template v-if="enabledValue">
      <Table
        class="mt-[12px]"
        :data="tableData"
        :row-config="{ isHover: true, keyField: 'id' }"
        :row-height="52"
      >
        <TableColumn
          field="periodText"
          :label="$t('执行周期')"
          min-width="160"
        />
        <TableColumn
          field="enabled"
          :label="$t('启用该策略')"
          min-width="120"
        >
          <template #default="{ row }">
            <StatusIcon
              :status="String(row.enabled)"
              :status-color-map="enabledStatusColorMap"
              :status-text-map="enabledStatusTextMap"
            />
          </template>
        </TableColumn>
        <TableColumn
          field="desiredReplicas"
          :label="$t('期望实例数')"
          min-width="140"
        />
        <TableColumn
          field="timeText"
          :label="$t('执行时间')"
          min-width="160"
        />
        <TableColumn
          field="remark"
          :label="$t('备注')"
          min-width="160"
          show-overflow="tooltip"
        >
          <template #default="{ row }">
            {{ row.remark || '--' }}
          </template>
        </TableColumn>
        <TableColumn
          fixed="right"
          :label="$t('操作')"
          :show-overflow="false"
          width="140"
        >
          <template #default="{ row }">
            <Button
              text
              theme="primary"
              @click="handleEdit(row.rangeIndex)"
            >
              {{ $t('编辑') }}
            </Button>
            <Button
              class="ml-[12px]"
              text
              theme="primary"
              @click="handleDelete(row.rangeIndex)"
            >
              {{ $t('删除') }}
            </Button>
          </template>
        </TableColumn>
      </Table>

      <Button
        class="mt-[12px]"
        text
        theme="primary"
        @click="handleAdd"
      >
        <i class="bkms-icon bkms-icon-plus-circle-shape mr-[4px]"></i>
        {{ $t('新增策略') }}
      </Button>
    </template>

    <Dialog
      v-model:is-show="dialogVisible"
      quick-close
      render-directive="if"
      :title="editIndex === -1 ? $t('新增定时扩缩容策略') : $t('编辑定时扩缩容策略')"
      :width="640"
    >
      <Form
        ref="dialogFormRef"
        form-type="vertical"
        :model="dialogForm"
      >
        <Form.FormItem
          :label="$t('启用该策略')"
          property="enabled"
        >
          <Switcher
            v-model="dialogForm.enabled"
            theme="primary"
          />
        </Form.FormItem>

        <Form.FormItem
          :label="$t('执行周期')"
          property="periodType"
          required
          :rules="periodTypeRules"
        >
          <Button.ButtonGroup class="flex items-center">
            <Button
              v-for="item in periodOptions"
              :key="item.value"
              class="flex-1"
              :selected="dialogForm.periodType === item.value"
              @click="handleChangePeriod(item.value)"
            >
              {{ item.label }}
            </Button>
          </Button.ButtonGroup>
        </Form.FormItem>

        <Form.FormItem
          v-if="dialogForm.periodType === 'weekly'"
          :label="$t('每周选择')"
          property="weekdays"
          required
          :rules="weekdaysRules"
        >
          <Select
            v-model="dialogForm.weekdays"
            class="w-full"
            multiple
            multiple-mode="tag"
            :placeholder="$t('请选择')"
            selected-style="checkbox"
          >
            <Select.Option
              v-for="item in weekdayOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </Select>
        </Form.FormItem>

        <Form.FormItem
          :label="$t('执行时间')"
          required
        >
          <div class="grid grid-cols-2 gap-[22px]">
            <Form.FormItem
              class="mb-0"
              property="startTime"
              :rules="startTimeRules"
            >
              <Select
                v-model="dialogForm.startTime"
                class="w-full"
                :placeholder="$t('开始时间')"
              >
                <Select.Option
                  v-for="item in hourOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </Select>
            </Form.FormItem>
            <Form.FormItem
              class="mb-0"
              property="endTime"
              :rules="endTimeRules"
            >
              <Select
                v-model="dialogForm.endTime"
                class="w-full"
                :placeholder="$t('结束时间')"
              >
                <Select.Option
                  v-for="item in hourOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </Select>
            </Form.FormItem>
          </div>
        </Form.FormItem>

        <Form.FormItem
          :label="$t('期望实例数')"
          property="desiredReplicas"
          required
          :rules="desiredReplicasRules"
        >
          <Input
            :model-value="dialogForm.desiredReplicas"
            :precision="0"
            type="number"
            @change="handleDesiredReplicasUpdate"
            @update:model-value="handleDesiredReplicasUpdate"
          />
        </Form.FormItem>

        <Form.FormItem
          :label="$t('备注')"
          property="remark"
        >
          <Input v-model="dialogForm.remark" />
        </Form.FormItem>
      </Form>

      <template #footer>
        <Button
          :loading="isDialogSubmitting"
          theme="primary"
          @click="handleDialogConfirm"
        >
          {{ $t('确定') }}
        </Button>
        <Button
          class="ml-[8px]"
          @click="dialogVisible = false"
        >
          {{ $t('取消') }}
        </Button>
      </template>
    </Dialog>
  </div>
</template>

<script lang="ts" setup>
  import { computed, nextTick, reactive, ref } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Checkbox, Dialog, Form, Input, Select, Switcher } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import StatusIcon from '~/components/status-icon.vue';

  import type { GPATimeRangeInput } from '~/@types/v1/gpa';

  /** 已解析的时间范围，包含原始数据、解析后的表单和索引 */
  interface ParsedTimeRange {
    form: null | ScheduleForm;
    range: GPATimeRangeInput;
    rangeIndex: number;
  }

  /** 执行周期类型 */
  type PeriodType = 'daily' | 'weekly';

  /** 弹窗表单数据结构 */
  interface ScheduleForm {
    desiredReplicas: number | string;
    enabled: boolean;
    endTime: string;
    periodType: PeriodType;
    remark: string;
    startTime: string;
    weekdays: number[];
  }

  // 组件 Props
  const props = defineProps<{
    maxReplicas: number;
    minReplicas: number;
  }>();

  // 双向绑定：策略列表 & 启用状态
  const modelValue = defineModel<GPATimeRangeInput[]>({ default: () => [] });
  const enabledValue = defineModel<boolean>('enabled', { default: false });

  const { t } = useI18n();

  const editIndex = ref(-1);
  const isDialogSubmitting = ref(false);
  const dialogVisible = ref(false);
  const dialogFormRef = ref<InstanceType<typeof Form>>();
  const dialogForm = reactive<ScheduleForm>(createDefaultForm());

  // 启用状态颜色和文案映射
  const enabledStatusColorMap = { true: 'green', false: 'gray' };
  const enabledStatusTextMap = { true: t('开启'), false: t('关闭') };
  const hourOptions = Array.from({ length: 24 }, (_, hour) => {
    const time = `${padNumber(hour)}:00`;
    return { label: time, value: time };
  });

  // 执行周期选项
  const periodOptions: Array<{ label: string; value: PeriodType }> = [
    { label: t('每天'), value: 'daily' },
    { label: t('每周'), value: 'weekly' },
  ];

  // 星期选项（0=周日，与 cron 一致）
  const weekdayOptions = [
    { label: t('周一'), value: 1 },
    { label: t('周二'), value: 2 },
    { label: t('周三'), value: 3 },
    { label: t('周四'), value: 4 },
    { label: t('周五'), value: 5 },
    { label: t('周六'), value: 6 },
    { label: t('周日'), value: 0 },
  ];

  /** 将原始 cron 策略解析为可编辑的表单数据 */
  const parsedTimeRanges = computed<ParsedTimeRange[]>(() =>
    modelValue.value.map((range, rangeIndex) => ({
      rangeIndex,
      range,
      form: parseTimeRange(range),
    })),
  );

  /** 表格展示数据，将解析后的策略转为展示文本 */
  const tableData = computed(() =>
    parsedTimeRanges.value.map(item => ({
      id: getTimeRangeRowId(item.range, item.rangeIndex),
      rangeIndex: item.rangeIndex,
      desiredReplicas: item.range.desiredReplicas,
      enabled: item.range.enabled ?? true,
      periodText: getPeriodText(item.form, item.range.schedule),
      remark: item.range.remark ?? '',
      timeText: getTimeText(item.form, item.range.schedule),
    })),
  );

  // ---- 表单校验规则 ----
  const periodTypeRules = [
    {
      validator: (value: PeriodType) => !!value,
      message: t('请选择执行周期'),
      trigger: 'change',
    },
  ];

  const weekdaysRules = [
    {
      validator: (value: number[]) => dialogForm.periodType !== 'weekly' || value.length > 0,
      message: t('请选择每周执行日期'),
      trigger: 'change',
    },
  ];

  const startTimeRules = [
    {
      validator: (value: string) => !!normalizeTime(value),
      message: t('请选择开始时间'),
      trigger: 'change',
    },
  ];

  const endTimeRules = [
    {
      validator: (value: string) => !!normalizeTime(value),
      message: t('请选择结束时间'),
      trigger: 'change',
    },
    {
      validator: () => getTimeRangeError() === '',
      message: () => getTimeRangeError() || t('执行时间不合法'),
      trigger: 'change',
    },
  ];

  const desiredReplicasRules = [
    {
      validator: (value: number | string) => isPositiveInteger(value),
      message: t('请输入正整数'),
      trigger: 'blur',
    },
    {
      validator: (value: number | string) => Number(value) >= props.minReplicas && Number(value) <= props.maxReplicas,
      message: () => t('期望实例数必须在 {0} ~ {1} 之间', [props.minReplicas, props.maxReplicas]),
      trigger: 'blur',
    },
  ];

  // ---- Cron 表达式构建 ----
  /** 根据表单数据生成 cron 调度表达式 */
  function buildSchedule(form: ScheduleForm) {
    const timeCron = buildTimeCron(form.startTime, form.endTime);
    if (!timeCron) return '';

    if (form.periodType === 'weekly') {
      return `${timeCron.minute} ${timeCron.hour} * * ${formatWeekdays(form.weekdays)}`;
    }

    // 每天执行
    return `${timeCron.minute} ${timeCron.hour} * * *`;
  }

  /** 将表单数据构建为提交用的 GPATimeRange 数组 */
  function buildSubmitRanges(form: ScheduleForm) {
    const desiredReplicas = Number(form.desiredReplicas);
    const baseRange = {
      desiredReplicas,
      enabled: form.enabled,
      remark: form.remark,
    };
    return [{ ...baseRange, schedule: buildSchedule(form) }];
  }

  /** 根据开始/结束时间构建 cron 的时和分字段 */
  function buildTimeCron(startTime: string, endTime: string) {
    const start = parseTime(startTime);
    const end = parseTime(endTime);
    if (!start || !end) return null;

    // 同小时：生成单小时执行；跨小时：生成小时范围执行。
    if (start.hour === end.hour) {
      return {
        hour: String(start.hour),
        minute: '0',
      };
    }

    return {
      hour: `${start.hour}-${end.hour}`,
      minute: '0',
    };
  }

  /** 将数字数组压缩为连续区间表示，如 [1,2,3,5,6] → "1-3,5-6" */
  function compressConsecutiveNumbers(values: number[]) {
    const sorted = [...new Set(values)].sort((a, b) => a - b);
    const ranges: string[] = [];
    let start = sorted[0];
    let prev = sorted[0];

    for (let index = 1; index <= sorted.length; index += 1) {
      const value = sorted[index];
      if (value === prev + 1) {
        prev = value;
        continue;
      }
      ranges.push(start === prev ? String(start) : `${start}-${prev}`);
      start = value;
      prev = value;
    }

    return ranges.join(',');
  }

  /** 创建默认空表单 */
  function createDefaultForm(): ScheduleForm {
    return {
      periodType: 'daily',
      weekdays: [],
      startTime: '',
      endTime: '',
      desiredReplicas: props?.minReplicas ?? 1,
      enabled: true,
      remark: '',
    };
  }

  /** 展开 cron 字段（支持 *、逗号分隔、范围），返回去重排序后的数值数组 */
  function expandCronField(field: string, min: number, max: number) {
    if (field === '*') return null;

    const values = field.split(',').flatMap(part => {
      if (part.includes('-')) {
        const [start, end] = part.split('-').map(Number);
        if (!Number.isInteger(start) || !Number.isInteger(end) || start > end) return [];
        return Array.from({ length: end - start + 1 }, (_, index) => start + index);
      }
      const value = Number(part);
      return Number.isInteger(value) ? [value] : [];
    });

    const uniqueValues = [...new Set(values)].filter(value => value >= min && value <= max).sort((a, b) => a - b);
    return uniqueValues.length > 0 ? uniqueValues : null;
  }

  /** 格式化星期字段（压缩连续数字） */
  function formatWeekdays(values: number[]) {
    return compressConsecutiveNumbers(values);
  }

  /** 获取执行周期的展示文本 */
  function getPeriodText(form: null | ScheduleForm, schedule: string) {
    if (!form) return schedule;
    if (form.periodType === 'daily') return t('每天');
    if (form.periodType === 'weekly') {
      return form.weekdays
        .map(value => weekdayOptions.find(item => item.value === value)?.label)
        .filter(Boolean)
        .join('，');
    }
    return schedule;
  }

  /** 校验时间范围的合法性 */
  function getTimeRangeError() {
    const start = parseTime(dialogForm.startTime);
    const end = parseTime(dialogForm.endTime);
    if (!start || !end) return t('请选择执行时间');

    const startTotal = start.hour * 60 + start.minute;
    const endTotal = end.hour * 60 + end.minute;
    if (startTotal >= endTotal) return t('结束时间必须大于开始时间');

    return '';
  }

  function getTimeRangeRowId(range: GPATimeRangeInput, rangeIndex: number) {
    return [rangeIndex, range.schedule, range.desiredReplicas, range.enabled ?? true, range.remark ?? ''].join('|');
  }

  /** 获取执行时间的展示文本 */
  function getTimeText(form: null | ScheduleForm, schedule: string) {
    if (!form) return schedule;
    return `${form.startTime} ~ ${form.endTime}`;
  }

  // ---- 事件处理 ----
  function handleAdd() {
    editIndex.value = -1;
    resetDialogForm(createDefaultForm());
    dialogVisible.value = true;
  }

  function handleChangePeriod(periodType: PeriodType) {
    dialogForm.periodType = periodType;
    nextTick(() => dialogFormRef.value?.clearValidate?.());
  }

  function handleDelete(index: number) {
    modelValue.value = modelValue.value.filter((_, itemIndex) => itemIndex !== index);
  }

  function handleDesiredReplicasUpdate(value: number | string) {
    dialogForm.desiredReplicas = value === '' ? '' : Number(value);
  }

  /** 弹窗确认：校验、构建数据并更新策略列表 */
  async function handleDialogConfirm() {
    if (isDialogSubmitting.value) return;
    isDialogSubmitting.value = true;
    try {
      normalizeDialogTimes();
      normalizeDesiredReplicas();
      const valid = await dialogFormRef.value?.validate().catch(() => false);
      if (!valid) return;

      const ranges = buildSubmitRanges(dialogForm).filter(item => item.schedule);
      if (editIndex.value === -1) {
        modelValue.value = [...modelValue.value, ...ranges];
      } else {
        // 编辑模式：替换对应索引的策略
        modelValue.value = modelValue.value.flatMap((item, index) => (index === editIndex.value ? ranges : [item]));
      }
      dialogVisible.value = false;
    } finally {
      isDialogSubmitting.value = false;
    }
  }

  function handleEdit(index: number) {
    const parsedForm = parsedTimeRanges.value.find(item => item.rangeIndex === index)?.form;
    editIndex.value = index;
    resetDialogForm(parsedForm ?? createDefaultForm());
    dialogVisible.value = true;
  }

  /** 判断是否为正整数 */
  function isPositiveInteger(value: number | string | undefined) {
    if (value === '' || value === undefined || value === null) return false;
    const numberValue = Number(value);
    return Number.isInteger(numberValue) && numberValue >= 1;
  }

  function normalizeDesiredReplicas() {
    if (isPositiveInteger(dialogForm.desiredReplicas)) {
      dialogForm.desiredReplicas = Number(dialogForm.desiredReplicas);
    }
  }

  function normalizeDialogTimes() {
    dialogForm.startTime = normalizeTime(dialogForm.startTime);
    dialogForm.endTime = normalizeTime(dialogForm.endTime);
  }

  /** 格式化时间为 HH:00 */
  function normalizeTime(value: string) {
    const time = parseTime(value);
    if (!time) return '';
    return `${padNumber(time.hour)}:00`;
  }

  /** 数字补零到 2 位 */
  function padNumber(value: number) {
    return String(value).padStart(2, '0');
  }

  /** 解析 HH 或 HH:mm 时间字符串 */
  function parseTime(value: string) {
    const match = /^(\d{1,2})(?::(\d{1,2}))?$/.exec(value);
    if (!match) return null;

    const hour = Number(match[1]);
    const minute = match[2] === undefined ? 0 : Number(match[2]);
    if (!Number.isInteger(hour) || !Number.isInteger(minute) || hour < 0 || hour > 23 || minute < 0 || minute > 59) {
      return null;
    }
    return { hour, minute };
  }

  // ---- Cron 表达式解析 ----
  /** 解析 cron 的时和分字段，返回 startTime/endTime */
  function parseTimeCron(minuteField: string, hourField: string) {
    const hours = expandCronField(hourField, 0, 23);
    if (!hours) return null;

    if (minuteField !== '*' && !expandCronField(minuteField, 0, 59)) return null;

    return {
      startTime: `${padNumber(hours[0])}:00`,
      endTime: `${padNumber(hours[hours.length - 1])}:00`,
    };
  }

  /**
   * 将 GPATimeRange 的 cron schedule 解析为 ScheduleForm
   * 支持两种周期类型：每天、每周
   */
  function parseTimeRange(range: GPATimeRangeInput): null | ScheduleForm {
    const parts = range.schedule.split(/\s+/);
    if (parts.length !== 5 || !isPositiveInteger(range.desiredReplicas)) return null;

    const [minuteField, hourField, dayOfMonthField, monthField, dayOfWeekField] = parts;
    const timeRange = parseTimeCron(minuteField, hourField);
    if (!timeRange) return null;

    const baseForm: ScheduleForm = {
      ...createDefaultForm(),
      desiredReplicas: Number(range.desiredReplicas),
      enabled: range.enabled ?? true,
      startTime: timeRange.startTime,
      endTime: timeRange.endTime,
      remark: range.remark ?? '',
    };

    // 指定了星期 → 每周
    if (dayOfWeekField !== '*') {
      const weekdays = expandCronField(dayOfWeekField, 0, 6);
      return weekdays ? { ...baseForm, periodType: 'weekly', weekdays } : null;
    }

    // 日期、月份、星期全为 * → 每天
    if (dayOfMonthField === '*' && monthField === '*' && dayOfWeekField === '*') {
      return { ...baseForm, periodType: 'daily' };
    }

    return null;
  }

  /** 重置弹窗表单并清除校验状态 */
  function resetDialogForm(form: ScheduleForm) {
    Object.assign(dialogForm, {
      ...form,
      weekdays: [...form.weekdays],
      desiredReplicas: Number(form.desiredReplicas) || props.minReplicas,
    });
    nextTick(() => dialogFormRef.value?.clearValidate?.());
  }
</script>
