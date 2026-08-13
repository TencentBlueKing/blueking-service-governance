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
  <div class="h-full py-[12px] px-[16px] flex flex-col bg-[#fff]">
    <div class="leading-[40px] flex items-center justify-between px-[10px]">
      <span class="text-[#313238] font-bold">{{ $t('状态') }}</span>
      <Button
        text
        theme="primary"
        @click="resetFilters"
        >{{ $t('全选') }}</Button
      >
    </div>
    <Select
      v-model="status"
      class="h-[32px]"
      :clearable="false"
      :placeholder="$t('请选择状态')"
      @toggle="isStatusPopoverShow = !isStatusPopoverShow"
    >
      <template #trigger>
        <div
          :class="[
            'px-[12px] flex items-center justify-between group text-[#4D4F56] border-[1px] h-full text-[14px] cursor-pointer rounded-[2px]',
            isStatusPopoverShow ? 'bg-[#fff] border-[#3A84FF]' : 'bg-[#F0F1F5] border-[transparent] hover:bg-[#EAEBF0]',
          ]"
        >
          <div class="flex items-center gap-[8px]">
            <TopologyStatusIcon
              :size="12"
              :type="status"
            />
            <span class="whitespace-nowrap text-[#4D4F56] text-[12px]">
              {{ statusLabel[status] }} ( {{ statusCounts[status] }} )
            </span>
          </div>
          <AngleDownFill
            :class="['text-[#C4C6CC] group-hover:text-[#979BA5]', isStatusPopoverShow ? '' : 'rotate-180']"
          />
        </div>
      </template>
      <Select.Option
        v-for="item in statusOptions"
        :id="item.value"
        :key="item.value"
        :name="item.label"
      >
        <div class="flex items-center gap-[8px]">
          <TopologyStatusIcon
            :size="12"
            :type="item.value"
          />
          <span>{{ item.label }} ( {{ item.count }} )</span>
        </div>
      </Select.Option>
    </Select>

    <div class="flex-1 overflow-y-auto">
      <div
        v-for="item in categorizedNodes"
        :key="item.id"
        class="mt-[4px]"
      >
        <div class="flex items-center h-[40px] px-[10px] text-[#313238] font-bold">{{ $t(item.label) }}</div>

        <FlexRow
          v-for="resource in item.kinds"
          :key="resource.kind"
          class="h-[32px] px-[10px] text-[#979BA5] hover:bg-[#F5F7FA] cursor-pointer"
          :class="{
            '!bg-[#F0F5FF] !text-[#3A84FF]': visibleResourceKinds.includes(resource.kind),
          }"
          lclass="flex items-center gap-[8px]"
          @click="toggleResourceNodes(resource)"
        >
          <template #left>
            <template v-if="KIND_ICON_ASIDE_MAP[resource.kind]">
              <component
                :is="KIND_ICON_ASIDE_MAP[resource.kind]"
                :class="['w-[16px] h-[16px]', visibleResourceKinds.includes(resource.kind) ? '' : 'text-[#979BA5]']"
              />
              <span
                class="max-w-[150px] ellipsis"
                :title="pluralize(resource.kind)"
                >{{ pluralize(resource.kind) }}</span
              >
            </template>
            <template v-else>
              <span
                :class="[
                  'bkms-icon bkms-icon-space-basic text-[16px]',
                  visibleResourceKinds.includes(resource.kind) ? '' : 'text-[#979BA5]',
                ]"
              ></span>
              <span
                class="max-w-[150px] ellipsis"
                :title="resource.kind"
                >{{ resource.kind }}</span
              >
            </template>
          </template>
          <template #right>
            <span
              class="bg-[#F0F1F5] px-[8px] py-[2px] rounded-[2px]"
              :class="{
                '!bg-[#E1ECFF] !text-[#3A84FF]': visibleResourceKinds.includes(resource.kind),
              }"
              >{{ resource.nodes.length }}</span
            >
          </template>
        </FlexRow>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Button, Select } from 'bkui-vue';
  import { AngleDownFill } from 'bkui-vue/lib/icon';
  import pluralize from 'pluralize';
  import { useI18n } from 'vue-i18n';
  import { TopologyNodeOutputObj } from '~/@types/v1/topology';

  /** 统计面板中使用的节点类型（id/kind 保证存在） */
  type SearchNode = Omit<TopologyNodeOutputObj, 'id' | 'kind'> & {
    id: string;
    kind: string;
  };

  import { getKindCategory, getTopologyNodeStatus, RESOURCE_CATEGORIES } from './constants';
  import { KIND_ICON_ASIDE_MAP } from './constants';
  import TopologyStatusIcon from './topology-status-icon.vue';
  import { type CategoryGroup, type NodeStatus, type StatusCounts, KindGroup, TOPO_NODE_STATUS_ORDER } from './types';

  const props = withDefaults(
    defineProps<{
      nodes?: SearchNode[];
    }>(),
    {
      nodes: () => [],
    },
  );

  const visibleNodeIds = defineModel<string[]>('visibleNodeIds', { required: true });

  const { t } = useI18n();

  // 状态筛选
  const status = ref<NodeStatus>('all');
  const statusLabel = computed(
    () =>
      ({
        all: t('全部'),
        healthy: t('正常'),
        error: t('异常'),
        warning: t('告警'),
        unknown: t('其他'),
      }) satisfies Record<NodeStatus, string>,
  );
  const statusCounts = computed<StatusCounts>(() => {
    const counts: StatusCounts = {
      all: props.nodes.length,
      healthy: 0,
      error: 0,
      warning: 0,
      unknown: 0,
    };
    for (const node of props.nodes) counts[getTopologyNodeStatus(node)] += 1;
    return counts;
  });
  const statusOptions = computed(() =>
    TOPO_NODE_STATUS_ORDER.map(status => ({
      label: statusLabel.value[status],
      value: status,
      count: statusCounts.value[status],
    })).filter(item => item.count > 0),
  );
  const isStatusPopoverShow = ref(false);

  // 状态改变时，重置资源过滤为全选并同步更新可见节点
  watch(status, () => {
    visibleResourceKinds.value = [...allKinds.value];
    syncVisibleNodeIds();
  });

  // 节点分类列表
  const nodesForCategoryList = computed(() => {
    if (status.value === 'all') return props.nodes || [];
    return props.nodes.filter(n => getTopologyNodeStatus(n) === status.value);
  });

  const categorizedNodes = computed<CategoryGroup[]>(() =>
    buildCategorizedGroups(nodesForCategoryList.value, t('其他')),
  );

  /** 按类别 -> kind 对节点分组，类别顺序固定，kind 按字母排序；基于当前状态筛选 */
  function buildCategorizedGroups(allNodes: SearchNode[], unknownLabel: string): CategoryGroup[] {
    const byKind = new Map<string, TopologyNodeOutputObj[]>();
    for (const n of allNodes) {
      const list = byKind.get(n.kind);
      if (list) list.push(n);
      else byKind.set(n.kind, [n]);
    }

    const byCategory = new Map(RESOURCE_CATEGORIES.map(c => [c.id, [] as KindGroup[]]));
    const unknownKindGroups: KindGroup[] = [];

    for (const [kind, kindNodes] of byKind) {
      const category = getKindCategory(kind);
      const group: KindGroup = { kind, nodes: kindNodes };
      if (category === 'unknown') unknownKindGroups.push(group);
      else byCategory.get(category)!.push(group);
    }

    const sortKindGroups = (groups: KindGroup[]) => {
      groups.sort((a, b) => a.kind.localeCompare(b.kind));
      return groups;
    };

    const ordered: CategoryGroup[] = RESOURCE_CATEGORIES.map(meta => ({
      icon: meta.icon,
      id: meta.id,
      kinds: sortKindGroups([...byCategory.get(meta.id)!]),
      label: meta.label,
    })).filter(g => g.kinds.length > 0);

    if (unknownKindGroups.length > 0) {
      ordered.push({
        id: 'unknown',
        kinds: sortKindGroups([...unknownKindGroups]),
        label: unknownLabel,
      });
    }

    return ordered;
  }

  /** 恢复默认：状态改为全部，资源过滤全选 */
  function resetFilters() {
    status.value = 'all';
    visibleResourceKinds.value = [...allKinds.value];
    visibleNodeIds.value = [];
  }

  /** 点击资源分类列表中的资源，更新可见节点列表 */
  const allKinds = computed(() => categorizedNodes.value.flatMap(g => g.kinds.map(k => k.kind)));
  const visibleResourceKinds = ref<string[]>([]);

  // 首次加载数据时默认全选
  const initialized = ref(false);
  watch(
    allKinds,
    kinds => {
      if (!initialized.value && kinds.length > 0) {
        initialized.value = true;
        visibleResourceKinds.value = [...kinds];
      }
    },
    { immediate: true },
  );
  /** 根据当前状态和资源过滤，同步可见节点列表 */
  function syncVisibleNodeIds() {
    const isAllSelected = visibleResourceKinds.value.length === allKinds.value.length;
    if (isAllSelected && status.value === 'all') {
      // 全部状态 + 全部资源选中 = 不过滤
      visibleNodeIds.value = [];
    } else if (isAllSelected) {
      // 非全部状态 + 全部资源选中 = 仅按状态过滤
      visibleNodeIds.value = nodesForCategoryList.value.map(node => node.id);
    } else {
      // 按当前状态和资源类型过滤
      const filtered = nodesForCategoryList.value
        .filter(node => visibleResourceKinds.value.includes(node.kind))
        .map(node => node.id);
      // 过滤结果为空时传入占位 ID，避免空数组被视为"不过滤"
      visibleNodeIds.value = filtered.length > 0 ? filtered : ['__none__'];
    }
  }

  function toggleResourceNodes(resource: KindGroup) {
    if (visibleResourceKinds.value.includes(resource.kind)) {
      visibleResourceKinds.value = visibleResourceKinds.value.filter(kind => kind !== resource.kind);
    } else {
      visibleResourceKinds.value.push(resource.kind);
    }
    syncVisibleNodeIds();
  }
</script>
