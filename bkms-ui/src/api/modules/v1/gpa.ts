/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { GetAppEnvGPAConfigRequest, GPAConfigOutputObj, UpsertAppEnvGPAConfigRequest, EmptyOutput, DeleteAppEnvGPAConfigRequest, ToggleAppEnvGPAConfigRequest } from '~/@types/v1/gpa';

export const GpaService = {
  /**
   * 查询应用在指定环境的 GPA 配置
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/autoscaler
   * @tag gpa
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 GetGPAConfigOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getAppEnvGPAConfig: async <Request extends GetAppEnvGPAConfigRequest = GetAppEnvGPAConfigRequest, ResponseData = GPAConfigOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/autoscaler')(params, config),
  /**
   * 创建或更新应用在指定环境的 GPA 配置
   *
   * @method PUT
   * @path /apps/{appID}/envs/{envName}/autoscaler
   * @tag gpa
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param body body UpsertGPAConfigInput required 请求体
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  upsertAppEnvGPAConfig: async <Request extends UpsertAppEnvGPAConfigRequest = UpsertAppEnvGPAConfigRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/envs/{envName}/autoscaler')(params, config),
  /**
   * 删除应用在指定环境的 GPA 配置
   *
   * @method DELETE
   * @path /apps/{appID}/envs/{envName}/autoscaler
   * @tag gpa
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteAppEnvGPAConfig: async <Request extends DeleteAppEnvGPAConfigRequest = DeleteAppEnvGPAConfigRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/envs/{envName}/autoscaler')(params, config),
  /**
   * 开关应用在指定环境的 GPA
   *
   * @method PATCH
   * @path /apps/{appID}/envs/{envName}/autoscaler/toggle
   * @tag gpa
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param body body ToggleGPAConfigInput required 请求体
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  toggleAppEnvGPAConfig: async <Request extends ToggleAppEnvGPAConfigRequest = ToggleAppEnvGPAConfigRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.patch<Request, ResponseData>('/apps/{appID}/envs/{envName}/autoscaler/toggle')(params, config),
};
