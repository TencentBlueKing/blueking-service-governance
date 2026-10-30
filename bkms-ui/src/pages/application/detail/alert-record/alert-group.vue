<template>
  <div
    ref="containerRef"
    class="rounded-[6px] flex flex-col gap-[16px]"
  >
    <!-- 告警组空间级资源提示 -->
    <div ref="alertRef">
      <Alert
        class="w-full"
        theme="info"
      >
        {{ $t('告警组空间级资源提示') }}
      </Alert>
    </div>

    <!-- 顶部工具栏 -->
    <div
      ref="searchBarRef"
      class="flex items-center justify-between gap-[16px]"
    >
      <Button
        theme="primary"
        @click="handleCreate"
      >
        <Plus
          class="mr-[4px]"
          :height="24"
          :width="24"
        />
        {{ $t('新建告警组') }}
      </Button>
      <SearchSelect
        v-model="searchValue"
        class="w-[480px] bg-[#fff]"
        :data="searchData"
        :placeholder="placeholder"
        unique-select
        value-behavior="need-key"
      />
    </div>

    <!-- 表格：max-height 限制在父容器剩余空间内 -->
    <div class="flex-1">
      <Table
        v-bkloading="{ loading: isLoading, zIndex: 6 }"
        :data="filteredData"
        :max-height="tableMaxHeight"
        :pagination="pagination"
        :row-config="{ isHover: true }"
        @page-limit-change="pageSizeChange"
        @page-value-change="pageChange"
      >
        <template #empty>
          <TableException
            :type="filteredData.length === 0 && hasFilter ? 'search' : 'empty'"
            @clear="handleClearFilters"
          />
        </template>
        <!-- 告警组名称 -->
        <TableColumn
          field="name"
          :label="$t('告警组名称')"
          :min-width="200"
          show-overflow="tooltip"
        >
          <template #default="{ row }">
            <span
              class="text-[#3B6EF5] cursor-pointer"
              @click="handleEdit(row)"
            >
              {{ row.name || '--' }}
            </span>
          </template>
        </TableColumn>
        <!-- 成员 -->
        <TableColumn
          :label="$t('成员')"
          :min-width="200"
        >
          <template #default="{ row }: { row: UserGroup }">
            <MoreTag
              v-if="row.users?.length"
              :data="row.users"
              overflow-mode="popover"
            >
              <template #default="{ item }: { item: UserGroupUser }">
                <span class="flex items-center gap-[4px] rounded-[6px] px-[8px] bg-[#E9EDF5] cursor-default">
                  <i class="bkms-icon bkms-icon-usergroup text-[#8F9FBD] text-[14px]"></i>
                  {{ item.display_name || item.id || '--' }}
                </span>
              </template>
            </MoreTag>
            <span v-else>--</span>
          </template>
        </TableColumn>
        <!-- 关联策略数 -->
        <TableColumn
          field="strategy_count"
          :width="120"
        >
          <template #header>
            <UnderLineTips :description="$t('关联策略数提示')">
              {{ $t('关联策略数') }}
            </UnderLineTips>
          </template>
          <template #default="{ row }: { row: UserGroup }">
            <span>{{ row.strategy_count || '--' }}</span>
          </template>
        </TableColumn>
        <!-- 操作 -->
        <TableColumn
          fixed="right"
          :label="$t('操作')"
          :width="140"
        >
          <template #default="{ row }">
            <div class="flex items-center gap-[12px]">
              <Button
                :disabled="!row.edit_allowed"
                text
                theme="primary"
                @click="handleEdit(row)"
              >
                {{ $t('编辑') }}
              </Button>
              <Button
                v-bk-tooltips="{
                  content: $t('存在关联的策略，不可删除'),
                  disabled: row.delete_allowed !== false,
                }"
                :disabled="row.delete_allowed === false"
                text
                theme="primary"
                @click="handleDelete(row)"
              >
                {{ $t('删除') }}
              </Button>
            </div>
          </template>
        </TableColumn>
      </Table>
    </div>

    <!-- 新建 / 编辑 / 查看 告警组侧滑表单 -->
    <AlertGroupForm
      v-model:is-show="isFormShow"
      :group-i-d="formGroupID"
      :mode="formMode"
      @edit-success="handleEditSuccess"
      @success="handleCreateSuccess"
    />
  </div>
