/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：bkintegrations-bkci

export interface ListBkCIOAuthGitProjectsRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 搜索关键词
   */
  keyword?: string;
}

export interface GetBkCIOAuthUrlRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export interface ListBkCIPipelinesRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 搜索关键词
   */
  keyword?: string;
  /**
   * 页码，最小为 1
   */
  page: number;
  /**
   * 每页数量，可选值: 5, 10, 20, 50, 100
   */
  pageSize: number;
}

export interface GetBkCIPipelineRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 流水线 ID
   */
  pipelineID: string;
}

export interface GetBkCIPipelineVariablesRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 流水线 ID
   */
  pipelineID: string;
}

export interface ListBkCIRepositoryBranchesRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 代码仓库 ID 或名称
   */
  repositoryID: string;
  /**
   * 代码仓库标识类型，可选值: ID, NAME
   */
  repositoryType: string;
  /**
   * 搜索关键词
   */
  search?: string;
  /**
   * 页码，最小为 1
   */
  page: number;
  /**
   * 每页数量，可选值: 5, 10, 20, 50, 100
   */
  pageSize: number;
}

export interface ListBkCIRepositoryTagsRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 代码仓库 ID 或名称
   */
  repositoryID: string;
  /**
   * 代码仓库标识类型，可选值: ID, NAME
   */
  repositoryType: string;
  /**
   * 搜索关键词
   */
  search?: string;
  /**
   * 页码，最小为 1
   */
  page: number;
  /**
   * 每页数量，可选值: 5, 10, 20, 50, 100
   */
  pageSize: number;
}

export interface ListBkCIOAuthGitProjectsOutput {
  data?: BkCIOAuthGitProjectOutput[];
}

export interface GetBkCIOAuthUrlOutput {
  data?: string;
}

export interface ListBkCIPipelinesOutput {
  data?: PaginatedBkCIPipelineOutput;
}

export interface GetBkCIPipelineOutput {
  data?: BkCIPipelineDetailOutput;
}

export interface GetBkCIPipelineVariablesOutput {
  data?: BkCIPipelineVariableOutput[];
}

export interface ListBkCIRepositoryBranchesOutput {
  data?: BkCIRepositoryRefOutput[];
}

export interface ListBkCIRepositoryTagsOutput {
  data?: BkCIRepositoryRefOutput[];
}

export interface BkCIRepositoryRefOutput {
  linkUrl?: string;
  name?: string;
  path?: string;
  sha?: string;
}

export interface BkCIPipelineVariableOutput {
  constant?: boolean;
  defaultValue?: string;
  description?: string;
  id?: string;
  name?: string;
  options?: BkCIPipelineVariableOptionOutput[];
  readOnly?: boolean;
  required?: boolean;
  type?: string;
}

export interface BkCIPipelineVariableOptionOutput {
  key?: string;
  value?: string;
}

export interface BkCIPipelineDetailOutput {
  createdAt?: string;
  creator?: string;
  description?: string;
  id?: string;
  name?: string;
  updatedAt?: string;
  updater?: string;
  variables?: BkCIPipelineVariableOutput[];
  version?: string;
}

export interface PaginatedBkCIPipelineOutput {
  count?: string;
  results?: BkCIPipelineOutput[];
}

export interface BkCIPipelineOutput {
  createdAt?: string;
  creator?: string;
  description?: string;
  id?: string;
  name?: string;
  updatedAt?: string;
  updater?: string;
  version?: string;
}

export interface BkCIOAuthGitProjectOutput {
  alias?: string;
  id?: string;
  name?: string;
  url?: string;
}
