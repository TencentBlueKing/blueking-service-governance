/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：deploy

export interface ListHelmDeployRecordsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
  /**
   * 部署的泳道名称（空字符串表示不使用泳道）
   */
  trafficLaneName?: string;
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
}

export type CreateHelmDeployRequest = CreateHelmDeployInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
};

export interface PreviewHelmDeployRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
  /**
   * 目标镜像 TAG
   */
  imageTag: string;
  /**
   * 指定的部署的 Chart 版本
   */
  chartVersion: string;
  /**
   * 部署使用的 ValuesFile ID
   */
  valuesFileID: string;
  /**
   * 部署的泳道名称（空字符串表示不使用泳道）
   */
  trafficLaneName?: string;
}

export type RollbackHelmDeployRequest = RollbackHelmDeployInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
  /**
   * 部署记录 ID
   */
  deployID: string;
};

export interface DeleteHelmDeployRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
  /**
   * 部署记录 ID
   */
  deployID: string;
  /**
   * 部署的泳道名称（空字符串表示不使用泳道）
   */
  trafficLaneName?: string;
}

export interface PreviewRollbackHelmDeployRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
  /**
   * 准备回滚到的记录 ID
   */
  deployID: string;
  /**
   * 部署的泳道名称（空字符串表示不使用泳道）
   */
  trafficLaneName?: string;
}

export interface ListTafDeployRecordsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
  /**
   * 部署的泳道名称（空字符串表示不使用泳道）
   */
  trafficLaneName?: string;
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
}

export type CreateTafDeployRequest = CreateAppModelDeployInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
};

export interface DeleteTafDeployRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
  /**
   * 部署的泳道名称（空字符串表示不使用泳道）
   */
  trafficLaneName?: string;
}

export interface PreCheckTafDeployEnvVarsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
}

export interface GetLatestTafDeployStatusRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
  /**
   * 部署的泳道名称（空字符串表示不使用泳道）
   */
  trafficLaneName?: string;
}

export interface ListTafResourceSnapshotsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
  /**
   * 部署记录 ID
   */
  deployID: string;
  /**
   * 分页页码（从 1 开始）
   */
  page: number;
  /**
   * 分页大小
   */
  pageSize: number;
}

export interface GetTafResourceSnapshotRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
  /**
   * 部署记录 ID
   */
  deployID: string;
  /**
   * 资源清单快照 ID
   */
  snapshotID: string;
}

export interface ListTrpcDeployRecordsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
  /**
   * 部署的泳道名称（空字符串表示不使用泳道）
   */
  trafficLaneName?: string;
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
}

export type CreateTrpcDeployRequest = CreateAppModelDeployInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
};

export interface DeleteTrpcDeployRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
  /**
   * 部署的泳道名称（空字符串表示不使用泳道）
   */
  trafficLaneName?: string;
}

export interface PreCheckTrpcDeployEnvVarsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
}

export interface GetLatestTrpcDeployStatusRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
  /**
   * 部署的泳道名称（空字符串表示不使用泳道）
   */
  trafficLaneName?: string;
}

export interface ListTrpcResourceSnapshotsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
  /**
   * 部署记录 ID
   */
  deployID: string;
  /**
   * 分页页码（从 1 开始）
   */
  page: number;
  /**
   * 分页大小
   */
  pageSize: number;
}

export interface GetTrpcResourceSnapshotRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
  /**
   * 部署记录 ID
   */
  deployID: string;
  /**
   * 资源清单快照 ID
   */
  snapshotID: string;
}

export interface ListHelmDeployRecordsOutput {
  /**
   * 分页 Helm 部署记录列表
   */
  data?: PaginatedHelmDeployRecordOutputObjs;
}

export interface CreateHelmDeployInput {
  /**
   * 指定的部署的 Chart 版本，目前要求必须提供版本（前端获取最新版本并提交）
   */
  chartVersion: string;
  /**
   * 目标镜像 TAG
   */
  imageTag: string;
  /**
   * 部署的泳道名称（空字符串表示不使用泳道）
   */
  trafficLaneName?: string;
  /**
   * 部署使用的 ValuesFile ID
   */
  valuesFileID: string;
}

export interface PreviewHelmDeployOutput {
  /**
   * 目前部署的 manifest
   */
  current?: string;
  /**
   * MissingEnvVars values 中引用但未定义的 env 命名空间变量
   */
  missingEnvVars?: string[];
  /**
   * MissingVars values 中引用但未定义的非 env 命名空间变量（以 "ns.var" 形式，如 bkms.BAR）
   */
  missingVars?: string[];
  /**
   * 部署或回滚操作下发的 manifest
   */
  target?: string;
}

