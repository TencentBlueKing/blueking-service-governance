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
  <div class="flex flex-col gap-[16px] p-[16px] bg-[#FFF] shadow-[0_2px_4px_0_#1919290d]">
    <Alert theme="info">
      {{ $t('应用变量优先级最高，同名时将覆盖该应用继承的环境级别变量。') }}
      <Button
        text
        theme="primary"
        @click="showEnvBgVarsSlider = true"
      >
        {{ $t('查看环境级变量') }}
      </Button>
    </Alert>
    <div class="flex items-center justify-between">
      <Button
        theme="primary"
        @click="handleAddVariable"
      >
        <Plus
          :height="24"
          :width="24"
        />
        {{ $t('新增应用变量') }}
      </Button>
      <Input
        v-model.trim="searchKeyword"
        class="w-[420px]"
        clearable
        :placeholder="$t('搜索变量名、变量值、描述')"
      >
        <template #suffix>
          <Search class="text-[16px] text-[#979BA5] mr-[6px] mt-[2px] hover:text-[#3A84FF]" />
        </template>
      </Input>
    </div>
    <EditableVariableTable
      ref="editableTableRef"
      :editable="true"
      :list="variableList"
      @add="handleAdd"
      @delete="handleDelete"
      @edit="handleEdit"
    />

    <!-- 环境级变量侧栏 -->
    <EnvBgVarsSideslider
      v-model:visible="showEnvBgVarsSlider"
      :alert-text="$t('应用从环境继承的所有生效变量（包含环境自定义、公共及系统内置变量）。')"
      :app-id="appDetailStore.appID"
      source="app"
      :title="$t('环境级变量')"
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Alert, Button, Input, Message } from 'bkui-vue';
  import { Plus, Search } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { AppEnvVarDetailedOutputObj } from '~/@types/v1/envvars';
  import { EnvvarsService } from '~/api/modules/v1';
  import { sortByDate } from '~/common/util';
  import EditableVariableTable, { type EnvVariableConfig } from '~/components/editable-variable-table/index.vue';
  import { type IInputKey, useTableSearchInput } from '~/composables/use-search';
  import { useAppDetail } from '~/stores/app-detail';

  import EnvBgVarsSideslider from './env-bg-vars-sideslider.vue';

  const appDetailStore = useAppDetail();
  const { t } = useI18n();

  // 自定义环境变量列表 - 通过 ListDetailedAppEnvVars 接口获取
  const customEnvVarList = ref<EnvVariableConfig[]>([]);
  async function fetchAppEnvVarList() {
    if (!appDetailStore.appID) return;
    const list: AppEnvVarDetailedOutputObj[] = await EnvvarsService.listDetailedAppEnvVars({
      appID: appDetailStore.appID,
    }).catch(() => []);
    sortByDate(list, item => item.appEnvVar?.createdAt);
    customEnvVarList.value = (list || []).map((item: AppEnvVarDetailedOutputObj) => ({
      key: item.appEnvVar?.key || '',
      value: item.appEnvVar?.value || '',
      description: item.appEnvVar?.description || '',
      isSensitive: item.appEnvVar?.isSensitive ?? false,
      conflictedInfo: item.conflictedInfo || undefined,
    }));
  }

  async function handleAdd(data: EnvVariableConfig) {
    if (!appDetailStore.appID) return;
    await EnvvarsService.createAppEnvVars({
      appID: appDetailStore.appID,
      key: data.key,
      value: data.value,
      description: data.description,
      isSensitive: data.isSensitive,
    });
    Message({ theme: 'success', message: t('新增成功') });
    await fetchAppEnvVarList();
  }
  async function handleDelete(data: EnvVariableConfig) {
    if (!appDetailStore.appID) return;
    await EnvvarsService.deleteAppEnvVars({
      appID: appDetailStore.appID,
      key: data.key,
    });
    Message({ theme: 'success', message: t('删除成功') });
    await fetchAppEnvVarList();
  }
  async function handleEdit(data: EnvVariableConfig, originalData: EnvVariableConfig) {
    if (!appDetailStore.appID) return;
    try {
      await EnvvarsService.updateAppEnvVars({
        appID: appDetailStore.appID,
        key: originalData.key, // 旧 key（路径参数）
        updatedKey: data.key, // 新 key（可能重命名）
        value: data.value,
        description: data.description,
        isSensitive: data.isSensitive,
      });
      Message({ theme: 'success', message: t('修改成功') });
      await fetchAppEnvVarList();
    } catch (error: unknown) {
      console.warn('error', error);
      // 更新失败，还原该行为编辑前的原始数据
      editableTableRef.value?.restoreData(customEnvVarList.value);
    }
  }

  // 暴露刷新方法供父组件调用
  function refreshData() {
    fetchAppEnvVarList();
  }

  const searchKeys = ref<IInputKey[]>([
    { field: 'key', id: 'key', fuzzy: true },
    { field: 'value', id: 'value', fuzzy: true },
    { field: 'description', id: 'description', fuzzy: true },
  ]);
  const { searchValue: searchKeyword, tableDataMatchSearch: filteredEnvVars } = useTableSearchInput(
    customEnvVarList,
    searchKeys,
  );

  // 展示的变量列表
  const variableList = computed(() => filteredEnvVars.value);

  const editableTableRef = ref<InstanceType<typeof EditableVariableTable>>();
  // 添加变量
  function handleAddVariable() {
    editableTableRef.value?.addVariable();
  }

  // 查看环境级变量侧栏
  const showEnvBgVarsSlider = ref(false);

  // 监听 appID 变化后请求环境变量列表，确保 appID 已就绪
  watch(
    () => appDetailStore.appID,
    newAppID => {
      if (newAppID) fetchAppEnvVarList();
    },
    { immediate: true },
  );

  defineExpose({ refreshData });
</script>
