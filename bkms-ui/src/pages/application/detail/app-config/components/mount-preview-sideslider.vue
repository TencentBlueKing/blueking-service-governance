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
    :width="960"
  >
    <template #header>
      <DividerHeader
        :title="$t('挂载预览')"
        :title-size="16"
      >
        <span class="truncate">{{ $t('按环境查看容器内最终挂载的配置文件') }}</span>
      </DividerHeader>
    </template>
    <div
      class="overflow-y-auto px-[24px] py-[20px]"
      style="max-height: calc(100vh - 106px)"
    >
      <Radio.Group
        v-model="previewEnvName"
        class="preview-env-radio mb-[16px] !flex w-full flex-wrap gap-[4px]"
        type="capsule"
      >
        <Radio.Button
          v-for="env in envList"
          :key="env.name"
          :label="env.name"
        >
          {{ env.displayName }}
        </Radio.Button>
      </Radio.Group>
      <Table
        v-bkloading="{ loading }"
        :data="rows"
        :row-height="48"
        :show-overflow="false"
      >
        <template #empty><TableException type="empty" /></template>
        <TableColumn
          :label="$t('容器内路径')"
          min-width="280"
        >
          <template #default="{ row }">
            <div class="leading-[18px]">
              <span class="text-[#313238]">{{ row.name || '--' }}</span>
              <div class="text-[#979BA5]">{{ row.mountDir || '--' }}</div>
            </div>
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('来源')"
          :width="150"
        >
          <template #default="{ row }">
            <Tag :theme="row.configKind === 'framework' ? 'info' : ''">
              {{ row.configKind === 'framework' ? $t('框架配置') : $t('其他配置') }}
            </Tag>
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('内容来源')"
          min-width="180"
        >
          <template #default="{ row }">
            {{ getContentSourceLabel(row) }}
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('环境独立实例')"
          :width="150"
        >
          <template #default="{ row }">
            <Tag :theme="row.hasEnvFile ? 'warning' : 'default'">
              {{ row.hasEnvFile ? $t('已创建') : $t('使用默认配置') }}
            </Tag>
          </template>
        </TableColumn>
      </Table>
      <div class="mt-[12px] text-[12px] text-[#979BA5]">
        {{ $t('共 {0} 个文件；未挂载到该环境的配置文件不会出现在容器内。', [rows.length]) }}
      </div>
    </div>
  </Sideslider>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Radio, Sideslider, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { AppConfigFileDefsService } from '~/api/modules/v1';

  import type { MountPreviewItemObj } from '~/@types/v1/app-config-file-defs';
  import type { EnvOutput } from '~/@types/v1/env';

  /** 预览表格行：补充框架配置的展示字段 */
  interface PreviewRow extends MountPreviewItemObj {
    mountDir: string;
    name: string;
  }

  const props = withDefaults(
    defineProps<{
      /** 应用 ID */
      appId: string;
      /** 可切换预览的环境列表 */
      envList: EnvOutput[];
      /** 框架配置文件名（用于展示兜底） */
      frameworkFileName?: string;
      /** 框架配置挂载路径 */
      frameworkFilePath?: string;
      isShow: boolean;
    }>(),
    { frameworkFileName: '', frameworkFilePath: '' },
  );
  const emit = defineEmits<{ 'update:isShow': [value: boolean] }>();
  const { t } = useI18n();

  // 抽屉显隐双向绑定
  const visible = computed({
    get: () => props.isShow,
    set: value => emit('update:isShow', value),
  });
  const loading = ref(false);
  /** 当前预览的环境名 */
  const previewEnvName = ref('');
  /** 预览数据行 */
  const rows = ref<PreviewRow[]>([]);
  let previewRequestID = 0;

  /** 按环境拉取挂载预览数据；框架配置使用外部传入的路径/文件名兜底 */
  async function fetchPreview() {
    const requestID = ++previewRequestID;
    const requestedAppID = props.appId;
    const requestedEnvName = previewEnvName.value;
    if (!requestedAppID || !visible.value) return;
    if (!previewEnvName.value) {
      rows.value = [];
      return;
    }
    loading.value = true;
    try {
      const result = await AppConfigFileDefsService.getMountPreview(
        { appID: requestedAppID, envName: requestedEnvName },
        { needRes: true },
      );
      if (
        requestID !== previewRequestID ||
        !visible.value ||
        requestedAppID !== props.appId ||
        requestedEnvName !== previewEnvName.value
      ) {
        return;
      }
      rows.value = (result.items || []).map(item => ({
        ...item,
        mountDir:
          item.configKind === 'framework' ? props.frameworkFilePath || item.mountDir || '' : item.mountDir || '',
        name: item.configKind === 'framework' ? props.frameworkFileName || item.name || '' : item.name || '',
      }));
    } catch {
      if (requestID === previewRequestID) rows.value = [];
    } finally {
      if (requestID === previewRequestID) loading.value = false;
    }
  }

  /** 根据内容来源类型返回展示文案 */
  function getContentSourceLabel(row: PreviewRow) {
    if (row.contentSource === 'overlay') return t('默认配置 + 本环境差异项');
    if (row.contentSource === 'overwrite') return t('本环境独立内容');
    if (row.contentSource === 'normal') return t('默认配置');
    return row.contentSource || '--';
  }

  // 打开抽屉或环境列表变化时，若当前环境已不在列表中则回退到第一个环境，并重新拉取数据。
  watch(
    [() => props.isShow, () => props.envList],
    ([isShow]) => {
      if (!isShow) {
        previewRequestID += 1;
        loading.value = false;
        return;
      }
      if (!props.envList.some(env => env.name === previewEnvName.value)) {
        const nextEnvName = props.envList[0]?.name || '';
        if (nextEnvName !== previewEnvName.value) {
          previewEnvName.value = nextEnvName;
          return;
        }
      }
      fetchPreview();
    },
    { deep: true },
  );
  // 切换环境时重新拉取预览
  watch(previewEnvName, fetchPreview);
</script>

<style lang="postcss" scoped>
  :deep(.preview-env-radio.bk-radio-capsule .bk-radio-button::before) {
    display: none;
  }
</style>
