/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：polaris-access-points

export interface GetEnvPolarisOperatorRequest {
  /**
   * 环境 ID
   */
  envID: string;
}

export interface ListPolarisAccessPointsRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export type CreatePolarisAccessPointRequest = AccessPointCreateInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export type TestPolarisAccessPointConnectivityRequest = ConnectivityTestInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export interface GetPolarisAccessPointNetworkInfoRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export interface DeletePolarisAccessPointRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 接入点 ID
   */
  accessPointID: string;
}

export type UpdatePolarisAccessPointRequest = AccessPointUpdateInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 接入点 ID
   */
  accessPointID: string;
};

export interface ListPolarisAccessPointNamespacesRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 接入点 ID
   */
  accessPointID: string;
}

export type InstallPolarisOperatorsRequest = OperatorInstallInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 接入点 ID
   */
  accessPointID: string;
};

export interface ListPolarisClustersRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export interface ListPolarisOperatorsRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export interface PolarisOperatorResponse {
  data?: PolarisOperatorOutput;
}

export interface AccessPointListResponse {
  data?: AccessPointOutput[];
}

export interface AccessPointCreateInput {
  /**
   * 客户端地址，IP 或域名，1 ~ 10 个；OpenAPI 端口固定为 8090，SDK 端口固定为 8091
   */
  clientHosts: string[];
  /**
   * 绑定到该接入点的集群 ID；缺省表示不绑定集群
   */
  clusterIDs?: string[];
  /**
   * 北极星控制台地址，IP:Port 或域名，缺省协议时补 http://
   */
  consoleURL: string;
  /**
   * 接入点名称，空间内唯一
   */
  name: string;
  /**
   * 北极星鉴权 Token，选填
   */
  token?: string;
}

export interface AccessPointResponse {
  data?: AccessPointOutput;
}

export interface ConnectivityTestInput {
  /**
   * 客户端地址
   */
  clientHosts: string[];
  /**
   * 北极星控制台地址
   */
  consoleURL: string;
  /**
   * 已有接入点的 ID
   */
  id?: string;
  /**
   * 北极星鉴权 Token
   */
  token?: string;
}

export interface ConnectivityTestResponse {
  data?: ConnectivityTestOutput;
}

export interface NetworkInfoResponse {
  data?: NetworkInfoOutput;
}

export interface EmptyResponse {
}

export interface AccessPointUpdateInput {
  /**
   * 客户端地址
   */
  clientHosts?: string[];
  /**
   * 绑定到该接入点的集群 ID，全量替换；显式传空数组表示解除全部绑定
   */
  clusterIDs?: string[];
  /**
   * 北极星控制台地址
   */
  consoleURL?: string;
  /**
   * 接入点名称
   */
  name?: string;
  /**
   * 北极星鉴权 Token；显式传空字符串表示清除
   */
  token?: string;
}

export interface NamespacesResponse {
  data?: string[];
}

export interface OperatorInstallInput {
  /**
   * chart 版本
   */
  chartVersion: string;
  /**
   * 目标集群 ID，必须已绑定该接入点
   */
  clusterIDs: string[];
}

export interface OperatorTaskListResponse {
  data?: OperatorTaskOutput[];
}

export interface PolarisClusterListResponse {
  data?: PolarisClusterOutput[];
}

export interface PolarisOperatorListResponse {
  data?: PolarisOperatorListOutput;
}

export interface PolarisOperatorListOutput {
  /**
   * 可选的 chart 版本，从新到旧
   */
  chartVersions?: string[];
  /**
   * 候选集群的组件状态
   */
  operators?: PolarisOperatorOutput[];
}

export interface PolarisOperatorOutput {
  /**
   * 集群绑定的自建接入点 ID，未绑定为空
   */
  accessPointID?: string;
  /**
   * 已安装的 chart 版本，未安装为空
   */
  chartVersion?: string;
  /**
   * 集群 ID
   */
  clusterID?: string;
  /**
   * 与接入点期望值不一致的 values 键，仅 outdated 时非空
   */
  driftFields?: string[];
  /**
   * 是否已安装
   */
  installed?: boolean;
  /**
   * 最近一次安装任务，没有任务时为 null
   */
  lastTask?: OperatorTaskOutput;
  /**
   * helm release 所在命名空间，未安装为空
   */
  namespace?: string;
  /**
   * helm release 名称，未安装为空
   */
  releaseName?: string;
  /**
   * 组件状态
   */
  status?: OperatorStatus;
}

