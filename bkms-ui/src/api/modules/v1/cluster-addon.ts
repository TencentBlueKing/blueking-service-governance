/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListClusterAddonsRequest, ListClusterAddonsOutput, UpsertClusterAddonRequest, DeleteClusterAddonRequest, DeleteClusterAddonOutput } from '~/@types/v1/cluster-addon';

export const ClusterAddonService = {
  /**
   * 查询可安装的集群插件列表
   *
   * @method GET
   * @path /envs/{envID}/cluster-addons
   * @tag cluster-addon
   * @param envID path string required 环境 ID
   * @param namespace query string 命名空间
   * @response 200 ListClusterAddonsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listClusterAddons: async <Request extends ListClusterAddonsRequest = ListClusterAddonsRequest, ResponseData = ListClusterAddonsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/envs/{envID}/cluster-addons')(params, config),
  /**
   * 部署/更新集群插件
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
  upsertClusterAddon: async <Request extends UpsertClusterAddonRequest = UpsertClusterAddonRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/envs/{envID}/cluster-addons/{addonName}')(params, config),
  /**
   * 卸载集群插件
   *
   * @method DELETE
   * @path /envs/{envID}/cluster-addons/{addonName}
   * @tag cluster-addon
   * @param envID path string required 环境 ID
   * @param addonName path string required 插件名称
   * @param namespace query string 命名空间
   * @response 200 DeleteClusterAddonOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteClusterAddon: async <Request extends DeleteClusterAddonRequest = DeleteClusterAddonRequest, ResponseData = DeleteClusterAddonOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/envs/{envID}/cluster-addons/{addonName}')(params, config),
};
