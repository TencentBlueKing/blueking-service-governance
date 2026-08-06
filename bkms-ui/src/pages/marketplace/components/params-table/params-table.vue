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
  <div :style="containerHeight">
    <Ediatable
      :key="tableKey"
      class="w-full"
    >
      <template #default>
        <HeadColumn :min-width="120">
          {{ $t('参数名称') }}
        </HeadColumn>
        <HeadColumn :min-width="120">
          {{ $t('类型') }}
        </HeadColumn>
        <HeadColumn
          :min-width="120"
          :required="false"
        >
          {{ $t('默认值') }}
        </HeadColumn>
        <HeadColumn
          :min-width="120"
          :required="false"
        >
          {{ $t('描述') }}
        </HeadColumn>
        <HeadColumn
          :min-width="80"
          :required="false"
        >
          {{ $t('操作') }}
        </HeadColumn>
      </template>
      <template #data>
        <RenderRow
          v-for="(item, index) in tableData"
          :key="item.rowKey"
          ref="rowRefs"
          v-model="tableData[index]"
          :existing-names="tableData.filter((_, i) => i !== index).map(r => r.name)"
          removeable
          @add="handleAddRow(index)"
          @copy="handleCopyRow(index)"
          @remove="handleRemoveRow(index)"
        />
      </template>
    </Ediatable>
    <div
      v-if="tableData.length === 0"
      class="mt-[12px]"
    >
      <Button
        text
        theme="primary"
        @click="handleAddRow(-1)"
      >
        <div class="flex items-center">
          <span class="bkms-icon bkms-icon-plus-circle-shape text-[14px]"></span>
          <span class="text-[12px] ml-[6px]">{{ $t('添加') }}</span>
        </div>
      </Button>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { type Ref, computed, inject, ref, watch } from 'vue';

  import Ediatable, { HeadColumn } from '@blueking/ediatable';
  import { Button } from 'bkui-vue';

  import { REFRESH_TABLE_SIGNAL, TYPE_MAP } from './constants';
  import RenderRow from './render-row.vue';

  import type { IRowData } from './render-row.vue';
  import type { PropertyDefInput } from '~/@types/v1/component-defs';
  import type { SimpleParamType } from '~/components/type-param-select.vue';

  import '@blueking/ediatable/vue3/vue3.css';

  interface IProps {
    /** 初始数据（PropertyDefInput 接口格式），传入后自动初始化表格 */
    initialData?: PropertyDefInput[];
  }

  const props = withDefaults(defineProps<IProps>(), {
    initialData: () => [],
  });

  /** 表单模式行数据 */
  let rowKeyCounter = 0;

  const tableKey = ref(0);

  const tableData = ref<IRowData[]>([]);
  const rowRefs = ref();

  /**
   * 表格容器高度
   * @description 由于外部设置了overflow-visible，以此达成描述栏textarea的abosulte溢出效果
   * 若不设置高度，则溢出的高度会覆盖 gap: 24px 的间距，与设计稿不一致
   * */
  const containerHeight = computed(() => {
    if (tableData.value.length === 0) {
      return {
        height: '60px',
      };
    }
    return {
      height: 42 * (tableData.value.length + 1) + 'px',
    };
  });

  // ======== 工具函数 ========

  function createRowData(partial?: Partial<IRowData>): IRowData {
    return {
      rowKey: String(++rowKeyCounter),
      name: '',
      typeConfig: { type: 'String' },
      defaultValue: '',
      description: '',
      ...partial,
    };
  }

  function getData(): IRowData[] {
    return tableData.value;
  }

  function handleAddRow(index: number) {
    tableData.value.splice(index + 1, 0, createRowData());
  }

  function handleCopyRow(index: number) {
    const { rowKey: _, ...copyData } = tableData.value[index];
    tableData.value.splice(index + 1, 0, createRowData(copyData));
  }

  function handleRemoveRow(index: number) {
    tableData.value.splice(index, 1);
  }

  /** 将 PropertyDefInput 转换为 IRowData */
  function propertyToRowData(prop: PropertyDefInput): IRowData {
    const targetType = TYPE_MAP[prop.type] ?? prop.type.toLowerCase();
    let typeConfig: IRowData['typeConfig'];

    if (targetType === 'Select') {
      typeConfig = {
        type: 'Select',
        options: prop.options || [],
      };
    } else {
      typeConfig = { type: targetType as SimpleParamType };
    }
    return {
      rowKey: String(++rowKeyCounter),
      name: prop.name,
      typeConfig,
      defaultValue: (prop.defaultValue as null | number | string) ?? null,
      description: prop.description ?? '',
    };
  }

  function resetData() {
    rowKeyCounter = 0;
    tableData.value = [createRowData()];
  }

  /** 外部设置数据（编辑模式回填） */
  function setData(data: PropertyDefInput[]) {
    rowKeyCounter = 0;
    tableData.value = data.length > 0 ? data.map(propertyToRowData) : [];
  }

  /** 校验所有行参数名并返回数据（异步） */
  async function validate(): Promise<boolean> {
    // 仅校验必填的参数名称字段
    const validates = await Promise.all(
      rowRefs.value.map((ref: { validateParamsName: () => Promise<void> }) => ref.validateParamsName()),
    );
    return validates.every(Boolean);
  }

  watch(inject<Ref<number>>(REFRESH_TABLE_SIGNAL)!, () => tableKey.value++);

  // 监听 initialData 变化自动初始化
  watch(
    () => props.initialData,
    val => {
      setData(val ?? []);
    },
    { immediate: true },
  );

  defineExpose({ getData, resetData, setData, validate });
</script>

<style lang="less" scoped>
  :deep(.bk-ediatable) {
    /* 允许描述列 absolute 内容视觉溢出 */
    overflow: visible;
  }

  :deep(.bk-ediatable-right-fixed-column) {
    &::before {
      display: none;
    }
  }
</style>
