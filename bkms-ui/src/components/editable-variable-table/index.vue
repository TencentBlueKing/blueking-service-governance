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
  <div class="editable-variable-table">
    <Form
      ref="formRef"
      form-type="vertical"
      :model="variableList"
    >
      <Table
        ref="tableRef"
        class="w-full"
        :data="variableList"
        row-class-name="group"
        :row-config="{
          isHover: true,
          useKey: true,
          keyField: 'id',
        }"
        :row-height="43"
        show-overflow-tooltip
        v-bind="$attrs"
        :sort-config="{ remote: false, trigger: 'cell' }"
      >
        <TableColumn
          field="key"
          label="Key"
          min-width="100"
          sortable
        >
          <template #default="{ row, rowIndex }: { row: VariableItem; rowIndex?: number }">
            <Form.FormItem
              v-if="editStates[row.id].editing"
              class="mb-0"
              error-display-type="tooltips"
              label=""
              :property="`${rowIndex}.key`"
              :rules="rules.key"
            >
              <Input
                ref="keyInputRef"
                v-model.trim="row.key"
                class="flex-1 w-full relative z-99"
                clearable
                :disabled="!editStates[row.id].add && props.disableKeyEdit"
                @enter="saveRow(row)"
              />
            </Form.FormItem>
            <div
              v-else
              class="text-[12px] flex items-center gap-[5px]"
            >
              <div class="whitespace-nowrap overflow-hidden text-ellipsis">{{ row?.key || '--' }}</div>
              <!-- 查看态：冲突警告图标 -->
              <InfoLine
                v-if="hasConflict(row)"
                v-bk-tooltips="buildEnvVarConflictTooltip(row.conflictedInfo!)"
                class="text-[#979BA5] shrink-0"
                :height="14"
                :width="14"
              />
              <!-- 复制 -->
              <Button
                class="ml-[6px] shrink-0"
                text
                @click="copyText(row.key)"
              >
                <Copy
                  class="group-hover:opacity-100 opacity-0 transition text-[#3A84FF]"
                  :title="$t('复制')"
                />
              </Button>
            </div>
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('是否敏感')"
          width="100"
        >
          <template #header>
            <span
              v-bk-tooltips="$t('敏感环境变量的值将在页面上以脱敏形式展示，只有应用进程内能够获取到这些变量的明文值。')"
              class="border-b border-dashed border-[#979ba5]"
            >
              {{ $t('是否敏感') }}
            </span>
          </template>
          <template #default="{ row }: { row: VariableItem }">
            <Switcher
              v-if="editStates[row.id].editing"
              v-model="row.isSensitive"
              :disabled="isSensitiveSwitcherDisabled(row)"
              theme="primary"
              @change="() => handleSensitiveChange(row)"
            />
            <Tag
              v-else
              :class="row.isSensitive ? 'text-[#2CAF5E] bg-[#DAF6E5]' : 'text-[#979BA5] bg-[#F0F1F5]'"
              :theme="row.isSensitive ? 'success' : 'default'"
            >
              {{ row.isSensitive ? $t('是') : $t('否') }}
            </Tag>
          </template>
        </TableColumn>
        <TableColumn
          label="Value"
          min-width="100"
        >
          <template #default="{ row, rowIndex }: { row: VariableItem; rowIndex?: number }">
            <Form.FormItem
              v-if="editStates[row.id].editing"
              class="mb-0"
              :class="{ 'has-sensitive-value-input': shouldOffsetSensitiveValueErrorIcon(row) }"
              error-display-type="tooltips"
              label=""
              :property="`${rowIndex}.value`"
            >
              <!-- 加密 -->
              <SensitiveValueInput
                v-if="shouldUseSensitiveValueInput(row)"
                v-model="row.value"
                v-model:modified="sensitiveValueModifiedStates[row.id]"
                :mode="getSensitiveValueInputMode(row)"
                @enter="saveRow(row)"
                @reset="clearValueValidate(rowIndex)"
              />
              <Input
                v-else
                v-model.trim="row.value"
                class="w-full relative z-99"
                clearable
                @enter="saveRow(row)"
              />
            </Form.FormItem>
            <template v-else>
              <!-- 是否敏感，占位展示 -->
              <SensitiveValuePlaceholder v-if="row.isSensitive" />
              <div
                v-else
                class="text-[12px] flex items-center gap-[5px]"
              >
                <div
                  class="whitespace-nowrap overflow-hidden text-ellipsis"
                  :class="{ 'text-[#DCDEE5]': !row?.value }"
                >
                  {{ row?.value || '--' }}
                </div>
                <!-- 复制 -->
                <Button
                  class="ml-[6px] shrink-0"
                  text
                  @click="copyText(row.value)"
                >
                  <Copy
                    class="group-hover:opacity-100 opacity-0 transition text-[#3A84FF]"
                    :title="$t('复制')"
                  />
                </Button>
              </div>
            </template>
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('描述')"
          min-width="100"
        >
          <template #default="{ row }: { row: VariableItem }">
            <Input
              v-if="editStates[row.id].editing"
              v-model.trim="row.description"
              class="w-full"
              @enter="saveRow(row)"
            />
            <span
              v-else
              class="text-[12px]"
              >{{ row?.description || '--' }}</span
            >
          </template>
        </TableColumn>
        <TableColumn
          v-if="props.editable !== false"
          :label="$t('操作')"
          width="160"
        >
          <template #default="{ row }">
            <template v-if="!editStates[row.id].editing">
              <Button
                class="mr-[10px]"
                text
                theme="primary"
                @click="editRow(row)"
              >
                {{ $t('编辑') }}
              </Button>
              <PopConfirm
                :content="$t('确认删除该环境变量？')"
                trigger="click"
                width="288"
                @confirm="deleteVariable(row)"
              >
                <Button
                  text
                  theme="primary"
                >
                  {{ $t('删除') }}
                </Button>
              </PopConfirm>
            </template>
            <!-- 编辑态：保存、取消 -->
            <template v-else>
              <Button
                text
                theme="primary"
                @click="saveRow(row)"
              >
                {{ $t('保存') }}
              </Button>
              <Button
                class="ml-[10px]"
                text
                theme="primary"
                @click="cancelEdit(row)"
              >
                {{ $t('取消') }}
              </Button>
            </template>
          </template>
        </TableColumn>
      </Table>
    </Form>
  </div>