export interface RollbackHelmDeployInput {
  /**
   * 部署的泳道名称（空字符串表示不使用泳道）
   */
  trafficLaneName?: string;
}

export interface EmptyOutput {
}

export interface ListAppModelDeployRecordsOutput {
  data?: PaginatedAppModelDeployRecordsOutputObjs;
}

export interface CreateAppModelDeployInput {
  /**
   * 部署的镜像版本
   */
  imageTag: string;
  /**
   * 副本数量
   */
  replicas: number;
  /**
   * 部署的泳道名称（空字符串表示不使用泳道）
   */
  trafficLaneName?: string;
}

export interface EnvVarPreCheckOutput {
  undefinedVars?: UndefinedEnvVarOutput[];
}

export interface GetLatestAppModelDeployStatusOutput {
  data?: LatestDeployStatus;
}

export interface ListAppModelResourceSnapshotsOutput {
  data?: PaginatedAppModelResourceSnapshotsOutputObjs;
}

export interface GetAppModelResourceSnapshotOutput {
  snapshot?: AppModelResourceSnapshot;
}

export interface AppModelResourceSnapshot {
  apiVersion?: string;
  createdAt?: string;
  id?: string;
  isTruncated?: boolean;
  kind?: string;
  manifest?: string;
  name?: string;
}

export interface PaginatedAppModelResourceSnapshotsOutputObjs {
  count?: string;
  results?: AppModelResourceSnapshot[];
}

export interface LatestDeployStatus {
  branch?: string;
  buildID?: string;
  deployID?: string;
  deploySource?: string;
  endedAt?: string;
  hasDeployRecord?: boolean;
  imageTag?: string;
  /**
   * 兼容字段：供前端迁移期使用，待前端全部改用 deploySource 后删除。
   */
  isBuildAutoDeploy?: boolean;
  message?: string;
  operator?: string;
  pipelineID?: string;
  stage?: string;
  startedAt?: string;
  status?: string;
}

export interface UndefinedEnvVarOutput {
  key?: string;
  sources?: EnvVarReferenceSourceOutput[];
}

export interface EnvVarReferenceSourceOutput {
  name?: string;
  type?: string;
}

export interface PaginatedAppModelDeployRecordsOutputObjs {
  count?: string;
  results?: AppModelDeployRecordOutputObj[];
}

export interface AppModelDeployRecordOutputObj {
  /**
   * 代码分支（仅构建+部署记录返回）
   */
  branch?: string;
  clusterID?: string;
  /**
   * Commit ID（仅构建+部署记录返回）
   */
  commitID?: string;
  createdAt?: string;
  /**
   * 部署来源
   */
  deploySource?: string;
  id?: string;
  imageTag?: string;
  /**
   * 兼容字段：供前端迁移期使用，待前端全部改用 deploySource 后删除。
   */
  isBuildAutoDeploy?: boolean;
  message?: string;
  namespace?: string;
  operator?: string;
  replicas?: number;
  status?: string;
  updatedAt?: string;
  /**
   * 本次部署的主工作负载类型（GameDeployment 或 Deployment）
   */
  workloadKind?: string;
}

export interface PaginatedHelmDeployRecordOutputObjs {
  /**
   * 总记录数
   */
  count?: string;
  /**
   * 当前页 Helm 部署记录列表
   */
  results?: HelmDeployRecordOutputObj[];
}

export interface HelmDeployRecordOutputObj {
  /**
   * Chart 名称
   */
  chartName?: string;
  /**
   * Chart 版本
   */
  chartVersion?: string;
  /**
   * 集群 ID
   */
  clusterID?: string;
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 部署环境名称
   */
  envName?: string;
  /**
   * 部署记录 ID
   */
  id?: string;
  /**
   * 镜像 TAG
   */
  imageTag?: string;
  /**
   * 部署消息
   */
  message?: string;
  /**
   * 命名空间
   */
  namespace?: string;
  /**
   * 操作人
   */
  operator?: string;
  /**
   * 蓝盾项目 ID
   */
  projectCode?: string;
  /**
   * Helm Release 名称
   */
  releaseName?: string;
  /**
   * Helm Revision
   */
  revision?: string;
  /**
   * 部署状态
   */
  status?: string;
  /**
   * 更新时间
   */
  updatedAt?: string;
  /**
   * Helm Release Values；受 Helm 历史存储限制，历史久远的数据可能为空
   */
  values?: string;
  /**
   * 部署使用的 ValuesFile ID
   */
  valuesFileID?: string;
}
