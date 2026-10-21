/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListReleaseStrategiesRequest, ReleaseStrategyOutputObj, CreateReleaseStrategyRequest, GetReleaseStrategyRequest, UpdateReleaseStrategyRequest, DeleteReleaseStrategyRequest, EmptyOutput } from '~/@types/v1/release-strategies';

export const ReleaseStrategiesService = {
  /**
   * 查询空间内全部发布策略
   *
   * @method GET
   * @path /workspaces/{workspaceID}/release-strategies
   * @tag release-strategies
   * @param workspaceID path string required 工作空间 ID
   * @response 200 ListReleaseStrategiesOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  listReleaseStrategies: async <Request extends ListReleaseStrategiesRequest = ListReleaseStrategiesRequest, ResponseData = ReleaseStrategyOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/release-strategies')(params, config),
  /**
   * 新建发布策略
   *
   * @method POST
   * @path /workspaces/{workspaceID}/release-strategies
   * @tag release-strategies
   * @param workspaceID path string required 工作空间 ID
   * @param body body ReleaseStrategyInput required 发布策略
   * @response 200 ReleaseStrategyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  createReleaseStrategy: async <Request extends CreateReleaseStrategyRequest = CreateReleaseStrategyRequest, ResponseData = ReleaseStrategyOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/release-strategies')(params, config),
  /**
   * 查询发布策略详情
   *
   * @method GET
   * @path /workspaces/{workspaceID}/release-strategies/{strategyID}
   * @tag release-strategies
   * @param workspaceID path string required 工作空间 ID
   * @param strategyID path string required 发布策略 ID
   * @response 200 ReleaseStrategyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  getReleaseStrategy: async <Request extends GetReleaseStrategyRequest = GetReleaseStrategyRequest, ResponseData = ReleaseStrategyOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/release-strategies/{strategyID}')(params, config),
  /**
   * 修改发布策略
   *
   * @method PUT
   * @path /workspaces/{workspaceID}/release-strategies/{strategyID}
   * @tag release-strategies
   * @param workspaceID path string required 工作空间 ID
   * @param strategyID path string required 发布策略 ID
   * @param body body ReleaseStrategyInput required 发布策略
   * @response 200 ReleaseStrategyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  updateReleaseStrategy: async <Request extends UpdateReleaseStrategyRequest = UpdateReleaseStrategyRequest, ResponseData = ReleaseStrategyOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/workspaces/{workspaceID}/release-strategies/{strategyID}')(params, config),
  /**
   * 删除发布策略
   *
   * @method DELETE
   * @path /workspaces/{workspaceID}/release-strategies/{strategyID}
   * @tag release-strategies
   * @param workspaceID path string required 工作空间 ID
   * @param strategyID path string required 发布策略 ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteReleaseStrategy: async <Request extends DeleteReleaseStrategyRequest = DeleteReleaseStrategyRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/workspaces/{workspaceID}/release-strategies/{strategyID}')(params, config),
};
