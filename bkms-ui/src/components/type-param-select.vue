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
  <Popover
    ref="popoverRef"
    placement="bottom-start"
    render-type="shown"
    theme="light no-padding-popover"
    trigger="click"
    :width="508"
    @after-hidden="handlePopoverHide"
    @after-show="handlePopoverShow"
  >
    <div
      :class="[
        'flex items-center px-[8px] rounded-[2px] cursor-pointer bg-white transition-all duration-200',
        { 'text-[#3a84ff]': isPopoverShow },
      ]"
      :style="styleVars"
    >
      <span
        :class="[
          'flex-1 text-[12px] overflow-hidden text-ellipsis whitespace-nowrap',
          displayText ? 'text-[#63656e]' : 'text-[#c4c6cc]',
        ]"
        :style="{ lineHeight: styleVars.lineHeight }"
      >
        {{ displayText || props.placeholder }}
      </span>
      <AngleDownLine
        class="text-[11px] text-[#979ba5] ml-[4px] transition-transform duration-200"
        :class="{ 'rotate-180': isPopoverShow }"
      />
      <!-- <i
        :class="[
          'bkms-icon bkms-icon-angle-down shrink-0 text-[12px] text-[#979ba5] ml-[4px] transition-transform duration-200',
          { 'rotate-180': isPopoverShow },
        ]"
      /> -->
    </div>
    <template #content>
      <div class="flex flex-col">
        <div class="py-[12px] px-[16px]">
          <Form
            ref="formRef"
            form-type="vertical"
            :model="editState"
          >
            <Form.FormItem
              class="mb-0 params-type"
              :label="t('选择参数类型')"
              property="type"
            >
              <Radio.Group
                v-model="editState.type"
                class="flex flex-nowrap"
                @change="handleTypeChange"
              >
                <Radio
                  v-for="item in typeOptions"
                  :key="item.value"
                  :label="item.value"
                >
                  <span class="text-[12px]">{{ item.label }}</span>
                </Radio>
              </Radio.Group>
            </Form.FormItem>
            <div
              v-if="'options' in editState.typeState"
              class="mt-[24px]"
            >
              <Form.FormItem class="mb-0 key-value-label">
                <template
                  v-if="editState.type === 'Select'"
                  #label
                >
                  <div class="grid grid-cols-2 w-[442px]">
                    <span class="text-[12px]">
                      {{ $t('选项文本（用于显示）') }}
                      <span class="text-red text-[14px]">*</span>
                    </span>
                    <span class="text-[12px]">
                      {{ $t('选项值') }}
                      <span class="text-red text-[14px]">*</span>
                    </span>
                  </div>
                </template>
                <KeyValue
                  ref="kvRef"
                  v-model="editState.typeState.options"
                  key-of-key="label"
                  :key-placeholder="$t('请输入选项文本')"
                  :key-rules="keyRules"
                  :max-height="245"
                  :min-rows="1"
                  :value-placeholder="$t('请输入选项值')"
                  :value-rules="valueRules"
                >
                  <template #middle>
                    <div class="w-[8px]"></div>
                  </template>
                </KeyValue>
              </Form.FormItem>
            </div>
          </Form>
        </div>
        <FlexRow class="h-[42px] px-[16px] bg-[#fafbfd] border-t border-solid border-[#eaebf0]">
          <template #right>
            <div class="flex gap-[8px]">
              <Button
                size="small"
                theme="primary"
                @click="handleConfirm"
              >
                {{ t('确定') }}
              </Button>
              <Button
                size="small"
                @click="handleClose"
              >
                {{ t('取消') }}
              </Button>
            </div>
          </template>
        </FlexRow>
      </div>
    </template>
  </Popover>
</template>

