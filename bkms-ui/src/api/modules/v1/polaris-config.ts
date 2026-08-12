/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListAppPolarisConfigsRequest, PolarisConfigOutputObj, CreateAppPolarisConfigRequest, PolarisNameOutputObj, ValidateAppPolarisConfigRequest, ValidateAppPolarisConfigOutput, DeleteAppPolarisConfigRequest, PatchAppPolarisConfigRequest, GetEnvInstanceStatsRequest, GetEnvInstanceStatsOutputObj, PutEnvWeightRequest, ListAppPolarisConfigVarsRequest, PolarisConfigVarOutput } from '~/@types/v1/polaris-config';

export const PolarisConfigService = {
  /**
   * 获取应用的北极星配置列表
   *
   * @method GET
   * @path /apps/{appID}/deps/polaris-configs
   * @tag polaris-config
   * @param appID path string required 应用 ID
   * @response 200 ListAppPolarisConfigsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listAppPolarisConfigs: async <Request extends ListAppPolarisConfigsRequest = ListAppPolarisConfigsRequest, ResponseData = PolarisConfigOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/deps/polaris-configs')(params, config),
  /**
   * 创建北极星配置
   *
   * @method POST
   * @path /apps/{appID}/deps/polaris-configs
   * @tag polaris-config
   * @param appID path string required 应用 ID
   * @param body body CreateAppPolarisConfigInput required 请求体
   * @response 200 CreateAppPolarisConfigOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createAppPolarisConfig: async <Request extends CreateAppPolarisConfigRequest = CreateAppPolarisConfigRequest, ResponseData = PolarisNameOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/deps/polaris-configs')(params, config),
  /**
   * 校验北极星配置（创建前预校验）
   *
   * @method POST
   * @path /apps/{appID}/deps/polaris-configs/validate
   * @tag polaris-config
   * @param appID path string required 应用 ID
   * @param body body CreateAppPolarisConfigInput required 请求体
   * @response 200 ValidateAppPolarisConfigOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  validateAppPolarisConfig: async <Request extends ValidateAppPolarisConfigRequest = ValidateAppPolarisConfigRequest, ResponseData = ValidateAppPolarisConfigOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/deps/polaris-configs/validate')(params, config),
  /**
   * 删除北极星配置
   *
   * @method DELETE
   * @path /apps/{appID}/deps/polaris-configs/{configName}
   * @tag polaris-config
   * @param appID path string required 应用 ID
   * @param configName path string required 配置名称
   * @response 200 unknown OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteAppPolarisConfig: async <Request extends DeleteAppPolarisConfigRequest = DeleteAppPolarisConfigRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/deps/polaris-configs/{configName}')(params, config),
  /**
   * 更新北极星配置
   *
   * @method PATCH
   * @path /apps/{appID}/deps/polaris-configs/{configName}
   * @tag polaris-config
   * @param appID path string required 应用 ID
   * @param configName path string required 配置名称
   * @param body body PatchAppPolarisConfigInput required 请求体
   * @response 200 PatchAppPolarisConfigOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  patchAppPolarisConfig: async <Request extends PatchAppPolarisConfigRequest = PatchAppPolarisConfigRequest, ResponseData = PolarisConfigOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.patch<Request, ResponseData>('/apps/{appID}/deps/polaris-configs/{configName}')(params, config),
  /**
   * 获取北极星配置各环境实例统计
   *
   * @method GET
   * @path /apps/{appID}/deps/polaris-configs/{configName}/env-instance-stats
   * @tag polaris-config
   * @param appID path string required 应用 ID
   * @param configName path string required 配置名称
   * @response 200 GetEnvInstanceStatsOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  getEnvInstanceStats: async <Request extends GetEnvInstanceStatsRequest = GetEnvInstanceStatsRequest, ResponseData = GetEnvInstanceStatsOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/deps/polaris-configs/{configName}/env-instance-stats')(params, config),
  /**
   * 更新指定环境的北极星实例权重
   *
   * @method PUT
   * @path /apps/{appID}/deps/polaris-configs/{configName}/envs/{envName}/weight
   * @tag polaris-config
   * @param appID path string required 应用 ID
   * @param configName path string required 配置名称
   * @param envName path string required 环境名称
   * @param body body PutEnvWeightInput required 请求体
   * @response 200 PutEnvWeightOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 500 GinErrorOutput Internal Server Error
   */
  putEnvWeight: async <Request extends PutEnvWeightRequest = PutEnvWeightRequest, ResponseData = PolarisConfigOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/deps/polaris-configs/{configName}/envs/{envName}/weight')(params, config),
  /**
   * 获取北极星配置变量列表
   *
   * @method GET
   * @path /apps/{appID}/deps/polaris-configs/{configName}/vars
   * @tag polaris-config
   * @param appID path string required 应用 ID
   * @param configName path string required 配置名称
   * @response 200 ListAppPolarisConfigVarsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listAppPolarisConfigVars: async <Request extends ListAppPolarisConfigVarsRequest = ListAppPolarisConfigVarsRequest, ResponseData = PolarisConfigVarOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/deps/polaris-configs/{configName}/vars')(params, config),
};