</template>

<script lang="ts" setup>
  import { computed, nextTick, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Form, Input, PopConfirm, Switcher, Tag } from 'bkui-vue';
  import { Copy, InfoLine } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { BKMS_REGEX } from '~/common/const';
  import { useCopy } from '~/composables/use-copy';

  import SensitiveValueInput from './sensitive-value-input.vue';
  import SensitiveValuePlaceholder from './sensitive-value-placeholder.vue';

  import type { EnvVarConflictedInfoOutputObj } from '~/@types/envvars';

  // 定义环境变量配置接口
  export interface EnvVariableConfig {
    // 冲突信息（可选）
    conflictedInfo?: EnvVarConflictedInfoOutputObj;
    // 变量描述
    description: string;
    // 是否为敏感变量
    isSensitive: boolean;
    // 变量名
    key: string;
    // 变量值
    value?: string;
  }

  // 扩展 Input 组件类型，包含 focus 方法
  interface InputWithFocus extends InstanceType<typeof Input> {
    focus: () => void;
  }

  const props = withDefaults(
    defineProps<{
      // 编辑时是否禁用 key 列，默认为 false
      disableKeyEdit?: boolean;
      // 是否可编辑，默认为true
      editable?: boolean;
      // 变量列表
      list: EnvVariableConfig[];
    }>(),
    {
      editable: true,
      disableKeyEdit: false,
    },
  );

  const emit = defineEmits<{
    add: [item: EnvVariableConfig];
    delete: [item: EnvVariableConfig];
    edit: [item: EnvVariableConfig, originalItem: EnvVariableConfig];
    update: [value: EnvVariableConfig[]];
  }>();

  const { copyText } = useCopy();
  const { t } = useI18n();

  // 编辑状态管理
  interface EditState {
    // 是否为新增行（未保存）- true: 取消时删除整行, false: 取消时恢复原始数据
    add: boolean;
    // 是否处于编辑状态
    editing: boolean;
    // 原始数据（用于取消编辑时恢复）
    originalData: EnvVariableConfig | null;
  }

  // 内部使用的变量项类型（带唯一ID）
  interface VariableItem extends Omit<EnvVariableConfig, 'value'> {
    id: string; // 唯一标识
    value: string;
  }

  // 表单校验规则
  const rules = ref({
    key: [
      { required: true, message: t('Key 必填'), trigger: 'blur' },
      {
        validator: (value: string) => BKMS_REGEX.envVarKeyRegex.test(value),
        message: t('字母或下划线开头，仅允许字母、数字、下划线'),
        trigger: 'blur',
      },
    ],
  });

  // 冲突来源映射表
  const conflictedSourceMap: Record<string, string> = {
    builtin: '内置',
    scopedWorkspace: '工作空间',
    scopedEnvType: '环境类型',
    scopedEnv: '环境',
    app: '应用',
  };

  // 拼接完整冲突 tooltip 文本
  function buildEnvVarConflictTooltip(conflictedInfo: EnvVarConflictedInfoOutputObj): string {
    const { conflictedSources = [], overrideConflicted, conflictedDetail } = conflictedInfo;
    const prefix = overrideConflicted
      ? t('当前值已覆盖其他来源的同名变量，来源')
      : t('当前值已被其他来源的同名变量覆盖，来源');
    const sourceText = conflictedSources
      .map(({ source, sourceValue }) => {
        const label = source ? conflictedSourceMap[source] : undefined;
        return label ? (sourceValue ? `${sourceValue} ${t(label)}` : t(label)) : source;
      })
      .join(', ');
    const firstLine = sourceText ? `${prefix}: ${sourceText}` : prefix;
    const detail = conflictedDetail?.trim();
    return detail ? `${firstLine}\n${detail}` : firstLine;
  }

  // 判断是否有冲突
  function hasConflict(row: VariableItem): boolean {
    return !!(row.conflictedInfo && row.conflictedInfo.conflictedSources?.length);
  }

  // 数据列表
  const variableList = ref<VariableItem[]>([]);

  // 编辑状态 Map（状态与数据分离）
  const editStateMap = ref<Map<string, EditState>>(new Map());

  const sensitiveValueModifiedStates = ref<Record<string, boolean>>({});
  const submittedRowId = ref<string>();

  // 计算属性：将编辑状态转换为响应式对象，便于模板使用且有缓存
  const editStates = computed(() => {
    const states: Record<string, EditState> = {};
    variableList.value.forEach(item => {
      states[item.id] = editStateMap.value.get(item.id) || {
        editing: false,
        add: false,
        originalData: null,
      };
    });
    return states;
  });

  const keyInputRef = ref<InputWithFocus>();

  // 清理指定行 Value 字段的校验状态
  function clearValueValidate(rowIndex?: number) {
    if (rowIndex === undefined) {
      return;
    }
    nextTick(() => {
      formRef.value?.clearValidate(`${rowIndex}.value`);
    });
  }

  function createVariableItem(item: EnvVariableConfig, id = generateId()): VariableItem {
    return {
      ...item,
      id,
      value: item.value ?? '',
    };
  }

  // 删除行的编辑状态
  function deleteEditState(id: string) {
    editStateMap.value.delete(id);
    delete sensitiveValueModifiedStates.value[id];
    if (submittedRowId.value === id) {
      submittedRowId.value = undefined;
    }
  }

  // 聚焦到 Key Input
  function focusKeyInput() {
    nextTick(() => {
      if (keyInputRef.value) {
        keyInputRef.value.focus();
      }
    });
  }

  // 移除内部 id，生成对外透传的数据
  function getCleanRow(row: VariableItem): EnvVariableConfig {
    const { id, ...cleanRow } = row;
    return cleanRow;
  }

  // 获取行的编辑状态
  function getEditState(id: string): EditState {
    return editStateMap.value.get(id) || { editing: false, add: false, originalData: null };
  }

  // 根据行状态决定敏感输入组件的新建或编辑模式
  function getSensitiveValueInputMode(row: VariableItem) {
    return getEditState(row.id).add ? 'create' : 'edit';
  }

  // 同步“是否敏感”切换后 Value 是否需要作为新值提交
  function handleSensitiveChange(row: VariableItem) {
    const state = getEditState(row.id);
    if (state.add) {
      sensitiveValueModifiedStates.value[row.id] = true;
      return;
    }
    if (row.isSensitive && state.originalData?.isSensitive === false) {
      sensitiveValueModifiedStates.value[row.id] = true;
      return;
    }
    sensitiveValueModifiedStates.value[row.id] = false;
  }

  // 已有敏感变量不允许在编辑态关闭敏感开关
  function isSensitiveSwitcherDisabled(row: VariableItem) {
    return getEditState(row.id).originalData?.isSensitive === true;
  }

  // 设置行的编辑状态
  function setEditState(id: string, state: Partial<EditState>) {
    const currentState = getEditState(id);
    editStateMap.value.set(id, { ...currentState, ...state });
  }

  // 敏感输入编辑态需要为错误图标预留恢复按钮空间
  function shouldOffsetSensitiveValueErrorIcon(row: VariableItem) {
    return (
      getEditState(row.id).editing &&
      shouldUseSensitiveValueInput(row) &&
      getSensitiveValueInputMode(row) === 'edit' &&
      sensitiveValueModifiedStates.value[row.id]
    );
  }

  // 判断当前行是否需要使用敏感 Value 输入组件
  function shouldUseSensitiveValueInput(row: VariableItem) {
    return row.isSensitive && (getEditState(row.id).add || getEditState(row.id).originalData?.isSensitive !== false);
  }

  function syncVariableList(newList: EnvVariableConfig[]) {
    const currentRows = variableList.value;
    const submittedId = submittedRowId.value;

    // 外部列表刷新时保留未提交的编辑行，避免输入框因行重建而失焦。
    const editingRow = currentRows.find(row => {
      const state = getEditState(row.id);
      return state.editing && submittedId !== row.id;
    });
    const editingOriginalKey = editingRow ? getEditState(editingRow.id).originalData?.key : undefined;
    const nextList: VariableItem[] = [];

    newList.forEach(item => {
      const shouldKeepEditingRow =
        editingRow &&
        !nextList.includes(editingRow) &&
        (item.key === editingRow.key || item.key === editingOriginalKey);

      if (shouldKeepEditingRow) {
        nextList.push(editingRow);
        return;
      }

      const reusableRow = currentRows.find(row => row.id !== editingRow?.id && row.key === item.key);
      nextList.push(createVariableItem(item, reusableRow?.id));
    });

    // 搜索过滤等场景下，编辑行可能不在外部列表中，仍需临时展示。
    if (editingRow && !nextList.includes(editingRow)) {
      nextList.unshift(editingRow);
    }
    variableList.value = nextList;

    const visibleIds = new Set(variableList.value.map(row => row.id));
    Array.from(editStateMap.value.keys()).forEach(id => {
      if (!visibleIds.has(id) || submittedId === id) {
        deleteEditState(id);
      }
    });
    submittedRowId.value = undefined;
  }

  // 监听props变化
  watch(
    () => props.list,
    newList => {
      syncVariableList(newList);
    },
    { immediate: true },
  );

  // 生成唯一ID
  function generateId(): string {
    return Date.now().toString() + Math.random().toString(36).slice(2, 11);
  }

  // 表格验证
  const formRef = ref<InstanceType<typeof Form>>();

  // 添加变量
  function addVariable() {
    // 是否已存在新加的行
    const hasAddRow = Array.from(editStateMap.value.values()).some(state => state.add);
    if (hasAddRow) {
      return;
    }

    // 取消其他行的编辑状态
    variableList.value.forEach(item => {
      const state = getEditState(item.id);
      if (state.editing) {
        if (state.originalData) {
          // 恢复原始数据
          Object.assign(item, state.originalData);
        }
        setEditState(item.id, { editing: false, originalData: null });
      }
    });

    const newId = generateId();
    const newVariable: VariableItem = {
      id: newId,
      isSensitive: false,
      key: '',
      value: '',
      description: '',
    };

    variableList.value.unshift(newVariable);
    setEditState(newId, { editing: true, add: true, originalData: null });
    sensitiveValueModifiedStates.value[newId] = true;

    // 聚焦到新添加行
    focusKeyInput();
  }

  // 取消当前行编辑，新增行删除，已有行恢复原始数据
  function cancelEdit(row: VariableItem) {
    const state = getEditState(row.id);

    // 新加行，直接删除
    if (state.add) {
      const index = variableList.value.findIndex(item => item.id === row.id);
      if (index > -1) {
        variableList.value.splice(index, 1);
      }
      deleteEditState(row.id);
      return;
    }

    // 非新加行，恢复原始数据
    if (state.originalData) {
      Object.assign(row, state.originalData);
    }
    sensitiveValueModifiedStates.value[row.id] = false;
    setEditState(row.id, { editing: false, originalData: null });
  }

  // 删除变量
  function deleteVariable(row: VariableItem) {
    // 保存被删除的项数据（纯数据，不包含id）
    const cleanRow = getCleanRow(row);

    // 单个删除事件
    emit('delete', cleanRow);
    // 注意：编辑状态清理和列表移除由外部事件处理方负责，
    // 以便在 API 调用失败时能够正确回滚。
  }

  // 编辑单行
  function editRow(row: VariableItem) {
    // 取消其他行的编辑状态
    const itemsToRemove: string[] = [];

    variableList.value.forEach(item => {
      if (item.id !== row.id) {
        const state = getEditState(item.id);
        if (state.editing) {
          // 如果是新增行，标记为删除
          if (state.add) {
            itemsToRemove.push(item.id);
          } else {
            // 恢复原始数据
            if (state.originalData) {
              Object.assign(item, state.originalData);
            }
            setEditState(item.id, { editing: false, originalData: null });
          }
        }
      }
    });

    // 删除标记的新增行
    itemsToRemove.forEach(id => {
      const index = variableList.value.findIndex(item => item.id === id);
      if (index > -1) {
        variableList.value.splice(index, 1);
      }
      deleteEditState(id);
    });

    // 保存原始数据（不包含id）
    const cleanRow = getCleanRow(row);
    setEditState(row.id, {
      editing: true,
      add: false,
      originalData: { ...cleanRow },
    });
    sensitiveValueModifiedStates.value[row.id] = false;

    focusKeyInput();
  }

  // 还原数据（用于更新失败时回滚）
  function restoreData(originalData: EnvVariableConfig[]) {
    editStateMap.value.clear();
    sensitiveValueModifiedStates.value = {};
    submittedRowId.value = undefined;
    variableList.value = originalData.map(item => createVariableItem(item));
  }

  // 保存单行
  async function saveRow(row: VariableItem) {
    const valid = await tableValidate();
    if (!valid) {
      return;
    }

    const state = getEditState(row.id);
    const cleanRow = getCleanRow(row);
    if (row.isSensitive && !state.add && !sensitiveValueModifiedStates.value[row.id]) {
      delete cleanRow.value;
    }
    submittedRowId.value = row.id;

    if (state.add) {
      // 单个新增事件
      emit('add', cleanRow);
    } else {
      // 修改操作 - 传递新数据和原始数据
      emit('edit', cleanRow, state.originalData as EnvVariableConfig);
    }
    // 注意：编辑状态重置和列表刷新由外部事件处理方负责，
    // 以便在 API 调用失败时能够正确回滚。
  }

  // 执行整表校验，失败时统一返回 false
  async function tableValidate() {
    try {
      const valid = await formRef.value?.validate();
      return valid;
    } catch {
      return false;
    }
  }

  defineExpose({
    addVariable,
    restoreData,
    cancelEdit,
  });
</script>

<style lang="postcss" scoped>
  .editable-variable-table {
    :deep(.bk-form-error-tips) {
      z-index: 999;
    }

    :deep(.has-sensitive-value-input .bk-form-error-tips) {
      right: 30px;
    }
  }
</style>