export interface OperatorTaskOutput {
  /**
   * chart 版本
   */
  chartVersion?: string;
  /**
   * 目标集群 ID
   */
  clusterID?: string;
  /**
   * 结束时间，运行中为 null
   */
  finishedAt?: string;
  /**
   * 任务 ID
   */
  id?: string;
  /**
   * 失败原因等补充信息
   */
  message?: string;
  /**
   * 发起人
   */
  operator?: string;
  /**
   * 开始时间
   */
  startedAt?: string;
  /**
   * 任务状态
   */
  status?: Status;
}

export type OperatorStatus = "ready" | "installing" | "failed" | "outdated" | "not_installed";

export type Status = "running" | "succeeded" | "failed";

export interface PolarisClusterOutput {
  /**
   * 绑定的自建接入点 ID；未绑定为空，此时集群不能使用北极星
   */
  accessPointID?: string;
  /**
   * 绑定的自建接入点名称，未绑定为空
   */
  accessPointName?: string;
  /**
   * 集群 ID
   */
  clusterID?: string;
  /**
   * 集群名称
   */
  clusterName?: string;
  /**
   * 集群状态
   */
  clusterStatus?: string;
  /**
   * 集群上的空间环境（含特性环境）
   */
  envs?: PolarisClusterEnvOutput[];
  /**
   * 集群上是否有环境被北极星配置使用；为 true 时不能解绑或换绑
   */
  inUse?: boolean;
}

export interface PolarisClusterEnvOutput {
  /**
   * 环境 ID
   */
  id?: string;
  /**
   * 是否为特性环境
   */
  isFeature?: boolean;
  /**
   * 环境名称
   */
  name?: string;
}

export interface NetworkInfoOutput {
  /**
   * 平台访问北极星的出口 IP，需加入北极星白名单
   */
  egressIPs?: string[];
}

export interface ConnectivityTestOutput {
  /**
   * 是否开启客户端鉴权
   */
  clientAuth?: boolean;
  /**
   * 是否开启控制台鉴权
   */
  consoleAuth?: boolean;
  /**
   * 失败原因或提示信息
   */
  message?: string;
  /**
   * 北极星上的命名空间（已过滤系统命名空间 Polaris），网络不通时为空
   */
  namespaces?: string[];
  /**
   * 是否通过
   */
  ok?: boolean;
  /**
   * 失败的原因，通过时为空：networkUnreachable 网络不通 / clientAuthEnabled 开启了客户端鉴权 / tokenInvalid Token 无效
   */
  reason?: FailureReason;
}

export type FailureReason = "networkUnreachable" | "clientAuthEnabled" | "tokenInvalid";

export interface AccessPointOutput {
  /**
   * 绑定的集群数量
   */
  boundClusterCount?: number;
  /**
   * 客户端地址
   */
  clientHosts?: string[];
  /**
   * 北极星控制台地址
   */
  consoleURL?: string;
  /**
   * 是否填写了 Token
   */
  hasToken?: boolean;
  /**
   * 接入点 ID
   */
  id?: string;
  /**
   * 最近一次连通测试结果，从未测试时为 null
   */
  lastConnectivity?: ConnectivityResultOutput;
  /**
   * 接入点名称
   */
  name?: string;
  /**
   * 最近修改时间
   */
  updatedAt?: string;
  /**
   * 最近修改人
   */
  updater?: string;
}

export interface ConnectivityResultOutput {
  /**
   * 测试时间
   */
  checkedAt?: string;
  /**
   * 是否开启客户端鉴权
   */
  clientAuth?: boolean;
  /**
   * 是否开启控制台鉴权
   */
  consoleAuth?: boolean;
  /**
   * 失败原因或提示信息
   */
  message?: string;
  /**
   * 是否通过
   */
  ok?: boolean;
  /**
   * 失败的原因，通过时为空：networkUnreachable 网络不通 / clientAuthEnabled 开启了客户端鉴权 / tokenInvalid Token 无效
   */
  reason?: FailureReason;
}
