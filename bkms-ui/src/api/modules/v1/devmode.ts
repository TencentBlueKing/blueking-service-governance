/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { DevModePublishPreflightRequest, PreflightData, ListDevModePublishRecordsRequest, PaginatedPublishRecords, CreateDevModePublishRecordsRequest, CreatePublishRecordsData } from '~/@types/v1/devmode';

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
   * 获取开发模式发布记录列表
   *
   * @method GET
   * @path /devmode/{appID}/envs/{envName}/publish-records
   * @tag devmode
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param keyword query string 搜索关键字
   * @param page query number required 分页页码（从 1 开始）
   * @param pageSize query number required 分页大小
   * @response 200 ListPublishRecordsOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   */
  listDevModePublishRecords: async <Request extends ListDevModePublishRecordsRequest = ListDevModePublishRecordsRequest, ResponseData = PaginatedPublishRecords>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/devmode/{appID}/envs/{envName}/publish-records')(params, config),
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
};
