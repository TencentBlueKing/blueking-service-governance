/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListBkCIOAuthGitProjectsRequest, BkCIOAuthGitProjectOutput, GetBkCIOAuthUrlRequest, ListBkCIPipelinesRequest, PaginatedBkCIPipelineOutput, GetBkCIPipelineRequest, BkCIPipelineDetailOutput, ListBkCIPipelineRepoRefsRequest, BkCIPipelineRepoRefOutput, ListBkCIPipelineRepoRefOptionsRequest, BkCIPipelineVariableOptionOutput, GetBkCIPipelineVariablesRequest, BkCIPipelineVariableOutput } from '~/@types/v1/bkintegrations-bkci';

export const BkintegrationsBkciService = {
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
   * 获取蓝盾流水线分支/Tag字段列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/bkci-pipelines/{pipelineID}/repo-refs
   * @tag bkintegrations-bkci
   * @param workspaceID path string required 工作空间 ID
   * @param pipelineID path string required 流水线 ID
   * @response 200 ListBkCIPipelineRepoRefsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBkCIPipelineRepoRefs: async <Request extends ListBkCIPipelineRepoRefsRequest = ListBkCIPipelineRepoRefsRequest, ResponseData = BkCIPipelineRepoRefOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/bkci-pipelines/{pipelineID}/repo-refs')(params, config),
  /**
   * 获取蓝盾流水线分支/Tag字段的可选项
   *
   * @method POST
   * @path /workspaces/{workspaceID}/bkci-pipelines/{pipelineID}/repo-refs/options
   * @tag bkintegrations-bkci
   * @param workspaceID path string required 工作空间 ID
   * @param pipelineID path string required 流水线 ID
   * @param input body BkCIPipelineRepoRefOptionsInput required 查询参数
   * @response 200 ListBkCIPipelineRepoRefOptionsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBkCIPipelineRepoRefOptions: async <Request extends ListBkCIPipelineRepoRefOptionsRequest = ListBkCIPipelineRepoRefOptionsRequest, ResponseData = BkCIPipelineVariableOptionOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/bkci-pipelines/{pipelineID}/repo-refs/options')(params, config),
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
};
