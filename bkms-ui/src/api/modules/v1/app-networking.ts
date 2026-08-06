/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListAppServicesRequest, AppServiceOutput, CreateAppServiceRequest, EmptyOutput, UpdateAppServiceRequest, DeleteAppServiceRequest, ListTrafficLaneCandidateAppsRequest, TrafficLaneCandidateAppOutput } from '~/@types/v1/app-networking';

export const AppNetworkingService = {
  /**
   * 获取应用下的 Service 列表
   *
   * @method GET
   * @path /apps/{appID}/services
   * @tag app-networking
   * @param appID path string required 应用 ID
   * @response 200 ListAppServicesOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 500 GinErrorOutput Internal Server Error
   */
  listAppServices: async <Request extends ListAppServicesRequest = ListAppServicesRequest, ResponseData = AppServiceOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/services')(params, config),
  /**
   * 创建应用下的 Service
   *
   * @method POST
   * @path /apps/{appID}/services
   * @tag app-networking
   * @param appID path string required 应用 ID
   * @param body body CreateAppServiceInput required 创建 Service 请求体
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 500 GinErrorOutput Internal Server Error
   */
  createAppService: async <Request extends CreateAppServiceRequest = CreateAppServiceRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/services')(params, config),
  /**
   * 更新应用下的 Service
   *
   * @method PUT
   * @path /apps/{appID}/services/{name}
   * @tag app-networking
   * @param appID path string required 应用 ID
   * @param name path string required Service 名称
   * @param body body UpdateAppServiceInput required 更新 Service 请求体
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 500 GinErrorOutput Internal Server Error
   */
  updateAppService: async <Request extends UpdateAppServiceRequest = UpdateAppServiceRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/services/{name}')(params, config),
  /**
   * 删除应用下的 Service
   *
   * @method DELETE
   * @path /apps/{appID}/services/{name}
   * @tag app-networking
   * @param appID path string required 应用 ID
   * @param name path string required Service 名称
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 500 GinErrorOutput Internal Server Error
   */
  deleteAppService: async <Request extends DeleteAppServiceRequest = DeleteAppServiceRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/services/{name}')(params, config),
  /**
   * 查询空间下的泳道候选应用列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/traffic-lanes/candidate-apps
   * @tag app-networking
   * @param workspaceID path string required 工作空间 ID
   * @response 200 ListTrafficLaneCandidateAppsOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 500 GinErrorOutput Internal Server Error
   */
  listTrafficLaneCandidateApps: async <Request extends ListTrafficLaneCandidateAppsRequest = ListTrafficLaneCandidateAppsRequest, ResponseData = TrafficLaneCandidateAppOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/traffic-lanes/candidate-apps')(params, config),
};
