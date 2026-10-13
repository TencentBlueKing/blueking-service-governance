/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：bscpcfg

export interface ListBscpCfgEnvBindingsRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export interface GetBscpCfgEnvBindingRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface CreateBscpCfgEnvBindingRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface DeleteBscpCfgEnvBindingRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetBscpCfgMetadataRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export interface InitBscpCfgMetadataRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export interface DeleteBscpCfgMetadataRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export type PatchBscpCfgMetadataRequest = PatchMetadataInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface EnvBindingListResponse {
  data?: EnvBindingOutput[];
}

export interface EnvBindingResponse {
  data?: EnvBindingOutput;
}

export interface MetadataResponse {
  data?: MetadataOutput;
}

export interface PatchMetadataInput {
  /**
   * MountPath 配置文件挂载路径（传入则更新）
   */
  mountPath?: string;
  /**
   * WorkloadKind 目标工作负载类型（传入则更新）
   */
  workloadKind?: string;
  /**
   * WorkloadName 指定被注入 bscp 配置的目标 workload 名称（传入则更新）
   */
  workloadName?: string;
}

export interface MetadataOutput {
  appID?: string;
  createdAt?: string;
  enable?: boolean;
  mountPath?: string;
  operator?: string;
  postHookID?: string;
  updatedAt?: string;
  workloadKind?: string;
  workloadName?: string;
}

export interface EnvBindingOutput {
  appID?: string;
  bscpAppID?: string;
  bscpEnvID?: string;
  bscpEnvName?: string;
  createdAt?: string;
  envName?: string;
  mountPath?: string;
  operator?: string;
  updatedAt?: string;
  workloadKind?: string;
  workloadName?: string;
}
