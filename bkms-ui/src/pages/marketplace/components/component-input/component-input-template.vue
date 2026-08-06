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
  <ToggleCard
    content-class="overflow-visible !m-0 pt-[16px]"
    :name="$t('组件输入模板')"
    type="normal"
  >
    <FlexRow>
      <template #left>
        <!-- 编辑模式：切换表单/YAML -->
        <Radio.Group
          v-model="editMode"
          class="w-full flex mb-[12px] w-[220px]"
          type="capsule"
          @change="handleEditModeChange"
        >
          <Radio.Button
            class="flex-1 flex items-center justify-center"
            label="form"
          >
            <i class="bkms-icon bkms-icon-form-mode text-[14px]"></i>
            <span class="text-[12px] pl-[4px]">{{ $t('表单模式') }}</span>
          </Radio.Button>
          <Radio.Button
            class="flex-1 flex items-center justify-center"
            label="yaml"
          >
            <i class="bkms-icon bkms-icon-code text-[14px]"></i>
            <span class="text-[12px] pl-[4px]">{{ $t('Yaml 模式') }}</span>
          </Radio.Button>
        </Radio.Group>
      </template>
      <template #right>
        <Button
          text
          theme="primary"
          @click="handleClickToDoc"
        >
          <Help class="mr-[4px] text-[14px]" />
          {{ $t('帮助文档') }}
        </Button>
      </template>
    </FlexRow>
    <ParamsTable
      v-show="editMode === 'form'"
      ref="paramsTableRef"
      :initial-data="initialData?.length ? initialData : DEFAULT_INPUT_TEMPLATE"
    />
    <ResizeLayout
      v-show="editMode === 'yaml'"
      :auto-minimize="true"
      class="!h-[500px] w-full"
      :collapsible="true"
      :disabled="!editorError.length"
      :is-collapsed="!editorError.length"
      :max="100"
      :min="100"
      placement="bottom"
    >
      <template #collapse-trigger />
      <template #aside>
        <EditorStatus :message="editorError" />
      </template>
      <template #main>
        <MsEditor
          ref="yamlEditorRef"
          class="h-[500px]"
          :model-value="yamlValue"
          :readonly="false"
          :title="t('输入模板')"
          @error="err => handleEditorErr(err)"
        />
      </template>
    </ResizeLayout>
  </ToggleCard>
</template>

