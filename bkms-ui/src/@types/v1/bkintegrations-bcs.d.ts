/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：bkintegrations-bcs

export interface ListBCSAuthorizedProjectsRequest {
}

export interface GetBCSProjectRequest {
  /**
   * BCS 项目 ID
   */
  projectID: string;
}

export interface ListClustersByProjectRequest {
  /**
   * BCS 项目 ID
   */
  projectID: string;
}

export interface ListNamespacesByClusterRequest {
  /**
   * BCS 项目 ID
   */
  projectID: string;
  /**
   * 集群 ID
   */
  clusterID: string;
}

export interface GetBCSUserTokenRequest {
}

export interface ListBCSAuthorizedProjectsOutput {
  data?: BCSProjectOutput[];
}

export interface GetBCSProjectOutput {
  data?: BCSProjectOutput;
}

export interface ListClustersByProjectOutput {
  data?: ClusterOutput[];
}

export interface ListNamespacesByClusterOutput {
  data?: NamespaceOutput[];
}

export interface GetBCSUserTokenOutput {
  data?: string;
}

export interface NamespaceOutput {
  name?: string;
  status?: string;
}

export interface ClusterOutput {
  description?: string;
  environment?: string;
  id?: string;
  isShared?: boolean;
  name?: string;
  status?: string;
  type?: string;
}

export interface BCSProjectOutput {
  bizID?: string;
  code?: string;
  description?: string;
  id?: string;
  isBoundWorkspace?: boolean;
  isOffline?: boolean;
  name?: string;
}