</template>

<script lang="ts" setup>
  import { computed, h, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Button, InfoBox, Message, SearchSelect } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { BkintegrationsBkmonitorService } from '~/api/modules/v1/bkintegrations-bkmonitor';
  import MoreTag from '~/components/more-tag.vue';
  import TableException from '~/components/table-exception.vue';
  import UnderLineTips from '~/components/under-line-tips.vue';
  import { useElementHeight } from '~/composables/use-element-height';
  import usePageConf from '~/composables/use-page';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import { useSpaceStore } from '~/stores/space';

  import AlertGroupForm from './components/alert-group-form.vue';

  import type { ISearchValue } from 'bkui-vue/lib/search-select/utils';
  import type { UserGroup, UserGroupUser } from '~/@types/v1/bkintegrations-bkmonitor';
  import type { ISelectKey } from '~/composables/use-search';

  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();

  const spaceStore = useSpaceStore();

  /** 搜索栏配置 */
  const searchData = shallowRef<ISelectKey<UserGroup>[]>([
    { id: 'name', name: t('告警组名称'), field: 'name', fuzzy: true },
    { id: 'users', name: t('成员'), field: 'users', fuzzy: true },
  ]);

  const placeholder = createPlaceholder({
    type: 'searchSelect',
    labels: [t('告警组名称'), t('成员')],
  });

  const searchValue = ref<ISearchValue[]>([]);

  const tableData = ref<UserGroup[]>([]);
  const isLoading = ref(false);

  /** 新建/编辑/查看 告警组侧滑表单显隐 */
  const isFormShow = ref(false);

  /** 侧滑表单模式 */
  const formMode = ref<'create' | 'edit'>('create');

  /** 编辑/查看模式下的告警组 ID */
  const formGroupID = ref<number>();

  const containerRef = ref<HTMLElement>();
  const alertRef = ref<HTMLElement>();
  const searchBarRef = ref<HTMLElement>();
  /** 卡片内容区高度。父级内边距在测量时扣除 */
  const parentContentHeight = ref(400);
  const { height: alertHeight } = useElementHeight(alertRef, {
    watchSource: isLoading,
    defaultHeight: 32,
  });
  const { height: searchBarHeight } = useElementHeight(searchBarRef, {
    watchSource: isLoading,
    defaultHeight: 40,
  });

  /** 测量父容器内容区高度 */
  function updateParentHeight() {
    const parent = containerRef.value?.parentElement;
    if (!parent) return;
    const rect = parent.getBoundingClientRect();
    const parentStyle = getComputedStyle(parent);
    const padTop = Number.parseFloat(parentStyle.paddingTop) || 0;
    const padBottom = Number.parseFloat(parentStyle.paddingBottom) || 0;
    parentContentHeight.value = rect.height - padTop - padBottom;
  }

  let parentObserver: ResizeObserver | undefined;

  onMounted(() => {
    updateParentHeight();
    const parent = containerRef.value?.parentElement;
    if (parent) {
      parentObserver = new ResizeObserver(updateParentHeight);
      parentObserver.observe(parent);
    }
    window.addEventListener('resize', updateParentHeight);
  });

  onBeforeUnmount(() => {
    parentObserver?.disconnect();
    window.removeEventListener('resize', updateParentHeight);
  });

  /** 是否有任意筛选条件 */
  const hasFilter = computed(() => searchValue.value.length > 0);

  /** 表格最大高度：卡片内容区 - 提示条 - 筛选栏 - 两段间距 16 */
  const tableMaxHeight = computed(() =>
    Math.max(parentContentHeight.value - alertHeight.value - searchBarHeight.value - 16 * 2, 0),
  );

  /** 前端筛选 */
  const filteredData = computed(() => {
    if (searchValue.value.length === 0) return tableData.value;

    return tableData.value.filter(row => {
      return searchValue.value.every(item => {
        const values = (item.values || []).map(v => v.id);
        if (values.length === 0) return true;

        if (item.id === 'name') {
          const keyword = values[0].toLowerCase();
          return String(row.name || '')
            .toLowerCase()
            .includes(keyword);
        }
        if (item.id === 'users') {
          const keyword = values[0].toLowerCase();
          return (row.users || []).some(u => {
            const name = (u.display_name || u.id || '').toLowerCase();
            return name.includes(keyword);
          });
        }
        return true;
      });
    });
  });

  const filteredCount = computed(() => filteredData.value.length);

  const { pagination, pageChange, pageSizeChange, handleResetPage } = usePageConf(
    filteredData,
    {
      current: 1,
      limit: 10,
      remote: false,
    },
    filteredCount,
  );

  /** 拉取告警组列表 */
  async function fetchList() {
    if (!spaceStore.currentSpace) return;
    isLoading.value = true;
    try {
      const res = await BkintegrationsBkmonitorService.listUserGroups({
        workspaceID: spaceStore.currentSpace,
      });
      tableData.value = res?.results ?? [];
    } catch {
      tableData.value = [];
    } finally {
      isLoading.value = false;
    }
  }

  /** 清空所有筛选 */
  function handleClearFilters() {
    searchValue.value = [];
    handleResetPage();
  }

  /** 新建告警组：打开侧滑表单 */
  function handleCreate() {
    formMode.value = 'create';
    formGroupID.value = undefined;
    isFormShow.value = true;
  }

  /** 新建告警组成功：刷新列表并回到第一页 */
  async function handleCreateSuccess() {
    handleResetPage();
    await fetchList();
  }

  /** 删除告警组 */
  function handleDelete(row: UserGroup) {
    if (!spaceStore.currentSpace || !row.id) return;
    InfoBox({
      title: t('确认删除该告警组？'),
      headerAlign: 'center',
      footerAlign: 'center',
      content: [
        h('div', [h('div', { class: 'mt-[14px] py-[12px] px-[16px]' }, [t('删除后，将不可恢复，请谨慎操作！')])]),
      ],
      confirmButtonTheme: 'danger',
      confirmText: t('删除'),
      cancelText: t('取消'),
      async onConfirm() {
        if (!spaceStore.currentSpace || !row.id) return;
        try {
          await BkintegrationsBkmonitorService.deleteUserGroup({
            workspaceID: spaceStore.currentSpace,
            groupID: row.id,
          });
          Message({ theme: 'success', message: t('删除成功') });
          if (
            pagination.value.current > 1 &&
            pagination.value.current > Math.ceil((filteredCount.value - 1) / pagination.value.limit)
          ) {
            pageChange(pagination.value.current - 1);
          }
          await fetchList();
        } catch {
          // 错误由拦截器统一处理
        }
      },
    });
  }

  /** 编辑告警组：设置编辑模式并打开侧滑表单 */
  function handleEdit(row: UserGroup) {
    if (!row.id) return;
    formMode.value = 'edit';
    formGroupID.value = row.id;
    isFormShow.value = true;
  }

  /** 编辑告警组成功：刷新列表 */
  async function handleEditSuccess() {
    await fetchList();
  }

  // 监听搜索值变化，重置分页
  watch(
    searchValue,
    () => {
      handleResetPage();
    },
    { deep: true },
  );

  // 监听 space 变化，首次加载
  watch(
    () => spaceStore.currentSpace,
    async space => {
      if (space) {
        handleResetPage();
        await fetchList();
      } else {
        tableData.value = [];
      }
    },
    { immediate: true },
  );
</script>

<style lang="postcss" scoped></style>
