/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

import { type PathArray, BaseBehavior, ExtensionCategory, Polyline, register } from '@antv/g6';

import type { Group } from '@antv/g';
import type {
  BaseBehaviorOptions,
  EdgeData,
  IElementEvent,
  IPointerEvent,
  PolylineStyleProps,
  RuntimeContext,
} from '@antv/g6';
interface ShowAuxiliaryEdgesOptions extends BaseBehaviorOptions {
  auxiliaryEdges?: EdgeData[];
}

/**
 * 辅助边 — 虚线折线，用于非主要依赖关系（hover 节点时显示）
 * 路径形状：根据源节点和目标节点的相对位置动态规划路径
 */
class AuxiliaryEdge extends Polyline {
  /** 平行判断阈值（坐标差小于此值算平行） */
  static parallelThreshold = 10;
  /** 非平行时起点偏移距离 */
  static startOffset = 20;
  /** 非平行时拐点偏移距离 */
  static turnOffset = 20;
  /** U 型线偏移距离 */
  static uShapeOffset = 20;

  protected getKeyPath(): PathArray {
    const { sourceNode, targetNode } = this;
    const sourceBBox = sourceNode.getBBox();
    const targetBBox = targetNode.getBBox();
    const [x1, y1] = sourceNode.getPosition();
    const [x2, y2] = targetNode.getPosition();

    const w1 = sourceBBox.width;
    const h1 = sourceBBox.height;
    const w2 = targetBBox.width;
    const h2 = targetBBox.height;

    // 计算中心点（用于判断方向）
    const sourceCenterX = x1 + w1 / 2;
    const sourceCenterY = y1 + h1 / 2;
    const targetCenterX = x2 + w2 / 2;
    const targetCenterY = y2 + h2 / 2;

    const dx = targetCenterX - sourceCenterX;
    const dy = targetCenterY - sourceCenterY;

    const threshold = AuxiliaryEdge.parallelThreshold;
    const uOffset = AuxiliaryEdge.uShapeOffset;
    const startOff = AuxiliaryEdge.startOffset;
    const turnOff = AuxiliaryEdge.turnOffset;

    // 情况1：左右平行（Y 坐标接近）
    if (Math.abs(dy) < threshold) {
      // 起点：source 顶部中心
      const startX = sourceCenterX;
      const startY = y1;
      // 终点：target 顶部中心
      const endX = targetCenterX;
      const endY = y2;
      // U 型线：向上偏移
      const offsetY = Math.min(y1, y2) - uOffset;

      return [
        ['M', startX, startY],
        ['L', startX, offsetY],
        ['L', endX, offsetY],
        ['L', endX, endY],
      ];
    }

    // 情况2：上下平行（X 坐标接近）
    if (Math.abs(dx) < threshold) {
      // 起点：source 右侧中心
      const startX = x1 + w1;
      const startY = sourceCenterY;
      // 终点：target 右侧中心
      const endX = x2 + w2;
      const endY = targetCenterY;
      // U 型线：向右偏移
      const offsetX = Math.max(x1 + w1, x2 + w2) + uOffset;

      return [
        ['M', startX, startY],
        ['L', offsetX, startY],
        ['L', offsetX, endY],
        ['L', endX, endY],
      ];
    }

    // 情况3：非平行（4 个方向）
    // 起点：source 的第一个方向对应的边
    // 终点：target 的第二个方向的相反边
    if (dx > 0 && dy > 0) {
      // 右下：起点在 source 右侧，终点在 target 顶部
      const startX = x1 + w1;
      const startY = sourceCenterY;
      const endX = targetCenterX;
      const endY = y2;

      const path: PathArray = [
        ['M', startX, startY],
        ['L', startX + startOff, startY],
        ['L', startX + startOff, endY - turnOff],
        ['L', endX, endY - turnOff],
        ['L', endX, endY],
      ];
      return path;
    }

    if (dx > 0 && dy < 0) {
      // 右上：起点在 source 右侧，终点在 target 底部
      const startX = x1 + w1;
      const startY = sourceCenterY;
      const endX = targetCenterX;
      const endY = y2 + h2;

      const path: PathArray = [
        ['M', startX, startY],
        ['L', startX + startOff, startY],
        ['L', startX + startOff, endY + turnOff],
        ['L', endX, endY + turnOff],
        ['L', endX, endY],
      ];
      return path;
    }

    if (dx < 0 && dy > 0) {
      // 左下：起点在 source 左侧，终点在 target 顶部
      const startX = x1;
      const startY = sourceCenterY;
      const endX = targetCenterX;
      const endY = y2;

      const path: PathArray = [
        ['M', startX, startY],
        ['L', startX - startOff, startY],
        ['L', startX - startOff, endY - turnOff],
        ['L', endX, endY - turnOff],
        ['L', endX, endY],
      ];
      return path;
    }

    if (dx < 0 && dy < 0) {
      // 左上：起点在 source 左侧，终点在 target 底部
      const startX = x1;
      const startY = sourceCenterY;
      const endX = targetCenterX;
      const endY = y2 + h2;

      const path: PathArray = [
        ['M', startX, startY],
        ['L', startX - startOff, startY],
        ['L', startX - startOff, endY + turnOff],
        ['L', endX, endY + turnOff],
        ['L', endX, endY],
      ];
      return path;
    }

    // 默认返回直线（理论上不会走到这里）
    return [
      ['M', x1 + w1, sourceCenterY],
      ['L', x2, targetCenterY],
    ];
  }

