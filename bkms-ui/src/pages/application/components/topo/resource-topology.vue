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
    class="relative bg-[#edf2fc] bg-[radial-gradient(circle,rgba(193,202,219,0.3)_1px,transparent_1px)] bg-[length:16px_16px]"
  >
    <div
      id="topology-graph"
      ref="graphContainerRef"
      class="w-full h-full custom-graph"
    ></div>
    <TopologyToolbar
      class="absolute top-[16px] left-[16px]"
      :graph="graphInstance"
      :minimap-disabled="minimapDisabled"
    />
    <TopologyContextMenu
      :menu-items="contextMenuList"
      :node-id="contextMenu.nodeId"
      :visible="contextMenu.visible"
      :x="contextMenu.x"
      :y="contextMenu.y"
      @close="contextMenu.visible = false"
      @menu-click="handleContextMenuClick"
    />
    <!-- 辅助边 tooltip -->
    <EdgeTooltip
      :reason="edgeTooltip.reason"
      :relation="edgeTooltip.relation"
      :visible="edgeTooltip.visible"
      :x="edgeTooltip.x"
      :y="edgeTooltip.y"
    />
  </div>
</template>

<script lang="ts" setup>
  import { computed, h, nextTick, onBeforeUnmount, onMounted, reactive, ref, shallowRef, watch } from 'vue';

  import { DisplayObject, Group, Rect } from '@antv/g';
  import { ExtensionCategory, Graph, register } from '@antv/g6';
  import { VueNode } from 'g6-extension-vue';
  import { useI18n } from 'vue-i18n';
  import { diffArrayFast } from '~/common/util';

  import {
    CUSTOM_NODE_TYPE,
    DAGRE_LAYOUT_OPTIONS,
    LOG_ALLOWED_STATUSES,
    MINIMAP_NODE_LIMIT,
    MINIMAP_NODE_MAX,
    NODE_HEIGHT,
    NODE_WIDTH,
    normalizeStatus,
    STATUS_CONFIG,
  } from './constants';
  import { registerCustomExtensions } from './custom-edge';
  import EdgeTooltip from './edge-tooltip.vue';
  import TopologyContextMenu from './topology-context-menu.vue';
  import TopoNodeComponent from './topology-node.vue';
  import TopologyToolbar from './topology-toolbar.vue';

  import type { TopoNodeData } from './types';
  import type { EdgeData, IElementEvent, State } from '@antv/g6';
  import type { EdgeReason, TopologyEdge, TopologyNode } from '~/@types/topology';

  const props = withDefaults(
    defineProps<{
      edges?: TopologyEdge[];
      focusedNodeId?: string;
      nodes?: TopologyNode[];
      rootId?: string;
      selectedNodeIds?: string | string[];
      showOnlyNodeIds?: string[];
    }>(),
    {
      nodes: () => [],
      edges: () => [],
      rootId: '',
      focusedNodeId: '',
      // 只展示指定节点
      showOnlyNodeIds: () => [],
      selectedNodeIds: () => [],
    },
  );

  const emit = defineEmits<{
    'menu-click': [action: string, nodeData: TopoNodeData];
    'node-click': [nodeData: TopoNodeData];
  }>();

  const { t } = useI18n();

  const graphContainerRef = ref<HTMLDivElement>();
  const graphInstance = shallowRef<Graph | null>(null);
  // 记录已折叠的节点 ID 集合（避免 computed 重置 collapsed 状态）
  const collapsedNodeIds = ref(new Set<string>());

  const contextMenu = reactive({
    visible: false,
    x: 0,
    y: 0,
    nodeId: '',
    nodeData: null as null | TopoNodeData,
  });

  /** 根据当前右键节点的 kind 动态计算菜单项（非 Pod 不显示日志） */
  const contextMenuList = computed(() => {
    const isPod = contextMenu.nodeData?.kind === 'Pod';
    const items = [
      { id: 'overview', label: t('概览') },
      { id: 'events', label: t('事件') },
      ...(isPod
        ? [
            {
              id: 'log',
              label: t('日志'),
              disabled: !LOG_ALLOWED_STATUSES.some(k => k === contextMenu.nodeData?.status?.toLowerCase()),
              tip: t('实例尚未创建成功或宿主机异常，暂无法获取日志'),
            },
          ]
        : []),
      { id: 'YAML', label: 'YAML' },
    ];
    return items;
  });

  // 辅助边 tooltip 状态
  const edgeTooltip = reactive({
    visible: false,
    x: 0,
    y: 0,
    relation: '',
    reason: undefined as EdgeReason | undefined,
  });

  let extensionsRegistered = false;
  // 节点数量超过 MINIMAP_NODE_MAX 时，禁用 minimap（性能考虑）
  const minimapDisabled = computed(() => props.nodes.length > MINIMAP_NODE_MAX);

  // 合并高亮/聚焦状态，统一为 {highlighted: [], focused: []} 格式
  const nodeStates = computed(() => {
    const highlighted = typeof props.selectedNodeIds === 'string' ? [props.selectedNodeIds] : props.selectedNodeIds;
    return {
      highlighted,
      focused: props.focusedNodeId ? [props.focusedNodeId] : [],
    };
  });
  // 去重后的可见节点 ID 集合
  const visibleNodeIds = computed(() => {
    const ids = props.showOnlyNodeIds;
    return ids.length > 0 ? new Set(ids) : null;
  });

  // 按 sourceID 聚合主边的子节点 ID
  const primaryChildrenMap = computed(() => {
    const map = new Map<string, string[]>();
    for (const edge of props.edges) {
      if (!edge.isPrimary) continue;
      const list = map.get(edge.sourceID!);
      if (list) {
        list.push(edge.targetID!);
      } else {
        map.set(edge.sourceID!, [edge.targetID!]);
      }
    }
    return map;
  });

  // 图节点数据（始终包含全量节点，布局不受 showOnlyNodeIds 影响）
  const graphNodes = computed(() => {
    const childrenMap = primaryChildrenMap.value;
    const collapsedIds = collapsedNodeIds.value;
    // const visible = visibleNodeIds.value;
    return props.nodes.map(node => {
      const children = childrenMap.get(node.id!) ?? [];
      // 有过滤条件时，只有存在可见子节点才认为 hasChildren
      // const hasChildren = visible ? children.some(id => visible.has(id)) : children.length > 0;
      const hasChildren = children.length > 0;
      return {
        id: node.id!,
        data: {
          ...node,
          nodeStatus: normalizeStatus(node.status ?? 'Unknown'),
          collapsed: collapsedIds.has(node.id!),
          hasChildren,
        },
        children,
      };
    });
  });

  // 图主边数据（始终包含全量主边）
  const graphEdges = computed(() =>
    props.edges
      .filter(edge => edge.isPrimary)
      .map(edge => ({
        id: edge.id!,
        source: edge.sourceID!,
        target: edge.targetID!,
        data: { ...edge },
      })),
  );

  // 辅助边数据（hover 时按需添加，也基于全量）
  const auxiliaryEdges = computed(() =>
    props.edges
      .filter(edge => !edge.isPrimary)
      .map(edge => ({
        id: edge.id!,
        source: edge.sourceID!,
        target: edge.targetID!,
        data: { ...edge },
      })),
  );

  // 初始渲染用（initGraph 中使用），后续更新走 watcher 直接监听 props
  const graphData = computed(() => ({
    nodes: graphNodes.value,
    edges: graphEdges.value,
  }));

  /** 根据 showOnlyNodeIds 设置节点/边的可见性 */
  async function applyVisibility() {
    const graph = graphInstance.value;
    if (!graph) return;

    const visible = visibleNodeIds.value;

    const all = graph.getData();
    // 收集需要显示/隐藏的 ID，批量操作
    const showIds: string[] = [];
    const hideIds: string[] = [];
    // 方案 3 优化：建立节点 ID → 节点数据的映射（O(n) 一次），避免后续的 find 查找
    const nodeDataMap = new Map<string, (typeof all.nodes)[0]>();

    if (!visible) {
      for (const n of all.nodes) {
        nodeDataMap.set(n.id!, n);
        showIds.push(n.id!);
      }
      for (const e of all.edges) {
        showIds.push(e.id!);
      }
    } else {
      for (const n of all.nodes) {
        nodeDataMap.set(n.id!, n);
        if (visible.has(n.id!)) {
          showIds.push(n.id!);
        } else {
          hideIds.push(n.id!);
        }
      }

      for (const e of all.edges) {
        const shouldShow = visible.has(e.source as string) && visible.has(e.target as string);
        if (shouldShow) {
          showIds.push(e.id!);
        } else {
          hideIds.push(e.id!);
        }
      }
    }

    // 路径 4: 过滤条件变化时，同步更新受影响节点的 hasChildren 字段
    // （之前通过 graphNodes computed → watcher 全量 diff 实现，现在改为精确更新）
    // 注意：必须在 show/hide 之后、draw 之前执行，且无论 visible 是否为 null 都要执行
    const childrenMap = primaryChildrenMap.value;
    const hasChildrenUpdates: Array<{ data: { hasChildren: boolean }; id: string }> = [];

    // 方案 3 优化：直接遍历有子节点的节点（O(k)，k = 有子节点的节点数），跳过叶子节点
    for (const [nodeId, children] of childrenMap) {
      const newHasChildren = !visible || children.some(id => visible.has(id));

      // 从映射中获取节点数据（O(1)），避免 Array.find() 的 O(n) 查找
      const nodeData = nodeDataMap.get(nodeId);
      const currentHasChildren = !!(nodeData?.data as Record<string, unknown>)?.hasChildren;

      if (currentHasChildren !== newHasChildren) {
        hasChildrenUpdates.push({ data: { hasChildren: newHasChildren }, id: nodeId });
      }
    }
    if (hasChildrenUpdates.length > 0) {
      graph.updateNodeData(hasChildrenUpdates);
    }

    // 根据 showIds 和 hideIds 设置节点/边的可见性
    graph.showElement(showIds, false);
    if (visible) {
      graph.hideElement(hideIds, false);
    }
  }

  // 构建图主边数据（供 updateGraphEdges 使用，不依赖 graphEdges computed）
  function buildGraphEdgeData() {
    return props.edges
      .filter(edge => edge.isPrimary)
      .map(edge => ({
        id: edge.id!,
        source: edge.sourceID!,
        target: edge.targetID!,
        data: { ...edge },
      }));
  }

  // 构建图节点数据（供 updateGraphNodes 使用，不依赖 graphNodes computed）
  function buildGraphNodeData() {
    const childrenMap = primaryChildrenMap.value;
    const collapsedIds = collapsedNodeIds.value;
    const visible = visibleNodeIds.value;
    return props.nodes.map(node => {
      const children = childrenMap.get(node.id!) ?? [];
      const hasChildren = visible ? children.some(id => visible.has(id)) : children.length > 0;
      return {
        id: node.id!,
        data: {
          ...node,
          nodeStatus: normalizeStatus(node.status ?? 'Unknown'),
          collapsed: collapsedIds.has(node.id!),
          hasChildren,
        },
        children,
      };
    });
  }

  function extractNodeData(nodeId: string): null | TopoNodeData {
    const data = graphInstance.value?.getNodeData(nodeId);
    return (data?.data as unknown as TopoNodeData) ?? null;
  }

  function handleContextMenuClick(action: string, nodeId: string) {
    const nodeData = extractNodeData(nodeId);
    if (nodeData) {
      emit('menu-click', action, nodeData);
    }
  }

  // 初始化图表
  function initGraph() {
    if (!graphContainerRef.value) return;

    registerExtensions();

    const graph = new Graph({
      container: graphContainerRef.value,
      autoFit: 'center',
      padding: [60, 40, 40, 40],
      data: graphData.value,
      // 限制缩放范围
      zoomRange: [0.25, 1.5],
      node: {
        type: CUSTOM_NODE_TYPE,
        style: {
          component: (data: Record<string, unknown>) =>
            h(TopoNodeComponent, {
              data: Object.assign({}, data) as InstanceType<typeof TopoNodeComponent>['$props']['data'],
              nodeCount: props.nodes.length,
              onToggleCollapse: togglePrimaryCollapse,
            }),
          size: [NODE_WIDTH, NODE_HEIGHT],
          ports: [
            { key: 'left', placement: [0, 0.5] },
            { key: 'right', placement: [1, 0.5] },
            { key: 'top', placement: [0.5, 0] },
            { key: 'bottom', placement: [0.5, 1] },
          ],
        },
      },
      edge: {
        type: (data: EdgeData) => (data.data?.isPrimary ? 'primary-edge' : 'auxiliary-edge'),
        style: {
          router: {
            type: 'orth',
          },
        },
      },
      layout: DAGRE_LAYOUT_OPTIONS,
      behaviors: [
        'drag-canvas',
        'zoom-canvas',
        // 点击选中节点
        {
          type: 'click-select',
          key: 'click-select',
          state: 'selected',
        },
        {
          type: 'show-auxiliary-edges',
          key: 'show-auxiliary-edges',
          auxiliaryEdges: auxiliaryEdges.value,
        },
        // hover高亮辅助边
        'highlight-auxiliary-edge',
      ],
      plugins: [
        // 节点数量超过 MINIMAP_NODE_MAX 时，禁用 minimap（性能考虑）
        ...(minimapDisabled.value
          ? []
          : [
              {
                key: 'minimap',
                type: 'minimap',
                size: [300, 180],
                // 缩略图挂载到toolbar中的custom-minimap上
                container: document.querySelector('.custom-minimap'),
                containerStyle: {
                  background: '#242A35',
                },
                maskStyle: {
                  background: 'transparent',
                  boxShadow: '0 0 10000px 10000px #0000002d',
                },
                filter: (id: string, elementType: string) => {
                  if (elementType !== 'node') return true;
                  const visible = visibleNodeIds.value;
                  return !visible || visible.has(id);
                },
                shape: (id: string, elementType: string, element: DisplayObject) => {
                  if (elementType === 'node') {
                    const nodeData = graphInstance.value?.getNodeData(id);
                    const status = (nodeData?.data as unknown as TopoNodeData)?.nodeStatus ?? 'unknown';
                    const fill = STATUS_CONFIG[status]?.miniColor || '#99B1E0';

                    const group = new Group();

                    // 白色圆角底板
                    group.appendChild(
                      new Rect({
                        style: {
                          x: 0,
                          y: 0,
                          width: NODE_WIDTH,
                          height: NODE_HEIGHT,
                          fill: '#FFFFFF',
                          radius: 10,
                        },
                      }),
                    );

                    // 节点数量大于 MINIMAP_NODE_LIMIT 时，不绘制内嵌左侧状态色块（minimap 性能优化）
                    if (props.nodes.length <= MINIMAP_NODE_LIMIT) {
                      const padding = 6;
                      const iconSize = NODE_HEIGHT - padding * 2;

                      // 内嵌左侧状态色块
                      group.appendChild(
                        new Rect({
                          style: {
                            x: padding,
                            y: padding,
                            width: iconSize,
                            height: iconSize,
                            fill,
                            radius: 6,
                          },
                        }),
                      );
                    }

                    return group;
                  }
                  // 非节点元素（边等）返回元素本身，minimap 会自动克隆
                  return element;
                },
              },
            ]),
      ],
      // 禁用动画
      animation: false,
    });

    graph.on('node:click', (event: IElementEvent) => {
      const nodeData = extractNodeData(event.target.id);
      if (nodeData && !isRootNode(nodeData)) emit('node-click', nodeData);
    });

    graph.on('node:contextmenu', (event: IElementEvent) => {
      event.preventDefault?.();
      const nodeId = event.target.id;
      const nodeData = extractNodeData(nodeId);
      if (!nodeData || isRootNode(nodeData)) return;

      contextMenu.visible = true;
      contextMenu.x = event.client.x;
      contextMenu.y = event.client.y;
      contextMenu.nodeId = nodeId;
      contextMenu.nodeData = nodeData;
    });

    // 辅助边 hover：在鼠标位置显示 tooltip
    graph.on('edge:pointerenter', (event: IElementEvent) => {
      const edgeId = event.target.id;
      const edgeData = graph.getEdgeData(edgeId);
      if (edgeData?.data?.isPrimary) return;

      const data = edgeData?.data as TopologyEdge | undefined;
      if (!data) return;

      // 鼠标位置（客户端坐标）→ 相对于 graph 容器
      const rect = graphContainerRef.value!.getBoundingClientRect();
      edgeTooltip.x = (event.client?.x ?? 0) - rect.left;
      edgeTooltip.y = (event.client?.y ?? 0) - rect.top;
      edgeTooltip.relation = data.relation || '';
      edgeTooltip.reason = data.reason;
      edgeTooltip.visible = true;
    });

    graph.on('edge:pointerleave', () => {
      edgeTooltip.visible = false;
    });

    graphInstance.value = graph;
    graph.render();

    // 修复：鼠标拖拽画布移出区域后，drag-canvas 拖拽状态无法正常终止
    // 离开画布时移除 drag-canvas 行为，回来时恢复
    const behaviorsCfg = graph.getOptions().behaviors!;
    graphContainerRef.value!.addEventListener('pointerleave', () => {
      graph.setBehaviors(behaviorsCfg.filter(b => !(typeof b === 'string' && b === 'drag-canvas')));
    });
    graphContainerRef.value!.addEventListener('pointerenter', () => {
      graph.setBehaviors(behaviorsCfg);
    });
  }

  /** 根节点（App）没有详情接口，不展示右键菜单和双击详情 */
  function isRootNode(nodeData: TopoNodeData): boolean {
    return nodeData.kind === 'App';
  }

  // 注册自定义节点，自定义边，自定义行为
  function registerExtensions() {
    if (extensionsRegistered) return;
    register(ExtensionCategory.NODE, CUSTOM_NODE_TYPE, VueNode);
    registerCustomExtensions();
    extensionsRegistered = true;
  }

  // 切换主边折叠状态
  async function togglePrimaryCollapse(nodeId: string) {
    const graph = graphInstance.value;
    if (!graph) return;
    const nodeData = graph.getNodeData(nodeId);
    if (!nodeData?.data) return;
    const collapsed = !nodeData.data.collapsed;
    // 更新折叠状态集合
    if (collapsed) {
      collapsedNodeIds.value.add(nodeId);
      await graph.collapseElement(nodeId);
    } else {
      collapsedNodeIds.value.delete(nodeId);
      await graph.expandElement(nodeId);
    }
    graph.updateNodeData([{ id: nodeId, data: { collapsed } }]);
    // 更新节点数据后，需要重新绘制
    await graph.draw();
  }

  // 监听图表容器大小变化
  let resizeObserver: null | ResizeObserver = null;
  function initResizeObserver() {
    if (!graphContainerRef.value) return;
    resizeObserver = new ResizeObserver(() => {
      if (!graphInstance.value || !graphContainerRef.value) return;
      const { clientWidth, clientHeight } = graphContainerRef.value;
      if (clientWidth > 0 && clientHeight > 0) {
        graphInstance.value.resize(clientWidth, clientHeight);
      }
    });
    resizeObserver.observe(graphContainerRef.value);
  }

  // 更新边策略（只 diff 主边：hover 动态添加的辅助边不在 graphEdges 里，否则会误判为 removed 并删掉，且与 Behavior 内 activeEdgeIds 不同步）
  async function updateGraphEdges() {
    if (!graphInstance.value) return;
    const edges = graphInstance.value.getEdgeData().filter(e => (e.data as TopologyEdge | undefined)?.isPrimary);
    const { added, removed, common } = diffArrayFast(edges, buildGraphEdgeData(), 'id');

    if (added.length > 0) {
      graphInstance.value.addEdgeData(added);
    }

    if (removed.length > 0) {
      const ids = removed.map(edge => edge.id!);
      graphInstance.value.removeEdgeData(ids as string[]);
    }

    if (common.length > 0) {
      graphInstance.value.updateEdgeData(common);
    }

    // 绘制边（不执行布局）
    await graphInstance.value.draw();

    // 如果新增或删除节点，则重新布局
    if (added.length > 0 || removed.length > 0) {
      await graphInstance.value.layout();
    }
  }

  // 更新节点策略
  async function updateGraphNodes() {
    if (!graphInstance.value) return;
    const nodes = graphInstance.value.getNodeData();
    const { added, removed, common } = diffArrayFast(nodes, buildGraphNodeData(), 'id');

    if (added.length > 0) {
      graphInstance.value.addNodeData(added);
    }

    if (removed.length > 0) {
      graphInstance.value.removeNodeData(removed.map(node => node.id!));
    }

    if (common.length > 0) {
      graphInstance.value.updateNodeData(common);
    }
    // 绘制节点（不执行布局）
    await graphInstance.value.draw();

    // 如果新增或删除节点，则重新布局
    if (added.length > 0 || removed.length > 0) {
      await graphInstance.value.layout();
    }
  }

  // 路径 1: 外部节点数据变化 → 全量 diff + draw + layout
  watch(
    () => props.nodes,
    () => updateGraphNodes(),
    { deep: true },
  );

  // 路径 1 (边): 外部边数据变化 → 全量 diff + draw + layout
  watch(
    () => props.edges,
    () => updateGraphEdges(),
    { deep: true },
  );
  watch(
    auxiliaryEdges,
    () => {
      if (!graphInstance.value) return;
      graphInstance.value.updateBehavior({
        key: 'show-auxiliary-edges',
        auxiliaryEdges: auxiliaryEdges.value,
      });
    },
    { deep: true },
  );

  watch(visibleNodeIds, () => {
    applyVisibility();
  });

  // 监听搜索命中节点变化：focused = 当前命中项（橙色），highlighted = 全部命中项（浅橙色）
  watch(nodeStates, async ({ focused, highlighted }) => {
    const graph = graphInstance.value;
    if (!graph) return;

    const { nodes } = graph.getData();
    const stateMap: Record<string, State[]> = {};

    // 先清除旧状态
    for (const n of nodes) {
      stateMap[n.id] = graph.getElementState(n.id).filter(s => s !== 'focused' && s !== 'highlighted');
    }

    // 构建新状态，格式与 G6 setElementState 对齐
    const states: Record<string, string[]> = { highlighted, focused };
    for (const [state, ids] of Object.entries(states)) {
      for (const id of ids) {
        stateMap[id]?.push(state as State);
      }
    }

    await graph.setElementState(stateMap, false);

    // 聚焦到目标节点（视口动画）
    if (focused.length > 0) {
      await graph.focusElement(focused[0], { duration: 300 });
    }
  });

  onMounted(() => {
    initResizeObserver();
    nextTick(() => initGraph());
  });

  onBeforeUnmount(() => {
    resizeObserver?.disconnect();
    resizeObserver = null;
    graphInstance.value?.destroy();
    graphInstance.value = null;
  });
</script>

<style scoped>
  :deep(.custom-graph) {
    canvas {
      image-rendering: -webkit-optimize-contrast;
      image-rendering: crisp-edges;
    }
  }
</style>
