/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：devmode

export type DevModePublishPreflightRequest = PreflightBodyInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
};

export type CreateDevModePublishRecordsRequest = CreatePublishRecordsInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
};

export interface ListDevModePublishRecordsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称，与 envTypes 二选一
   */
  envNames?: string[];
  /**
   * 环境类型，与 envNames 二选一
   */
  envTypes?: string[];
  /**
   * 搜索关键字
   */
  keyword?: string;
  /**
   * 分页页码（从 1 开始）
   */
  page: number;
  /**
   * 分页大小
   */
  pageSize: number;
  /**
   * 只返回创建时间不早于该时间的记录（RFC3339）
   */
  since?: string;
  /**
   * 状态过滤
   */
  statuses?: string[];
}

export type RestartDevModeProcessesRequest = RestartInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface PreflightBodyInput {
  /**
   * InstanceIDs 指定的实例 ID 列表（与 PublishAll 二选一）
   */
  instanceIDs?: string[];
  /**
   * PublishAll 是否发布到所有 Running 状态的实例（与 InstanceIDs 二选一）
   */
  publishAll?: boolean;
}

export interface PreflightOutput {
  data?: PreflightData;
}

export interface CreatePublishRecordsInput {
  /**
   * BinaryName 发布的二进制名称
   */
  binaryName: string;
  /**
   * FileSize 文件大小（字节）
   */
  fileSize?: number;
  /**
   * MD5 文件 MD5 值
   */
  md5: string;
  /**
   * Results 逐实例发布结果
   */
  results: PublishInstanceResult[];
}

export interface CreatePublishRecordsOutput {
  data?: CreatePublishRecordsData;
}

export interface ListPublishRecordsOutput {
  data?: PaginatedPublishRecords;
}

export interface RestartInput {
  /**
   * EnvTypes 按环境类型重启（匹配环境下所有 Running 实例）
   */
  envTypes?: RestartEnvTypesTarget;
  /**
   * Envs 按环境名称重启（各环境下所有 Running 实例）
   */
  envs?: RestartEnvsTarget;
  /**
   * Instances 重启指定环境下的指定实例
   */
  instances?: RestartInstancesTarget;
}

export interface RestartEnvTypesTarget {
  /**
   * Types 环境类型列表，不允许 production
   */
  types: string[];
}

export interface RestartEnvsTarget {
  /**
   * Names 环境名称列表
   */
  names: string[];
}

export interface RestartInstancesTarget {
  /**
   * EnvName 环境名称
   */
  envName: string;
  /**
   * InstanceIDs 实例 ID 列表，须属于 EnvName 环境
   */
  instanceIDs: string[];
}

export interface PaginatedPublishRecords {
  count?: string;
  results?: PublishRecordOutputObj[];
}

export interface PublishRecordOutputObj {
  action?: string;
  binaryName?: string;
  createdAt?: string;
  envName?: string;
  fileSize?: number;
  id?: string;
  instance?: string;
  md5?: string;
  message?: string;
  operator?: string;
  status?: string;
  updatedAt?: string;
}

export interface CreatePublishRecordsData {
  /**
   * RecordIDs 新建的发布记录 ID 列表
   */
  recordIDs?: string[];
}

export interface PublishInstanceResult {
  /**
   * Instance 目标实例（pod）名称
   */
  instance: string;
  /**
   * Message 失败原因等附加信息
   */
  message?: string;
  /**
   * Status 发布状态：success / failed
   */
  status: "success" | "failed";
}

export interface PreflightData {
  /**
   * Address 已组装好的集群完整地址；如 {baseUrl}/clusters/{clusterID}/
   */
  address?: string;
  /**
   * DevMode 开发模式相关路径配置
   */
  devMode?: PreflightDevMode;
  /**
   * InstanceIDs 校验通过的实例 ID 列表
   */
  instanceIDs?: string[];
  /**
   * Namespace 目标命名空间
   */
  namespace?: string;
  /**
   * Token 用户 Token，用于访问集群 API
   */
  token?: string;
}

export interface PreflightDevMode {
  /**
   * MountPath 脚本挂载路径
   */
  mountPath?: string;
  /**
   * WorkPath 开发模式根目录
   */
  workPath?: string;
}
