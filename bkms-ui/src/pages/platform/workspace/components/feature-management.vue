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
  <div
    v-show="active"
    class="p-[24px]"
  >
    <Input
      v-model="search"
      class="mb-[16px] max-w-[360px]"
      clearable
      :placeholder="t('请输入特性名称')"
      type="search"
    />
    <Table
      v-bkloading="{ loading: isLoading }"
      class="bg-white"
      :data="hasLoadError ? [] : visibleFeatures"
      :row-config="{ keyField: 'key' }"
      :row-height="48"
    >
      <template #empty>
        <TableException
          v-if="!isLoading"
          :type="tableExceptionType"
          @clear="search = ''"
          @refresh="fetchFeatures"
        />
      </template>
      <TableColumn
        field="displayName"
        :label="t('特性名称')"
        :min-width="240"
      />
      <TableColumn
        :label="t('默认值')"
        :width="160"
      >
        {{ t('关闭（开关状态）') }}
      </TableColumn>
      <TableColumn
        :label="t('状态')"
        :width="160"
      >
        <template #default="{ row }: { row: FeatureRow }">
          <Switcher
            :aria-checked="row.enabled === true"
            :aria-disabled="isRowLocked(row.key)"
            :aria-label="row.displayName"
            :disabled="isRowLocked(row.key)"
            role="switch"
            theme="primary"
            :value="row.enabled === true"
            @change="requestFeatureChange(row, $event)"
            @keydown.space.prevent="requestFeatureChange(row, !row.enabled)"
          />
        </template>
      </TableColumn>
      <TableColumn
        :label="t('操作')"
        :width="160"
      >
        <template #default="{ row }: { row: FeatureRow }">
          <Button
            :disabled="!row.enabled || isRowLocked(row.key)"
            text
            theme="primary"
            @click="requestFeatureChange(row, false)"
          >
            {{ t('恢复默认值') }}
          </Button>
        </template>
      </TableColumn>
    </Table>

    <Dialog
      v-model:is-show="isBscpConfirmVisible"
      :cancel-text="t('取消')"
      confirm-button-theme="danger"
      :confirm-text="t('关闭')"
      :quick-close="false"
      render-directive="if"
      :title="t('关闭 BSCP')"
      :width="480"
      @confirm="updateFeature('bscp', false)"
    >
      {{ t('关闭后，此空间下的应用将无法使用 BSCP 接口，下次部署将移除 BSCP 容器。') }}
    </Dialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Dialog, Input, Message, Switcher } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { PlatmgtService } from '~/api/modules/v1';
  import TableException from '~/components/table-exception.vue';

  import type { PlatCfgSwitchOutput } from '~/@types/v1/platmgt';

  type Feature = PlatCfgSwitchOutput & { key: string };
  type FeatureRow = Feature & { displayName: string };

  const props = defineProps<{ active: boolean; workspaceId: string }>();
  const { t } = useI18n();

  const features = ref<Feature[]>([]);
  const search = ref('');
  const isLoading = ref(false);
  const hasLoadError = ref(false);
  const pendingKeys = ref(new Set<string>());
  const isBscpConfirmVisible = ref(false);
  // 父级按 workspaceID 挂载组件。切换空间后忽略旧组件的回包。
  let isDisposed = false;

  const featureNames = computed(
    () =>
      new Map([
        ['bscp', t('BSCP')],
        ['rainbowTConf', t('七彩石')],
      ]),
  );
  const visibleFeatures = computed(() => {
    const keyword = search.value.trim().toLowerCase();
    return features.value
      .map(feature => ({ ...feature, displayName: featureNames.value.get(feature.key) || feature.name || feature.key }))
      .filter(feature => feature.displayName.toLowerCase().includes(keyword));
  });

  const tableExceptionType = computed(() => {
    if (hasLoadError.value) return 'error';
    return search.value.trim() ? 'search' : 'empty';
  });

  async function fetchFeatures() {
    // 当前组件有提交进行时跳过查询，避免较早读到的状态覆盖提交结果。
    if (isLoading.value || pendingKeys.value.size) return;
    isLoading.value = true;
    hasLoadError.value = false;
    try {
      const result = await PlatmgtService.listPlatCfgSwitches({ workspaceID: props.workspaceId });
      if (!isDisposed) features.value = result.filter((feature): feature is Feature => !!feature.key);
    } catch {
      if (!isDisposed) hasLoadError.value = true;
    } finally {
      if (!isDisposed) isLoading.value = false;
    }
  }

  function isRowLocked(key: string) {
    return isLoading.value || pendingKeys.value.has(key) || (key === 'bscp' && isBscpConfirmVisible.value);
  }

  /** 恢复默认值等同于关闭。开关和恢复入口共用 BSCP 关闭确认。 */
  function requestFeatureChange(row: FeatureRow, enabled: boolean) {
    if (isRowLocked(row.key) || (row.enabled === true) === enabled) return;
    if (row.key === 'bscp' && !enabled) {
      isBscpConfirmVisible.value = true;
      return;
    }
    void updateFeature(row.key, enabled);
  }

  async function updateFeature(key: string, enabled: boolean) {
    if (isDisposed || !props.active || isLoading.value || hasLoadError.value || pendingKeys.value.has(key)) return;
    pendingKeys.value.add(key);
    try {
      const result = await PlatmgtService.updatePlatCfgSwitch({ workspaceID: props.workspaceId, key, enabled });
      if (isDisposed) return;
      // 用服务端确认的状态更新对应行，提交失败时保留上一次确认的状态。
      features.value = features.value.map(feature => (feature.key === key ? { ...result, key } : feature));
      Message({ theme: 'success', message: t('操作成功') });
    } catch {
      // 请求拦截器统一提示提交失败。
    } finally {
      pendingKeys.value.delete(key);
    }
  }

  // 确认弹窗挂载在 body 上，离开页签时单独关闭尚未确认的弹窗。
  watch(
    () => props.active,
    active => {
      if (active) void fetchFeatures();
      else isBscpConfirmVisible.value = false;
    },
    { immediate: true },
  );
  onBeforeUnmount(() => {
    isDisposed = true;
  });
</script>
