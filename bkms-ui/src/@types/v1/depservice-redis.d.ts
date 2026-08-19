/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：depservice-redis

export interface ListRedisInstancesRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 实例状态
   */
  status?: string;
  /**
   * 作用域类型
   */
  scopeType?: string;
}

export type CreateRedisInstanceRequest = CreateRedisInstanceInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export interface GetRedisInstanceRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 实例 ID
   */
  instanceID: string;
}

export interface DeleteRedisInstanceRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 实例 ID
   */
  instanceID: string;
}

export interface ListRedisInstancesOutput {
  /**
   * 实例列表
   */
  data?: RedisInstanceOutputObj[];
}

export interface CreateRedisInstanceInput {
  /**
   * --- Redis / DBM 创建参数 ---
   * 业务 ID
   */
  bkBizID: number;
  /**
   * 云区域 ID
   */
  bkCloudID?: number;
  /**
   * 集群别名（集群模式）
   */
  clusterAlias?: string;
  /**
   * 集群名称（小写字母开头，仅小写字母/数字/连字符）
   */
  clusterName: string;
  /**
   * 集群分片数
   */
  clusterShardNum?: number;
  /**
   * 集群类型
   */
  clusterType: string;
  /**
   * DB 数量
   */
  databases?: number;
  /**
   * 业务英文缩写
   */
  dbAppAbbr: string;
  /**
   * 版本号（如 Redis-6）
   */
  dbVersion: string;
  /**
   * 描述
   */
  description?: string;
  /**
   * 容灾级别
   */
  disasterToleranceLevel?: string;
  /**
   * 主机来源
   */
  ipSource?: string;
  /**
   * 实例名称（同 workspace 下唯一）
   */
  name: string;
  /**
   * 主从起始端口
   */
  port?: number;
  /**
   * 集群接入层端口
   */
  proxyPort?: number;
  /**
   * Redis 访问密码
   */
  redisPwd?: string;
  /**
   * 资源池申请规格
   */
  resourceSpec?: ResourceSpecInput;
  /**
   * 作用域类型: workspace / envType / env
   */
  scopeType: "workspace" | "envType" | "env";
  /**
   * 作用域值；workspace 时为空，envType 为环境类型，env 为环境名
   */
  scopeValue?: string;
}

export interface GetRedisInstanceOutput {
  data?: RedisInstanceOutputObj;
}

export interface EmptyOutput {
}

export interface RedisInstanceOutputObj {
  /**
   * 非敏感配置
   */
  config?: RedisInstanceConfigOutput;
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 描述
   */
  description?: string;
  /**
   * 实例 ID
   */
  id?: string;
  /**
   * 状态详情
   */
  message?: string;
  /**
   * 实例名称
   */
  name?: string;
  /**
   * 操作人
   */
  operator?: string;
  /**
   * Provider 类型
   */
  providerType?: string;
  /**
   * 作用域类型
   */
  scopeType?: string;
  /**
   * 作用域值
   */
  scopeValue?: string;
  /**
   * 服务名
   */
  serviceName?: string;
  /**
   * 实例状态
   */
  status?: string;
  /**
   * 更新时间
   */
  updatedAt?: string;
  /**
   * 引用该实例的应用 ID 列表（由绑定反查）
   */
  usedAppIDs?: string[];
  /**
   * 工作空间 ID
   */
  workspaceID?: string;
}

export interface RedisInstanceConfigOutput {
  bkBizID?: number;
  clusterID?: number;
  clusterName?: string;
  clusterType?: string;
  domain?: string;
  port?: number;
}

export interface ResourceSpecInput {
  backendGroup?: ResourceSpecItemInput;
  proxy?: ResourceSpecItemInput;
}

export interface ResourceSpecItemInput {
  count?: number;
  locationSpec?: LocationSpecInput;
  specID?: number;
}

export interface LocationSpecInput {
  city?: string;
  subZoneIDs?: number[];
}
