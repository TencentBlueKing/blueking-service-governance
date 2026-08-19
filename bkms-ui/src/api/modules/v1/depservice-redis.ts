/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListRedisInstancesRequest, RedisInstanceOutputObj, CreateRedisInstanceRequest, GetRedisInstanceRequest, DeleteRedisInstanceRequest, EmptyOutput } from '~/@types/v1/depservice-redis';

export const DepserviceRedisService = {
  /**
   * 查询 Redis 依赖服务实例列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/deps/redis
   * @tag depservice-redis
   * @param workspaceID path string required 工作空间 ID
   * @param status query string 实例状态
   * @param scopeType query string 作用域类型
   * @response 200 ListRedisInstancesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listRedisInstances: async <Request extends ListRedisInstancesRequest = ListRedisInstancesRequest, ResponseData = RedisInstanceOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/deps/redis')(params, config),
  /**
   * 创建 Redis 依赖服务实例
   *
   * @method POST
   * @path /workspaces/{workspaceID}/deps/redis
   * @tag depservice-redis
   * @param workspaceID path string required 工作空间 ID
   * @param body body CreateRedisInstanceInput required 请求体
   * @response 201 CreateRedisInstanceOutput Created
   * @response 400 GinErrorOutput Bad Request
   */
  createRedisInstance: async <Request extends CreateRedisInstanceRequest = CreateRedisInstanceRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/deps/redis')(params, config),
  /**
   * 查询 Redis 依赖服务实例详情
   *
   * @method GET
   * @path /workspaces/{workspaceID}/deps/redis/{instanceID}
   * @tag depservice-redis
   * @param workspaceID path string required 工作空间 ID
   * @param instanceID path string required 实例 ID
   * @response 200 GetRedisInstanceOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getRedisInstance: async <Request extends GetRedisInstanceRequest = GetRedisInstanceRequest, ResponseData = RedisInstanceOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/deps/redis/{instanceID}')(params, config),
  /**
   * 删除 Redis 依赖服务实例
   *
   * @method DELETE
   * @path /workspaces/{workspaceID}/deps/redis/{instanceID}
   * @tag depservice-redis
   * @param workspaceID path string required 工作空间 ID
   * @param instanceID path string required 实例 ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteRedisInstance: async <Request extends DeleteRedisInstanceRequest = DeleteRedisInstanceRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/workspaces/{workspaceID}/deps/redis/{instanceID}')(params, config),
};
