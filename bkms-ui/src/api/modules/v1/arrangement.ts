/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListPlaceholderVarsRequest, PlaceholderVarOutputObj } from '~/@types/v1/arrangement';

export const ArrangementService = {
  /**
   * 获取编排可用的应用占位符变量列表
   *
   * @method GET
   * @path /placeholder-vars
   * @tag arrangement
   * @response 200 ListPlaceholderVarsOutput OK
   */
  listPlaceholderVars: async <Request extends ListPlaceholderVarsRequest = ListPlaceholderVarsRequest, ResponseData = PlaceholderVarOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/placeholder-vars')(params, config),
};
