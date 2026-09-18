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
  <!-- 发布操作展示字段表格，其余操作展示操作前后 YAML Diff -->
  <Sideslider
    v-model:is-show="isShow"
    class="record-detail"
    :title="$t('操作详情')"
    :width="960"
  >
    <!-- 发布操作：以表格形式展示发布详情字段 -->
    <div
      v-if="record?.operationType === 'publish'"
      class="p-[20px]"
    >
      <Table :data="publishDetailRows">
        <TableColumn
          field="label"
          :label="$t('字段')"
          width="180"
        />
        <TableColumn
          field="value"
          :label="$t('值')"
          min-width="300"
        >
          <template #default="{ row }">
            <span
              v-if="row.field === 'result' && (record?.result === 'success' || record?.result === 'failed')"
              class="inline-flex items-center"
            >
              <StatusDotIcon
                :icon="record.result === 'success' ? 'normal' : 'abnormal'"
                :size="12"
              />
              {{ row.value }}
            </span>
            <span
              v-else
              class="break-all"
              >{{ row.value }}</span
            >
          </template>
        </TableColumn>
      </Table>
    </div>
    <MsEditor
      v-else
      class="w-[100%] !h-[calc(100vh-52px)] p-[20px]"
      is-diff
      :model-value="yamlAfter"
      :options="{
        enableSplitViewResizing: false,
        lineNumbersMinChars: 2,
      }"
      :original="yamlBefore"
      readonly
    >
      <template #title>
        <div class="grid grid-cols-2 gap-[18px] text-[12px] text-center leading-[22px]">
          <div class="w-[52px] h-[22px] bg-[#1E3567] rounded-[2px]">
            {{ $t('操作前') }}
          </div>
          <div class="w-[52px] h-[22px] bg-[#144628] rounded-[2px] text-[#3FC362]">
            {{ $t('操作后') }}
          </div>
        </div>
      </template>
    </MsEditor>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { convertToYaml, formatTimeByTimezone } from '~/common/util';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';
  import StatusDotIcon from '~/components/status-dot-icon.vue';

  import type { OperationRecordOutputObj } from '~/@types/v1/operation-audit';

  const props = defineProps<{
    envNameMapping: Record<string, string>;
    record?: OperationRecordOutputObj;
    resultDisplayName: string;
  }>();

  const isShow = defineModel<boolean>('isShow', { default: false });
  const { t } = useI18n();

  // 操作前 YAML 内容（发布操作无 diff）
  const yamlBefore = computed(() =>
    props.record?.operationType === 'publish' ? undefined : convertToYaml(props.record?.data?.before || ''),
  );

  // 操作后 YAML 内容
  const yamlAfter = computed(() =>
    props.record?.operationType === 'publish' ? undefined : convertToYaml(props.record?.data?.after || ''),
  );

  // 发布操作详情表格行数据；失败时追加"失败原因"
  const publishDetailRows = computed(() => {
    const row = props.record;
    if (!row || row.operationType !== 'publish') return [];
    const after = parsePublishAuditData(row.data?.after);
    const resultLabel =
      row.result === 'success'
        ? t('成功')
        : row.result === 'failed'
          ? t('失败')
          : props.resultDisplayName || row.result || '--';
    const details = [
      { field: 'binaryName', label: t('二进制'), value: textOrPlaceholder(after.binaryName) },
      { field: 'md5', label: 'MD5', value: textOrPlaceholder(after.md5) },
      {
        field: 'envName',
        label: t('环境'),
        value: props.envNameMapping[row.group?.envName || ''] || row.group?.envName || '--',
      },
      { field: 'resourceID', label: t('目标实例'), value: row.resourceID || '--' },
      { field: 'username', label: t('操作人'), value: row.username || '--' },
      {
        field: 'createdAt',
        label: t('操作时间'),
        value: row.createdAt ? formatTimeByTimezone(row.createdAt) : '--',
      },
      { field: 'result', label: t('操作结果'), value: resultLabel },
    ];
    if (row.result === 'failed') {
      details.push({ field: 'message', label: t('失败原因'), value: textOrPlaceholder(after.message) });
    }
    return details;
  });

  // 解析发布审计数据（JSON 字符串），非法或非对象内容返回空对象
  function parsePublishAuditData(value?: string): Record<string, unknown> {
    if (!value) return {};
    try {
      const parsed: unknown = JSON.parse(value);
      return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? (parsed as Record<string, unknown>) : {};
    } catch {
      return {};
    }
  }

  function textOrPlaceholder(value: unknown): string {
    return typeof value === 'string' && value ? value : '--';
  }
</script>
