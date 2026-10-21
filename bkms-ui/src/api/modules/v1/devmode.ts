/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { DevModePublishPreflightRequest, PreflightData, CreateDevModePublishRecordsRequest, CreatePublishRecordsData, ListDevModePublishRecordsRequest, PaginatedPublishRecords, RestartDevModeProcessesRequest } from '~/@types/v1/devmode';

export const DevmodeService = {
  /**
   * 开发模式 Publish 预检
   *
   * @method POST
   * @path /devmode/{appID}/envs/{envName}/preflight
   * @tag devmode
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param body body PreflightBodyInput required 预检请求体
   * @response 200 PreflightOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  devModePublishPreflight: async <Request extends DevModePublishPreflightRequest = DevModePublishPreflightRequest, ResponseData = PreflightData>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/devmode/{appID}/envs/{envName}/preflight')(params, config),
  /**
   * 上报开发模式发布结果
   *
   * @method POST
   * @path /devmode/{appID}/envs/{envName}/publish-records
   * @tag devmode
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param body body CreatePublishRecordsInput required 发布结果上报请求体
   * @response 200 CreatePublishRecordsOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  createDevModePublishRecords: async <Request extends CreateDevModePublishRecordsRequest = CreateDevModePublishRecordsRequest, ResponseData = CreatePublishRecordsData>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/devmode/{appID}/envs/{envName}/publish-records')(params, config),
  /**
   * 获取开发模式发布记录列表
   *
   * @method GET
   * @path /devmode/{appID}/publish-records
   * @tag devmode
   * @param appID path string required 应用 ID
   * @param envNames query string[] 环境名称，与 envTypes 二选一
   * @param envTypes query string[] 环境类型，与 envNames 二选一
   * @param keyword query string 搜索关键字
   * @param page query number required 分页页码（从 1 开始）
   * @param pageSize query number required 分页大小
   * @param since query string 只返回创建时间不早于该时间的记录（RFC3339）
   * @param statuses query string[] 状态过滤
   * @response 200 ListPublishRecordsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listDevModePublishRecords: async <Request extends ListDevModePublishRecordsRequest = ListDevModePublishRecordsRequest, ResponseData = PaginatedPublishRecords>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/devmode/{appID}/publish-records')(params, config),
  /**
   * 开发模式进程重启
   *
   * @method POST
   * @path /devmode/{appID}/restart
   * @tag devmode
   * @param appID path string required 应用 ID
   * @param body body RestartInput required 进程重启请求体
   * @response 202 RestartOutput Accepted
   * @response 400 GinErrorOutput Bad Request
   */
  restartDevModeProcesses: async <Request extends RestartDevModeProcessesRequest = RestartDevModeProcessesRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/devmode/{appID}/restart')(params, config),
};
