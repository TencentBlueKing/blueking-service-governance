/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { CreateAppConfigFileDefRequest, CreateDefOutput, ListDefaultFilesWithDefRequest, ListDefsOutput, GetAppConfigFileDefDetailRequest, GetDefDetailOutput, UpdateAppConfigFileDefRequest, UpdateDefOutput, DeleteAppConfigFileDefRequest, DeleteDefOutput, UpdateAppConfigFileDefContentRequest, UpdateContentOutput, ListAppConfigFileDefEnvInstancesRequest, ListEnvInstancesOutput, ResetAppConfigFileDefEnvToDefaultRequest, ResetEnvOutput, GetMountPreviewRequest, MountPreviewOutput } from '~/@types/v1/app-config-file-defs';

export const AppConfigFileDefsService = {
  /**
   * 创建配置文件定义及默认文件
   *
   * @method POST
   * @path /apps/{appID}/app-config-file-defs
   * @tag app-config-file-defs
   * @param appID path string required 应用 ID
   * @param body body CreateDefInput required 创建配置文件定义请求
   * @response 200 CreateDefOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createAppConfigFileDef: async <Request extends CreateAppConfigFileDefRequest = CreateAppConfigFileDefRequest, ResponseData = CreateDefOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/app-config-file-defs')(params, config),
  /**
   * 列出应用下所有配置文件定义及其默认文件信息
   *
   * @method GET
   * @path /apps/{appID}/app-config-file-defs/defaults
   * @tag app-config-file-defs
   * @param appID path string required 应用 ID
   * @response 200 ListDefsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listDefaultFilesWithDef: async <Request extends ListDefaultFilesWithDefRequest = ListDefaultFilesWithDefRequest, ResponseData = ListDefsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/app-config-file-defs/defaults')(params, config),
  /**
   * 获取配置文件定义详情
   *
   * @method GET
   * @path /apps/{appID}/app-config-file-defs/{id}
   * @tag app-config-file-defs
   * @param appID path string required 应用 ID
   * @param id path string required 配置文件定义 ID
   * @param envName query string 环境名称，为空返回默认文件
   * @response 200 GetDefDetailOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getAppConfigFileDefDetail: async <Request extends GetAppConfigFileDefDetailRequest = GetAppConfigFileDefDetailRequest, ResponseData = GetDefDetailOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/app-config-file-defs/{id}')(params, config),
  /**
   * 更新配置文件定义信息
   *
   * @method PUT
   * @path /apps/{appID}/app-config-file-defs/{id}
   * @tag app-config-file-defs
   * @param appID path string required 应用 ID
   * @param id path string required 配置文件定义 ID
   * @param body body UpdateDefInput required 更新配置文件定义请求
   * @response 200 UpdateDefOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateAppConfigFileDef: async <Request extends UpdateAppConfigFileDefRequest = UpdateAppConfigFileDefRequest, ResponseData = UpdateDefOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/app-config-file-defs/{id}')(params, config),
  /**
   * 删除配置文件定义及关联数据
   *
   * @method DELETE
   * @path /apps/{appID}/app-config-file-defs/{id}
   * @tag app-config-file-defs
   * @param appID path string required 应用 ID
   * @param id path string required 配置文件定义 ID
   * @response 200 DeleteDefOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteAppConfigFileDef: async <Request extends DeleteAppConfigFileDefRequest = DeleteAppConfigFileDefRequest, ResponseData = DeleteDefOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/app-config-file-defs/{id}')(params, config),
  /**
   * 更新配置文件内容
   *
   * @method PUT
   * @path /apps/{appID}/app-config-file-defs/{id}/content
   * @tag app-config-file-defs
   * @param appID path string required 应用 ID
   * @param id path string required 配置文件定义 ID
   * @param envName query string 环境名称，为空更新默认文件
   * @param body body UpdateContentInput required 更新配置内容请求
   * @response 200 UpdateContentOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateAppConfigFileDefContent: async <Request extends UpdateAppConfigFileDefContentRequest = UpdateAppConfigFileDefContentRequest, ResponseData = UpdateContentOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/app-config-file-defs/{id}/content')(params, config),
  /**
   * 列出配置文件定义的环境实例
   *
   * @method GET
   * @path /apps/{appID}/app-config-file-defs/{id}/env-instances
   * @tag app-config-file-defs
   * @param appID path string required 应用 ID
   * @param id path string required 配置文件定义 ID
   * @response 200 ListEnvInstancesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listAppConfigFileDefEnvInstances: async <Request extends ListAppConfigFileDefEnvInstancesRequest = ListAppConfigFileDefEnvInstancesRequest, ResponseData = ListEnvInstancesOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/app-config-file-defs/{id}/env-instances')(params, config),
  /**
   * 恢复环境配置为默认值
   *
   * @method DELETE
   * @path /apps/{appID}/app-config-file-defs/{id}/envs/{envName}
   * @tag app-config-file-defs
   * @param appID path string required 应用 ID
   * @param id path string required 配置文件定义 ID
   * @param envName path string required 环境名称
   * @response 200 ResetEnvOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  resetAppConfigFileDefEnvToDefault: async <Request extends ResetAppConfigFileDefEnvToDefaultRequest = ResetAppConfigFileDefEnvToDefaultRequest, ResponseData = ResetEnvOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/app-config-file-defs/{id}/envs/{envName}')(params, config),
  /**
   * 获取配置文件挂载预览
   *
   * @method GET
   * @path /apps/{appID}/mount-preview
   * @tag app-config-file-defs
   * @param appID path string required 应用 ID
   * @param envName query string 环境名称
   * @response 200 MountPreviewOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getMountPreview: async <Request extends GetMountPreviewRequest = GetMountPreviewRequest, ResponseData = MountPreviewOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/mount-preview')(params, config),
};
