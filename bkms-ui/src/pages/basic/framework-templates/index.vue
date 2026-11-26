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
  <div class="flex h-full flex-col overflow-hidden">
    <Skeleton
      class="h-full"
      :loading="loading"
      theme="gray"
    >
      <template #loading>
        <div class="mb-[16px] h-[42px]">
          <Layout.shape
            height="42px"
            type="rect"
            width="100%"
          />
        </div>
        <div class="mb-[16px] flex items-center gap-[12px]">
          <Layout.shape
            height="32px"
            type="rect"
            width="104px"
          />
          <Layout.shape
            height="32px"
            type="rect"
            width="188px"
          />
          <Layout.shape
            class="ml-auto"
            height="32px"
            type="rect"
            width="520px"
          />
        </div>
        <div class="flex flex-col gap-[16px] overflow-hidden">
          <div
            v-for="group in 2"
            :key="group"
            class="overflow-hidden rounded-[2px] bg-[#fff] shadow-[0_2px_4px_0_#1919290d]"
          >
            <div class="flex h-[42px] items-center bg-[#f0f1f5] px-[16px]">
              <Layout.shape
                height="20px"
                type="rect"
                width="96px"
              />
            </div>
            <div class="bg-[#fff] px-[16px]">
              <div
                class="grid min-h-[44px] items-center gap-[24px] border-b border-[#f0f1f5]"
                :style="skeletonTableStyle"
              >
                <Layout.shape
                  v-for="(width, column) in skeletonHeaderWidths"
                  :key="`header-${group}-${column}`"
                  height="16px"
                  type="rect"
                  :width="`${width}px`"
                />
              </div>
              <div
                v-for="(row, rowIndex) in skeletonRowWidths"
                :key="`row-${group}-${rowIndex}`"
                class="grid min-h-[42px] items-center gap-[24px] border-b border-[#f0f1f5]"
                :style="skeletonTableStyle"
              >
                <Layout.shape
                  v-for="(width, column) in row"
                  :key="`row-${group}-${rowIndex}-${column}`"
                  height="16px"
                  type="rect"
                  :width="`${width}px`"
                />
              </div>
            </div>
          </div>
        </div>
      </template>

      <Alert
        class="mb-[16px]"
        theme="info"
        :title="tipText"
      />
      <div class="mb-[16px] flex items-center gap-[12px]">
        <Button
          theme="primary"
          @click="handleCreate"
        >
          <Plus
            :height="24"
            :width="24"
          />
          <span class="h-[24px] leading-[24px]">{{ $t('新建模板') }}</span>
        </Button>
        <Radio.Group
          v-model="appType"
          type="capsule"
        >
          <Radio.Button :label="APP_FRAMEWORKS.TRPC">
            {{ `tRPC ( ${displayCounts.trpc} )` }}
          </Radio.Button>
          <Radio.Button :label="APP_FRAMEWORKS.TAF">
            {{ `TAF ( ${displayCounts.taf} )` }}
          </Radio.Button>
        </Radio.Group>
        <Input
          v-model.trim="keyword"
          class="ml-auto w-[520px] shrink-0"
          clearable
          :placeholder="$t('搜索模板名称')"
          type="search"
        />
      </div>

      <div class="min-h-0 flex-1 overflow-auto flex flex-col gap-[16px]">
        <component
          :is="group.label ? BkmsContent : 'div'"
          v-for="group in templateGroups"
          :key="group.key"
          class="framework-template-card rounded-[2px] bg-[#fff]"
          :collapsible="Boolean(group.label)"
          :title="group.label"
        >
          <div class="bg-[#fff]">
            <Table
              v-bkloading="{ loading, zIndex: 8 }"
              :class="{ 'framework-template-table': Boolean(group.label) }"
              :data="group.rows"
              :pagination="paginationOf(group.key, group.rows.length)"
              :row-config="{ isHover: true }"
              :sort-config="sortConfig"
              @page-value-change="page => setPage(group.key, page)"
              @sort-change="params => handleSortChange(group.key, params)"
            >
              <template #empty>
                <TableException
                  :type="keyword ? 'search' : 'empty'"
                  @clear="keyword = ''"
                />
              </template>
              <TableColumn
                :label="$t('模板名称')"
                min-width="320"
              >
                <template #default="{ row }: { row: FrameworkTemplate }">
                  <div class="flex items-center gap-[8px]">
                    <span
                      v-overflow-title
                      class="truncate"
                    >
                      {{ displayName(row) }}
                    </span>
                    <Tag
                      v-if="row.source === 'platform'"
                      class="!bg-[#F0F1F5] !text-[#63656E] shrink-0"
                      size="small"
                    >
                      {{ $t('内置') }}
                    </Tag>
                    <Tag
                      v-if="row.isDefault"
                      class="shrink-0"
                      size="small"
                      theme="info"
                    >
                      {{ $t('默认') }}
                    </Tag>
                  </div>
                </template>
              </TableColumn>
              <TableColumn
                field="fileName"
                :label="$t('配置文件名称')"
                min-width="180"
              />
              <TableColumn
                :label="$t('来源')"
                min-width="120"
              >
                <template #default="{ row }: { row: FrameworkTemplate }">
                  {{ row.source === 'platform' ? $t('平台默认') : $t('空间自定义') }}
                </template>
              </TableColumn>
              <TableColumn
                field="updater"
                :label="$t('更新人')"
                min-width="140"
              >
                <template #default="{ row }: { row: FrameworkTemplate }">
                  {{ row.updater || '--' }}
                </template>
              </TableColumn>
              <TableColumn
                field="updatedAt"
                :label="$t('更新时间')"
                min-width="160"
                sortable
              >
                <template #default="{ row }: { row: FrameworkTemplate }">
                  {{ row.source === 'workspace' && row.updatedAt ? formatTimeByTimezone(row.updatedAt) : '--' }}
                </template>
              </TableColumn>
              <TableColumn
                fixed="right"
                :label="$t('操作')"
                :width="150"
              >
                <template #default="{ row }: { row: FrameworkTemplate }">
                  <div class="flex items-center gap-[12px]">
                    <Button
                      v-if="row.source === 'platform'"
                      text
                      theme="primary"
                      @click="openSlider('view', row)"
                    >
                      {{ $t('查看') }}
                    </Button>
                    <Button
                      v-else
                      text
                      theme="primary"
                      @click="openSlider('edit', row)"
                    >
                      {{ $t('编辑') }}
                    </Button>
                    <Button
                      v-if="row.source === 'workspace'"
                      text
                      theme="primary"
                      @click="handleDelete(row)"
                    >
                      {{ $t('删除') }}
                    </Button>
                    <Button
                      v-if="!row.isDefault"
                      v-bk-tooltips="$t('创建应用时默认使用该模板')"
                      text
                      theme="primary"
                      @click="handleSetDefault(row)"
                    >
                      {{ $t('设为默认') }}
                    </Button>
                  </div>
                </template>
              </TableColumn>
            </Table>
          </div>
        </component>
      </div>
    </Skeleton>

    <TemplateEditor
      v-model:is-show="sliderVisible"
      :app-type="appType"
      :scene="sliderScene"
      :template="editingTemplate"
      :templates="sliderTemplates"
      @success="handleTemplateSuccess"
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, h, reactive, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Button, InfoBox, Input, Message, Radio, Tag } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { FrameworkConfigTemplatesService } from '~/api/modules/v1';
  import { APP_FRAMEWORKS } from '~/common/app-types';
  import { formatTimeByTimezone } from '~/common/util';
  import BkmsContent from '~/components/bkms-content.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import TableException from '~/components/table-exception.vue';
  import { useSpaceStore } from '~/stores/space';

  import TemplateEditor from './template-editor.vue';
  import {
    type FrameworkAppType,
    type FrameworkTemplateRecord as FrameworkTemplate,
    normalizeTemplate,
    TEMPLATE_LANGUAGE_LABELS,
    TEMPLATE_LANGUAGES,
  } from './types';

  type Scene = 'create' | 'edit' | 'view';

  const { t } = useI18n();
  const spaceStore = useSpaceStore();
  const tipText = t('这里新建的模板用于创建 tRPC、TAF 应用时选择框架配置，只影响本空间之后新建的应用。');

  const frameworkAppTypes = [APP_FRAMEWORKS.TRPC, APP_FRAMEWORKS.TAF] as const satisfies readonly FrameworkAppType[];

  const PAGE_SIZE = 10;
  const skeletonTableStyle = {
    gridTemplateColumns:
      'minmax(220px, 2fr) minmax(180px, 1.4fr) minmax(120px, 0.8fr) minmax(160px, 1fr) minmax(120px, 0.8fr)',
  };
  const skeletonHeaderWidths = [96, 112, 48, 80, 48];
  const skeletonRowWidths = [
    [168, 128, 72, 112, 64],
    [132, 104, 84, 96, 56],
    [156, 136, 72, 120, 64],
    [120, 112, 84, 88, 56],
  ];

  type SortOrder = 'asc' | 'desc' | null;

  const appType = ref<FrameworkAppType>(APP_FRAMEWORKS.TRPC);
  const keyword = ref('');
  const loading = ref(false);
  // 分组 key：tRPC 按语言分组（go/cpp），TAF 单组 all
  const groupKeys = ['all', ...TEMPLATE_LANGUAGES] as const;
  const sortOrdersByGroup = reactive<Record<string, SortOrder>>(Object.fromEntries(groupKeys.map(key => [key, null])));
  const sortConfig = { remote: true, trigger: 'cell' as const };
  const paginationByGroup = reactive<
    Record<string, { count: number; current: number; limit: number; showLimit: boolean }>
  >({});
  const templatesByType = ref<Record<FrameworkAppType, FrameworkTemplate[]>>({
    [APP_FRAMEWORKS.TRPC]: [],
    [APP_FRAMEWORKS.TAF]: [],
  });
  const templateCounts = ref<Record<FrameworkAppType, number>>({
    [APP_FRAMEWORKS.TRPC]: 0,
    [APP_FRAMEWORKS.TAF]: 0,
  });
  // 追踪已加载的类型，避免 Tab 切换时重复请求已有数据
  const loadedTypes = new Set<FrameworkAppType>();
  const templates = computed(() => templatesByType.value[appType.value]);
  const sliderTemplates = computed(() => [...templatesByType.value.trpc, ...templatesByType.value.taf]);
  let fetchSeq = 0;
  const sliderVisible = ref(false);
  const sliderScene = ref<Scene>('create');
  const editingTemplate = ref<FrameworkTemplate | null>(null);
  const visibleTemplates = computed(() => {
    const text = keyword.value.trim().toLowerCase();
    return text ? templates.value.filter(item => displayName(item).toLowerCase().includes(text)) : templates.value;
  });
  const displayCounts = computed<Record<FrameworkAppType, number>>(() => ({
    taf: appType.value === APP_FRAMEWORKS.TAF ? visibleTemplates.value.length : templateCounts.value.taf,
    trpc:
      appType.value === APP_FRAMEWORKS.TRPC
        ? countTemplates(APP_FRAMEWORKS.TRPC, visibleTemplates.value)
        : templateCounts.value.trpc,
  }));
  const templateGroups = computed(() => {
    if (appType.value !== APP_FRAMEWORKS.TRPC) {
      return [{ key: 'all', label: '', rows: sortTemplates(visibleTemplates.value, 'all') }];
    }
    return TEMPLATE_LANGUAGES.map(language => ({
      key: language,
      label: TEMPLATE_LANGUAGE_LABELS[language],
      rows: sortTemplates(
        visibleTemplates.value.filter(item => item.language === language),
        language,
      ),
    }));
  });

  function compareCreatedAt(left: FrameworkTemplate, right: FrameworkTemplate) {
    const leftRank = left.source === 'platform' ? 0 : 1;
    const rightRank = right.source === 'platform' ? 0 : 1;
    if (leftRank !== rightRank) return leftRank - rightRank;
    return (left.createdAt || '').localeCompare(right.createdAt || '');
  }

  function compareUpdatedAt(left: FrameworkTemplate, right: FrameworkTemplate, order: 'asc' | 'desc') {
    const diff = (left.updatedAt || '').localeCompare(right.updatedAt || '');
    return order === 'asc' ? diff : -diff;
  }

  function countTemplates(type: FrameworkAppType, list: FrameworkTemplate[]) {
    if (type !== APP_FRAMEWORKS.TRPC) return list.length;
    return list.filter(item => (TEMPLATE_LANGUAGES as readonly string[]).includes(item.language)).length;
  }

  function displayName(row: FrameworkTemplate) {
    return row.source === 'platform' ? t('平台默认') : row.name;
  }

  async function fetchTemplates(type?: FrameworkAppType) {
    if (!spaceStore.currentSpace) {
      fetchSeq += 1;
      templatesByType.value = { [APP_FRAMEWORKS.TRPC]: [], [APP_FRAMEWORKS.TAF]: [] };
      templateCounts.value = { [APP_FRAMEWORKS.TRPC]: 0, [APP_FRAMEWORKS.TAF]: 0 };
      loadedTypes.clear();
      return;
    }

    const types: readonly FrameworkAppType[] = type ? [type] : frameworkAppTypes;
    const seq = ++fetchSeq;
    const workspaceID = spaceStore.currentSpace;
    loading.value = true;
    try {
      const results = await Promise.allSettled(
        types.map(templateType =>
          FrameworkConfigTemplatesService.listWorkspaceFrameworkConfigTemplates({
            workspaceID,
            framework: templateType,
          }),
        ),
      );
      if (seq !== fetchSeq || spaceStore.currentSpace !== workspaceID) return;
      let hasFailure = false;
      results.forEach((result, index) => {
        const templateType = types[index];
        if (result.status === 'fulfilled') {
          // 仅成功时标记已加载：失败后切换 Tab 可重新请求，避免一次接口故障导致列表持续显示为空
          loadedTypes.add(templateType);
          const list = (Array.isArray(result.value) ? result.value : []).map(item => normalizeTemplate(item));
          templatesByType.value[templateType] = list;
          templateCounts.value[templateType] = countTemplates(templateType, list);
          return;
        }
        hasFailure = true;
        templatesByType.value[templateType] = [];
        templateCounts.value[templateType] = 0;
      });
      if (hasFailure) {
        Message({ theme: 'warning', message: t('框架配置模板加载失败') });
      }
    } finally {
      if (seq === fetchSeq) loading.value = false;
    }
  }

  function handleCreate() {
    openSlider('create');
  }

  function handleDelete(row: FrameworkTemplate) {
    if (!spaceStore.currentSpace) return;
    InfoBox({
      title: t('确认删除该模板吗？'),
      cancelText: t('取消'),
      headerAlign: 'left',
      contentAlign: 'left',
      confirmButtonTheme: 'danger',
      content: h('div', [
        h('div', [h('span', `${t('模板名称')}：`), h('span', row.name)]),
        h('div', { class: 'mt-[10px]' }, t('删除后，将不可恢复，请谨慎操作！')),
      ]),
      async onConfirm() {
        await FrameworkConfigTemplatesService.deleteWorkspaceFrameworkConfigTemplate({
          workspaceID: spaceStore.currentSpace,
          templateID: row.id,
        });
        Message({ theme: 'success', message: t('删除成功') });
        await fetchTemplates(row.appType);
      },
    });
  }

  async function handleSetDefault(row: FrameworkTemplate) {
    if (!spaceStore.currentSpace) return;
    await FrameworkConfigTemplatesService.setWorkspaceFrameworkConfigTemplateDefault({
      workspaceID: spaceStore.currentSpace,
      templateID: row.id,
    });
    Message({ theme: 'success', message: t('保存成功') });
    await fetchTemplates(row.appType);
  }

  function handleSortChange(groupKey: string, params: { field?: string; order?: '' | SortOrder }) {
    sortOrdersByGroup[groupKey] = params.field === 'updatedAt' && params.order ? params.order : null;
    const pagination = paginationByGroup[groupKey];
    if (pagination) pagination.current = 1;
  }

  function handleTemplateSuccess(type: FrameworkAppType) {
    void fetchTemplates(type);
  }

  function openSlider(scene: Scene, template: FrameworkTemplate | null = null) {
    sliderScene.value = scene;
    editingTemplate.value = template;
    sliderVisible.value = true;
  }

  function paginationOf(key: string, count: number) {
    if (count <= PAGE_SIZE) return undefined;
    const current = paginationByGroup[key] || { count, current: 1, limit: PAGE_SIZE, showLimit: false };
    current.count = count;
    current.limit = PAGE_SIZE;
    const maxPage = Math.max(1, Math.ceil(count / PAGE_SIZE));
    if (current.current > maxPage) current.current = 1;
    paginationByGroup[key] = current;
    return current;
  }

  function setPage(key: string, page: number) {
    const current = paginationByGroup[key];
    if (current) current.current = page;
  }

  function sortTemplates(list: FrameworkTemplate[], groupKey: string) {
    const order = sortOrdersByGroup[groupKey];
    return [...list].sort((left, right) =>
      order ? compareUpdatedAt(left, right, order) : compareCreatedAt(left, right),
    );
  }

  watch(
    () => spaceStore.currentSpace,
    () => {
      templatesByType.value = { [APP_FRAMEWORKS.TRPC]: [], [APP_FRAMEWORKS.TAF]: [] };
      templateCounts.value = { [APP_FRAMEWORKS.TRPC]: 0, [APP_FRAMEWORKS.TAF]: 0 };
      loadedTypes.clear();
      void fetchTemplates();
    },
    { immediate: true },
  );

  watch(appType, type => {
    // 仅在该类型尚未加载时才请求（首次全量加载后切换 Tab 不再重复请求）
    if (!loadedTypes.has(type)) void fetchTemplates(type);
  });

  watch(keyword, () => {
    Object.values(paginationByGroup).forEach(item => {
      item.current = 1;
    });
  });
</script>

<style lang="postcss" scoped>
  .framework-template-card {
    box-shadow: 0 2px 4px 0 #1919290d;
  }

  :deep(.bkms-content-title) {
    height: 42px;
    background-color: #eaebf0;
  }

  /* 表头：背景白色 + 默认文字颜色（与「应用默认配置」卡片表格一致） */
  :deep(.framework-template-table .vxe-header--column) {
    color: #4d4f56;
    background-color: #fff;
  }

  /* 表格正文：默认文字颜色 */
  :deep(.framework-template-table .vxe-body--column .vxe-cell) {
    color: #4d4f56;
  }
</style>
