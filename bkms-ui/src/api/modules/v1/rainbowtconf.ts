/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { GetRainbowTConfRequest, RainbowTConfOutput, PutRainbowTConfRequest } from '~/@types/v1/rainbowtconf';

export const RainbowtconfService = {
  /**
   * 获取应用的七彩石入口
   *
   * 未配置时 url 为空；应用所在空间未开启七彩石配置开关时不可用
   *
   * @method GET
   * @path /apps/{appID}/rainbow-tconf
   * @tag rainbowtconf
   * @param appID path string required 应用 ID
   * @response 200 RainbowTConfResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  getRainbowTConf: async <Request extends GetRainbowTConfRequest = GetRainbowTConfRequest, ResponseData = RainbowTConfOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/rainbow-tconf')(params, config),
  /**
   * 设置应用的七彩石入口
   *
   * 地址仅支持 https；应用所在空间未开启七彩石配置开关时不可用
   *
   * @method PUT
   * @path /apps/{appID}/rainbow-tconf
   * @tag rainbowtconf
   * @param appID path string required 应用 ID
   * @param body body PutRainbowTConfInput required 七彩石页面地址
   * @response 200 RainbowTConfResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  putRainbowTConf: async <Request extends PutRainbowTConfRequest = PutRainbowTConfRequest, ResponseData = RainbowTConfOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/rainbow-tconf')(params, config),
};
