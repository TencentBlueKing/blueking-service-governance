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

import type { Component } from 'vue';

import cRoleIcon2 from '~/assets/k8s-icons-2/c-role.svg';
import cmIcon2 from '~/assets/k8s-icons-2/cm.svg';
import crbIcon2 from '~/assets/k8s-icons-2/crb.svg';
import crdIcon2 from '~/assets/k8s-icons-2/crd.svg';
import cronjobIcon2 from '~/assets/k8s-icons-2/cronjob.svg';
import deployIcon2 from '~/assets/k8s-icons-2/deploy.svg';
import dsIcon2 from '~/assets/k8s-icons-2/ds.svg';
import epIcon2 from '~/assets/k8s-icons-2/ep.svg';
import groupIcon2 from '~/assets/k8s-icons-2/group.svg';
import hpaIcon2 from '~/assets/k8s-icons-2/hpa.svg';
import ingIcon2 from '~/assets/k8s-icons-2/ing.svg';
import jobIcon2 from '~/assets/k8s-icons-2/job.svg';
import limitsIcon2 from '~/assets/k8s-icons-2/limits.svg';
import netpolIcon2 from '~/assets/k8s-icons-2/netpol.svg';
import nsIcon2 from '~/assets/k8s-icons-2/ns.svg';
import podIcon2 from '~/assets/k8s-icons-2/pod.svg';
import pspIcon2 from '~/assets/k8s-icons-2/psp.svg';
import pvIcon2 from '~/assets/k8s-icons-2/pv.svg';
import pvcIcon2 from '~/assets/k8s-icons-2/pvc.svg';
import quotaIcon2 from '~/assets/k8s-icons-2/quota.svg';
import rbIcon2 from '~/assets/k8s-icons-2/rb.svg';
import roleIcon2 from '~/assets/k8s-icons-2/role.svg';
import rsIcon2 from '~/assets/k8s-icons-2/rs.svg';
import saIcon2 from '~/assets/k8s-icons-2/sa.svg';
import scIcon2 from '~/assets/k8s-icons-2/sc.svg';
import secretIcon2 from '~/assets/k8s-icons-2/secret.svg';
import stsIcon2 from '~/assets/k8s-icons-2/sts.svg';
import svcIcon2 from '~/assets/k8s-icons-2/svc.svg';
import userIcon2 from '~/assets/k8s-icons-2/user.svg';
import volIcon2 from '~/assets/k8s-icons-2/vol.svg';
import cRoleIcon from '~/assets/k8s-icons/c-role.svg';
import cmIcon from '~/assets/k8s-icons/cm.svg';
import crbIcon from '~/assets/k8s-icons/crb.svg';
import crdIcon from '~/assets/k8s-icons/crd.svg';
import cronjobIcon from '~/assets/k8s-icons/cronjob.svg';
import deployIcon from '~/assets/k8s-icons/deploy.svg';
import dsIcon from '~/assets/k8s-icons/ds.svg';
import epIcon from '~/assets/k8s-icons/ep.svg';
import groupIcon from '~/assets/k8s-icons/group.svg';
import hpaIcon from '~/assets/k8s-icons/hpa.svg';
import ingIcon from '~/assets/k8s-icons/ing.svg';
import jobIcon from '~/assets/k8s-icons/job.svg';
import limitsIcon from '~/assets/k8s-icons/limits.svg';
import netpolIcon from '~/assets/k8s-icons/netpol.svg';
import nsIcon from '~/assets/k8s-icons/ns.svg';
import pcIcon from '~/assets/k8s-icons/pc.svg';
import podIcon from '~/assets/k8s-icons/pod.svg';
import pspIcon from '~/assets/k8s-icons/psp.svg';
import pvIcon from '~/assets/k8s-icons/pv.svg';
import pvcIcon from '~/assets/k8s-icons/pvc.svg';
import quotaIcon from '~/assets/k8s-icons/quota.svg';
import rbIcon from '~/assets/k8s-icons/rb.svg';
import roleIcon from '~/assets/k8s-icons/role.svg';
import rsIcon from '~/assets/k8s-icons/rs.svg';
import saIcon from '~/assets/k8s-icons/sa.svg';
import scIcon from '~/assets/k8s-icons/sc.svg';
import secretIcon from '~/assets/k8s-icons/secret.svg';
import stsIcon from '~/assets/k8s-icons/sts.svg';
import svcIcon from '~/assets/k8s-icons/svc.svg';
import userIcon from '~/assets/k8s-icons/user.svg';
import volIcon from '~/assets/k8s-icons/vol.svg';

