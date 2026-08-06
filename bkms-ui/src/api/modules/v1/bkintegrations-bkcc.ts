/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListBKCCAuthorizedBusinessesRequest, BusinessInfoOutput } from '~/@types/v1/bkintegrations-bkcc';

export const BkintegrationsBkccService = {
  /**
   * 获取用户有权限的 BKCC 业务列表
   *
   * @method GET
   * @path /bkcc/businesses/authorized
   * @tag bkintegrations-bkcc
   * @response 200 ListBKCCAuthorizedBusinessesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listBKCCAuthorizedBusinesses: async <Request extends ListBKCCAuthorizedBusinessesRequest = ListBKCCAuthorizedBusinessesRequest, ResponseData = BusinessInfoOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/bkcc/businesses/authorized')(params, config),
};
