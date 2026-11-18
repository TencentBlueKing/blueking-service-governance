/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：bkintegrations-bkci

export interface ListBkCIBuildNodesRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

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

export interface GetBuildNodeBindingRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export type BindBuildNodeRequest = BindBuildNodeInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export interface UnbindBuildNodeRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export interface ListBuildNodeCandidatesOutput {
  data?: BuildNodeCandidateOutput[];
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

export interface GetBuildNodeBindingOutput {
  data?: BuildNodeBindingStateOutput;
}

export interface BindBuildNodeInput {
  nodeHashID: string;
}

export interface EmptyOutput {
}

export interface BuildNodeBindingStateOutput {
  /**
   * 当前关联的构建机，未关联时为 null，表示使用公共构建机
   */
  binding?: BuildNodeBindingOutput;
  /**
   * 当前用户能否关联、更换或解除构建机
   */
  editable?: boolean;
  /**
   * 工作空间的蓝盾项目 Code，为空表示尚未初始化蓝盾项目
   */
  projectCode?: string;
}

export interface BuildNodeBindingOutput {
  displayName?: string;
  ip?: string;
  /**
   * 蓝盾节点 HashId，用于更换时回显当前选项，页面不展示
   */
  nodeHashID?: string;
  osName?: string;
  /**
   * 实时状态：online 在线；offline 离线；notFound 已从蓝盾删除；unknown 查询失败
   */
  state?: "online" | "offline" | "notFound" | "unknown";
  updatedAt?: string;
  updater?: string;
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

export interface BuildNodeCandidateOutput {
  displayName?: string;
  ip?: string;
  /**
   * 蓝盾节点 HashId，只作为选项的值，页面不展示
   */
  nodeHashID?: string;
  online?: boolean;
  osName?: string;
  selectable?: boolean;
  /**
   * 不可关联的原因：offline 离线；osUnsupported 非 Linux
   */
  unselectableReason?: "offline" | "osUnsupported";
}
