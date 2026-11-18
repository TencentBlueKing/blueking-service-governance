/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { GetCfgSwitchesRequest, CfgSwitchesOutput } from '~/@types/v1/cfgswitch';

export const CfgswitchService = {
  /**
   * 获取空间的配置开关
   *
   * 开关由平台管理员在平台管理中设置，前端据此决定是否展示配置管理（BSCP）、七彩石等 tab；未设置时为 false
   *
   * @method GET
   * @path /workspaces/{workspaceID}/cfg-switches
   * @tag cfgswitch
   * @param workspaceID path string required 工作空间 ID
   * @response 200 CfgSwitchesResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  getCfgSwitches: async <Request extends GetCfgSwitchesRequest = GetCfgSwitchesRequest, ResponseData = CfgSwitchesOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/cfg-switches')(params, config),
};
