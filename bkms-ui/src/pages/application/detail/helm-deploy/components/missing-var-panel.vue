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
  <div class="rounded-[2px] border border-[#F9D090] bg-[#FDF4E8] px-[12px]">
    <!-- Header -->
    <div
      class="flex cursor-pointer items-center justify-between py-[8px]"
      @click="collapsed = !collapsed"
    >
      <div class="flex items-center">
        <i class="bkms-icon bkms-icon-triangle-warning text-[12px] text-[#F59500]" />

        <span class="mx-[8px] text-base font-bold text-[#F59500]">
          {{ title }}
        </span>

        <span class="rounded-[2px] bg-[#FCE5C0] px-[8px] py-[2px] text-[12px] text-[#E38B02]">
          <i18n-t keypath="{0} 个变量">
            <span class="pl-[2px]">{{ data.length }}</span>
          </i18n-t>
        </span>
      </div>

      <div
        class="flex items-center text-[#979BA5] transition-transform duration-200"
        :class="{ 'rotate-180': !collapsed }"
      >
        <AngleDown class="text-[22px]" />
      </div>
    </div>

    <!-- Body -->
    <transition name="collapse">
      <div
        v-show="!collapsed"
        class="border-t border-[#F9D090] py-[8px]"
      >
        <!-- description -->
        <slot />

        <!-- table -->
        <Table
          class="missing-var-panel__table"
          :data="data"
          :pagination="showPagination ? pagination : undefined"
          @page-limit-change="handlePageLimitChange"
          @page-value-change="handlePageChange"
        >
          <TableColumn
            field="key"
            :label="$t('变量名')"
            :min-width="200"
            show-overflow-tooltip
          />
        </Table>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { AngleDown } from 'bkui-vue/lib/icon';

  interface MissingVarItem {
    key: string;
  }

  interface Props {
    data: MissingVarItem[];
    defaultExpanded?: boolean;
    pageSize?: number;
    title: string;
  }

  const props = withDefaults(defineProps<Props>(), {
    defaultExpanded: true,
    pageSize: 10,
  });

  const collapsed = ref(!props.defaultExpanded);

  const current = ref(1);

  const limit = ref(props.pageSize);

  const showPagination = computed(() => props.data.length > props.pageSize);

  const pagination = computed(() => ({
    current: current.value,
    limit: limit.value,
    count: props.data.length,
    limitList: [10, 20, 50],
  }));

  function handlePageChange(page: number) {
    current.value = page;
  }

  function handlePageLimitChange(limitValue: number) {
    limit.value = limitValue;
    current.value = 1;
  }

  /**
   * 数据变化时重置分页
   */
  watch(
    () => props.data,
    () => {
      current.value = 1;
    },
  );
</script>

<style scoped lang="postcss">
  /* 表格边框色 */
  :deep(.missing-var-panel__table) {
    --vxe-ui-table-border-color: #dcdee5;
  }

  /* 表头背景色 */
  :deep(.missing-var-panel__table .vxe-header--column) {
    background-color: #fafbfd;
  }

  /* 表格正文文字色 */
  :deep(.missing-var-panel__table .vxe-body--column .vxe-cell) {
    color: #4d4f56;
  }
</style>