<script lang="ts" setup>
  import { ref } from 'vue';

  import { Button, Radio, ResizeLayout } from 'bkui-vue';
  import { Help } from 'bkui-vue/lib/icon';
  import yaml from 'js-yaml';
  import { useI18n } from 'vue-i18n';
  import EditorStatus from '~/components/monaco-editor/editor-status.vue';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';
  import ToggleCard from '~/components/toggle-card.vue';
  import { TYPE_MAP } from '~/pages/marketplace/components/params-table/constants';
  import ParamsTable from '~/pages/marketplace/components/params-table/params-table.vue';

  import type { PropertyDefInput, PropertyOptionInput } from '~/@types/v1/component-defs';
  import type { IMonacoEditorErrorMarkerItem } from '~/common/util';
  import type { SelectTypeConfig } from '~/components/type-param-select.vue';
  import type { IRowData } from '~/pages/marketplace/components/params-table/render-row.vue';

  /** 输入参数默认模板 */
  const DEFAULT_INPUT_TEMPLATE: PropertyDefInput[] = [
    { name: 'graceSeconds', type: 'INT', options: [], defaultValue: 30, description: '优雅停机等待秒数' },
  ];

  interface IProps {
    /** 初始属性数据（PropertyDefInput 接口格式），用于编辑回填 */
    initialData?: PropertyDefInput[];
  }

  const yamlValue = defineModel<string>('yamlValue', { default: '' });

  withDefaults(defineProps<IProps>(), {
    initialData: () => [],
  });

  const emit = defineEmits<{
    (e: 'error', errors: IMonacoEditorErrorMarkerItem[]): void;
  }>();

  const { t } = useI18n();

  const editMode = ref<'form' | 'yaml'>('form');
  const paramsTableRef = ref<InstanceType<typeof ParamsTable>>();
  const yamlEditorRef = ref();
  const editorError = ref<string[]>([]);

  /** 获取原始表单数据 */
  function getFormData() {
    return paramsTableRef.value?.getData() ?? [];
  }

  /** 获取当前值（PropertyDefInput[]，提交用） */
  function getValue(): PropertyDefInput[] {
    if (editMode.value === 'yaml') {
      return yamlToPropertyDefs(yamlEditorRef.value!.getValue());
    }
    return tableDataToPropertyDefs(paramsTableRef.value?.getData() ?? []);
  }

  function handleEditModeChange(val: 'form' | 'yaml') {
    if (val === 'yaml') {
      // 表单 → YAML：把表单数据同步到 yamlValue
      yamlValue.value = tableDataToYaml(paramsTableRef.value?.getData() ?? []);
    } else {
      // YAML → 表单：从编辑器实例获取最新内容，解析后回填到 ParamsTable
      const editorContent = yamlEditorRef.value!.getValue();
      yamlValue.value = editorContent;
      const parsed = yamlToPropertyDefs(editorContent);
      paramsTableRef.value?.setData(parsed);
    }
  }

  function handleEditorErr(err: IMonacoEditorErrorMarkerItem[]) {
    editorError.value = err.map(item => item.message);
    emit('error', err);
  }

  /** 是否有编辑器错误（供父组件查询） */
  async function isValid(): Promise<boolean> {
    if (editMode.value === 'yaml') return !editorError.value.length;
    return (await paramsTableRef.value?.validate()) ?? false;
  }

  /** 重置数据 */
  function resetData() {
    editMode.value = 'form';
    editorError.value = [];
    paramsTableRef.value?.setData(DEFAULT_INPUT_TEMPLATE);
  }

  /** 设置数据（编辑模式回填，空时用默认模板） */
  function setData(data?: PropertyDefInput[]) {
    paramsTableRef.value?.setData(data?.length ? data : DEFAULT_INPUT_TEMPLATE);
  }

  /** 将表单行数据转为 PropertyDefInput 数组 */
  function tableDataToPropertyDefs(data: IRowData[]): PropertyDefInput[] {
    if (!data.length) return [];
    return data.map(row => ({
      name: row.name,
      type: (TYPE_REVERSE_MAP[row.typeConfig?.type]?.toUpperCase() ??
        row.typeConfig?.type ??
        'STRING') as PropertyDefInput['type'],
      options: row.typeConfig?.type === 'Select' ? (row.typeConfig as SelectTypeConfig).options : [],
      defaultValue: row.defaultValue ?? undefined,
      description: row.description,
    }));
  }

  /** 将表单行数据转为 YAML 字符串 */
  function tableDataToYaml(data: IRowData[]): string {
    if (!data.length) return '';
    return yaml.dump(
      data.map(row => ({
        name: row.name,
        type: TYPE_REVERSE_MAP[row.typeConfig?.type]?.toUpperCase() ?? row.typeConfig?.type,
        options: row.typeConfig?.type === 'Select' ? (row.typeConfig as SelectTypeConfig).options : [],
        defaultValue: row.defaultValue,
        description: row.description,
      })),
    );
  }

  /** 前端类型 -> 后端类型 反向映射 */
  const TYPE_REVERSE_MAP: Record<string, string> = Object.fromEntries(Object.entries(TYPE_MAP).map(([k, v]) => [v, k]));

  /** 查看帮助文档 */
  function handleClickToDoc() {
    const COMPONENTS_INPUT_DOC_PATH = '/p/4022526963';
    window.open(window.BK_DOC_URL + COMPONENTS_INPUT_DOC_PATH, '_blank');
  }

  /** 将 YAML 字符串解析为 PropertyDefInput 数组 */
  function yamlToPropertyDefs(yamlStr: string): PropertyDefInput[] {
    if (!yamlStr?.trim()) return [];
    try {
      const items = yaml.load(yamlStr, { schema: yaml.JSON_SCHEMA });
      if (!Array.isArray(items)) return [];
      return items
        .filter((item): item is Record<string, unknown> => !!item && typeof item === 'object')
        .map(item => ({
          name: String(item.name ?? ''),
          type: String(item.type ?? 'STRING') as PropertyDefInput['type'],
          options: Array.isArray(item.options) ? (item.options as PropertyOptionInput[]) : [],
          defaultValue: item.defaultValue ?? undefined,
          description: String(item.description ?? ''),
        }));
    } catch {
      return [];
    }
  }

  defineExpose({ getValue, getFormData, setData, resetData, isValid });
</script>