import type { NodeStatus } from './types';
import type { TopologyNode } from '~/@types/topology';

/** K8s 资源 Kind 到 SVG 图标组件的映射 */
export const KIND_ICON_MAP: Record<string, Component> = {
  ClusterRole: cRoleIcon,
  ConfigMap: cmIcon,
  ClusterRoleBinding: crbIcon,
  CustomResourceDefinition: crdIcon,
  CronJob: cronjobIcon,
  Deployment: deployIcon,
  DaemonSet: dsIcon,
  Endpoints: epIcon,
  Group: groupIcon,
  HorizontalPodAutoscaler: hpaIcon,
  Ingress: ingIcon,
  Job: jobIcon,
  LimitRange: limitsIcon,
  NetworkPolicy: netpolIcon,
  Namespace: nsIcon,
  PolarisConfig: pcIcon,
  Pod: podIcon,
  PodSecurityPolicy: pspIcon,
  PersistentVolume: pvIcon,
  PersistentVolumeClaim: pvcIcon,
  ResourceQuota: quotaIcon,
  RoleBinding: rbIcon,
  Role: roleIcon,
  ReplicaSet: rsIcon,
  ServiceAccount: saIcon,
  StorageClass: scIcon,
  Secret: secretIcon,
  StatefulSet: stsIcon,
  Service: svcIcon,
  User: userIcon,
  Volume: volIcon,
  Helm: deployIcon,
  HookTemplate: deployIcon,
  GameDeployment: deployIcon,
  GameStatefulSet: dsIcon,
};

/** K8s 资源 Kind 到 SVG 图标组件的映射 */
export const KIND_ICON_ASIDE_MAP: Record<string, Component> = {
  ClusterRole: cRoleIcon2,
  ConfigMap: cmIcon2,
  ClusterRoleBinding: crbIcon2,
  CustomResourceDefinition: crdIcon2,
  CronJob: cronjobIcon2,
  Deployment: deployIcon2,
  DaemonSet: dsIcon2,
  Endpoints: epIcon2,
  Group: groupIcon2,
  HorizontalPodAutoscaler: hpaIcon2,
  Ingress: ingIcon2,
  Job: jobIcon2,
  LimitRange: limitsIcon2,
  NetworkPolicy: netpolIcon2,
  Namespace: nsIcon2,
  PolarisConfig: pcIcon,
  Pod: podIcon2,
  PodSecurityPolicy: pspIcon2,
  PersistentVolume: pvIcon2,
  PersistentVolumeClaim: pvcIcon2,
  ResourceQuota: quotaIcon2,
  RoleBinding: rbIcon2,
  Role: roleIcon2,
  ReplicaSet: rsIcon2,
  ServiceAccount: saIcon2,
  StorageClass: scIcon2,
  Secret: secretIcon2,
  StatefulSet: stsIcon2,
  Service: svcIcon2,
  User: userIcon2,
  Volume: volIcon2,
  Helm: deployIcon2,
  HookTemplate: deployIcon2,
  GameDeployment: deployIcon2,
  GameStatefulSet: dsIcon2,
};

// 状态映射
export const HEALTHY_KEYWORDS = [
  'running',
  'healthy',
  'active',
  'bound',
  'available',
  'ready',
  'completed',
  'succeeded',
];
export const WARNING_KEYWORDS = ['deployed', 'warning', 'pending', 'progressing', 'syncing', 'applying'];
export const ERROR_KEYWORDS = [
  'degraded',
  'notFound',
  'failed',
  'error',
  'crashloopbackoff',
  'imagepullbackoff',
  'evicted',
  'terminated',
];

// 日志：Running、CrashLoopBackOff、Error、Completed、Succeeded 状态可查看日志
export const LOG_ALLOWED_STATUSES = ['running', 'crashloopbackoff', 'error', 'completed', 'succeeded'];

