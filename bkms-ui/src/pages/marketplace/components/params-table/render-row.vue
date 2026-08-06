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
  <tr>
    <td>
      <InputColumn
        ref="inputColumnRef"
        v-model="model.name"
        :clearable="false"
        :placeholder="t('请输入参数名')"
        :rules="rules"
      />
    </td>
    <td>
      <TypeParamSelect
        v-model="model.typeConfig"
        :placeholder="t('请选择类型')"
      />
    </td>
    <td>
      <InputColumn
        v-if="!['Select', 'Map', 'Boolean'].includes(currentType)"
        v-model="model.defaultValue as string"
        :clearable="false"
        :placeholder="t('请输入默认值')"
        :type="currentType === 'Number' ? 'number' : undefined"
      />
      <SelectColumn
        v-else-if="currentType === 'Select'"
        v-model="model.defaultValue as string"
        :clearable="false"
        :list="selectList"
        :placeholder="t('请选择默认值')"
      />
      <SelectColumn
        v-else-if="currentType === 'Boolean'"
        v-model="model.defaultValue as string"
        :allow-empty-values="[false]"
        :clearable="false"
        :list="booleanList"
        :placeholder="t('请选择默认值')"
      />
      <Popover
        v-else-if="currentType === 'Map'"
        ref="mapPopoverRef"
        placement="bottom-start"
        theme="light no-padding-popover"
        trigger="click"
        :width="400"
      >
        <div
          class="flex items-center px-[8px] rounded-[2px] cursor-pointer bg-white"
          :style="{ height: '30px', lineHeight: '28px' }"
        >
          <span
            :class="[
              'flex-1 text-[12px] px-[8px] overflow-hidden text-ellipsis whitespace-nowrap',
              mapDisplayText ? 'text-[#63656e]' : 'text-[#c4c6cc]',
            ]"
          >
            {{ mapDisplayText || t('请配置键值对') }}
          </span>
          <i
            :class="[
              'bkms-icon bkms-icon-angle-down shrink-0 text-[12px] text-[#979ba5] ml-[4px] transition-transform duration-200',
              { 'rotate-180': isMapPopoverShow },
            ]"
          />
        </div>
        <template #content>
          <div class="p-[12px]">
            <KeyValue
              v-model="mapDefaultValue"
              key-placeholder="key"
              :min-rows="1"
              value-placeholder="value"
            />
          </div>
        </template>
      </Popover>
    </td>
    <td>
      <InputColumn
        v-model="model.description"
        autosize
        class="description-textarea"
        :clearable="false"
        :placeholder="t('请输入描述内容')"
        :resize="false"
        type="textarea"
      />
    </td>
    <td>
      <div class="flex items-center gap-x-[18px] px-[16px]">
        <i
          v-bk-tooltips="$t('克隆')"
          class="bkms-icon bkms-icon-clone cursor-pointer text-[16px] text-[#c4c6cc] hover:text-[#979ba5]"
          @click="emit('copy')"
        />
        <i
          v-bk-tooltips="$t('新增')"
          class="bkms-icon bkms-icon-plus-circle-shape cursor-pointer text-[16px] text-[#c4c6cc] hover:text-[#979ba5]"
          @click="emit('add')"
        />
        <i
          v-bk-tooltips="$t('删除')"
          class="bkms-icon bkms-icon-minus-circle-shape cursor-pointer text-[16px] text-[#c4c6cc] hover:text-[#979ba5]"
          @click="emit('remove')"
        />
      </div>
    </td>
  </tr>
</template>

