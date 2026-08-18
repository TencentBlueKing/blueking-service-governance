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
        <span class="px-[4px] font-bold">1</span>
      </i18n-t>
    </template>

    <div class="rounded-[2px] border border-[#F9D090] bg-[#FDF4E8] px-[12px]">
      <div
        class="flex cursor-pointer items-center justify-between py-[8px]"
        @click="collapsed = !collapsed"
      >
        <div class="flex items-center">
          <i class="bkms-icon bkms-icon-triangle-warning text-[12px] text-[#F59500]" />
          <span class="mx-[8px] text-base font-bold text-[#F59500]">{{ $t('环境变量未定义') }}</span>
          <span class="rounded-[2px] bg-[#FCE5C0] px-[8px] py-[2px] text-[12px] text-[#E38B02]">
            <i18n-t keypath="{0} 个变量">
              <span class="pl-[2px]">{{ rows.length }}</span>
            </i18n-t>
          </span>
        </div>
        <AngleDown
          class="text-[22px] text-[#979BA5] transition-transform duration-200"
          :class="{ 'rotate-180': !collapsed }"
        />
      </div>

      <transition name="collapse">
        <div
          v-show="!collapsed"
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
              @click="handleGoModify"
            >
              「{{ $t('环境管理') }} / {{ $t('环境变量') }}」
            </span>
          </i18n-t>

          <Table
            class="env-var-precheck-dialog-table"
            :data="rows"
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
      <div class="flex justify-end">
        <Button
          class="mr-[8px]"
          theme="primary"
          @click="handleGoModify"
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

  import type { UndefinedEnvVarOutput } from '~/@types/v1/deploy';

  /** 表格展示行数据结构 */
  interface DisplayRow {
    key: string;
    sources: DisplaySource[];
  }

  /** 变量来源标签数据结构 */
  interface DisplaySource {
    className: string;
    id: string;
    text: string;
  }

  /** 组件 Props */
  const props = defineProps<{
    /** 当前部署环境名称，用于跳转时定位 */
    envName: string;
    /** 未定义的环境变量列表 */
    undefinedVars: UndefinedEnvVarOutput[];
  }>();
  const emit = defineEmits<{
    cancel: [];
    goModify: [];
    stillDeploy: [];
  }>();

  /** 弹窗显示状态（双向绑定） */
  const isShow = defineModel<boolean>('isShow', { default: false });
  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();

  /** 警告区域折叠状态 */
  const collapsed = ref(false);
  /** 分页当前页码 */
  const current = ref(1);
  /** 分页每页条数 */
  const limit = ref(10);

  /** 变量来源类型映射：样式类名 + 标签文本 */
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

  /**
   * 格式化变量来源列表
   * 去重合并同类型来源，生成带样式的标签数组
   */
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

  /** 表格展示数据：将原始数据转换为带格式化来源的行 */
  const rows = computed<DisplayRow[]>(() =>
    props.undefinedVars.map(item => ({
      key: item.key || '--',
      sources: formatSources(item),
    })),
  );

  /** 是否显示分页（超过10条时启用） */
  const showPagination = computed(() => rows.value.length > 10);

  /** 分页配置 */
  const pagination = computed(() => ({
    current: current.value,
    limit: limit.value,
    count: rows.value.length,
    limitList: [10, 20, 50],
  }));

  /** 取消本次部署。 */
  function handleCancel() {
    emit('cancel');
    isShow.value = false;
  }

  /**
   * 跳转到环境管理页面
   * 新窗口打开并定位到当前环境的环境变量配置
   */
  function handleGoModify() {
    const resolved = router.resolve({
      name: 'env',
      params: { space: route.params.space },
      query: {
        active: props.envName,
        activeTab: 'setting',
      },
    });
    window.open(resolved.href, '_blank');
    emit('goModify');
    isShow.value = false;
  }

  /** 分页页码切换 */
  function handlePageChange(page: number) {
    current.value = page;
  }

  /** 每页条数切换，重置到第一页 */
  function handlePageLimitChange(limitValue: number) {
    limit.value = limitValue;
    current.value = 1;
  }

  /** 忽略本次环境变量警告，继续原部署流程。 */
  function handleStillDeploy() {
    emit('stillDeploy');
    isShow.value = false;
  }

  /** 弹窗打开时重置折叠状态和分页 */
  watch(isShow, show => {
    if (show) {
      collapsed.value = false;
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

  :deep(.env-var-precheck-dialog-table) {
    --vxe-ui-table-border-color: #dcdee5;
    --vxe-ui-table-header-font-weight: 400;
    --vxe-ui-table-header-font-color: #4d4f56;
    --vxe-ui-table-header-background-color: #fafbfd;
    --vxe-ui-font-color: #4d4f56;
  }
</style>