<script lang="ts" setup>
  import { computed, reactive, ref } from 'vue';

  import { Button, Form, Popover, Radio } from 'bkui-vue';
  import { AngleDownLine } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import FlexRow from '~/components/flex-row.vue';
  import KeyValue, { type FormRule } from '~/components/key-value.vue';

  /** Select 类型配置 */
  export interface SelectTypeConfig {
    type: 'Select';
    options: {
      label: string;
      value: string;
    }[];
  }

  // ======== TS 类型定义 ========
  /** 简单参数类型 */
  export type SimpleParamType = 'Boolean' | 'Map' | 'Number' | 'String' | 'Text';

  /** 简单类型配置 */
  export interface SimpleTypeConfig {
    type: SimpleParamType;
  }

  /** 类型参数值联合类型 */
  export type TypeParamValue = SelectTypeConfig | SimpleTypeConfig;

  // ======== 类型注册表 ========
  type Option = { label: string; value: string };

  interface TypeDefinition {
    /** Radio 上的显示名 */
    label: string;
    /** 从编辑态构建输出值 */
    buildOutput: (state: TypeState) => TypeParamValue;
    /** 该类型的默认编辑态 */
    defaultState: () => TypeState;
    /** 从 modelValue 恢复编辑态 */
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    syncFromModel: (val: any) => TypeState;
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  type TypeState = Record<string, any>;

  /** 类型定义：集中声明一种类型的所有知识，新增类型只需在此加一条 */
  const TYPE_DEFINITIONS: Record<string, TypeDefinition> = {
    String: {
      label: 'String',
      defaultState: () => ({}),
      syncFromModel: () => ({}),
      buildOutput: () => ({ type: 'String' }),
    },
    Number: {
      label: 'Number',
      defaultState: () => ({}),
      syncFromModel: () => ({}),
      buildOutput: () => ({ type: 'Number' }),
    },
    Boolean: {
      label: 'Boolean',
      defaultState: () => ({}),
      syncFromModel: () => ({}),
      buildOutput: () => ({ type: 'Boolean' }),
    },
    Text: {
      label: 'Text',
      defaultState: () => ({}),
      syncFromModel: () => ({}),
      buildOutput: () => ({ type: 'Text' }),
    },
    Select: {
      label: 'Select',
      defaultState: () => ({ options: [] as Option[] }),
      syncFromModel: (val: SelectTypeConfig) => ({
        options: [...val.options],
      }),
      buildOutput: state => ({
        type: 'Select',
        options: state.options,
      }),
    },
    Map: {
      label: 'Map',
      defaultState: () => ({}),
      syncFromModel: () => ({}),
      buildOutput: () => ({ type: 'Map' }),
    },
  };

  /** 所有前端参数类型 — 从注册表 key 自动派生，新增类型无需手动维护 */
  export type ParamType = keyof typeof TYPE_DEFINITIONS;

  /** Radio 选项直接从注册表派生 */
  const typeOptions = computed(() =>
    Object.entries(TYPE_DEFINITIONS).map(([value, def]) => ({
      label: def.label,
      value,
    })),
  );

  /** 当前类型定义 */
  const currentDef = computed(() => TYPE_DEFINITIONS[editState.type as ParamType]);

  interface IProps {
    height?: number;
    placeholder?: string;
  }

  const props = withDefaults(defineProps<IProps>(), {
    height: 42,
    placeholder: '',
  });

  const modelValue = defineModel<TypeParamValue>({ required: true });
  const emit = defineEmits<{
    (e: 'change'): void;
    (e: 'close'): void;
    (e: 'open'): void;
  }>();

  const { t } = useI18n();

  const popoverRef = ref();
  const kvRef = ref();
  const isPopoverShow = computed(() => popoverRef.value?.localIsShow);

  /** 编辑态：type + 动态类型状态 */
  const editState = reactive({
    type: 'String' as string,
    typeState: {} as TypeState,
  });

  /** Key 校验规则：必填 + 不重复 */
  const keyRules = computed<FormRule[]>(() => [
    {
      required: true,
      message: t('不能为空'),
      trigger: 'blur',
    },
    {
      validator: (val: unknown) => {
        if (!val) return true;
        const options = (editState.typeState.options as Option[] | undefined) ?? [];
        const labels = options.filter(item => item.label).map(item => item.label);
        return labels.filter(k => k === val).length <= 1;
      },
      message: t('选项文本不能重复'),
      trigger: 'blur',
    },
  ]);

  /** Value 校验规则：必填 + 不重复 */
  const valueRules = computed<FormRule[]>(() => [
    {
      required: true,
      message: t('不能为空'),
      trigger: 'blur',
    },
    {
      validator: (val: unknown) => {
        if (!val) return true;
        const options = (editState.typeState.options as Option[] | undefined) ?? [];
        const values = options.filter(item => item.value).map(item => item.value);
        return values.filter(v => v === val).length <= 1;
      },
      message: t('选项值不能重复'),
      trigger: 'blur',
    },
  ]);

  const styleVars = computed(() => ({
    height: `${props.height}px`,
    lineHeight: `${props.height - 2}px`,
  }));

  /** 显示文本 */
  const displayText = computed(() => {
    const val = modelValue.value;
    return val.type || '';
  });

  /** 根据编辑态构建输出值 */
  function buildOutputValue(): TypeParamValue {
    return currentDef.value.buildOutput(editState.typeState);
  }

  /** 关闭 Popover 弹层 */
  function handleClose() {
    popoverRef.value?.hide();
  }

  /** 确认并输出当前编辑值 */
  async function handleConfirm() {
    if ('options' in editState.typeState) {
      const valid = await kvRef.value?.validate().catch(() => false);
      if (!valid) return;
    }
    modelValue.value = buildOutputValue();
    emit('change');
    handleClose();
  }

  /** Popover 隐藏时通知父组件 */
  function handlePopoverHide() {
    emit('close');
  }

  /** Popover 显示时同步编辑态 */
  function handlePopoverShow() {
    syncEditStateFromModel();
    emit('open');
  }

  /** 切换类型时，重置 typeState */
  function handleTypeChange() {
    editState.typeState = currentDef.value.defaultState();
  }

  /** 从 modelValue 恢复编辑态 */
  function syncEditStateFromModel() {
    const val = modelValue.value;
    if (!val || !val.type) {
      editState.type = 'String';
      editState.typeState = TYPE_DEFINITIONS.string.defaultState();
      return;
    }
    editState.type = val.type;
    editState.typeState = currentDef.value.syncFromModel(val);
  }
</script>

<style lang="postcss" scoped>
  :deep(.bk-radio-group) {
    flex-wrap: nowrap;
  }

  :deep(.params-type) {
    .bk-form-label {
      color: #313238;
      font-weight: bold;
    }
  }

  :deep(.key-value-label) {
    .bk-form-label {
      font-size: 12px;
    }
  }
</style>
