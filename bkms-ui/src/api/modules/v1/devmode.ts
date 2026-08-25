/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { DevModePublishPreflightRequest, PreflightData } from '~/@types/v1/devmode';

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
};
