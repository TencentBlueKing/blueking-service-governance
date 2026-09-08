/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListAppDashboardsRequest, DashboardOutput, CreateAppDashboardRequest, EmptyOutput, UpdateAppDashboardRequest, DeleteAppDashboardRequest } from '~/@types/v1/bkmonitor-dashboard';

export const BkmonitorDashboardService = {
  /**
   * 获取应用绑定的仪表盘列表
   *
   * @method GET
   * @path /apps/{appID}/bkmonitor/dashboards
   * @tag bkmonitor-dashboard
   * @param appID path string required 应用 ID
   * @response 200 ListDashboardsResp OK
   * @response 400 GinErrorOutput Bad Request
   */
  listAppDashboards: async <Request extends ListAppDashboardsRequest = ListAppDashboardsRequest, ResponseData = DashboardOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/bkmonitor/dashboards')(params, config),
  /**
   * 创建应用仪表盘绑定
   *
   * @method POST
   * @path /apps/{appID}/bkmonitor/dashboards
   * @tag bkmonitor-dashboard
   * @param appID path string required 应用 ID
   * @param body body AppDashboardCreateInput required 绑定请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 409 GinErrorOutput Conflict
   */
  createAppDashboard: async <Request extends CreateAppDashboardRequest = CreateAppDashboardRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/bkmonitor/dashboards')(params, config),
  /**
   * 更新应用仪表盘绑定
   *
   * @method PUT
   * @path /apps/{appID}/bkmonitor/dashboards/{uid}
   * @tag bkmonitor-dashboard
   * @param appID path string required 应用 ID
   * @param uid path string required 仪表盘 uid
   * @param body body AppDashboardUpdateInput required 更新请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  updateAppDashboard: async <Request extends UpdateAppDashboardRequest = UpdateAppDashboardRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/bkmonitor/dashboards/{uid}')(params, config),
  /**
   * 删除应用仪表盘绑定
   *
   * @method DELETE
   * @path /apps/{appID}/bkmonitor/dashboards/{uid}
   * @tag bkmonitor-dashboard
   * @param appID path string required 应用 ID
   * @param uid path string required 仪表盘 uid
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  deleteAppDashboard: async <Request extends DeleteAppDashboardRequest = DeleteAppDashboardRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/bkmonitor/dashboards/{uid}')(params, config),
};
