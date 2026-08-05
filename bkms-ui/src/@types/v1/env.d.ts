/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：env

export interface ListAppEnvsRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export interface ListFeatureEnvsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 是否返回当前应用在每个特性环境下的部署状态
   */
  with_deploy_status?: boolean;
}

export type CreateFeatureEnvRequest = CreateFeatureEnvInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface GetEnvRequest {
  /**
   * 环境 ID
   */
  envID: string;
}

export interface DeleteEnvRequest {
  /**
   * 环境 ID
   */
  envID: string;
}

export type UpdateEnvBasicInfoRequest = UpdateEnvBasicInfoInput & {
  /**
   * 环境 ID
   */
  envID: string;
};

export type UpdateEnvClusterRequest = UpdateEnvClusterInput & {
  /**
   * 环境 ID
   */
  envID: string;
};

export interface ListEnvsRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export type CreateEnvRequest = CreateEnvInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export interface ListEnvTrafficLanesRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface ListEnvsOutput {
  data?: EnvOutput[];
}

export interface ListFeatureEnvsOutput {
  data?: FeatureEnvOutput[];
}

export interface CreateFeatureEnvInput {
  /**
   * 特性环境展示名称
   */
  displayName: string;
  /**
   * 来源标准环境 ID
   */
  sourceEnvID: string;
}

export interface CreateFeatureEnvOutput {
  data?: EnvOutput;
}

export interface GetEnvOutput {
  data?: EnvDetailOutput;
}

export interface EmptyOutput {
}

export interface UpdateEnvBasicInfoInput {
  /**
   * 环境显示名称
   */
  displayName?: string;
  /**
   * 环境类型, 可选值 development、test、staging 或 production
   */
  type?: string;
}

export interface UpdateEnvClusterInput {
  /**
   * 集群 ID
   */
  clusterID: string;
  /**
   * 集群类型
   */
  clusterType: string;
  /**
   * 集群命名空间
   */
  namespace: string;
}

export interface CreateEnvInput {
  /**
   * 绑定的 APM ID（可选，为空则创建同名 APM，不为空则使用共享 APM）
   */
  apmID?: number;
  /**
   * 环境关联的业务集群信息
   */
  cluster: CreateEnvClusterInput;
  /**
   * 环境描述
   */
  description?: string;
  /**
   * 环境显示名称
   */
  displayName: string;
  /**
   * 环境名称
   */
  name: string;
  /**
   * 环境类型, 可选值 development、test、staging 或 production
   */
  type: string;
}

export interface CreateEnvOutput {
  data?: EnvIDOutput;
}

export interface ListEnvTrafficLanesOutput {
  data?: TrafficLaneOutput[];
}

export interface TrafficLaneOutput {
  /**
   * 注解是泳道扩展字段, 针对不同产品通过不同的 k-v 扩展
   */
  annotations?: Record<string, string>;
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 泳道描述
   */
  description?: string;
  /**
   * 泳道 ID
   */
  id?: string;
  /**
   * 标签中包含泳道所属的微服务信息，类似于泳道组的概念
   */
  labels?: Record<string, string>;
  /**
   * 泳道名称
   */
  name?: string;
  /**
   * 泳道服务版本标签, 平台注入或者用户自定义
   */
  serviceVersionLabels?: Record<string, string>;
  /**
   * 泳道类型
   */
  type?: string;
  /**
   * 更新时间
   */
  updatedAt?: string;
}

export interface EnvIDOutput {
  /**
   * 环境 ID
   */
  id?: string;
}

export interface CreateEnvClusterInput {
  /**
   * 集群 ID
   */
  clusterID: string;
  /**
   * 集群类型
   */
  clusterType: string;
  /**
   * 集群命名空间
   */
  namespace: string;
}