  protected getKeyStyle(attributes: Required<PolylineStyleProps>) {
    return {
      ...super.getKeyStyle(attributes),
      stroke: '#ABB5CC',
      lineWidth: 2,
      lineDash: [4, 4],
    };
  }

  render(attributes: Required<PolylineStyleProps>, container: Group) {
    super.render(attributes, container);
    this.drawTargetArrow({
      ...attributes,
      endArrow: true,
    });
  }
}

/**
 * 自定义 Behavior：hover 辅助边时高亮（添加 active 状态），离开时取消高亮
 */
class HighlightAuxiliaryEdgeOnHover extends BaseBehavior {
  private onEdgePointerEnter = (event: IElementEvent) => {
    const { graph } = this.context;
    const edgeId = event.target.id;
    const edgeData = graph.getEdgeData(edgeId);
    if (edgeData?.data?.isPrimary) return;

    const currentStates = graph.getElementState(edgeId);
    if (!currentStates.includes('active')) {
      graph.setElementState(edgeId, [...currentStates, 'active']);
    }
  };

  private onEdgePointerLeave = (event: IElementEvent) => {
    const { graph } = this.context;
    const edgeId = event.target.id;
    const edgeData = graph.getEdgeData(edgeId);
    if (edgeData && !edgeData.data?.isPrimary) {
      const currentStates = graph.getElementState(edgeId);
      graph.setElementState(
        edgeId,
        currentStates.filter(s => s !== 'active'),
      );
    }
  };

  private bindEvents() {
    const { graph } = this.context;
    graph.on('edge:pointerenter', this.onEdgePointerEnter);
    graph.on('edge:pointerleave', this.onEdgePointerLeave);
  }

  private unbindEvents() {
    const { graph } = this.context;
    graph.off('edge:pointerenter', this.onEdgePointerEnter);
    graph.off('edge:pointerleave', this.onEdgePointerLeave);
  }

  constructor(context: RuntimeContext, options: BaseBehaviorOptions) {
    super(context, options);
    this.bindEvents();
  }

  destroy(): void {
    this.unbindEvents();
    super.destroy();
  }
}

/**
 * 主边 — 用于树形结构主关系
 * 路径形状：源节点右侧 → 水平 → 圆弧转角 → 竖直 → 圆弧转角 → 水平 → 目标节点左侧
 */
