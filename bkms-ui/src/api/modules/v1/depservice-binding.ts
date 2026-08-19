/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListServiceBindingsRequest, BindingOutputObj, CreateServiceBindingRequest, GetServiceBindingRequest, UpdateServiceBindingRequest, DeleteServiceBindingRequest, EmptyOutput } from '~/@types/v1/depservice-binding';

export const DepserviceBindingService = {
  /**
   * 查询依赖服务绑定列表
   *
   * @method GET
   * @path /apps/{appID}/deps/{serviceName}/bindings
   * @tag depservice-binding
   * @param appID path string required 应用 ID
   * @param serviceName path "redis" required 依赖服务名，目前仅支持 redis
   * @response 200 ListBindingsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listServiceBindings: async <Request extends ListServiceBindingsRequest = ListServiceBindingsRequest, ResponseData = BindingOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/deps/{serviceName}/bindings')(params, config),
  /**
   * 创建依赖服务绑定
   *
   * @method POST
   * @path /apps/{appID}/deps/{serviceName}/bindings
   * @tag depservice-binding
   * @param appID path string required 应用 ID
   * @param serviceName path "redis" required 依赖服务名，目前仅支持 redis
   * @param body body CreateBindingInput required 请求体
   * @response 201 BindingOutput Created
   * @response 400 GinErrorOutput Bad Request
   */
  createServiceBinding: async <Request extends CreateServiceBindingRequest = CreateServiceBindingRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/deps/{serviceName}/bindings')(params, config),
  /**
   * 查询依赖服务绑定详情
   *
   * @method GET
   * @path /apps/{appID}/deps/{serviceName}/bindings/{name}
   * @tag depservice-binding
   * @param appID path string required 应用 ID
   * @param serviceName path "redis" required 依赖服务名，目前仅支持 redis
   * @param name path string required 绑定名称
   * @response 200 BindingOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getServiceBinding: async <Request extends GetServiceBindingRequest = GetServiceBindingRequest, ResponseData = BindingOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/deps/{serviceName}/bindings/{name}')(params, config),
  /**
   * 更新依赖服务绑定
   *
   * @method PUT
   * @path /apps/{appID}/deps/{serviceName}/bindings/{name}
   * @tag depservice-binding
   * @param appID path string required 应用 ID
   * @param serviceName path "redis" required 依赖服务名，目前仅支持 redis
   * @param name path string required 绑定名称
   * @param body body UpdateBindingInput required 请求体
   * @response 200 BindingOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateServiceBinding: async <Request extends UpdateServiceBindingRequest = UpdateServiceBindingRequest, ResponseData = BindingOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/deps/{serviceName}/bindings/{name}')(params, config),
  /**
   * 删除依赖服务绑定
   *
   * @method DELETE
   * @path /apps/{appID}/deps/{serviceName}/bindings/{name}
   * @tag depservice-binding
   * @param appID path string required 应用 ID
   * @param serviceName path "redis" required 依赖服务名，目前仅支持 redis
   * @param name path string required 绑定名称
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteServiceBinding: async <Request extends DeleteServiceBindingRequest = DeleteServiceBindingRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/deps/{serviceName}/bindings/{name}')(params, config),
};
