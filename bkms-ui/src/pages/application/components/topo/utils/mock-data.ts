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

import type {
  RelationType,
  ResourceTopologyDataOutputObj,
  TopologyEdgeOutputObj,
  TopologyMetadataOutputObj,
  TopologyNodeOutputObj,
} from '~/@types/v1/topology';

// ============================================================================
// 类型定义
// ============================================================================

export interface MockTopologyOptions {
  /** 应用 ID（默认随机生成） */
  appID?: string;
  /** 集群 ID（默认 'BCS-K8S-12345'） */
  clusterID?: string;
  /** 环境名称（默认 'dev-bkms'） */
  envName?: string;
  /** 命名空间（默认 'dev-bkms'） */
  namespace?: string;
}

interface NodeDistribution {
  app: number;
  deployment: number;
  pod: number;
  replicaSet: number;
}

// ============================================================================
// 常量
// ============================================================================

const STATUS = {
  App: 'Active',
  Deployment: 'Available',
  ReplicaSet: 'Available',
  Pod: 'Running',
} as const;

const IMAGES = [
  'mirrors.tencent.com/tlinux/tlinux3.1-minimal:latest',
  'mirrors.tencent.com/nginx/nginx:1.25',
  'mirrors.tencent.com/redis/redis:7.2',
  'mirrors.tencent.com/nodejs/nodejs:20',
  'mirrors.tencent.com/postgres/postgres:16',
  'mirrors.tencent.com/go/golang:1.22',
  'mirrors.tencent.com/python/python:3.11',
] as const;

// ============================================================================
// ID 编码工具
// ============================================================================

interface GeneratedNodes {
  app: TopologyNodeOutputObj;
  deployments: TopologyNodeOutputObj[];
  pods: TopologyNodeOutputObj[];
  podsPerRS: number[]; // 每个 ReplicaSet 对应的 Pod 数量
  replicaSets: TopologyNodeOutputObj[];
}

/**
 * 生成大规模 mock 数据（用于性能测试）
 * 默认生成 500 个节点
 */
export function generateLargeMockData(nodeCount = 500, options?: MockTopologyOptions): ResourceTopologyDataOutputObj {
  return generateMockTopologyData(nodeCount, options);
}

/**
 * 生成 mock 资源拓扑数据
 * @param totalNodes 总节点数（至少 4：1 App + 1 Deployment + 1 ReplicaSet + 1 Pod）
 * @param options 可选配置
 * @returns 符合 ResourceTopologyDataOutputObj 结构的数据
 *
 * @example
 * // 生成 100 个节点的拓扑数据
 * const data = generateMockTopologyData(100);
 *
 * @example
 * // 自定义应用信息
 * const data = generateMockTopologyData(50, {
 *   appID: 'my-app-123',
 *   envName: 'production',
 *   namespace: 'prod-ns',
 * });
 */
export function generateMockTopologyData(
  totalNodes: number,
  options?: MockTopologyOptions,
): ResourceTopologyDataOutputObj {
  const opts: Required<MockTopologyOptions> = {
    appID: options?.appID ?? `test-helm-7r7a1b`,
    envName: options?.envName ?? 'dev-bkms',
    namespace: options?.namespace ?? 'dev-bkms',
    clusterID: options?.clusterID ?? 'BCS-K8S-12345',
  };

  const distribution = calculateDistribution(totalNodes);
  const nodes = generateNodes(distribution, opts);
  const edges = generateEdges(nodes);

  // 构建 metadata
  const metadata: TopologyMetadataOutputObj = {
    appID: opts.appID,
    envName: opts.envName,
    trafficLaneName: '',
    clusterID: opts.clusterID,
    namespace: opts.namespace,
  };

  // 构建完整数据
  const result: ResourceTopologyDataOutputObj = {
    metadata,
    nodes: [nodes.app, ...nodes.deployments, ...nodes.replicaSets, ...nodes.pods],
    edges,
    rootID: nodes.app.id!,
    generatedAt: new Date().toISOString(),
    isPartial: false,
    warnings: undefined,
    dataVersion: '1',
  };

  return result;
}

/**
 * 生成简化版的 mock 数据（用于快速测试）
 * 固定生成 10 个节点：1 App + 2 Deployment + 2 ReplicaSet + 5 Pods
 */
export function generateSimpleMockData(options?: MockTopologyOptions): ResourceTopologyDataOutputObj {
  return generateMockTopologyData(10, options);
}

/**
 * 将标准 base64 转换为 base64url（无填充）
 */