/** Kind 的缩写映射，用于节点展示 */
export const KIND_SHORT_MAP: Record<string, string> = {
  ClusterRole: 'ClusterRole',
  ConfigMap: 'ConfigMap',
  ClusterRoleBinding: 'CRB',
  CustomResourceDefinition: 'CRD',
  CronJob: 'CronJob',
  Deployment: 'Deployment',
  DaemonSet: 'DaemonSet',
  Endpoints: 'Endpoints',
  HorizontalPodAutoscaler: 'HPA',
  Ingress: 'Ingress',
  Job: 'Job',
  LimitRange: 'LimitRange',
  NetworkPolicy: 'NetworkPolicy',
  Namespace: 'Namespace',
  Pod: 'Pod',
  PodSecurityPolicy: 'PSP',
  PersistentVolume: 'PV',
  PersistentVolumeClaim: 'PVC',
  ResourceQuota: 'Quota',
  RoleBinding: 'RoleBinding',
  Role: 'Role',
  ReplicaSet: 'ReplicaSet',
  ServiceAccount: 'SA',
  StorageClass: 'StorageClass',
  Secret: 'Secret',
  StatefulSet: 'StatefulSet',
  Service: 'Service',
  Helm: 'Helm',
  HookTemplate: 'HookTemplate',
  GameDeployment: 'GameDeployment',
  GameStatefulSet: 'GameStatefulSet',
};

/** 节点状态颜色配置 */
export const STATUS_CONFIG: Record<
  NodeStatus,
  {
    badgeIconColor?: string;
    bgColor: string;
    color?: string;
    iconBgColor: string;
    miniColor?: string;
  }
> = {
  all: {
    bgColor: '#fff',
    iconBgColor: 'linear-gradient(90deg, #748EC1 0%, #99B1E0 100%)',
  },
  healthy: {
    bgColor: '#fff',
    iconBgColor: 'linear-gradient(90deg, #5FB8AC 0%, #61C7B9 100%)',
    color: '#61C7B9',
    miniColor: '#DAF6E5',
  },
  warning: {
    bgColor: 'linear-gradient(90deg, #FFF5F5 0%, #FFFDFC 50%, #FFFFFF 100%)',
    iconBgColor: 'linear-gradient(90deg, #F59500 0%, #F8B64F 100%)',
    badgeIconColor: '#F59500',
    color: '#F8B64F',
    miniColor: '#FDEED8',
  },
  error: {
    bgColor: 'linear-gradient(90deg, #FFF5F5 0%, #FFFDFC 50%, #FFFFFF 100%)',
    iconBgColor: 'linear-gradient(90deg, #FA4E41 0%, #F07474 100%)',
    badgeIconColor: '#FA4E41',
    color: '#F07474',
    miniColor: '#FEDDDD',
  },
  unknown: {
    bgColor: '#fff',
    iconBgColor: 'linear-gradient(90deg, #748EC1 0%, #99B1E0 100%)',
    badgeIconColor: '#748EC1',
    color: '#99B1E0',
    miniColor: '#99B1E0',
  },
};

/** 拓扑节点归一化状态（与图上节点展示一致） */
export function getTopologyNodeStatus(node: TopologyNode): NodeStatus {
  return normalizeStatus(node.status);
}

/** 后端 status 字段到前端 NodeStatus 的映射 */
export function normalizeStatus(status?: string): NodeStatus {
  if (!status) return 'unknown';
  const lower = status.toLowerCase();

  if (HEALTHY_KEYWORDS.some(k => lower.includes(k))) return 'healthy';
  if (WARNING_KEYWORDS.some(k => lower.includes(k))) return 'warning';
  if (ERROR_KEYWORDS.some(k => lower.includes(k))) return 'error';
  return 'unknown';
}

/** 节点尺寸（视觉尺寸，即用户看到的实际大小） */
export const NODE_WIDTH = 240;
export const NODE_HEIGHT = 48;
/**
 * HTML 节点超采样倍率。
 * G6 Canvas 渲染器中，HTML 节点通过 CSS transform 缩放，会导致放大后文字模糊。
 * 在 Vue 组件内部使用 CSS zoom: SCALE_FACTOR + transform: scale(1/SCALE_FACTOR)，
 * 让浏览器以更高像素密度渲染 DOM，画布放大 SCALE_FACTOR 倍以内仍然清晰。
 */
