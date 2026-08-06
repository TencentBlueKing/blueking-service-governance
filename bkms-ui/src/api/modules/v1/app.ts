/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { GetAppIDAutoSuffixRequest, GetAppIDAutoSuffixOutput, GetAppRequest, AppDetailOutputObj, DeleteAppRequest, EmptyOutput, GetAppDeployStatusesRequest, AppDeployedEnvOutputObj, UpdateAppDisplayNameRequest, UpdateHelmSpecRequest, UpdateAppTafSpecRequest, UpdateAppTrpcSpecRequest, ListAppsRequest, AppInfoOutputObj, CreateAppRequest, AppOutputObj } from '~/@types/v1/app';

export const AppService = {
  /**
   * 获取创建应用时使用的自动 ID 后缀
   *
   * @method GET
   * @path /apps/auto-id-suffix
   * @tag app
   * @response 200 GetAppIDAutoSuffixOutput OK
   */
  getAppIDAutoSuffix: async <Request extends GetAppIDAutoSuffixRequest = GetAppIDAutoSuffixRequest, ResponseData = GetAppIDAutoSuffixOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/auto-id-suffix')(params, config),
  /**
   * 查询单个应用详情
   *
   * @method GET
   * @path /apps/{appID}
   * @tag app
   * @param appID path string required 应用 ID
   * @response 200 GetAppOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getApp: async <Request extends GetAppRequest = GetAppRequest, ResponseData = AppDetailOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}')(params, config),
  /**
   * 删除单个应用
   *
   * @method DELETE
   * @path /apps/{appID}
   * @tag app
   * @param appID path string required 应用 ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteApp: async <Request extends DeleteAppRequest = DeleteAppRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}')(params, config),
  /**
   * 查询应用在各环境及各泳道上的部署状态
   *
   * @method GET
   * @path /apps/{appID}/deploy-statuses
   * @tag app
   * @param appID path string required 应用 ID
   * @response 200 GetAppDeployStatusesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getAppDeployStatuses: async <Request extends GetAppDeployStatusesRequest = GetAppDeployStatusesRequest, ResponseData = AppDeployedEnvOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/deploy-statuses')(params, config),
  /**
   * 更新应用显示名称
   *
   * @method PUT
   * @path /apps/{appID}/display-name
   * @tag app
   * @param appID path string required 应用 ID
   * @param body body UpdateAppDisplayNameInput required 更新显示名请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateAppDisplayName: async <Request extends UpdateAppDisplayNameRequest = UpdateAppDisplayNameRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/display-name')(params, config),
  /**
   * 更新应用 Helm Chart 配置
   *
   * @method PUT
   * @path /apps/{appID}/helm-spec
   * @tag app
   * @param appID path string required 应用 ID
   * @param body body UpdateHelmSpecInput required 更新 Helm 配置请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateHelmSpec: async <Request extends UpdateHelmSpecRequest = UpdateHelmSpecRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/helm-spec')(params, config),
  /**
   * 更新应用 Taf 配置
   *
   * @method PUT
   * @path /apps/{appID}/taf-spec
   * @tag app
   * @param appID path string required 应用 ID
   * @param body body UpdateAppModelSpecInput required 更新 Taf 配置请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateAppTafSpec: async <Request extends UpdateAppTafSpecRequest = UpdateAppTafSpecRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/taf-spec')(params, config),
  /**
   * 更新应用 Trpc 配置
   *
   * @method PUT
   * @path /apps/{appID}/trpc-spec
   * @tag app
   * @param appID path string required 应用 ID
   * @param body body UpdateAppModelSpecInput required 更新 Trpc 配置请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateAppTrpcSpec: async <Request extends UpdateAppTrpcSpecRequest = UpdateAppTrpcSpecRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/trpc-spec')(params, config),
  /**
   * 查询 app 列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/apps
   * @tag app
   * @param workspaceID path string required 工作空间 ID
   * @param appName query string 应用名过滤
   * @response 200 ListAppsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listApps: async <Request extends ListAppsRequest = ListAppsRequest, ResponseData = AppInfoOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/apps')(params, config),
  /**
   * 创建应用
   *
   * @method POST
   * @path /workspaces/{workspaceID}/apps
   * @tag app
   * @param workspaceID path string required 工作空间 ID
   * @param body body CreateAppInput required 创建应用请求
   * @response 200 CreateAppOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createApp: async <Request extends CreateAppRequest = CreateAppRequest, ResponseData = AppOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/apps')(params, config),
};