class PrimaryEdge extends Polyline {
  static edgeOffset = -1;
  static lineWidth = 2;
  /** 转角圆弧半径 */
  static radius = 8;

  protected getKeyPath(): PathArray {
    const { sourceNode, targetNode } = this;
    const { width, height } = sourceNode.getBBox();
    const [x1, y1] = sourceNode.getPosition();
    const [x2, y2] = targetNode.getPosition();

    const offset = PrimaryEdge.edgeOffset;
    const r = PrimaryEdge.radius;

    // 起点：源节点右侧中心
    const startX = x1 + width;
    const startY = y1 + height / 2 - offset;
    // 终点：目标节点左侧中心
    const endX = x2;
    const endY = y2 + height / 2 - offset;
    // 竖直主干的 X 坐标（源和目标中间）
    const midX = (x1 + x2) / 2 + width / 2 - PrimaryEdge.lineWidth / 2;

    // 如果 Y 坐标相同（水平直线），无需转角
    if (Math.abs(startY - endY) < 1) {
      return [
        ['M', startX, startY],
        ['L', endX, endY],
      ];
    }

    const dy = endY - startY;
    const dirY = dy > 0 ? 1 : -1;
    const absDy = Math.abs(dy);
    // 如果垂直距离不足两倍圆弧半径，缩小半径以适应
    const effectiveR = Math.min(r, absDy / 2, Math.abs(midX - startX), Math.abs(endX - midX));

    const path: PathArray = [
      // 起点
      ['M', startX, startY],
      // 水平线到第一个转角前
      ['L', midX - effectiveR, startY],
      // 第一个圆弧转角（水平 → 竖直）
      ['A', effectiveR, effectiveR, 0, 0, dirY > 0 ? 1 : 0, midX, startY + dirY * effectiveR],
      // 竖直线到第二个转角前
      ['L', midX, endY - dirY * effectiveR],
      // 第二个圆弧转角（竖直 → 水平）
      ['A', effectiveR, effectiveR, 0, 0, dirY > 0 ? 0 : 1, midX + effectiveR, endY],
      // 水平线到终点
      ['L', endX, endY],
    ];

    return path;
  }

  protected getKeyStyle(attributes: Required<PolylineStyleProps>) {
    return {
      ...super.getKeyStyle(attributes),
      lineWidth: PrimaryEdge.lineWidth,
      stroke: '#ABB5CC',
    };
  }

  render(attributes: Required<PolylineStyleProps>, container: Group) {
    super.render(attributes, container);
    this.drawTargetArrow({
      ...attributes,
      endArrow: true,
    });
  }
}

let selectedNodeId: null | string = null;
let isDragging = false;
/**
 * 自定义 Behavior：hover 节点时动态绘制/移除辅助边
 * 辅助边不参与布局计算，仅在 hover 时按需添加到画布
 */
class ShowAuxiliaryEdgesOnHover extends BaseBehavior<ShowAuxiliaryEdgesOptions> {
  private activeEdgeIds: string[] = [];
  private onCanvasClick = () => {
    selectedNodeId = null;
    this.removeActiveAuxiliaryEdges();
  };

  private onDragend = () => {
    isDragging = false;
  };

  private onDragstart = () => {
    isDragging = true;
  };

  private onNodeClick = (event: IElementEvent) => {
    const nodeId = (event.target as unknown as { id?: string })?.id;
    if (!nodeId) return;
    this.addAuxiliaryEdges(nodeId);
    // 上面方法是异步的，这里调用添加边逻辑后再更新选中节点ID
    selectedNodeId = selectedNodeId === nodeId ? null : nodeId;
  };

