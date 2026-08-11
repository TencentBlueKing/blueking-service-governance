<template>
  <Dialog
    v-model:is-show="visible"
    :title="$t('添加告警组')"
    width="1000"
    @closed="handleClosed"
    @shown="handleShown"
  >
    <div class="flex flex-col gap-[12px]">
      <!-- 搜索框 -->
      <Input
        v-model="keyword"
        class="ml-auto w-[480px]"
        clearable
        :placeholder="$t('搜索告警组名称、成员、通知渠道配置')"
        type="search"
      >
      </Input>

      <!-- 告警组勾选表格 -->
      <Table
        ref="tableRef"
        :data="filteredGroups"
        :max-height="500"
        @checkbox-all="handleSelectAll"
        @checkbox-change="handleSelectChange"
        @row-click="handleRowClick"
        key="tableKey"
      >
        <template #empty>
          <TableException type="empty" />
        </template>
        <!-- 复选框列 -->
        <TableColumn
          type="checkbox"
          width="50"
        />
        <!-- 告警组名称 -->
        <TableColumn
          field="name"
          :label="$t('告警组名称')"
          :min-width="160"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            <span class="text-[#313238]">{{ row.name || '--' }}</span>
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
              <template #default="{ item }">
                <span class="flex items-center gap-[4px] rounded-[2px] px-[8px] bg-[#F0F1F5] cursor-default">
                  <i class="bkms-icon bkms-icon-usergroup text-[#979BA5] text-[14px]"></i>
                  {{ item.display_name || item.id || '--' }}
                </span>
              </template>
            </MoreTag>
            <span v-else>--</span>
          </template>
        </TableColumn>
        <!-- 通知渠道配置 -->
        <!-- <TableColumn
          :label="$t('通知渠道配置')"
          :min-width="200"
        >
          <template #default="{ row }">
            <div
              v-if="row.channels?.length"
              class="flex items-center gap-[6px]"
            >
              <Tag
                v-for="ch in row.channels"
                :key="ch"
                class="!mr-0"
              >
                {{ channelTextMap[ch] || ch }}
              </Tag>
            </div>
            <span v-else>--</span>
          </template>
        </TableColumn> -->
      </Table>
    </div>

    <template #footer>
      <div class="flex items-center justify-end gap-[12px]">
        <span class="text-[12px] text-[#979BA5]">
          {{ $t('已选 {n} 个告警组', { n: checkedIds.length }) }}
        </span>
        <Button
          theme="primary"
          @click="handleConfirm"
        >
          {{ $t('确定') }}
        </Button>
        <Button @click="visible = false">
          {{ $t('关闭') }}
        </Button>
      </div>
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Dialog, Input } from 'bkui-vue';
  import TableException from '~/components/table-exception.vue';

  import type { UserGroup } from '~/@types/v1/bkintegrations-bkmonitor';

  interface Emits {
    (e: 'update:isShow', value: boolean): void;
    (e: 'confirm', ids: number[]): void;
  }

  /** vxe-table 实例引用，用于行点击时联动 checkbox */
  interface ITableRef {
    getVxeTableInstance?: () => { setCheckboxRow?: (row: UserGroup, checked: boolean) => void };
  }

  interface Props {
    isShow: boolean;
    selectedIds?: number[];
    userGroups: UserGroup[];
  }

  const props = withDefaults(defineProps<Props>(), {
    selectedIds: () => [],
  });

  const emit = defineEmits<Emits>();

  const visible = computed({
    get: () => props.isShow,
    set: (val: boolean) => emit('update:isShow', val),
  });

  const keyword = ref('');
  /** 当前弹窗内勾选的告警组 ID 列表（确认前为本地态，确认后 emit） */
  const checkedIds = ref<number[]>([]);
  const tableRef = ref<ITableRef>();

  /** 按关键词过滤告警组（名称、成员、渠道） */
  const filteredGroups = computed(() => {
    const kw = keyword.value.trim().toLowerCase();
    if (!kw) return props.userGroups;
    return props.userGroups.filter(g => {
      const name = (g.name || '').toLowerCase();
      const users = (g.users || [])
        .map(u => u.display_name || u.id || '')
        .join('')
        .toLowerCase();
      const channels = (g.channels || []).join('').toLowerCase();
      return name.includes(kw) || users.includes(kw) || channels.includes(kw);
    });
  });

  function handleClosed() {
    keyword.value = '';
  }

  function handleConfirm() {
    emit('confirm', [...checkedIds.value]);
    visible.value = false;
  }

  /** 点击表格行切换该行的选中状态，并联动 checkbox */
  function handleRowClick(_event: Event, row: UserGroup) {
    const id = row.id as number;
    const willChecked = !checkedIds.value.includes(id);
    // 同步表格 checkbox 视觉状态
    tableRef.value?.getVxeTableInstance?.()?.setCheckboxRow?.(row, willChecked);
    // 更新本地选中集合
    if (willChecked) {
      checkedIds.value.push(id);
    } else {
      checkedIds.value = checkedIds.value.filter(i => i !== id);
    }
  }

  function handleSelectAll({ checked }: { checked: boolean }) {
    if (checked) {
      checkedIds.value = filteredGroups.value.map(g => g.id as number);
    } else {
      checkedIds.value = [];
    }
  }

  function handleSelectChange({ checked, row }: { checked: boolean; row: UserGroup }) {
    const id = row.id as number;
    if (checked) {
      if (!checkedIds.value.includes(id)) {
        checkedIds.value.push(id);
      }
    } else {
      checkedIds.value = checkedIds.value.filter(i => i !== id);
    }
  }

  // 弹窗打开时，用外部传入的已选 ID 初始化本地勾选态
  watch(
    () => props.isShow,
    val => {
      if (val) {
        checkedIds.value = [...props.selectedIds];
        keyword.value = '';
      }
    },
  );

  /** Dialog 内容渲染完成后同步 checkbox 勾选状态（首次打开时 nextTick 时机可能过早） */
  function handleShown() {
    syncCheckboxState();
  }

  /** 根据 checkedIds 同步表格 checkbox 的勾选状态 */
  function syncCheckboxState() {
    const vxeTable = tableRef.value?.getVxeTableInstance?.();
    if (!vxeTable?.setCheckboxRow) return;
    filteredGroups.value.forEach(g => {
      const id = g.id as number;
      vxeTable.setCheckboxRow?.(g, checkedIds.value.includes(id));
    });
  }
</script>
