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
  <!-- 目前一个空间最多一台，用表格是为了后续支持多台时直接加行 -->
  <Table
    :data="tableData"
    :row-config="{ isHover: true }"
  >
    <template #empty>
      <div class="py-[16px] text-[14px] text-[#979ba5]">
        {{ $t('未关联构建机，当前使用公共构建机') }}
      </div>
    </template>
    <TableColumn
      field="displayName"
      :label="$t('构建机名称')"
      min-width="200"
    >
      <template #default="{ row }">
        {{ row.displayName || '--' }}
      </template>
    </TableColumn>
    <TableColumn
      field="ip"
      label="IP"
      min-width="140"
    >
      <template #default="{ row }">
        {{ row.ip || '--' }}
      </template>
    </TableColumn>
    <TableColumn
      field="osName"
      :label="$t('操作系统')"
      min-width="100"
    >
      <template #default="{ row }">
        {{ row.osName || '--' }}
      </template>
    </TableColumn>
    <TableColumn
      field="state"
      :label="$t('Agent 状态')"
      min-width="140"
    >
      <template #default="{ row }">
        <StatusIcon
          :status="row.state || 'unknown'"
          :status-color-map="BUILD_NODE_STATE_COLOR_MAP"
          :status-text-map="stateTextMap"
        />
      </template>
    </TableColumn>
    <TableColumn
      field="updater"
      :label="$t('最后操作人')"
      min-width="120"
    >
      <template #default="{ row }">
        {{ row.updater || '--' }}
      </template>
    </TableColumn>
    <TableColumn
      field="updatedAt"
      :label="$t('更新时间')"
      min-width="180"
    >
      <template #default="{ row }">
        {{ row.updatedAt ? filterTimeFormat(row.updatedAt) : '--' }}
      </template>
    </TableColumn>
    <TableColumn
      v-if="editable"
      :label="$t('操作')"
      width="120"
    >
      <template #default>
        <Button
          text
          theme="primary"
          @click="emit('unbind')"
        >
          {{ $t('解除关联') }}
        </Button>
      </template>
    </TableColumn>
  </Table>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { filterTimeFormat } from '~/common/util';
  import StatusIcon from '~/components/status-icon.vue';

  import { BUILD_NODE_STATE_COLOR_MAP } from '../constants';

  import type { BuildNodeBindingOutput } from '~/@types/v1/bkintegrations-bkci';

  const props = defineProps<{
    /** 当前关联的构建机，为空表示使用公共构建机 */
    binding?: BuildNodeBindingOutput | null;
    /** 是否展示操作列，没有编辑权限时不展示 */
    editable?: boolean;
  }>();

  const emit = defineEmits<{ unbind: [] }>();

  const { t } = useI18n();

  const tableData = computed(() => (props.binding ? [props.binding] : []));
  const stateTextMap = computed(() => ({
    online: t('在线'),
    offline: t('离线'),
    notFound: t('构建机已不存在'),
    unknown: t('未知'),
  }));
</script>