  private onNodePointerEnter = (event: IPointerEvent) => {
    // 画布拖拽时, 不触发节点相关事件
    if (isDragging) return;
    const nodeId = (event.target as unknown as { id?: string })?.id;
    const domTarget = (event as IPointerEvent & { originalEvent?: PointerEvent }).originalEvent
      ?.srcElement as unknown as HTMLElement | undefined;

    // 防止点击折叠按钮时触发节点相关事件
    if (domTarget?.classList?.contains('custom-collapse') || !nodeId || selectedNodeId) return;
    this.addAuxiliaryEdges(nodeId);
  };

  private onNodePointerLeave = () => {
    if (selectedNodeId) return;
    this.removeActiveAuxiliaryEdges();
  };

  static defaultOptions: Partial<ShowAuxiliaryEdgesOptions> = {
    auxiliaryEdges: [],
  };

  /**
   * 添加辅助边
   * @param nodeId 节点ID
   */
  private async addAuxiliaryEdges(nodeId: string) {
    await this.removeActiveAuxiliaryEdges();

    const { graph } = this.context;
    const allAuxEdges = this.options.auxiliaryEdges ?? [];
    // 过滤：只保留 source 和 target 都可见的辅助边
    const related = allAuxEdges.filter(e => {
      if (e.source !== nodeId && e.target !== nodeId) return false;
      const sourceVisible = graph.getElementVisibility(e.source as string) !== 'hidden';
      const targetVisible = graph.getElementVisibility(e.target as string) !== 'hidden';
      return sourceVisible && targetVisible;
    });

    if (!related.length) return;

    this.activeEdgeIds = related.map(e => e.id!);
    graph.addEdgeData(related);
    await graph.draw();
  }

  private bindEvents() {
    const { graph } = this.context;
    graph.on('canvas:click', this.onCanvasClick);
    graph.on('node:click', this.onNodeClick);
    graph.on('node:pointerenter', this.onNodePointerEnter);
    graph.on('node:pointerleave', this.onNodePointerLeave);
    graph.on('canvas:dragstart', this.onDragstart);
    graph.on('canvas:dragend', this.onDragend);
  }

  /**
   * 移除辅助边
   */
  private async removeActiveAuxiliaryEdges() {
    if (!this.activeEdgeIds.length) return;

    const { graph } = this.context;
    const existing = new Set(
      graph
        .getEdgeData()
        .map(e => e.id)
        .filter(Boolean) as string[],
    );
    const toRemove = this.activeEdgeIds.filter(id => existing.has(id));
    this.activeEdgeIds = [];
    if (toRemove.length) {
      graph.removeEdgeData(toRemove);
      await graph.draw();
    }
  }

  private unbindEvents() {
    const { graph } = this.context;
    graph.off('canvas:click', this.onCanvasClick);
    graph.off('node:click', this.onNodeClick);
    graph.off('node:pointerenter', this.onNodePointerEnter);
    graph.off('node:pointerleave', this.onNodePointerLeave);
    graph.off('canvas:dragstart', this.onDragstart);
    graph.off('canvas:dragend', this.onDragend);
  }

  constructor(context: RuntimeContext, options: Partial<ShowAuxiliaryEdgesOptions>) {
    super(context, Object.assign({}, ShowAuxiliaryEdgesOnHover.defaultOptions, options));
    this.bindEvents();
  }

  destroy(): void {
    this.removeActiveAuxiliaryEdges();
    this.unbindEvents();
    super.destroy();
  }

  update(options: Partial<ShowAuxiliaryEdgesOptions>) {
    this.options.auxiliaryEdges = options.auxiliaryEdges ?? this.options.auxiliaryEdges;
  }
}

export function registerCustomExtensions() {
  register(ExtensionCategory.EDGE, 'primary-edge', PrimaryEdge);
  register(ExtensionCategory.EDGE, 'auxiliary-edge', AuxiliaryEdge);
  register(ExtensionCategory.BEHAVIOR, 'show-auxiliary-edges', ShowAuxiliaryEdgesOnHover);
  register(ExtensionCategory.BEHAVIOR, 'highlight-auxiliary-edge', HighlightAuxiliaryEdgeOnHover);
}