<script lang="ts" setup>
  import { computed, inject, Ref, ref, watch } from 'vue';

  import { InputColumn, SelectColumn } from '@blueking/ediatable';
  import { Popover } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import KeyValue from '~/components/key-value.vue';
  import TypeParamSelect, { type SelectTypeConfig, type TypeParamValue } from '~/components/type-param-select.vue';

  import { BUILTIN_VARS_SYMBOL } from './constants';

  import type { BuiltinVarOutputObj } from '~/@types/v1/component-defs';

  export interface IRowData {
    defaultValue: null | number | string;
    description: string;
    name: string;
    rowKey: string;
    typeConfig: TypeParamValue;
  }

  const model = defineModel<IRowData>({ required: true });

  const props = defineProps<{
    /** 其他行的参数名称列表（不含当前行），用于去重校验 */
    existingNames?: string[];
    removeable?: boolean;
  }>();

  const emit = defineEmits<{
    (e: 'add'): void;
    (e: 'copy'): void;
    (e: 'remove'): void;
  }>();

  const { t } = useI18n();

  const builtinVars = inject<Ref<BuiltinVarOutputObj[]>>(BUILTIN_VARS_SYMBOL, ref([]));

  const booleanList = [
    { label: 'true', value: true },
    { label: 'false', value: false },
  ];

  const inputColumnRef = ref();
  const mapPopoverRef = ref();
  const isMapPopoverShow = computed(() => mapPopoverRef.value?.localIsShow);

  const currentType = computed(() => model.value.typeConfig?.type ?? 'String');

  const selectList = computed(() => {
    if (currentType.value === 'Select') {
      return (model.value.typeConfig as SelectTypeConfig).options;
    }
    return [];
  });

  const rules = computed(() => [
    {
      validator: (value: string) => {
        return Boolean(value);
      },
      message: t('参数名称不能为空'),
    },
    {
      validator: (value: string) => {
        if (!value) return true;
        const names = props.existingNames ?? [];
        return !names.includes(value);
      },
      message: t('不能和已有的参数名称重复'),
    },
    {
      validator: (value: string) => {
        if (!value) return true;
        return !builtinVars.value.some(v => v.key === value);
      },
      message: t('不能和内置系统变量重复'),
    },
  ]);

  /** Map 类型默认值：KeyValue 绑定对象格式 { [key]: value }，空时保留一个空行便于编辑 */
  const mapDefaultValue = computed({
    get: () => {
      try {
        const raw = model.value.defaultValue;
        const parsed: Record<string, string> = raw ? JSON.parse(raw as string) : {};
        return parsed;
      } catch {
        return { '': '' };
      }
    },
    set: (val: Record<string, string>) => {
      const filtered = Object.fromEntries(Object.entries(val).filter(([k]) => k !== ''));
      model.value.defaultValue = Object.keys(filtered).length ? JSON.stringify(filtered) : '';
    },
  });

  /** Map 默认值显示文本 */
  const mapDisplayText = computed(() => {
    const obj = mapDefaultValue.value;
    const keys = Object.keys(obj);
    if (!keys.length) return '';
    return keys
      .filter(k => k)
      .map(k => `${k}:${obj[k]}`)
      .join(', ');
  });

  function resetDefaultValue() {
    if (currentType.value === 'Select' || currentType.value === 'Boolean') {
      model.value.defaultValue = null;
    } else {
      model.value.defaultValue = '';
    }
  }

  // 类型变化时重置默认值，首次初始化时不覆盖外部传入的值
  watch(
    () => model.value.typeConfig?.type,
    () => {
      resetDefaultValue();
    },
  );

  defineExpose({
    /** 校验参数名称是否合法 */
    validateParamsName: async () => {
      const res = await inputColumnRef.value.getValue().catch(() => false);
      return res;
    },
  });
</script>

<style lang="postcss" scoped>
  :deep(.description-textarea) {
    &.bk-textarea {
      min-height: 32px;
      transition: all 0.3s ease;
      position: absolute;
      top: 0;
      left: 0;
      border: 1px solid transparent;
      &.is-focused {
        z-index: 9999;
        height: auto;
        box-shadow: 0px 0px 3px 0px #a3c5fd;

        textarea {
          overflow: unset;
          text-overflow: unset;
          white-space: unset;
          max-height: 300px;
          width: 100%;
        }
      }

      &:hover {
        border: 1px solid #a3c5fd;
        background-color: #fafbfd;

        textarea {
          background-color: #fafbfd;
        }
      }

      textarea {
        cursor: pointer;
        min-height: 32px !important;
        height: 42px;
        width: calc(100% - 16px);
        padding: 12px 16px !important;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        scrollbar-width: none;
      }
    }
  }
</style>