function base64ToBase64URL(base64: string): string {
  return base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

// ============================================================================
// 随机数据生成工具
// ============================================================================

/**
 * base64 编码并转换为 URL-safe 格式（无填充）
 * 浏览器环境使用 btoa
 */
function btoaToBase64URL(input: string): string {
  const base64 = btoa(input);
  return base64ToBase64URL(base64);
}

/**
 * 根据总节点数计算各类型节点数量
 * 层级结构：App(1) → Deployment(N) → ReplicaSet(N) → Pod(M)
 * 每个 Deployment 分支消耗：1 Deployment + 1 ReplicaSet + N Pods
 */
function calculateDistribution(totalNodes: number): NodeDistribution {
  if (totalNodes < 4) {
    throw new Error('totalNodes must be at least 4 (1 App + 1 Deployment + 1 ReplicaSet + 1 Pod)');
  }

  // 每个 Deployment 分支平均消耗: 1 Dep + 1 RS + ~6.5 Pods(3~10随机) ≈ 8.5
  const AVG_BRANCH_SIZE = 8.5;
  const remaining = totalNodes - 1; // 减去 App 根节点
  const deploymentCount = Math.max(1, Math.min(Math.round(remaining / AVG_BRANCH_SIZE), Math.floor(remaining / 3)));

  return {
    app: 1,
    deployment: deploymentCount,
    replicaSet: deploymentCount,
    // pod 数量在生成时由 distributePods 随机决定（每个 RS 3-10 个）
    pod: 0, // 占位，实际由 distributePods 动态生成
  };
}

/**
 * 解码 base64url（浏览器环境）
 */
function decodeBase64URL(input: string): string {
  // 先将 base64url 转回标准 base64
  const base64 = input.replace(/-/g, '+').replace(/_/g, '/');
  // 添加填充
  const padded = base64 + '=='.slice(0, (4 - (base64.length % 4)) % 4);
  return atob(padded);
}

/**
 * 为每个 ReplicaSet 随机生成 Pod 数量（3-10），使分布更真实
 */
function distributePods(rsCount: number): number[] {
  return Array.from({ length: rsCount }, () => randomInt(3, 10));
}

// ============================================================================
// 节点分配算法
// ============================================================================

/**
 * 生成边 ID：base64url({sourceShortID}->{targetShortID}:{relation})，无填充
 * 使用短 ID（kind/namespace/name）而非完整 base64，与真实数据格式一致
 */
function encodeEdgeID(sourceID: string, targetID: string, relation: string): string {
  // 从 base64url 解码获取原始路径作为短标识
  const sourcePath = decodeBase64URL(sourceID);
  const targetPath = decodeBase64URL(targetID);
  const raw = `${sourcePath}->${targetPath}:${relation}`;
  return btoaToBase64URL(raw);
}

// ============================================================================
// 节点数据生成
// ============================================================================

/**
 * 生成节点 ID：base64url({kind}/{namespace}/{name})，无填充
 */
function encodeNodeID(kind: string, namespace: string, name: string): string {
  const raw = `${kind}/${namespace}/${name}`;
  return btoaToBase64URL(raw);
}

/**
 * 生成所有边数据
 */
function generateEdges(nodes: GeneratedNodes): TopologyEdgeOutputObj[] {
  const edges: TopologyEdgeOutputObj[] = [];
  const { app, deployments, replicaSets, pods } = nodes;

  // 1. 主边：App → MANAGES → Deployment
  deployments.forEach(deployment => {
    edges.push({
      id: encodeEdgeID(app.id!, deployment.id!, 'MANAGES'),
      sourceID: app.id!,
      targetID: deployment.id!,
      relation: 'MANAGES',
      isPrimary: true,
      reason: {
        type: 'app_root' as RelationType,
        summary: `app ${app.name} manages ${deployment.displayName}`,
        matchedLabels: undefined,
        sourceFieldPath: '',
        targetFieldPath: '',
      },
    });
  });

  // 映射 Deployment → ReplicaSet → Pods
  const deploymentToPods = new Map<string, TopologyNodeOutputObj[]>();
  let podOffset = 0;
  deployments.forEach((deployment, i) => {
    const rs = replicaSets[i];
    const rsPodCount = nodes.podsPerRS[i];
    const rsPods = pods.slice(podOffset, podOffset + rsPodCount);
    podOffset += rsPodCount;
    deploymentToPods.set(deployment.id!, rsPods);

    // 2. 主边：Deployment → CREATES → ReplicaSet
    edges.push({
      id: encodeEdgeID(deployment.id!, rs.id!, 'CREATES'),
      sourceID: deployment.id!,
      targetID: rs.id!,
      relation: 'CREATES',
      isPrimary: true,
      reason: {
        type: 'owner_reference' as RelationType,
        summary: `${deployment.displayName} owns ${rs.displayName}`,
        matchedLabels: undefined,
        sourceFieldPath: '',
        targetFieldPath: '',
      },
    });

    // 3. 主边：ReplicaSet → CREATES → Pod
    rsPods.forEach(pod => {
      edges.push({
        id: encodeEdgeID(rs.id!, pod.id!, 'CREATES'),
        sourceID: rs.id!,
        targetID: pod.id!,
        relation: 'CREATES',
        isPrimary: true,
        reason: {
          type: 'owner_reference' as RelationType,
          summary: `${rs.displayName} owns ${pod.displayName}`,
          matchedLabels: undefined,
          sourceFieldPath: '',
          targetFieldPath: '',
        },
      });
    });
  });

  // 4. 辅助边：Deployment → SELECTS → Pod（label_selector）
  // 每个 Deployment 对其下所有 Pod 建立 SELECTS 边
  deployments.forEach(deployment => {
    const rsPods = deploymentToPods.get(deployment.id!) ?? [];
    rsPods.forEach(pod => {
      // 80% 的 Pod 拥有此辅助边（模拟 K8s label_selector 机制）
      if (Math.random() < 0.8) {
        edges.push({
          id: encodeEdgeID(deployment.id!, pod.id!, 'SELECTS'),
          sourceID: deployment.id!,
          targetID: pod.id!,
          relation: 'SELECTS',
          isPrimary: false,
          reason: {
            type: 'label_selector' as RelationType,
            summary: `${deployment.displayName} selects Pods matching app=${deployment.name}`,
            matchedLabels: {
              app: deployment.name || '',
            },
            sourceFieldPath: '',
            targetFieldPath: '',
          },
        });
      }
    });
  });

  return edges;
}

/**
 * 生成所有节点数据
 */
function generateNodes(distribution: NodeDistribution, options: Required<MockTopologyOptions>): GeneratedNodes {
  const { appID, envName, namespace } = options;

  // 生成 App 根节点
  const appNode: TopologyNodeOutputObj = {
    id: encodeNodeID('App', '', appID),
    kind: 'App',
    namespace: '',
    name: appID,
    displayName: appID,
    status: STATUS.App,
    reason: '',
    isManaged: true,
    extras: undefined,
  };

  // 生成 Deployment 节点
  const deployments: TopologyNodeOutputObj[] = Array.from({ length: distribution.deployment }, (_, i) => {
    const name =
      distribution.deployment === 1 ? `trpc-debug-${envName}-${appID}` : `trpc-debug-${envName}-${appID}-${i + 1}`;
    const displayName = `Deployment/${name}`;
    const replicas = String(randomInt(1, 3));
    return {
      id: encodeNodeID('Deployment', namespace, name),
      kind: 'Deployment',
      namespace,
      name,
      displayName,
      status: STATUS.Deployment,
      reason: '',
      isManaged: true,
      extras: {
        availableReplicas: replicas,
        image: randomItem([...IMAGES]),
        readyReplicas: replicas,
        replicas,
        strategy: 'RollingUpdate',
      },
    };
  });

  // 生成 ReplicaSet 节点
  const replicaSets: TopologyNodeOutputObj[] = deployments.map((deployment, i) => {
    // 使用确定性后缀避免重复，同时模拟真实 K8s 的 hash 后缀
    const hashSuffix = ['796496bcc8', '7d8f9a2b3c', '5e6f1a2b3d', '9a8b7c6d5e'][i % 4];
    const rsName = `${deployment.name}-${hashSuffix}`;
    const displayName = `ReplicaSet/${rsName}`;
    const replicas = (deployment.extras as Record<string, string> | undefined)?.replicas ?? '1';
    return {
      id: encodeNodeID('ReplicaSet', namespace, rsName),
      kind: 'ReplicaSet',
      namespace,
      name: rsName,
      displayName,
      status: STATUS.ReplicaSet,
      reason: '',
      isManaged: false,
      extras: {
        image: (deployment.extras as Record<string, string> | undefined)?.image ?? IMAGES[0],
        ownerDeployment: deployment.name ?? '',
        readyReplicas: replicas,
        replicas,
      },
    };
  });

  // 分配 Pod 到各 ReplicaSet
  const pods: TopologyNodeOutputObj[] = [];
  const podsPerRS: number[] = distributePods(distribution.deployment);

  replicaSets.forEach((rs, index) => {
    const podCount = podsPerRS[index];
    for (let j = 0; j < podCount; j++) {
      // 使用确定性后缀确保唯一性，格式为 5 位随机字符串
      const podSuffix = randomNodeName();
      const podName = `${rs.name}-${podSuffix}`;
      const displayName = `Pod/${podName}`;
      const ip = randomIP();
      const nodeName = `30.161.${randomInt(0, 255)}.${randomInt(1, 254)}`;
      pods.push({
        id: encodeNodeID('Pod', namespace, podName),
        kind: 'Pod',
        namespace,
        name: podName,
        displayName,
        status: STATUS.Pod,
        reason: '',
        isManaged: false,
        extras: {
          image: (rs.extras as Record<string, string> | undefined)?.image ?? IMAGES[0],
          ip,
          nodeName,
          phase: 'Running',
          ready: 'true',
          restartCount: String(randomInt(0, 2)),
        },
      });
    }
  });

  return { app: appNode, deployments, replicaSets, pods, podsPerRS };
}

// ============================================================================
// 边数据生成
// ============================================================================

function randomInt(min: number, max: number): number {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

// ============================================================================
// 主函数
// ============================================================================

function randomIP(): string {
  return `9.165.${randomInt(0, 255)}.${randomInt(1, 254)}`;
}

function randomItem<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)];
}

function randomNodeName(): string {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < 5; i++) {
    result += chars[Math.floor(Math.random() * chars.length)];
  }
  return result;
}
