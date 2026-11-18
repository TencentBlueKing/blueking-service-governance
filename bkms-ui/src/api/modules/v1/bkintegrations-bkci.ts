/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListBkCIBuildNodesRequest, BuildNodeCandidateOutput, ListBkCIOAuthGitProjectsRequest, BkCIOAuthGitProjectOutput, GetBkCIOAuthUrlRequest, ListBkCIPipelinesRequest, PaginatedBkCIPipelineOutput, GetBkCIPipelineRequest, BkCIPipelineDetailOutput, GetBkCIPipelineVariablesRequest, BkCIPipelineVariableOutput, ListBkCIRepositoryBranchesRequest, BkCIRepositoryRefOutput, ListBkCIRepositoryTagsRequest, GetBuildNodeBindingRequest, BuildNodeBindingStateOutput, BindBuildNodeRequest, UnbindBuildNodeRequest, EmptyOutput } from '~/@types/v1/bkintegrations-bkci';

export const BkintegrationsBkciService = {
  /**
   * 获取工作空间蓝盾项目下可关联的第三方构建机
   *
   * @method GET
   * @path /workspaces/{workspaceID}/bkci-build-nodes
   * @tag bkintegrations-bkci
   * @param workspaceID path string required 工作空间 ID
   * @response 200 ListBuildNodeCandidatesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBkCIBuildNodes: async <Request extends ListBkCIBuildNodesRequest = ListBkCIBuildNodesRequest, ResponseData = BuildNodeCandidateOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/bkci-build-nodes')(params, config),
  /**
   * 获取蓝盾 OAuth 授权 Git 项目列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/bkci-git-projects
   * @tag bkintegrations-bkci
   * @param workspaceID path string required 工作空间 ID
   * @param keyword query string 搜索关键词
   * @response 200 ListBkCIOAuthGitProjectsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBkCIOAuthGitProjects: async <Request extends ListBkCIOAuthGitProjectsRequest = ListBkCIOAuthGitProjectsRequest, ResponseData = BkCIOAuthGitProjectOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/bkci-git-projects')(params, config),
  /**
   * 获取蓝盾 OAuth 授权 URL
   *
   * @method GET
   * @path /workspaces/{workspaceID}/bkci-oauth-url
   * @tag bkintegrations-bkci
   * @param workspaceID path string required 工作空间 ID
   * @response 200 GetBkCIOAuthUrlOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getBkCIOAuthUrl: async <Request extends GetBkCIOAuthUrlRequest = GetBkCIOAuthUrlRequest, ResponseData = string>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/bkci-oauth-url')(params, config),
  /**
   * 获取蓝盾流水线列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/bkci-pipelines
   * @tag bkintegrations-bkci
   * @param workspaceID path string required 工作空间 ID
   * @param keyword query string 搜索关键词
   * @param page query number required 页码，最小为 1
   * @param pageSize query number required 每页数量，可选值: 5, 10, 20, 50, 100
   * @response 200 ListBkCIPipelinesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBkCIPipelines: async <Request extends ListBkCIPipelinesRequest = ListBkCIPipelinesRequest, ResponseData = PaginatedBkCIPipelineOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/bkci-pipelines')(params, config),
  /**
   * 获取蓝盾流水线详情
   *
   * @method GET
   * @path /workspaces/{workspaceID}/bkci-pipelines/{pipelineID}
   * @tag bkintegrations-bkci
   * @param workspaceID path string required 工作空间 ID
   * @param pipelineID path string required 流水线 ID
   * @response 200 GetBkCIPipelineOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getBkCIPipeline: async <Request extends GetBkCIPipelineRequest = GetBkCIPipelineRequest, ResponseData = BkCIPipelineDetailOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/bkci-pipelines/{pipelineID}')(params, config),
  /**
   * 获取蓝盾流水线变量列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/bkci-pipelines/{pipelineID}/variables
   * @tag bkintegrations-bkci
   * @param workspaceID path string required 工作空间 ID
   * @param pipelineID path string required 流水线 ID
   * @response 200 GetBkCIPipelineVariablesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getBkCIPipelineVariables: async <Request extends GetBkCIPipelineVariablesRequest = GetBkCIPipelineVariablesRequest, ResponseData = BkCIPipelineVariableOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/bkci-pipelines/{pipelineID}/variables')(params, config),
  /**
   * 获取代码仓库分支列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/bkci-repositories/branches
   * @tag bkintegrations-bkci
   * @param workspaceID path string required 工作空间 ID
   * @param repositoryID query string required 代码仓库 ID 或名称
   * @param repositoryType query string required 代码仓库标识类型，可选值: ID, NAME
   * @param search query string 搜索关键词
   * @param page query number required 页码，最小为 1
   * @param pageSize query number required 每页数量，可选值: 5, 10, 20, 50, 100
   * @response 200 ListBkCIRepositoryBranchesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBkCIRepositoryBranches: async <Request extends ListBkCIRepositoryBranchesRequest = ListBkCIRepositoryBranchesRequest, ResponseData = BkCIRepositoryRefOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/bkci-repositories/branches')(params, config),
  /**
   * 获取代码仓库标签列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/bkci-repositories/tags
   * @tag bkintegrations-bkci
   * @param workspaceID path string required 工作空间 ID
   * @param repositoryID query string required 代码仓库 ID 或名称
   * @param repositoryType query string required 代码仓库标识类型，可选值: ID, NAME
   * @param search query string 搜索关键词
   * @param page query number required 页码，最小为 1
   * @param pageSize query number required 每页数量，可选值: 5, 10, 20, 50, 100
   * @response 200 ListBkCIRepositoryTagsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBkCIRepositoryTags: async <Request extends ListBkCIRepositoryTagsRequest = ListBkCIRepositoryTagsRequest, ResponseData = BkCIRepositoryRefOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/bkci-repositories/tags')(params, config),
  /**
   * 查询工作空间关联的构建机
   *
   * @method GET
   * @path /workspaces/{workspaceID}/build-node-binding
   * @tag bkintegrations-bkci
   * @param workspaceID path string required 工作空间 ID
   * @response 200 GetBuildNodeBindingOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getBuildNodeBinding: async <Request extends GetBuildNodeBindingRequest = GetBuildNodeBindingRequest, ResponseData = BuildNodeBindingStateOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/build-node-binding')(params, config),
  /**
   * 关联或更换工作空间的构建机
   *
   * @method PUT
   * @path /workspaces/{workspaceID}/build-node-binding
   * @tag bkintegrations-bkci
   * @param workspaceID path string required 工作空间 ID
   * @param body body BindBuildNodeInput required 要关联的构建机
   * @response 204 unknown No Content
   * @response 400 GinErrorOutput Bad Request
   */
  bindBuildNode: async <Request extends BindBuildNodeRequest = BindBuildNodeRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/workspaces/{workspaceID}/build-node-binding')(params, config),
  /**
   * 解除工作空间的构建机关联
   *
   * @method DELETE
   * @path /workspaces/{workspaceID}/build-node-binding
   * @tag bkintegrations-bkci
   * @param workspaceID path string required 工作空间 ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  unbindBuildNode: async <Request extends UnbindBuildNodeRequest = UnbindBuildNodeRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/workspaces/{workspaceID}/build-node-binding')(params, config),
};
