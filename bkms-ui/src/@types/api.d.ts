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

export type ComponentType = 'Component' | 'Deploy' | 'Storage' | 'Strategy';

export interface ICluster {
  // 从clusterExtraInfo中merge过来的
  autoScale: boolean;
  cloudAccountID: string;
  cluster_id: string; // 兼容旧版数据（不要再使用）
  clusterAdvanceSettings: any;
  clusterBasicSettings: any;
  clusterCategory: string;
  clusterID: string;
  clusterName: string;
  clusterType: string;
  createTime: string;
  creator: string;
  description: string;
  environment: 'debug' | 'prod' | 'stag';
  extraInfo?: Record<string, any>;
  importCategory: string;
  is_shared: boolean;
  labels: Record<string, string>;
  manageType: 'INDEPENDENT_CLUSTER' | 'MANAGED_CLUSTER';
  master: any;
  networkType: string;
  provider: CloudID;
  providerType: string;
  region: string;
  status: 'DELETING' | 'INITIALIZATION' | 'RUNNING';
  systemID: string;
  updateTime: string;
  vpcID: string;
  networkSettings: {
    cidrStep: number;
    clusterIPv4CIDR: string;
    enableVPCCni: boolean;
    eniSubnetIDs: string[];
    isStaticIpMode: boolean;
    maxNodePodNum: number;
    maxServiceNum: number;
    multiClusterCIDR: string[];
    networkMode: 'tke-direct-eni' | 'tke-route-eni';
    serviceIPv4CIDR: string;
    status: string;
  };

  sharedRanges?: {
    bizs: string[];
    projectIdOrCodes: string[];
  };
}

export interface IProject {
  annotations: Labels;
  businessID: string;
  businessName: string;
  createTime: string;
  creator: string;
  description: string;
  isOffline: boolean;
  kind: string;
  labels: Labels;
  managers: string;
  name: string;
  projectCode: string;
  projectID: string;
  updater: string;
  updateTime: string;
  useBKRes: boolean;
}

export interface IProjectPerm {
  project_create: boolean;
  project_delete: boolean;
  project_edit: boolean;
  project_view: boolean;
}

type ExtractData<T> = T extends { data: infer D } ? D : undefined;

interface IUser {
  user_id: string;
}

interface Labels {
  [key: string]: string;
}
