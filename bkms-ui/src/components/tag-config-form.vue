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
  <Form.FormItem
    class="tag-config-form-item"
    :label="label"
    :property="property"
  >
    <Switcher
      v-model="enabled"
      theme="primary"
      @change="handleEnabledChange"
    />
    <!-- 版本规则 -->
    <div
      v-if="enabled"
      class="mt-[8px] p-[16px] bg-[#F5F7FA] rounded-[2px]"
    >
      <div class="flex">
        <div class="w-[90px] text-align-right pr-[22px] text-[14px] text-[#63656e]">
          {{ $t('版本规则') }}
          <span class="text-[#ea3636]">*</span>
        </div>
        <Radio.Group
          v-model="configData.type"
          class="flex flex-col flex-1"
        >
          <Radio label="semver">
            {{ $t('语义化版本（格式为：主版本.特性版本.修正版本，示例：1.0.1）') }}
          </Radio>
          <Radio
            class="!ml-0"
            label="custom"
          >
            {{ $t('自定义版本') }}
          </Radio>
        </Radio.Group>
      </div>
      <div
        v-if="configData.type === 'custom'"
        class="flex items-center gap-[12px] ml-[112px]"
      >
        <Input
          v-model.trim="configData.prefix"
          class="w-[120px]"
          :placeholder="$t('自定义前缀')"
        />
        +
        <Checkbox
          v-model="configData.withRevision"
          :false-label="false"
          :true-label="true"
        >
          {{ $t('分支/Tag') }}
        </Checkbox>
        +
        <Checkbox
          v-model="configData.withBuildTime"
          disabled
          :false-label="false"
          :true-label="true"
        >
          {{ $t('构建时间') }}
        </Checkbox>
        <span class="text-[14px] text-[#63656e]">{{ $t('(示例：{0})', [example]) }}</span>
      </div>
    </div>
  </Form.FormItem>
</template>

<script setup lang="ts">
  import { computed, nextTick, ref, watch } from 'vue';

  import { Checkbox, Form, Input, Radio, Switcher } from 'bkui-vue';
  import dayjs from 'dayjs';

  import type { CustomTagOpts, TagConfig } from '~/@types/build';

  interface IProps {
    /** 表单项标签 */
    label?: string;
    /** 初始化时的 tagConfig 值（用于回显） */
    modelValue?: null | TagConfig;
    /** 表单校验的 property 路径 */
    property?: string;
  }

  interface TagConfigFormData {
    prefix: string;
    type: TagType;
    withBuildTime: boolean;
    withRevision: boolean;
  }

  type TagType = 'custom' | 'semver';

  // 默认表单数据
  const DEFAULT_FORM_DATA: TagConfigFormData = {
    type: 'semver',
    prefix: '',
    withRevision: true,
    withBuildTime: true,
  };

  const props = defineProps<IProps>();

  const emit = defineEmits<{
    /** 返回格式化后的 tagConfig 或 null */
    (e: 'update:modelValue', value: null | TagConfig): void;
  }>();

  // 是否启用推荐版本号
  const enabled = ref(false);

  // 表单数据
  const configData = ref<TagConfigFormData>({ ...DEFAULT_FORM_DATA });

  // 标记是否处于初始化阶段（防止初始化时触发 emit）
  let isInitializing = true;

  // 示例
  const example = computed(() => {
    const pieces: string[] = [
      configData.value.prefix,
      configData.value.withRevision ? 'master' : '',
      dayjs().format('YYYYMMDDHHmm'),
    ].filter(item => item !== '');

    return pieces.join('-');
  });

  /** 格式化并发出 tagConfig 值 */
  function emitValue() {
    if (isInitializing) return;
    emit('update:modelValue', enabled.value ? formatTagConfig() : null);
  }

  /** 将表单数据转换为 TagConfig 格式 */
  function formatTagConfig(): TagConfig {
    const { type, prefix, withRevision, withBuildTime } = configData.value;
    const customOpts: CustomTagOpts | null =
      type === 'custom' ? { prefix: prefix || '', withRevision, withBuildTime } : null;

    return { type, customOpts } as TagConfig;
  }

  /** 开关变化时的处理 */
  function handleEnabledChange(val: boolean) {
    if (!val) {
      resetFormData();
    }
    emit('update:modelValue', val ? formatTagConfig() : null);
  }

  /** 从 TagConfig 解析表单数据 */
  function parseTagConfig(val: TagConfig): TagConfigFormData {
    const { customOpts } = val;
    return {
      type: (val.type as TagType) || 'semver',
      prefix: customOpts?.prefix || '',
      withRevision: customOpts?.withRevision ?? true,
      withBuildTime: customOpts?.withBuildTime ?? true,
    };
  }

  /** 重置表单数据为默认值 */
  function resetFormData() {
    configData.value = { ...DEFAULT_FORM_DATA };
  }

  // 初始化：从 modelValue 回显数据
  watch(
    () => props.modelValue,
    val => {
      isInitializing = true;
      // type 为空时也视为未开启
      const isEnabled = !!val && !!val.type;
      enabled.value = isEnabled;
      configData.value = isEnabled ? parseTagConfig(val) : { ...DEFAULT_FORM_DATA };
      nextTick(() => {
        isInitializing = false;
      });
    },
    { immediate: true },
  );

  // 监听表单数据变化，通知父组件
  watch([enabled, configData], emitValue, { deep: true });
</script>

<style lang="postcss" scoped>
  :deep(.bk-radio) {
    margin-left: 0;
  }
  :deep(.bk-checkbox) {
    margin-left: 0;
  }
  .tag-config-form-item :deep(.bk-form-label::after) {
    content: '*';
    color: #ea3636;
    font-size: 12px;
    margin-left: 4px;
  }
</style>
