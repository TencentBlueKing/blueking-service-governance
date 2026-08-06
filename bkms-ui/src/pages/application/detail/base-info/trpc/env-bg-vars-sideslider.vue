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
  <Sideslider
    v-model:is-show="visible"
    quick-close
    render-directive="if"
    :title="title"
    :width="680"
    @hidden="handleHidden"
  >
    <div class="flex flex-col w-full p-[24px] h-full gap-[16px]">
      <Alert
        v-if="props.alertText"
        theme="info"
        :title="$t(props.alertText)"
      />
      <!-- 传入 envID 时，不展示环境选择 -->
      <OverflowCtrl v-if="!props.envId">
        <Radio.Group
          v-model="curEnvID"
          type="capsule"
        >
          <Radio.Button
            v-for="item in envList"
            :key="item.id"
            :label="item.id"
          >
            {{ item.displayName }}
          </Radio.Button>
        </Radio.Group>
      </OverflowCtrl>
      <Input
        v-model.trim="searchKeyword"
        class="w-full"
        clearable
        :placeholder="$t('搜索变量名、变量值、描述')"
      >
        <template #suffix>
          <Search class="text-[16px] text-[#979BA5] mr-[6px] mt-[2px] hover:text-[#3A84FF]" />
        </template>
      </Input>

      <!-- 单表格展示 -->
      <Table
        v-bkloading="{ loading: isLoading }"
        :data="filteredList"
      >
        <TableColumn
          field="key"
          label="Key"
          :min-width="220"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            <HoverCopy
              :copy-value="row.key"
              :text="row.key"
              :tooltip="row.description"
            >
              <Tag
                v-if="row.source === 'builtin'"
                class="ml-[4px]"
                size="small"
                theme="primary"
              >
                {{ $t('内置') }}
              </Tag>
            </HoverCopy>
          </template>
        </TableColumn>
        <TableColumn
          field="value"
          label="Value"
          :min-width="260"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            <HoverCopy
              :copy-value="row.value"
              :text="row.value"
            />
          </template>
        </TableColumn>
      </Table>
    </div>
  </Sideslider>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Input, Radio, Sideslider, Tag } from 'bkui-vue';
  import { Search } from 'bkui-vue/lib/icon';
  import { EnvOutput } from '~/@types/v1/env';
  import { BgEnvVarOutputObj } from '~/@types/v1/envvars';
  import { EnvService, EnvvarsService } from '~/api/modules/v1';
  import HoverCopy from '~/components/hover-copy.vue';
  import { type IInputKey, useTableSearchInput } from '~/composables/use-search';
  import { useSpaceStore } from '~/stores/space';

  const props = withDefaults(
    defineProps<{
      alertText?: string;
      appId?: string;
      envId?: string;
      source?: 'app' | 'env';
      title: string;
      visible: boolean;
    }>(),
    {
      source: 'env',
    },
  );
  const emit = defineEmits<{ (e: 'update:visible', value: boolean): void }>();

  const spaceStore = useSpaceStore();

  const visible = computed({
    get: () => props.visible,
    set: (val: boolean) => emit('update:visible', val),
  });

  const envList = ref<EnvOutput[]>([]);
  const curEnvID = ref('');
  const list = ref<BgEnvVarOutputObj[]>([]);
  const isLoading = ref(false);

  /** 搜索配置 */
  const searchKeys = ref<IInputKey[]>([
    { field: 'key', id: 'key', fuzzy: true },
    { field: 'value', id: 'value', fuzzy: true },
    { field: 'description', id: 'description', fuzzy: true },
  ]);
  const { searchValue: searchKeyword, tableDataMatchSearch: filteredList } = useTableSearchInput(list, searchKeys);

  /** 获取环境列表 */
  async function fetchEnvList() {
    const data = await EnvService.listEnvs({ workspaceID: spaceStore.currentSpace }).catch(() => []);
    envList.value = data;
    curEnvID.value = envList.value?.[0]?.id || '';
  }

  /** 获取环境级变量 */
  async function fetchList() {
    if (!curEnvID.value) return;
    isLoading.value = true;
    try {
      let data: BgEnvVarOutputObj[] = [];
      if (props.source === 'app') {
        const envName = envList.value.find(item => item.id === curEnvID.value)?.name;
        if (props.appId && envName) {
          data = await EnvvarsService.listAppBgEnvVars(
            {
              appID: props.appId,
              envName,
            },
            { validateCode: false },
          );
        }
      } else {
        data = await EnvvarsService.listEnvBgEnvVars({ envID: curEnvID.value }, { validateCode: false });
      }
      list.value = data;
    } catch {
      list.value = [];
    } finally {
      isLoading.value = false;
    }
  }

  /** 面板关闭时重置状态 */
  function handleHidden() {
    curEnvID.value = '';
    list.value = [];
    searchKeyword.value = '';
    if (!props.envId) {
      envList.value = [];
    }
  }

  /** 监听侧栏打开 */
  watch(
    () => props.visible,
    val => {
      if (val) {
        if (props.envId) {
          // 传入 envID 时, 直接展示当前环境下的变量列表
          curEnvID.value = props.envId;
        } else {
          fetchEnvList();
        }
      }
    },
  );

  /** 监听环境切换 */
  watch(curEnvID, () => {
    if (props.visible) fetchList();
  });

  /** appID 异步就绪后重新拉取应用视角的背景变量 */
  watch(
    () => props.appId,
    val => {
      if (props.visible && props.source === 'app' && curEnvID.value && val) fetchList();
    },
  );
</script>
