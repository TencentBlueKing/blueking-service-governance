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
  <Dialog
    v-model:is-show="isShow"
    :quick-close="false"
    :width="700"
  >
    <template #header>
      <i18n-t
        class="text-[20px] text-[#313238]"
        keypath="部署预检发现{0}类问题"
      >
        <span class="px-[4px] font-bold">{{ problemCount }}</span>
      </i18n-t>
    </template>

    <!-- 资源规格不一致 -->
    <div
      v-if="hasResourceBlocker"
      class="overflow-hidden rounded-[2px] border border-[#F8B4B4]"
      :class="{ 'mb-[16px]': hasUndefinedVars }"
    >
      <div
        class="flex cursor-pointer items-center justify-between bg-[#FEEBEA] px-[12px] py-[8px]"
        @click="resourceCollapsed = !resourceCollapsed"
      >
        <div class="flex items-center">
          <i class="bkms-icon bkms-icon-close-circle-shape text-[14px] text-[#EA3636]" />
          <span class="mx-[8px] text-base font-bold text-[#EA3636]">{{ $t('联邦集群资源规格不一致') }}</span>
        </div>
        <AngleDown
          class="text-[22px] text-[#979BA5] transition-transform duration-200"
          :class="{ 'rotate-180': !resourceCollapsed }"
        />
      </div>

      <transition name="collapse">
        <div
          v-show="!resourceCollapsed"
          class="border-t border-[#F8B4B4] bg-[#FFF5F4] px-[12px] py-[8px]"
        >
          <p class="text-[12px] leading-[20px] text-[#4D4F56]">
            {{ $t('当前环境已绑定联邦集群。联邦集群要求 CPU、内存的 Requests 与 Limits 必须一致，否则无法部署。') }}
          </p>
          <i18n-t
            class="mb-[8px] text-[12px] leading-[20px] text-[#4D4F56]"
            keypath="建议前往 {0} 将 Requests 调整为与 Limits 一致后再部署。"
            tag="p"
          >
            <span
              class="cursor-pointer text-[#3A84FF]"
              @click="handleGoResourceModify"
            >
              「{{ $t('应用配置') }} / {{ $t('部署配置') }}」
            </span>
          </i18n-t>

          <Table
            class="deploy-precheck-dialog-table"
            :data="resourceRows"
          >
            <TableColumn
              field="label"
              :label="$t('资源')"
              :min-width="120"
            />
            <TableColumn
              field="requests"
              :label="$t('Requests')"
              :min-width="140"
            />
            <TableColumn
              field="limits"
              :label="$t('Limits')"
              :min-width="140"
            />
          </Table>
        </div>
      </transition>
    </div>

    <!-- 环境变量未定义 -->
    <div
      v-if="hasUndefinedVars"
      class="rounded-[2px] border border-[#F9D090] bg-[#FDF4E8] px-[12px]"
    >
      <div
        class="flex cursor-pointer items-center justify-between py-[8px]"
        @click="envVarCollapsed = !envVarCollapsed"
      >
        <div class="flex items-center">
          <i class="bkms-icon bkms-icon-triangle-warning text-[12px] text-[#F59500]" />
          <span class="mx-[8px] text-base font-bold text-[#F59500]">{{ $t('环境变量未定义') }}</span>
          <span class="rounded-[2px] bg-[#FCE5C0] px-[8px] py-[2px] text-[12px] text-[#E38B02]">
            <i18n-t keypath="{0} 个变量">
              <span class="pl-[2px]">{{ envVarRows.length }}</span>
            </i18n-t>
          </span>
        </div>
        <AngleDown
          class="text-[22px] text-[#979BA5] transition-transform duration-200"
          :class="{ 'rotate-180': !envVarCollapsed }"
        />
      </div>

      <transition name="collapse">
        <div
          v-show="!envVarCollapsed"
          class="border-t border-[#F9D090] py-[8px]"
        >
          <i18n-t
            class="text-[12px] leading-[20px] text-[#4D4F56]"
            keypath="以下环境变量在当前配置中被引用，但在目标部署环境中未定义，部署后将被渲染为空值，{0}"
            tag="p"
          >
            <span class="font-bold">{{ $t('可能导致服务异常！') }}</span>
          </i18n-t>
          <i18n-t
            class="mb-[8px] text-[12px] leading-[20px] text-[#4D4F56]"
            keypath="建议前往 {0} 补充配置后再部署，避免服务注册异常或配置错误。"
            tag="p"
          >
            <span
              class="cursor-pointer text-[#3A84FF]"
              @click="handleGoEnvVarModify"
            >
              「{{ $t('环境管理') }} / {{ $t('环境变量') }}」
            </span>
          </i18n-t>

          <Table
            class="deploy-precheck-dialog-table"
            :data="envVarRows"
            :pagination="showPagination ? pagination : undefined"
            @page-limit-change="handlePageLimitChange"
            @page-value-change="handlePageChange"
          >
            <TableColumn
              field="key"
              :label="$t('变量名')"
              :min-width="180"
              show-overflow="tooltip"
            />
            <TableColumn
              :label="$t('来源')"
              :min-width="220"
            >
              <template #default="{ row }: { row: DisplayRow }">
                <div class="flex flex-wrap gap-[4px]">
                  <span
                    v-for="source in row.sources"
                    :key="source.id"
                    class="max-w-full break-all rounded-[2px] px-[6px] py-[2px] text-[12px] leading-[18px]"
                    :class="source.className"
                  >
                    {{ source.text }}
                  </span>
                </div>
              </template>
            </TableColumn>
          </Table>
        </div>
      </transition>
    </div>

    <template #footer>
      <div
        v-if="hasResourceBlocker"
        class="flex items-center justify-between"
      >
        <span class="text-[12px] leading-[20px] text-[#EA3636]">
          {{ $t('存在阻断问题，请先解决后再部署') }}
        </span>
        <Button
          class="bg-[#fff] text-[#4D4F56] !min-w-[60px]"
          @click="handleCancel"
        >
          {{ $t('关闭') }}
        </Button>
      </div>
      <div
        v-else
        class="flex justify-end"
      >
        <Button
          class="mr-[8px]"
          theme="primary"
          @click="handleGoEnvVarModify"
        >
          {{ $t('前往修改') }}
        </Button>
        <Button
          class="mr-[8px] bg-[#fff] text-[#4D4F56]"
          @click="handleStillDeploy"
        >
          {{ $t('仍然部署') }}
        </Button>
        <Button
          class="bg-[#fff] text-[#4D4F56] !min-w-[60px]"
          @click="handleCancel"
        >
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Dialog } from 'bkui-vue';
  import { AngleDown } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';

  import type { FederationResourceMismatch } from './use-federation-resource-precheck';
  import type { UndefinedEnvVarOutput } from '~/@types/v1/deploy';

  interface DisplayRow {
    key: string;
    sources: DisplaySource[];
  }

  interface DisplaySource {
    className: string;
    id: string;
    text: string;
  }

  const props = defineProps<{
    envName: string;
    mismatches: FederationResourceMismatch[];
    undefinedVars: UndefinedEnvVarOutput[];
  }>();
  const emit = defineEmits<{
    cancel: [];
    goModify: [];
    stillDeploy: [];
  }>();

  const isShow = defineModel<boolean>('isShow', { default: false });
  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const resourceCollapsed = ref(false);
  const envVarCollapsed = ref(false);
  const current = ref(1);
  const limit = ref(10);

  const hasResourceBlocker = computed(() => props.mismatches.length > 0);
  const hasUndefinedVars = computed(() => props.undefinedVars.length > 0);
  const problemCount = computed(() => Number(hasResourceBlocker.value) + Number(hasUndefinedVars.value));

  const sourceTypeMap: Record<string, { className: string; label: string }> = {
    appConfigFile: {
      className: 'bg-[#F0F1F5] text-[#63656E]',
      label: '框架配置文件',
    },
    component: {
      className: 'bg-[#E5F6EA] text-[#299E56]',
      label: '组件配置',
    },
    polaris: {
      className: 'bg-[#E1ECFF] text-[#3A84FF]',
      label: '北极星',
    },
  };

  function formatSources(item: UndefinedEnvVarOutput): DisplaySource[] {
    const sources = item.sources ?? [];
    if (sources.length === 0) {
      return [{ className: 'bg-[#F0F1F5] text-[#63656E]', id: 'unknown', text: '--' }];
    }

    const sourceMap = new Map<string, DisplaySource>();
    sources.forEach((source, index) => {
      const config = sourceTypeMap[source.type ?? ''];
      const typeLabel = config ? t(config.label) : source.type || t('未知来源');
      const text = source.name ? `${typeLabel}：${source.name}` : typeLabel;
      sourceMap.set(text, {
        className: config?.className ?? 'bg-[#F0F1F5] text-[#63656E]',
        id: `${source.type ?? 'unknown'}:${source.name ?? index}`,
        text,
      });
    });
    return [...sourceMap.values()];
  }

  const resourceRows = computed(() =>
    props.mismatches.map(item => ({
      key: item.key,
      label: item.key === 'cpu' ? 'CPU' : t('内存'),
      limits: item.limits || '--',
      requests: item.requests || '--',
    })),
  );

  const envVarRows = computed<DisplayRow[]>(() =>
    props.undefinedVars.map(item => ({
      key: item.key || '--',
      sources: formatSources(item),
    })),
  );

  const showPagination = computed(() => envVarRows.value.length > 10);
  const pagination = computed(() => ({
    current: current.value,
    limit: limit.value,
    count: envVarRows.value.length,
    limitList: [10, 20, 50],
  }));

  function closeAfterModify() {
    emit('goModify');
    isShow.value = false;
  }

  function handleCancel() {
    emit('cancel');
    isShow.value = false;
  }

  function handleGoEnvVarModify() {
    const resolved = router.resolve({
      name: 'env',
      params: { space: route.params.space },
      query: {
        active: props.envName,
        activeTab: 'setting',
      },
    });
    window.open(resolved.href, '_blank');
    closeAfterModify();
  }

  function handleGoResourceModify() {
    const resolved = router.resolve({
      name: 'detail',
      params: {
        ...route.params,
        menuName: 'appConfig',
      },
      query: {
        ...route.query,
        activeTab: 'deploy-config',
        envName: props.envName,
      },
    });
    window.open(resolved.href, '_blank');
    closeAfterModify();
  }

  function handlePageChange(page: number) {
    current.value = page;
  }

  function handlePageLimitChange(limitValue: number) {
    limit.value = limitValue;
    current.value = 1;
  }

  function handleStillDeploy() {
    if (hasResourceBlocker.value) return;
    emit('stillDeploy');
    isShow.value = false;
  }

  watch(isShow, show => {
    if (show) {
      resourceCollapsed.value = false;
      envVarCollapsed.value = false;
      current.value = 1;
      limit.value = 10;
    }
  });
</script>

<style scoped lang="postcss">
  :deep(.bk-dialog-content) {
    max-height: calc(80vh - 135px);
    overflow-y: auto;
  }

  :deep(.bk-dialog-footer) {
    border-top-color: #eaebf0;
  }

  :deep(.deploy-precheck-dialog-table) {
    --vxe-ui-table-border-color: #dcdee5;
    --vxe-ui-table-header-font-weight: 400;
    --vxe-ui-table-header-font-color: #4d4f56;
    --vxe-ui-table-header-background-color: #fafbfd;
    --vxe-ui-font-color: #4d4f56;
  }
</style>
