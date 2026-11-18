/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { GetFrontendFeaturesRequest, FrontendFeaturesOutput } from '~/@types/v1/frontend-features';

export const FrontendFeaturesService = {
  /**
   * 获取前端功能开关
   *
   * 返回由部署配置决定的功能开关，前端据此决定页面入口是否展示
   *
   * @method GET
   * @path /frontend-features
   * @tag frontend-features
   * @response 200 FrontendFeaturesResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  getFrontendFeatures: async <Request extends GetFrontendFeaturesRequest = GetFrontendFeaturesRequest, ResponseData = FrontendFeaturesOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/frontend-features')(params, config),
};
