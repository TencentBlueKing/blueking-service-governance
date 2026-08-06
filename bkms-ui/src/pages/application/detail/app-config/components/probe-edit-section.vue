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
  <div class="probe-card border rounded-[8px] flex-1 min-w-0 overflow-hidden">
    <!-- 卡片标题行 -->
    <div class="flex items-center justify-between h-[32px] px-[12px] bg-[#F5F7FA] border-b border-[#EAEBF0]">
      <div class="flex items-center gap-[6px]">
        <i
          v-if="isFieldModified"
          class="w-[3px] h-[18px] bg-[#ff9c01] flex-shrink-0"
        ></i>
        <span class="text-[12px] font-bold text-[#4D4F56]">{{ label }}</span>
      </div>
    </div>

    <!-- 表单内容 -->
    <div class="px-[12px] py-[8px]">
      <Form
        ref="formRef"
        form-type="vertical"
        :model="probeData"
      >
        <!-- 探测方法 -->
        <Form.FormItem
          class="!w-full"
          :label="$t('探测方法')"
          property="probeHandler.type"
          required
        >
          <Select
            v-model="probeType"
            :clearable="false"
          >
            <Select.Option
              v-for="opt in PROBE_TYPE_OPTIONS"
              :key="opt"
              :label="opt"
              :value="opt"
            />
          </Select>
        </Form.FormItem>

        <!-- TCP 类型：检查端口 -->
        <Form.FormItem
          v-if="probeType === ProbeType.TCP"
          class="!w-full"
          :label="$t('检查端口')"
          property="probeHandler.port"
          required
          :rules="rules.port"
        >
          <Input
            v-model.trim="probeData.probeHandler!.port"
            :max="65535"
            :min="1"
            :placeholder="$t('1 ~ 65535')"
            :precision="0"
            type="number"
          />
        </Form.FormItem>

        <!-- EXEC 类型：执行命令 -->
        <div
          v-if="probeType === ProbeType.EXEC"
          class="mb-[24px]"
        >
          <div class="mb-[4px] text-[12px]">
            {{ $t('执行命令') }}
            <span class="text-[#EA3636]">*</span>
          </div>
          <Radio.Group
            v-model="execMode"
            class="mb-[8px]"
            type="capsule"
            @change="handleExecModeChange"
          >
            <Radio.Button label="shell">shell</Radio.Button>
            <Radio.Button label="exec">exec</Radio.Button>
          </Radio.Group>
          <!-- shell 模式：textarea 输入 shCommand -->
          <Input
            v-if="execMode === 'shell'"
            v-model="probeData.probeHandler!.shCommand"
            :rows="4"
            type="textarea"
          />
          <!-- exec 模式：RepeatableInput 输入 command -->
          <RepeatableInput
            v-else
            ref="repeatableInputRef"
            v-model="probeData.probeHandler!.command"
            required
            trim-on-input
          />
        </div>

        <!-- HTTP 类型：检查路径 + 检查端口 -->
        <template v-if="probeType === ProbeType.HTTP">
          <Form.FormItem
            class="!w-full"
            :label="$t('检查路径')"
            property="probeHandler.url"
            required
            :rules="rules.url"
          >
            <Input
              v-model.trim="probeData.probeHandler!.url"
              :placeholder="$t('请输入，如 /healthz')"
            />
          </Form.FormItem>
          <Form.FormItem
            class="!w-full"
            :label="$t('检查端口')"
            property="probeHandler.port"
            required
            :rules="rules.port"
          >
            <Input
              v-model.trim="probeData.probeHandler!.port"
              :max="65535"
              :min="1"
              placeholder="1 ~ 65535"
              :precision="0"
              type="number"
            />
          </Form.FormItem>
        </template>

        <!-- 通用参数配置 -->
        <Form.FormItem
          :label="$t('延迟探测时间')"
          property="initialDelaySeconds"
          required
          :rules="rules.initialDelaySeconds"
        >
          <Input
            v-model.trim="probeData.initialDelaySeconds"
            :min="0"
            :precision="0"
            :suffix="$t('秒')"
            type="number"
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('探测超时时间')"
          property="timeoutSeconds"
          required
          :rules="rules.positiveInteger"
        >
          <Input
            v-model.trim="probeData.timeoutSeconds"
            :min="1"
            :precision="0"
            :suffix="$t('秒')"
            type="number"
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('探测频率')"
          property="periodSeconds"
          required
          :rules="rules.positiveInteger"
        >
          <Input
            v-model.trim="probeData.periodSeconds"
            :min="1"
            :precision="0"
            :suffix="$t('秒')"
            type="number"
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('连续探测成功次数')"
          property="successThreshold"
          required
          :rules="rules.positiveInteger"
        >
          <Input
            v-model.trim="probeData.successThreshold"
            :min="1"
            :precision="0"
            :suffix="$t('次')"
            type="number"
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('连续探测失败次数')"
          property="failureThreshold"
          required
          :rules="rules.positiveInteger"
        >
          <Input
            v-model.trim="probeData.failureThreshold"
            :min="1"
            :precision="0"
            :suffix="$t('次')"
            type="number"
          />
        </Form.FormItem>
      </Form>

      <!-- 操作按钮 -->
      <div class="flex items-center gap-[8px] mt-[8px]">
        <Button
          class="!w-[64px] !min-w-[64px]"
          :loading="saving"
          theme="primary"
          @click="$emit('save')"
        >
          {{ $t('保存') }}
        </Button>
        <PopConfirm
          v-if="isFieldModified"
          :confirm-text="$t('确认恢复')"
          :content="resetConfirmContent"
          :title="$t('确认恢复默认配置？')"
          trigger="click"
          :width="280"
          @confirm="$emit('reset-default')"
        >
          <Button :loading="resetting">
            {{ $t('恢复默认配置') }}
          </Button>
        </PopConfirm>
        <Button
          class="!w-[64px] !min-w-[64px]"
          @click="$emit('cancel')"
        >
          {{ $t('取消') }}
        </Button>
        <Divider
          class="h-[14px] !mx-[0]"
          color="#EAEBF0"
          direction="vertical"
          type="solid"
        />
        <PopConfirm
          :confirm-text="$t('确认停用')"
          :content="disableConfirmContent"
          :title="disableConfirmTitle"
          trigger="click"
          :width="280"
          @confirm="$emit('disable')"
        >
          <Button
            class="!w-[64px] !min-w-[64px]"
            :disabled="isProbeNew"
          >
            {{ $t('停用') }}
          </Button>
        </PopConfirm>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Button, Divider, Form, Input, PopConfirm, Radio, Select } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { ProbeOutput } from '~/@types/v1/app-spec';
  import RepeatableInput from '~/components/repeatable-input.vue';

  import { ProbeType } from './types';

  const PROBE_TYPE_OPTIONS = [ProbeType.HTTP, ProbeType.TCP, ProbeType.EXEC];

  interface Props {
    /** 停用确认弹窗的副标题 */
    disableConfirmContent?: string;
    /** 停用确认弹窗的标题 */
    disableConfirmTitle?: string;
    /** 是否显示环境覆盖标识 */
    isFieldModified?: boolean;
    /** 是否为新建探针（原本未配置），为 true 时禁用停用按钮 */
    isProbeNew?: boolean;
    /** 探针标题 */
    label: string;
    /** 探针数据 */
    modelValue: ProbeOutput;
    /** 恢复默认配置确认弹窗的副标题 */
    resetConfirmContent?: string;
    /** 恢复默认按钮 loading */
    resetting?: boolean;
    /** 保存按钮 loading */
    saving?: boolean;
  }

  const props = withDefaults(defineProps<Props>(), {
    disableConfirmContent: '',
    disableConfirmTitle: '',
    isFieldModified: false,
    isProbeNew: false,
    resetConfirmContent: '',
    resetting: false,
    saving: false,
  });

  const emit = defineEmits<{
    cancel: [];
    disable: [];
    'reset-default': [];
    save: [];
    'update:modelValue': [value: ProbeOutput];
  }>();

  const { t } = useI18n();

  const formRef = ref<InstanceType<typeof Form>>();
  const repeatableInputRef = ref<InstanceType<typeof RepeatableInput>>();

  const probeData = computed({
    get: () => props.modelValue,
    set: val => emit('update:modelValue', val),
  });

  const probeType = computed({
    get: () => probeData.value.probeHandler?.type || ProbeType.HTTP,
    set: (val: string) => {
      probeData.value.probeHandler!.type = val;
    },
  });

  /** EXEC 模式下 shell / exec 子类型切换 */
  const execMode = ref<'exec' | 'shell'>('exec');

  /** 根据数据初始化 execMode（shCommand 非空则为 shell 模式） */
  function syncExecModeFromData() {
    execMode.value = probeData.value.probeHandler?.shCommand ? 'shell' : 'exec';
  }
  syncExecModeFromData();

  /** 切换 shell/exec 时不清空另一方数据，保留用户输入 */
  function handleExecModeChange(mode: boolean | number | string) {
    const handler = probeData.value.probeHandler;
    if (mode === 'exec' && handler?.command?.length === 0) {
      setTimeout(() => repeatableInputRef.value?.add(), 0);
    }
  }

  const rules = {
    url: [
      {
        validator: (value: string) => !!value?.trim(),
        message: t('检查路径不能为空'),
        trigger: 'blur',
      },
    ],
    port: [
      {
        validator: (value: number | string | undefined) => {
          const num = Number(value);
          return !isNaN(num) && num >= 1 && num <= 65535;
        },
        message: t('请输入 1-65535 之间的端口号'),
        trigger: 'blur',
      },
    ],
    initialDelaySeconds: [
      {
        validator: (value: number | string) => Number(value) >= 0,
        message: t('不能小于 {0}', [0]),
        trigger: 'blur',
      },
    ],
    positiveInteger: [
      {
        validator: (value: number | string) => Number(value) >= 1,
        message: t('不能小于 {0}', [1]),
        trigger: 'blur',
      },
    ],
  };

  /** 校验方法，供父组件调用 */
  async function validate(): Promise<boolean> {
    const formValid = await formRef.value?.validate().catch(() => false);
    if (formValid === false) return false;

    if (probeType.value !== ProbeType.EXEC) return true;

    const handler = probeData.value.probeHandler;
    if (execMode.value === 'shell') {
      if (!handler?.shCommand?.trim()) return false;
      handler.command = [];
    } else {
      const execValid = repeatableInputRef.value ? await repeatableInputRef.value.validate().catch(() => false) : false;
      if (execValid === false) return false;
      handler!.shCommand = '';
    }

    return true;
  }

  // 切换探针类型时清空对应类型的专属字段
  watch(probeType, (newType, oldType) => {
    if (newType === oldType) return;

    const handler = probeData.value.probeHandler;
    if (newType !== ProbeType.EXEC) {
      handler!.command = [];
      handler!.shCommand = '';
    }
    if (newType !== ProbeType.HTTP) {
      handler!.url = '';
      handler!.headers = {};
    }
    if (oldType === ProbeType.HTTP || oldType === ProbeType.TCP) {
      handler!.port = '' as unknown as number;
    }

    setTimeout(() => formRef.value?.clearValidate?.(), 0);

    if (newType === ProbeType.EXEC && !handler?.shCommand && handler?.command?.length === 0) {
      execMode.value = 'exec';
      setTimeout(() => repeatableInputRef.value?.add(), 0);
    } else if (newType === ProbeType.EXEC) {
      syncExecModeFromData();
    }
  });

  defineExpose({ validate });
</script>

<style lang="postcss" scoped>
  :deep(.bk-form-label) {
    color: #4d4f56;
  }
</style>