export const NODE_SCALE_FACTOR = 2;
/** 节点之间边的长度 */
export const EDGE_LENGTH = 100;
// minimap 超过一定数量节点时，不绘制内嵌状态色块
export const MINIMAP_NODE_LIMIT = 100;
// 超过一定数量节点时，不使用 minimap
export const MINIMAP_NODE_MAX = 1000;

/** 缩进树布局（G6 hierarchy indented） */
export const DAGRE_LAYOUT_OPTIONS = {
  type: 'indented' as const,
  direction: 'LR' as const,
  indent: NODE_WIDTH + EDGE_LENGTH,
  getHeight: () => NODE_HEIGHT,
  getWidth: () => NODE_WIDTH,
  // 每个节点的第一个子节点是否换行
  dropCap: false,
  // 使用 Web Worker 加速布局计算
  useWorker: true,
};

/** 自定义节点类型 */
export const CUSTOM_NODE_TYPE = 'resource-node';

/** 资源类别 */
export type ResourceCategory =
  | 'config'
  | 'crd'
  | 'hpa'
  | 'network'
  | 'rbac'
  | 'storage'
  | 'tkex-crd'
  | 'unknown'
  | 'workload';

export interface ResourceCategoryMeta {
  icon: string;
  id: ResourceCategory;
  kinds: Set<string>;
  label: string;
}

// 工作负载
const WORKLOAD_KINDS = new Set(['Deployment', 'StatefulSet', 'DaemonSet', 'Job', 'CronJob', 'Pod', 'ReplicaSet']);

// 网络
const NETWORK_KINDS = new Set(['Ingress', 'Service', 'Endpoints', 'NetworkPolicy']);

// 配置
const CONFIG_KINDS = new Set(['BscpConfig', 'ConfigMap', 'Secret']);

// 存储
const STORAGE_KINDS = new Set(['PersistentVolume', 'PersistentVolumeClaim', 'StorageClass', 'Volume']);

// RBAC
const RBAC_KINDS = new Set(['ServiceAccount', 'Role', 'ClusterRole', 'RoleBinding', 'ClusterRoleBinding']);

// HPA
const HPA_KINDS = new Set(['HorizontalPodAutoscaler', 'GeneralPodAutoscaler']);

// CRD
const CRD_KINDS = new Set(['CustomResourceDefinition']);

// 游戏场景CRD
const TKEX_CRD_KINDS = new Set(['GameDeployment', 'GameStatefulSet', 'HookTemplate']);

/** 资源分类元信息，按展示顺序排列 */
export const RESOURCE_CATEGORIES: ResourceCategoryMeta[] = [
  { id: 'workload', label: '工作负载', icon: 'icon-apps', kinds: WORKLOAD_KINDS },
  { id: 'network', label: '网络', icon: 'icon-wangluoguanli', kinds: NETWORK_KINDS },
  { id: 'config', label: '配置', icon: 'icon-peizhiguanli', kinds: CONFIG_KINDS },
  { id: 'storage', label: '存储', icon: 'icon-data', kinds: STORAGE_KINDS },
  { id: 'rbac', label: 'RBAC', icon: 'icon-lock-shape', kinds: RBAC_KINDS },
  { id: 'hpa', label: 'HPA', icon: 'icon-hpa', kinds: HPA_KINDS },
  { id: 'crd', label: 'CRD', icon: 'icon-crd', kinds: CRD_KINDS },
  { id: 'tkex-crd', label: '游戏场景CRD', icon: 'icon-crd', kinds: TKEX_CRD_KINDS },
];

/** Kind -> 资源类别 快速查表 */
export const KIND_CATEGORY_MAP: Record<string, ResourceCategory> = Object.fromEntries(
  RESOURCE_CATEGORIES.flatMap(cat => [...cat.kinds].map(kind => [kind, cat.id])),
) as Record<string, ResourceCategory>;

/** 根据 kind 获取所属类别，未匹配归入 unknown */
export function getKindCategory(kind: string): ResourceCategory {
  return KIND_CATEGORY_MAP[kind] ?? 'unknown';
}
