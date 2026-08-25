/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListHostPortsRequest, HostPortsOutput, PutHostPortsRequest } from '~/@types/v1/hostport';

export const HostportService = {
  /**
   * 获取应用 HostPort 列表及联邦环境待部署状态
   *
   * @method GET
   * @path /apps/{appID}/hostports
   * @tag hostport
   * @param appID path string required 应用 ID
   * @response 200 HostPortsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listHostPorts: async <Request extends ListHostPortsRequest = ListHostPortsRequest, ResponseData = HostPortsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/hostports')(params, config),
  /**
   * 全量保存应用 HostPort 端口列表
   *
   * @method PUT
   * @path /apps/{appID}/hostports
   * @tag hostport
   * @param appID path string required 应用 ID
   * @param body body PutHostPortsInput required 请求体
   * @response 200 HostPortsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  putHostPorts: async <Request extends PutHostPortsRequest = PutHostPortsRequest, ResponseData = HostPortsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/hostports')(params, config),
};
