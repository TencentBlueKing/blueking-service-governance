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
  <div class="h-full overflow-hidden flex flex-col">
    <Alert
      class="mb-[16px] shrink-0"
      closable
      theme="info"
      :title="t('发布策略在空间内共享，应用部署时可选用。平台内置策略只读，可克隆后按需调整。')"
    />
    <Alert
      v-if="isError"
      class="mb-[16px] shrink-0"
      theme="danger"
      :title="t('发布策略加载失败，请稍后重试')"
    />
    <FlexRow
      average
      class="mb-[16px] shrink-0"
    >
      <template #left>
        <Button
          theme="primary"
          @click="openCreate"
        >
          {{ t('新建策略') }}
        </Button>
      </template>
      <template #right>
        <Input
          v-model="keyword"
          class="w-[520px] float-right"
          clearable
          :placeholder="createPlaceholder({ type: 'input', labels: ['策略名称'] })"
          type="search"
        />
      </template>
    </FlexRow>
    <Loading :loading="isLoading">
      <Table
        :data="strategies"
        :max-height="maxHeight"
        :pagination="pagination"
        :row-config="{
          isHover: true,
          isCurrent: true,
        }"
        :row-height="40"
        :virtual-y-config="{ enabled: true, gt: 0 }"
        @page-limit-change="handlePageLimitChange"
        @page-value-change="handlePageValueChange"
      >
        <template #empty>
          <TableException
            :type="curExceptionType"
            @clear="handleClearSearch"
            @refresh="fetchList"
          />
        </template>
        <TableColumn
          field="name"
          fixed="left"
          :label="t('策略名称')"
          min-width="300"
        >
          <template #default="{ row }">
            <div class="flex items-center">
              <!-- 名称可点开抽屉：内置策略只读走查看，空间策略走编辑 -->
              <Button
                text
                theme="primary"
                @click.stop="row.isBuiltin ? openView(row) : openEdit(row)"
              >
                {{ row.name }}
              </Button>
              <Tag
                v-if="row.isBuiltin"
                class="ml-[8px]"
                size="small"
              >
                {{ t('平台内置') }}
              </Tag>
            </div>
          </template>
        </TableColumn>
        <TableColumn
          field="batches"
          :label="t('批次')"
          min-width="320"
        >
          <template #default="{ row }">
            <Popover
              placement="bottom-start"
              theme="light !px-[16px]"
              trigger="hover"
            >
              <div class="flex items-center gap-[8px]">
                <span class="text-[#4D4F56]">{{ t('{0} 批', [batchCount(row)]) }}</span>
                <div class="flex items-center gap-[2px] w-[100px]">
                  <span
                    v-for="(ratio, index) in batchSegments(row)"
                    :key="index"
                    class="h-[6px] bg-[#3A84FF] rounded-[2px]"
                    :style="{ flexGrow: ratio }"
                  ></span>
                </div>
                <div class="text-[#979BA5] text-[12px] leading-[20px] no-wrap overflow-hidden text-ellipsis">
                  {{ formatBatchesSummary(row.batches, targetTexts) }}
                </div>
              </div>
              <template #content>
                <div class="batch-popover">
                  <div class="batch-popover__title">{{ row.name }} · {{ t('{0} 批', [batchCount(row)]) }}</div>
                  <div class="batch-popover__grid">
                    <div class="batch-popover__head">{{ t('批次') }}</div>
                    <div class="batch-popover__head">{{ t('类型') }}</div>
                    <div class="batch-popover__head">{{ t('累计数') }}</div>
                    <div class="batch-popover__head">{{ t('暂停方式') }}</div>
                    <template
                      v-for="detail in batchDetails(row)"
                      :key="detail.key"
                    >
                      <div class="batch-popover__cell">{{ detail.label }}</div>
                      <div class="batch-popover__cell">{{ t('更新') }}</div>
                      <div class="batch-popover__cell">{{ detail.accumulated }}</div>
                      <div class="batch-popover__cell">{{ detail.pauseMode }}</div>
                    </template>
                  </div>
                </div>
              </template>
            </Popover>
          </template>
        </TableColumn>
        <!-- <TableColumn
          field="failureStrategy"
          :label="t('失败策略')"
          width="120"
        >
          <template #default>—</template>
        </TableColumn> -->
        <TableColumn
          field="description"
          :label="t('策略说明')"
          min-width="200"
        >
          <template #default="{ row }">
            <span class="text-[#4D4F56]">{{ row.description || '--' }}</span>
          </template>
        </TableColumn>
        <TableColumn
          fixed="right"
          :label="t('操作')"
          :show-overflow="false"
          width="160"
        >
          <template #default="{ row }">
            <div class="flex items-center gap-[8px]">
              <Button
                v-if="row.isBuiltin"
                text
                theme="primary"
                @click.stop="openView(row)"
              >
                {{ t('查看') }}
              </Button>
              <Button
                v-else
                text
                theme="primary"
                @click.stop="openEdit(row)"
              >
                {{ t('编辑') }}
              </Button>
              <Button
                text
                theme="primary"
                @click.stop="openClone(row)"
              >
                {{ t('克隆') }}
              </Button>
              <Button
                v-if="!row.isBuiltin"
                text
                theme="primary"
                @click.stop="handleDelete(row)"
              >
                {{ t('删除') }}
              </Button>
            </div>
          </template>
        </TableColumn>
      </Table>
    </Loading>

    <StrategySideslider
      v-model:is-show="sidesliderVisible"
      :existing-names="existingNames"
      :mode="sidesliderMode"
      :strategy="currentStrategy"
      @success="handleSaved"
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, h, onMounted, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Button, InfoBox, Input, Loading, Message, Popover, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { ReleaseStrategiesService } from '~/api/modules/v1/release-strategies';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import useDynamicsHeight from '~/composables/use-table-height';
  import { useSpaceStore } from '~/stores/space';

  import StrategySideslider from './components/strategy-sideslider.vue';
  import useReleaseStrategies from './use-release-strategies';
  import { buildBatchDetailRows, calcBatchSegmentRatios, formatBatchesSummary } from './utils';

  import type { ReleaseStrategyOutputObj } from '~/@types/v1/release-strategies';

  type SidesliderMode = 'clone' | 'create' | 'edit' | 'view';

  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const spaceStore = useSpaceStore();
  const { maxHeight } = useDynamicsHeight(96, ['.header-right', '.container-header']);

  const {
    isLoading,
    isError,
    keyword,
    pagination,
    allStrategies,
    strategies,
    fetchList,
    handlePageLimitChange,
    handlePageValueChange,
  } = useReleaseStrategies();

  const { curExceptionType, setTypeToError, clearErrorType } = useTableEmpty({ filters: keyword });

  watch(isError, value => {
    if (value) setTypeToError();
    else clearErrorType();
  });

  const sidesliderVisible = ref(false);
  const sidesliderMode = ref<SidesliderMode>('create');
  const currentStrategy = ref<ReleaseStrategyOutputObj>();

  const targetTexts = computed(() => ({ full: t('全量'), unit: t('个') }));

  /** 空间内已有策略名称：克隆时用于生成不重名的默认名称 */
  const existingNames = computed(() => allStrategies.value.map(strategy => strategy.name ?? ''));

  function batchCount(row: ReleaseStrategyOutputObj): number {
    return row.batches?.length ?? 0;
  }

  /** 批次明细 Popover 的行数据（第 N 批 / 累计数 / 暂停方式） */
  function batchDetails(row: ReleaseStrategyOutputObj) {
    return buildBatchDetailRows(row.batches, targetTexts.value, t);
  }

  function batchSegments(row: ReleaseStrategyOutputObj): number[] {
    return calcBatchSegmentRatios(row.batches);
  }

  function handleClearSearch() {
    keyword.value = '';
  }

  function handleDelete(row: ReleaseStrategyOutputObj) {
    const { id, name } = row;
    if (!id) return;
    InfoBox({
      width: 480,
      title: t('确定删除该发布策略？'),
      content: renderDeleteInfoBoxContent(name ?? '--'),
      headerAlign: 'center',
      contentAlign: 'left',
      footerAlign: 'center',
      confirmButtonTheme: 'danger',
      confirmText: t('删除'),
      cancelText: t('取消'),
      async onConfirm() {
        try {
          await ReleaseStrategiesService.deleteReleaseStrategy({
            workspaceID: spaceStore.currentSpace,
            strategyID: id,
          });
          Message({ theme: 'success', message: t('策略已删除') });
          fetchList();
        } catch (error) {
          console.error(error);
          Message({ theme: 'error', message: t('请求异常') });
        }
      },
    });
  }

  function handleSaved() {
    fetchList();
  }

  function openClone(row: ReleaseStrategyOutputObj) {
    sidesliderMode.value = 'clone';
    currentStrategy.value = row;
    sidesliderVisible.value = true;
  }

  function openCreate() {
    sidesliderMode.value = 'create';
    currentStrategy.value = undefined;
    sidesliderVisible.value = true;
  }

  function openEdit(row: ReleaseStrategyOutputObj) {
    sidesliderMode.value = 'edit';
    currentStrategy.value = row;
    sidesliderVisible.value = true;
  }

  function openView(row: ReleaseStrategyOutputObj) {
    sidesliderMode.value = 'view';
    currentStrategy.value = row;
    sidesliderVisible.value = true;
  }

  /**
   * 删除确认框内容（对齐设计稿 InfoBox/删除策略 2480:1171）：
   * 上方名称行常规文案 + 下方灰底警示块，仅强调不可恢复与部署侧影响。
   */
  function renderDeleteInfoBoxContent(name: string) {
    return h('div', [
      h('div', { class: 'text-[14px] leading-[22px] text-[#313238]' }, [`${t('策略名称')}：${name}`]),
      h('div', { class: 'mt-[16px] bg-[#F5F7FA] px-[16px] py-[12px] text-[14px] leading-[22px] text-[#4D4F56]' }, [
        t('删除后无法恢复，部署应用时将不能再选择该策略。'),
      ]),
    ]);
  }

  onMounted(fetchList);
</script>

<!-- Popover 内容会 Teleport 到 body，样式需写在非 scoped 中 -->
<style lang="postcss">
  .batch-popover {
    padding: 4px;
  }

  .batch-popover__title {
    margin-bottom: 8px;
    color: #313238;
    font-size: 16px;
    font-weight: 700;
    line-height: 24px;
  }

  /* 表头与数据行共用一个 grid，列宽由内容自适应，保证跨行对齐且无边框 */
  .batch-popover__grid {
    display: grid;
    grid-template-columns: max-content max-content max-content max-content;
    align-items: center;
    column-gap: 24px;
    row-gap: 6px;
    width: max-content;
    font-size: 12px;
    line-height: 20px;
  }

  .batch-popover__head,
  .batch-popover__cell {
    white-space: nowrap;
  }

  .batch-popover__head {
    padding-bottom: 4px;
    color: #979ba5;
  }

  .batch-popover__cell {
    color: #4d4f56;
  }
</style>