export interface EnvDetailOutput {
  /**
   * 当前环境已部署应用及其部署状态
   */
  appDeployStatuses?: EnvAppDeployStatusOutput[];
  /**
   * 业务集群信息
   */
  cluster?: EnvClusterOutput;
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 创建者
   */
  creator?: string;
  /**
   * 环境描述
   */
  description?: string;
  /**
   * 环境显示名称
   */
  displayName?: string;
  /**
   * 环境 ID
   */
  id?: string;
  /**
   * 环境名称
   */
  name?: string;
  /**
   * 环境状态, 取值: Ready(就绪), NotReady(未就绪)
   */
  status?: string;
  /**
   * 环境类型
   */
  type?: string;
  /**
   * 更新时间
   */
  updatedAt?: string;
}

export interface EnvAppDeployStatusOutput {
  /**
   * 应用 ID
   */
  appID?: string;
  /**
   * 应用名称
   */
  appName?: string;
  /**
   * 应用类型
   */
  appType?: string;
  /**
   * 部署状态
   */
  deployStatus?: string;
  /**
   * 部署的镜像 Tag
   */
  imageTag?: string;
  /**
   * 泳道名称
   */
  trafficLaneName?: string;
}

export interface EnvClusterOutput {
  /**
   * 集群 ID
   */
  clusterID?: string;
  /**
   * 集群类型
   */
  clusterType?: string;
  /**
   * 集群命名空间
   */
  namespace?: string;
  /**
   * 项目 code
   */
  projectCode?: string;
}

export interface EnvOutput {
  /**
   * 已部署的应用 ID 列表
   */
  appIDs?: string[];
  /**
   * 业务集群信息
   */
  cluster?: EnvClusterOutput;
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 环境显示名称
   */
  displayName?: string;
  /**
   * 环境 ID
   */
  id?: string;
  /**
   * 环境类别，standard 或 feature
   */
  kind?: string;
  /**
   * 环境名称
   */
  name?: string;
  /**
   * 特性环境所属应用 ID，仅特性环境返回
   */
  ownerAppID?: string;
  /**
   * 特性环境来源环境 ID，仅特性环境返回
   */
  sourceEnvID?: string;
  /**
   * 环境状态, 取值: Ready(就绪), NotReady(未就绪)
   */
  status?: string;
  /**
   * 环境类型
   */
  type?: string;
  /**
   * 更新时间
   */
  updatedAt?: string;
}

export interface FeatureEnvOutput {
  /**
   * 部署位置
   */
  cluster?: FeatureEnvClusterOutput;
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 创建人
   */
  creator?: string;
  /**
   * 当前应用在该特性环境下各泳道的部署状态；未显式请求时为 null，显式请求时为空数组或状态列表
   */
  deployStatuses?: FeatureEnvDeployStatusOutput[];
  /**
   * 特性环境展示名称
   */
  displayName?: string;
  /**
   * 特性环境 ID
   */
  id?: string;
  /**
   * 特性环境内部名称
   */
  name?: string;
  /**
   * 来源标准环境
   */
  sourceEnv?: FeatureEnvSourceOutput;
  /**
   * 环境状态，取值 Ready（就绪）或 NotReady（未就绪）
   */
  status?: string;
  /**
   * 环境类型，可选值 development、test、staging 或 production
   */
  type?: string;
}

export interface FeatureEnvClusterOutput {
  /**
   * 集群 ID
   */
  clusterID?: string;
  /**
   * 特性环境独占的命名空间
   */
  namespace?: string;
}

export interface FeatureEnvDeployStatusOutput {
  /**
   * 部署状态
   */
  deployStatus?: string;
  /**
   * 部署的镜像 Tag
   */
  imageTag?: string;
  /**
   * 泳道名称
   */
  trafficLaneName?: string;
}

export interface FeatureEnvSourceOutput {
  /**
   * 来源环境展示名称，来源环境已删除时为空
   */
  displayName?: string;
  /**
   * 来源环境 ID
   */
  id?: string;
  /**
   * 来源环境是否已删除
   */
  isDeleted?: boolean;
  /**
   * 来源环境名称，来源环境已删除时为空
   */
  name?: string;
}
