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
  <div :class="['flex flex-col h-full min-w-0 w-full rounded-[2px]', border ? 'border-[1px] border-[#DCDEE5]' : '']">
    <div
      v-if="showHeader"
      class="flex-shrink-0 h-[40px] pl-[16px] rounded-t-sm border-b-[1px] text-[14px] font-bold border-[#DCDEE5] bg-[#F5F7FA] flex items-center"
    >
      {{ $t('环境变量') }}
    </div>
    <div class="p-[16px] overflow-auto flex-1 flex flex-col gap-[12px]">
      <slot name="alert">
        <Alert
          closable
          theme="info"
          :title="currentAlertTitle"
        />
      </slot>
      <Radio.Group
        v-model="curEnv"
        class="flex-wrap"
        type="capsule"
        @change="handleChange"
      >
        <Radio.Button
          v-for="item in envList"
          :key="item.id"
          :label="item.name"
        >
          {{ item.displayName }}
        </Radio.Button>
      </Radio.Group>
      <Input
        v-model.trim="searchValue"
        class="w-full flex-shrink-0"
        clearable
        :placeholder="$t('搜索变量名、变量值、描述')"
      >
        <template #suffix>
          <Search class="text-[16px] text-[#979BA5] mr-[6px] mt-[2px] hover:text-[#3A84FF]" />
        </template>
      </Input>
      <div
        ref="tableContentRef"
        v-bkloading="{ loading }"
        class="flex-1 min-h-0 min-w-0"
      >
        <Table
          ref="tableRef"
          :auto-resize="false"
          :data="tableDataMatchSearch"
          :height="tableHeight"
          row-class-name="group"
          :row-config="{
            isHover: true,
          }"
          :virtual-y-config="{ enabled: true, gt: 10 }"
        >
          <template #empty>
            <TableException
              :type="curExceptionType"
              @clear="() => (searchValue = '')"
              @refresh="fetchWorkspaceEnvVarList"
            >
            </TableException>
          </template>
          <TableColumn
            field="name"
            label="Key"
            min-width="80"
          >
            <template #default="{ row }">
              <div class="text-[12px] flex items-center gap-[5px]">
                <div
                  v-bk-tooltips="{
                    content: row?.description,
                    disabled: !row?.description,
                  }"
                  :class="[
                    'whitespace-nowrap overflow-hidden text-ellipsis',
                    row?.description ? 'border-b-1 border-b-dashed border-b-[#313238]' : '',
                  ]"
                >
                  {{ row?.key || '--' }}
                </div>
                <Tag
                  v-if="row.isBuiltin"
                  size="small"
                >
                  {{ $t('系统内置') }}
                </Tag>
                <Tag
                  v-if="row.isSensitive"
                  size="small"
                  theme="warning"
                >
                  {{ $t('敏感') }}
                </Tag>
                <!-- 多格式复制 -->
                <EnvVarCopyDropdown
                  v-if="hasMultipleCopyOptions"
                  :options="copyOptions!"
                  :selected-option-id="selectedCopyOptionId"
                  :variable-key="row.key"
                  @update:selected-option-id="selectedCopyOptionId = $event"
                />
                <!-- 单格式复制 -->
                <Button
                  v-else
                  class="ml-[6px] shrink-0"
                  text
                  @click="copyText(defaultCopyFormat(row.key))"
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
            field="value"
            label="Value"
            min-width="80"
          >
            <template #default="{ row }">
              <span class="text-[12px]">{{ row?.value || '--' }}</span>
            </template>
          </TableColumn>
        </Table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref, shallowRef, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Button, Input, Radio, Tag } from 'bkui-vue';
  import { Copy, Search } from 'bkui-vue/lib/icon';
  import { throttle } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { AppEnvVarOutputObj } from '~/@types/app';
  import { useCopy } from '~/composables/use-copy';
  import { useElementHeight } from '~/composables/use-element-height';
  import { type IInputKey, useTableSearchInput } from '~/composables/use-search';
  import useTableEmpty from '~/composables/use-table-empty';

  import EnvVarCopyDropdown from './env-var-copy-dropdown.vue';

  import type { EnvOutputObj } from '~/@types/env';
  import type { IAppType } from '~/composables/app-type';

  interface ICopyOption {
    description: string;
    id: string;
    recommended?: boolean;
    format: (key: string) => string;
  }

  interface IProps {
    /** 自定义提示文案 */
    alertTitle?: string;
    /** 应用类型，用于判断格式（如果不传则从 store 获取） */
    appType?: IAppType;
    /** 是否展示外层边框 */
    border?: boolean;
    /** 可选的环境变量复制格式，传入两个及以上选项时展示下拉复制控件 */
    copyOptions?: ICopyOption[];
    envList: EnvOutputObj[];
    /** 可通过表达式 expressTemplate 引用。 */
    expressTemplate?: string;
    /** 是否展示头部标题区域 */
    showHeader?: boolean;
    /** hoverCopy时 复制的文本format */
    copyFormat?: (key: string) => string;
    /** 自定义请求函数 */
    customRequestFn: (env: string) => Promise<AppEnvVarOutputObj[]>;
  }
  const props = withDefaults(defineProps<IProps>(), {
    showHeader: true,
    border: true,
  });
  const { t } = useI18n();

  const defaultExpressTemplate = computed(() => {
    if (props.expressTemplate) return props.expressTemplate;
    return '${{ env.<Key> }}';
  });

  const currentAlertTitle = computed(() => {
    if (props.alertTitle) return props.alertTitle;
    return t('可通过表达式 {0} 引用。', [defaultExpressTemplate.value]);
  });

  const defaultCopyFormat = computed(() => {
    if (props.copyFormat) return props.copyFormat;
    return (key: string) => `\${{ env.${key} }}`;
  });

  const hasMultipleCopyOptions = computed(() => props.copyOptions && props.copyOptions.length > 1);
  const selectedCopyOptionId = ref<string>();

  watch(
    () => props.copyOptions,
    options => {
      if (!options?.some(option => option.id === selectedCopyOptionId.value)) {
        selectedCopyOptionId.value = options?.[0]?.id;
      }
    },
    { immediate: true },
  );

  const emit = defineEmits<{
    envChange: [envName: string];
  }>();

  const { copyText } = useCopy();

  // 当前环境
  const curEnv = ref<string>('');
  const tableRef = ref();
  const loading = ref(false);

  // 搜索字段配置
  const searchKeys = shallowRef<IInputKey[]>([
    { field: 'key', id: 'key', fuzzy: true },
    { field: 'value', id: 'value', fuzzy: true },
    { field: 'description', id: 'description', fuzzy: true },
  ]);

  const envVariableList = ref<AppEnvVarOutputObj[]>([]);

  // 获取当前空间下的环境变量列表[环境变量]
  async function fetchWorkspaceEnvVarList() {
    loading.value = true;
    try {
      envVariableList.value = await props
        .customRequestFn(curEnv.value)
        .then(data => {
          clearErrorType();
          return data;
        })
        .catch(() => {
          setTypeToError();
          return [];
        });
    } finally {
      loading.value = false;
    }
  }

  // 节流处理，并触发 envChange 事件
  const handleChange = throttle(() => {
    fetchWorkspaceEnvVarList();
    emit('envChange', curEnv.value);
  }, 500);

  // 设置当前环境
  function setCurEnv(envName: string) {
    curEnv.value = envName;
    fetchWorkspaceEnvVarList();
  }

  const { searchValue, tableDataMatchSearch } = useTableSearchInput(envVariableList, searchKeys);

  const tableContentRef = ref<HTMLElement>();
  const { height: tableContentHeight } = useElementHeight(tableContentRef, { defaultHeight: 300 });
  // 计算表格高度
  const tableHeight = computed(() => Math.min(tableContentHeight.value, (tableDataMatchSearch.value.length + 1) * 40));

  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });

  watch(
    () => props.envList,
    val => {
      if (val.length > 0) {
        curEnv.value = val[0]!.name ?? '';
        fetchWorkspaceEnvVarList();
      }
    },
    { deep: true, immediate: true },
  );

  defineExpose({
    reRefreshTable: () => tableRef.value?.getVxeTableInstance().recalculate(true),
    setCurEnv,
  });
</script>

<style lang="postcss" scoped>
  :deep(.vxe-table--body) {
    width: 100%;
  }
</style>
