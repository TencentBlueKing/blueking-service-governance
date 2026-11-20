/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { GetEnvPolarisOperatorRequest, PolarisOperatorOutput, ListPolarisAccessPointsRequest, AccessPointOutput, CreatePolarisAccessPointRequest, TestPolarisAccessPointConnectivityRequest, ConnectivityTestOutput, GetPolarisAccessPointNetworkInfoRequest, NetworkInfoOutput, DeletePolarisAccessPointRequest, EmptyResponse, UpdatePolarisAccessPointRequest, ListPolarisAccessPointNamespacesRequest, InstallPolarisOperatorsRequest, OperatorTaskOutput, ListPolarisClustersRequest, PolarisClusterOutput, ListPolarisOperatorsRequest, PolarisOperatorListOutput } from '~/@types/v1/polaris-access-points';

export const PolarisAccessPointsService = {
  /**
   * 获取环境的北极星组件状态
   *
   * @method GET
   * @path /envs/{envID}/polaris-operator
   * @tag polaris-access-points
   * @param envID path string required 环境 ID
   * @response 200 PolarisOperatorResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  getEnvPolarisOperator: async <Request extends GetEnvPolarisOperatorRequest = GetEnvPolarisOperatorRequest, ResponseData = PolarisOperatorOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/envs/{envID}/polaris-operator')(params, config),
  /**
   * 获取空间的北极星接入点列表
   *
   * 允许空间自建接入点时返回空间自建接入点
   *
   * @method GET
   * @path /workspaces/{workspaceID}/polaris-access-points
   * @tag polaris-access-points
   * @param workspaceID path string required 工作空间 ID
   * @response 200 AccessPointListResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  listPolarisAccessPoints: async <Request extends ListPolarisAccessPointsRequest = ListPolarisAccessPointsRequest, ResponseData = AccessPointOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/polaris-access-points')(params, config),
  /**
   * 创建北极星接入点
   *
   * 保存前服务端重新执行连通测试，失败或北极星开启了客户端鉴权时拒绝；Token 选填，加密存储。
   * 传入 clusterIDs 时同时把这些集群绑定到该接入点
   *
   * @method POST
   * @path /workspaces/{workspaceID}/polaris-access-points
   * @tag polaris-access-points
   * @param workspaceID path string required 工作空间 ID
   * @param body body AccessPointCreateInput required 接入点
   * @response 200 AccessPointResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  createPolarisAccessPoint: async <Request extends CreatePolarisAccessPointRequest = CreatePolarisAccessPointRequest, ResponseData = AccessPointOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/polaris-access-points')(params, config),
  /**
   * 测试北极星接入点连通性
   *
   * 依次检查网络、鉴权开关、Token，返回失败的原因与北极星上的命名空间，整体 10 秒超时。
   * 传入已有接入点 ID 且 token 为空时使用已保存的 Token，并记录本次结果
   *
   * @method POST
   * @path /workspaces/{workspaceID}/polaris-access-points/connectivity-test
   * @tag polaris-access-points
   * @param workspaceID path string required 工作空间 ID
   * @param body body ConnectivityTestInput required 接入地址
   * @response 200 ConnectivityTestResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  testPolarisAccessPointConnectivity: async <Request extends TestPolarisAccessPointConnectivityRequest = TestPolarisAccessPointConnectivityRequest, ResponseData = ConnectivityTestOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/polaris-access-points/connectivity-test')(params, config),
  /**
   * 获取平台访问北极星的网络信息
   *
   * 返回平台访问北极星的出口 IP，用户需要把它们加入自建北极星的白名单
   *
   * @method GET
   * @path /workspaces/{workspaceID}/polaris-access-points/network-info
   * @tag polaris-access-points
   * @param workspaceID path string required 工作空间 ID
   * @response 200 NetworkInfoResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  getPolarisAccessPointNetworkInfo: async <Request extends GetPolarisAccessPointNetworkInfoRequest = GetPolarisAccessPointNetworkInfoRequest, ResponseData = NetworkInfoOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/polaris-access-points/network-info')(params, config),
  /**
   * 删除北极星接入点
   *
   * 接入点仍绑定集群，或仍有北极星配置固定使用它时拒绝删除
   *
   * @method DELETE
   * @path /workspaces/{workspaceID}/polaris-access-points/{accessPointID}
   * @tag polaris-access-points
   * @param workspaceID path string required 工作空间 ID
   * @param accessPointID path string required 接入点 ID
   * @response 200 EmptyResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  deletePolarisAccessPoint: async <Request extends DeletePolarisAccessPointRequest = DeletePolarisAccessPointRequest, ResponseData = EmptyResponse>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/workspaces/{workspaceID}/polaris-access-points/{accessPointID}')(params, config),
  /**
   * 更新北极星接入点
   *
   * 字段缺省表示不修改；客户端地址或 Token 变化时重新执行连通测试，失败时拒绝保存。
   * 接入点被北极星配置使用时仍允许修改地址，集群组件会显示为需要更新。
   * 传入 clusterIDs 时全量替换该接入点绑定的集群
   *
   * @method PATCH
   * @path /workspaces/{workspaceID}/polaris-access-points/{accessPointID}
   * @tag polaris-access-points
   * @param workspaceID path string required 工作空间 ID
   * @param accessPointID path string required 接入点 ID
   * @param body body AccessPointUpdateInput required 待修改的字段
   * @response 200 AccessPointResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  updatePolarisAccessPoint: async <Request extends UpdatePolarisAccessPointRequest = UpdatePolarisAccessPointRequest, ResponseData = AccessPointOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.patch<Request, ResponseData>('/workspaces/{workspaceID}/polaris-access-points/{accessPointID}')(params, config),
  /**
   * 获取北极星接入点的命名空间
   *
   * 实时读取空间自建接入点上的北极星命名空间，并过滤系统命名空间 Polaris
   *
   * @method GET
   * @path /workspaces/{workspaceID}/polaris-access-points/{accessPointID}/namespaces
   * @tag polaris-access-points
   * @param workspaceID path string required 工作空间 ID
   * @param accessPointID path string required 接入点 ID
   * @response 200 NamespacesResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  listPolarisAccessPointNamespaces: async <Request extends ListPolarisAccessPointNamespacesRequest = ListPolarisAccessPointNamespacesRequest, ResponseData = string[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/polaris-access-points/{accessPointID}/namespaces')(params, config),
  /**
   * 安装或更新北极星组件
   *
   * 集群必须已绑定该接入点；每个集群创建一个异步任务，同一集群同时只允许一个运行中的任务。
   *
   * @method POST
   * @path /workspaces/{workspaceID}/polaris-access-points/{accessPointID}/operator-installs
   * @tag polaris-access-points
   * @param workspaceID path string required 工作空间 ID
   * @param accessPointID path string required 接入点 ID
   * @param body body OperatorInstallInput required 目标集群与 chart 版本
   * @response 200 OperatorTaskListResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  installPolarisOperators: async <Request extends InstallPolarisOperatorsRequest = InstallPolarisOperatorsRequest, ResponseData = OperatorTaskOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/polaris-access-points/{accessPointID}/operator-installs')(params, config),
  /**
   * 获取北极星集群绑定视图
   *
   * 候选集群为空间 BCS 项目下的全部非共享集群，含没有环境的集群；先列出有环境的集群，同类按集群 ID 排序。
   *
   * @method GET
   * @path /workspaces/{workspaceID}/polaris-clusters
   * @tag polaris-access-points
   * @param workspaceID path string required 工作空间 ID
   * @response 200 PolarisClusterListResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  listPolarisClusters: async <Request extends ListPolarisClustersRequest = ListPolarisClustersRequest, ResponseData = PolarisClusterOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/polaris-clusters')(params, config),
  /**
   * 获取北极星组件状态视图
   *
   * 返回候选集群的组件状态、与接入点不一致的参数、最近的安装任务，以及可选的 chart 版本。
   *
   * @method GET
   * @path /workspaces/{workspaceID}/polaris-operators
   * @tag polaris-access-points
   * @param workspaceID path string required 工作空间 ID
   * @response 200 PolarisOperatorListResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  listPolarisOperators: async <Request extends ListPolarisOperatorsRequest = ListPolarisOperatorsRequest, ResponseData = PolarisOperatorListOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/polaris-operators')(params, config),
};
