/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListPortPoolsRequest, PortPoolConfigOutputObj, CreatePortPoolRequest, UpdatePortPoolRequest, DeletePortPoolRequest } from '~/@types/v1/port-pool';

export const PortPoolService = {
  /**
   * 获取端口池列表
   *
   * @method GET
   * @path /envs/{envID}/port-pools
   * @tag port-pool
   * @param envID path string required 环境 ID
   * @response 200 ListPortPoolsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listPortPools: async <Request extends ListPortPoolsRequest = ListPortPoolsRequest, ResponseData = PortPoolConfigOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/envs/{envID}/port-pools')(params, config),
  /**
   * 创建端口池
   *
   * @method POST
   * @path /envs/{envID}/port-pools
   * @tag port-pool
   * @param envID path string required 环境 ID
   * @param body body CreatePortPoolInput required 请求体
   * @response 200 unknown OK
   * @response 400 GinErrorOutput Bad Request
   */
  createPortPool: async <Request extends CreatePortPoolRequest = CreatePortPoolRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/envs/{envID}/port-pools')(params, config),
  /**
   * 更新端口池
   *
   * @method PUT
   * @path /envs/{envID}/port-pools/{name}
   * @tag port-pool
   * @param envID path string required 环境 ID
   * @param name path string required 端口池名称
   * @param body body UpdatePortPoolInput required 请求体
   * @response 200 unknown OK
   * @response 400 GinErrorOutput Bad Request
   */
  updatePortPool: async <Request extends UpdatePortPoolRequest = UpdatePortPoolRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/envs/{envID}/port-pools/{name}')(params, config),
  /**
   * 删除端口池
   *
   * @method DELETE
   * @path /envs/{envID}/port-pools/{name}
   * @tag port-pool
   * @param envID path string required 环境 ID
   * @param name path string required 端口池名称
   * @response 200 unknown OK
   * @response 400 GinErrorOutput Bad Request
   */
  deletePortPool: async <Request extends DeletePortPoolRequest = DeletePortPoolRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/envs/{envID}/port-pools/{name}')(params, config),
};
