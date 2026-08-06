/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { GetResourceTopologyRequest, ResourceTopologyDataOutputObj, GetTopologyNodeDetailRequest, TopologyNodeDetailOutputObj, ListTopologyNodeEventsRequest, PaginatedTopologyNodeEventsOutputObj, GetTopologyNodeManifestRequest, TopologyNodeManifestOutputObj } from '~/@types/v1/topology';

export const TopologyService = {
  /**
   * 获取应用资源拓扑
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/resource-topology
   * @tag topology
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @response 200 GetResourceTopologyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getResourceTopology: async <Request extends GetResourceTopologyRequest = GetResourceTopologyRequest, ResponseData = ResourceTopologyDataOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/resource-topology')(params, config),
  /**
   * 获取节点详情
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/resource-topology/nodes/{nodeID}
   * @tag topology
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param nodeID path string required 拓扑节点 ID（base64url 无填充编码）
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @response 200 GetTopologyNodeDetailOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getTopologyNodeDetail: async <Request extends GetTopologyNodeDetailRequest = GetTopologyNodeDetailRequest, ResponseData = TopologyNodeDetailOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/resource-topology/nodes/{nodeID}')(params, config),
  /**
   * 获取节点事件列表
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/resource-topology/nodes/{nodeID}/events
   * @tag topology
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param nodeID path string required 拓扑节点 ID（base64url 无填充编码）
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @param level query string 事件级别（可选过滤参数，可选值：Normal, Warning）
   * @param startedAt query number 起始时间戳（可选过滤参数，如：1772223278）
   * @param endedAt query number 结束时间戳（可选过滤参数，如：1772223278）
   * @param page query number required 分页页码（从 1 开始）
   * @param pageSize query number required 每页数量
   * @response 200 ListTopologyNodeEventsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listTopologyNodeEvents: async <Request extends ListTopologyNodeEventsRequest = ListTopologyNodeEventsRequest, ResponseData = PaginatedTopologyNodeEventsOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/resource-topology/nodes/{nodeID}/events')(params, config),
  /**
   * 获取节点 Manifest
   *
   * @method GET
   * @path /apps/{appID}/envs/{envName}/resource-topology/nodes/{nodeID}/manifest
   * @tag topology
   * @param appID path string required 应用 ID
   * @param envName path string required 部署环境名称
   * @param nodeID path string required 拓扑节点 ID（base64url 无填充编码）
   * @param trafficLaneName query string 部署的泳道名称（空字符串表示不使用泳道）
   * @response 200 GetTopologyNodeManifestOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getTopologyNodeManifest: async <Request extends GetTopologyNodeManifestRequest = GetTopologyNodeManifestRequest, ResponseData = TopologyNodeManifestOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/envs/{envName}/resource-topology/nodes/{nodeID}/manifest')(params, config),
};
