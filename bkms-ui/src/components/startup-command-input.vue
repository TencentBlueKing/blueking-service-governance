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
  <div>
    <Radio.Group
      v-model="mode"
      class="mb-[8px]"
      type="capsule"
    >
      <Radio.Button label="shell">shell</Radio.Button>
      <Radio.Button label="exec">exec</Radio.Button>
    </Radio.Group>
    <CommandModeHint :mode="mode" />
    <template v-if="mode === 'shell'">
      <Form
        ref="shellFormRef"
        form-type="vertical"
        :model="formData"
      >
        <Form.FormItem
          class="!mb-0"
          property="shCommand"
          :rules="shellRules"
        >
          <Input
            v-model="formData.shCommand"
            :rows="4"
            type="textarea"
          />
        </Form.FormItem>
      </Form>
    </template>
    <div
      v-else
      class="grid grid-cols-2 gap-x-[24px]"
      :class="{ 'bg-[#F5F7FA] p-[16px] py-[12px]': execBackground }"
    >
      <div class="min-w-0">
        <div class="mb-[8px] text-[12px] leading-[22px] text-[#4D4F56]">{{ $t('启动命令') }}</div>
        <RepeatableInput
          ref="commandRef"
          v-model="formData.command"
          :add-text="$t('添加')"
          :required="formData.command.length > 0"
          trim-on-input
        />
      </div>
      <div class="min-w-0">
        <div class="mb-[8px] text-[12px] leading-[22px] text-[#4D4F56]">{{ $t('命令参数') }}</div>
        <RepeatableInput
          ref="argsRef"
          v-model="formData.args"
          :add-text="$t('添加')"
          :required="formData.args.length > 0"
          trim-on-input
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref } from 'vue';

  import { Form, Input, Radio } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import CommandModeHint from '~/components/command-mode-hint.vue';
  import RepeatableInput from '~/components/repeatable-input.vue';

  import type { AppModelSpecInput } from '~/@types/v1/app';

  const props = defineProps<{
    defaultMode?: 'exec' | 'shell';
    execBackground?: boolean;
    value?: Pick<AppModelSpecInput, 'args' | 'command' | 'shCommand'>;
  }>();

  const { t } = useI18n();
  const mode = ref<'exec' | 'shell'>(
    props.value?.shCommand
      ? 'shell'
      : props.value?.command?.length || props.value?.args?.length
        ? 'exec'
        : props.defaultMode || 'exec',
  );
  // 两种模式的输入分别保留，提交时仅使用当前模式的数据。
  const formData = ref({
    command: [...(props.value?.command || [])],
    args: [...(props.value?.args || [])],
    shCommand: props.value?.shCommand || '',
  });
  const shellFormRef = ref<InstanceType<typeof Form> | null>(null);
  const commandRef = ref<InstanceType<typeof RepeatableInput> | null>(null);
  const argsRef = ref<InstanceType<typeof RepeatableInput> | null>(null);
  const shellRules = [
    {
      validator: (value: string) => !!value.trim(),
      message: t('必填项'),
      trigger: 'blur',
    },
  ];

  function getValue() {
    return {
      command: mode.value === 'exec' ? [...formData.value.command] : [],
      args: mode.value === 'exec' ? [...formData.value.args] : [],
      shCommand: mode.value === 'shell' ? formData.value.shCommand : '',
    };
  }

  async function validate() {
    if (mode.value === 'shell') {
      return !!(await shellFormRef.value?.validate().catch(() => false));
    }
    const [commandValid, argsValid] = await Promise.all([commandRef.value?.validate(), argsRef.value?.validate()]);
    return !!commandValid && !!argsValid;
  }

  defineExpose({ getValue, validate });
</script>

<style lang="less" scoped>
  .err-relative :deep(.bk-form-item .bk-form-error) {
    position: relative !important;
  }
</style>
