/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListEnvClusterAddonsRequest, ListClusterAddonsOutput, UpsertEnvClusterAddonRequest, DeleteEnvClusterAddonRequest, DeleteClusterAddonOutput, ListClusterAddonsRequest, UpsertClusterAddonRequest } from '~/@types/v1/cluster-addon';

export const ClusterAddonService = {
  /**
   * 查询环境所在集群的集群插件列表
   *
   * @method GET
   * @path /envs/{envID}/cluster-addons
   * @tag cluster-addon
   * @param envID path string required 环境 ID
   * @param namespace query string 命名空间，默认为插件定义中的 defaultNamespace
   * @response 200 ListClusterAddonsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listEnvClusterAddons: async <Request extends ListEnvClusterAddonsRequest = ListEnvClusterAddonsRequest, ResponseData = ListClusterAddonsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/envs/{envID}/cluster-addons')(params, config),
  /**
   * 在环境所在集群上部署/更新集群插件
   *
   * @method POST
   * @path /envs/{envID}/cluster-addons/{addonName}
   * @tag cluster-addon
   * @param envID path string required 环境 ID
   * @param addonName path string required 插件名称
   * @param body body UpsertClusterAddonInput required 请求体
   * @response 201 unknown Created
   * @response 400 GinErrorOutput Bad Request
   */
  upsertEnvClusterAddon: async <Request extends UpsertEnvClusterAddonRequest = UpsertEnvClusterAddonRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/envs/{envID}/cluster-addons/{addonName}')(params, config),
  /**
   * 卸载环境所在集群上的集群插件
   *
   * @method DELETE
   * @path /envs/{envID}/cluster-addons/{addonName}
   * @tag cluster-addon
   * @param envID path string required 环境 ID
   * @param addonName path string required 插件名称
   * @param namespace query string 命名空间，默认为插件定义中的 defaultNamespace
   * @response 200 DeleteClusterAddonOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteEnvClusterAddon: async <Request extends DeleteEnvClusterAddonRequest = DeleteEnvClusterAddonRequest, ResponseData = DeleteClusterAddonOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/envs/{envID}/cluster-addons/{addonName}')(params, config),
  /**
   * 查询空间集群的集群插件及安装状态
   *
   * 返回适用于该集群的全部插件（含未安装的），每项带安装状态、当前 chart 版本与 values。
   * 集群必须是空间绑定的 BCS 项目下的非共享集群
   *
   * @method GET
   * @path /workspaces/{workspaceID}/clusters/{clusterID}/cluster-addons
   * @tag cluster-addon
   * @param workspaceID path string required 工作空间 ID
   * @param clusterID path string required BCS 集群 ID
   * @param namespace query string 命名空间，默认为插件定义中的 defaultNamespace
   * @response 200 ListClusterAddonsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listClusterAddons: async <Request extends ListClusterAddonsRequest = ListClusterAddonsRequest, ResponseData = ListClusterAddonsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/clusters/{clusterID}/cluster-addons')(params, config),
  /**
   * 在空间集群上部署/更新集群插件
   *
   * 集群必须是空间绑定的 BCS 项目下的非共享集群，可以没有环境。
   * 与环境维度的接口共用安装锁，同一集群命名空间内的同一插件同时只允许一个安装或卸载操作。
   *
   * @method POST
   * @path /workspaces/{workspaceID}/clusters/{clusterID}/cluster-addons/{addonName}
   * @tag cluster-addon
   * @param workspaceID path string required 工作空间 ID
   * @param clusterID path string required BCS 集群 ID
   * @param addonName path string required 插件名称
   * @param body body UpsertClusterAddonInput required 请求体
   * @response 201 unknown Created
   * @response 400 GinErrorOutput Bad Request
   */
  upsertClusterAddon: async <Request extends UpsertClusterAddonRequest = UpsertClusterAddonRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/clusters/{clusterID}/cluster-addons/{addonName}')(params, config),
};
