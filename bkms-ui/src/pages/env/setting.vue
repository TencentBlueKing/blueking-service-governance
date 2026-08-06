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
  <Skeleton :loading="isLoading">
    <template #loading>
      <Layout.table />
      <Layout.shape class="mt-[16px]" />
    </template>
    <div class="bg-[#fff]">
      <Alert
        class="mb-[16px]"
        theme="info"
      >
        {{
          $t(
            '环境最终生效的变量由系统内置、公共与自定义变量共同组成。此处配置的自定义变量优先级最高，同名时将覆盖其他配置。',
          )
        }}
        <Button
          text
          theme="primary"
          @click="showEnvBgVarsSlider = true"
        >
          {{ $t('查看内置与公共环境变量') }}
        </Button>
      </Alert>

      <div class="flex justify-between">
        <Button
          class="mb-[16px]"
          theme="primary"
          @click="handleAddVariable"
        >
          <Plus
            :height="24"
            :width="24"
          />
          {{ $t('新增自定义变量') }}
        </Button>
        <Input
          v-model.trim="searchKeyword"
          class="w-[430px]"
          clearable
          :placeholder="$t('搜索变量名、变量值、描述')"
        >
          <template #suffix>
            <Search class="text-[16px] text-[#979BA5] mr-[6px] mt-[2px] hover:text-[#3A84FF]" />
          </template>
        </Input>
      </div>

      <!-- 变量管理-可编辑 -->
      <EditableVariableTable
        ref="editableTableRef"
        :disable-key-edit="true"
        :list="filteredList"
        @add="handleAddItem"
        @delete="handleDeleteItem"
        @edit="handleEditItem"
      />

      <!-- 环境变量侧栏 -->
      <EnvBgVarsSideslider
        v-model:visible="showEnvBgVarsSlider"
        :env-id="env"
        :title="$t('内置与公共环境变量')"
      />
    </div>
  </Skeleton>
</template>
<script lang="ts" setup>
  import { ref, watch } from 'vue';

  import { Alert, Button, Input, Message } from 'bkui-vue';
  import { Plus, Search } from 'bkui-vue/lib/icon';
  import { cloneDeep } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { EnvvarsService } from '~/api/modules/v1';
  import { sortByDate } from '~/common/util';
  import EditableVariableTable, { type EnvVariableConfig } from '~/components/editable-variable-table/index.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { type IInputKey, useTableSearchInput } from '~/composables/use-search';
  import { useSpaceStore } from '~/stores/space';

  import EnvBgVarsSideslider from '../application/detail/base-info/trpc/env-bg-vars-sideslider.vue';

  const props = defineProps<{
    env: string;
    envName: string;
    workspace: string;
  }>();

  const { t } = useI18n();
  const spaceStore = useSpaceStore();

  // 环境变量列表
  const variableList = ref<EnvVariableConfig[]>([]);
  const isLoading = ref(false);
  const showEnvBgVarsSlider = ref(false);

  /** 搜索配置 */
  const searchKeys = ref<IInputKey[]>([
    { field: 'key', id: 'key', fuzzy: true },
    { field: 'value', id: 'value', fuzzy: true },
    { field: 'description', id: 'description', fuzzy: true },
  ]);
  const { searchValue: searchKeyword, tableDataMatchSearch: filteredList } = useTableSearchInput(
    variableList,
    searchKeys,
  );
  // key → scopedEnvVarID 映射，用于编辑/删除时获取 ID
  const scopedEnvVarIdMap = ref<Map<string, string>>(new Map());

  // 获取环境变量列表
  async function getEnvConfigList() {
    if (!props.env) return;
    isLoading.value = true;
    const list = await EnvvarsService.listDetailedEnvScopedEnvVars({
      envID: props.env,
    }).catch(() => []);
    sortByDate(list, item => item.scopedEnvVar?.createdAt);
    // 维护 key → scopedEnvVarID 映射
    const idMap = new Map<string, string>();
    const mappedList: EnvVariableConfig[] = [];
    list.forEach(item => {
      const { scopedEnvVar, conflictedInfo } = item;
      if (!scopedEnvVar) return;
      idMap.set(scopedEnvVar.key ?? '', scopedEnvVar.id ?? '');
      mappedList.push({
        key: scopedEnvVar.key ?? '',
        value: scopedEnvVar.value ?? '',
        description: scopedEnvVar.description ?? '',
        isSensitive: scopedEnvVar.isSensitive ?? false,
        conflictedInfo: conflictedInfo || undefined,
      });
    });
    scopedEnvVarIdMap.value = idMap;
    variableList.value = mappedList;
    isLoading.value = false;
  }

  const editableTableRef = ref<InstanceType<typeof EditableVariableTable>>();

  // 新增
  function handleAddItem(item: EnvVariableConfig) {
    handleEnvVarOperation(
      () =>
        EnvvarsService.createScopedEnvVar({
          workspaceID: props.workspace,
          scopeType: 'env',
          scopeValue: props.envName,
          key: item.key,
          value: item.value,
          description: item.description,
          isSensitive: item.isSensitive,
        }),
      t('新增成功'),
    );
  }

  // 添加变量
  function handleAddVariable() {
    editableTableRef.value?.addVariable();
  }

  // 删除
  function handleDeleteItem(item: EnvVariableConfig) {
    const scopedEnvVarID = scopedEnvVarIdMap.value.get(item.key);
    if (!scopedEnvVarID) return;
    handleEnvVarOperation(
      () =>
        EnvvarsService.deleteScopedEnvVar({
          workspaceID: props.workspace,
          scopedEnvVarID,
        }),
      t('删除成功'),
    );
  }

  // 编辑 - 当前版本 key 不可修改
  function handleEditItem(newItem: EnvVariableConfig, originalItem: EnvVariableConfig) {
    const scopedEnvVarID = scopedEnvVarIdMap.value.get(originalItem.key);
    if (!scopedEnvVarID) return;

    handleEnvVarOperation(
      () =>
        EnvvarsService.updateScopedEnvVar({
          workspaceID: props.workspace,
          scopedEnvVarID,
          key: newItem.key,
          value: newItem.value,
          description: newItem.description,
          isSensitive: newItem.isSensitive,
        }),
      t('修改成功'),
    );
  }

  // 环境变量操作
  async function handleEnvVarOperation(operation: () => Promise<void>, successText: string) {
    try {
      await operation();
      getEnvConfigList();
      Message({
        message: successText,
        theme: 'success',
        delay: 1500,
      });
    } catch (err: unknown) {
      const error = err as { error?: { message?: string }; message?: string };
      Message({
        message: error?.message || error?.error?.message,
        theme: 'error',
      });
      variableList.value = cloneDeep(variableList.value);
    }
  }

  // 初始化
  watch(
    [() => spaceStore.currentSpace, () => props.env],
    async () => {
      await getEnvConfigList();
    },
    { immediate: true },
  );
</script>
