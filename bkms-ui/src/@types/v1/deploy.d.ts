/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：deploy

export interface GetBatchReleaseProgressRequest {
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

export interface PreCheckTafDeployRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
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

export interface PreCheckTrpcDeployRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 部署环境名称
   */
  envName: string;
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

export interface GetBatchReleaseProgressOutput {
  /**
   * 最新部署记录的分批发布进度；没有部署记录或最新记录不是分批发布时为 null
   */
  data?: BatchReleaseProgressOutputObj;
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
   * 发布策略 ID：指定时按该策略分批更新实例（仅支持非联邦集群），不指定时一次更新全部实例
   */
  releaseStrategyID?: string;
  /**
   * 副本数量
   */
  replicas: number;
  /**
   * 部署的泳道名称（空字符串表示不使用泳道）
   */
  trafficLaneName?: string;
}

export interface GetLatestAppModelDeployStatusOutput {
  data?: LatestDeployStatus;
}

export interface DeployPreCheckOutput {
  /**
   * 缺失的必选集群组件标识及展示名
   */
  missingRequiredClusterAddons?: ClusterAddonReferenceOutput[];
  undefinedVars?: UndefinedEnvVarOutput[];
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

export interface ClusterAddonReferenceOutput {
  displayName?: string;
  name?: string;
}

export interface UndefinedEnvVarOutput {
  key?: string;
  sources?: EnvVarReferenceSourceOutput[];
}

export interface EnvVarReferenceSourceOutput {
  name?: string;
  type?: string;
}

export interface LatestDeployStatus {
  /**
   * 兼容字段：仅 revisionType 为 branch 时返回
   */
  branch?: string;
  buildID?: string;
  deployID?: string;
  deploySource?: string;
  endedAt?: string;
  hasDeployRecord?: boolean;
  imageTag?: string;
  /**
   * 进行中的渐进式发布的发布方式：空表示没有，canary 表示分批发布进行中（通过分批发布进度接口查询进度）。
   * 按最新部署记录计算，与本接口返回的是构建记录还是部署记录无关
   */
  inProgressReleaseMode?: string;
  /**
   * 兼容字段：供前端迁移期使用，待前端全部改用 deploySource 后删除。
   */
  isBuildAutoDeploy?: boolean;
  message?: string;
  operator?: string;
  pipelineID?: string;
  /**
   * 本条记录的发布方式：all 一次更新全部 / canary 分批发布；构建记录固定为 all
   */
  releaseMode?: string;
  /**
   * 分支名、Tag 名或 CommitID
   */
  revision?: string;
  /**
   * 代码来源类型 branch / tag / commit；历史记录为 branch
   */
  revisionType?: string;
  stage?: string;
  startedAt?: string;
  status?: string;
}

export interface PaginatedAppModelDeployRecordsOutputObjs {
  count?: string;
  results?: AppModelDeployRecordOutputObj[];
}

export interface AppModelDeployRecordOutputObj {
  /**
   * 兼容字段：仅 revisionType 为 branch 时返回
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
  /**
   * 发布方式：all 一次更新全部 / canary 分批发布
   */
  releaseMode?: string;
  /**
   * 分批发布时发起时的策略名称（快照），一次更新全部时为空
   */
  releaseStrategyName?: string;
  replicas?: number;
  /**
   * 分支名、Tag 名或 CommitID（仅构建+部署记录返回）
   */
  revision?: string;
  /**
   * 代码来源类型 branch / tag / commit；历史记录为 branch
   */
  revisionType?: string;
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

export interface BatchReleaseProgressOutputObj {
  /**
   * 当前批次状态：pending / updating / timed-paused / manual-paused / completed
   */
  batchPhase?: string;
  /**
   * 下发时的批次（快照）
   */
  batches?: BatchOutputObj[];
  /**
   * 当前批次序号，从 0 开始，-1 表示回滚；尚未进入批次或全部批次已走完时为 null
   */
  currentBatchIndex?: number;
  /**
   * 部署记录 ID
   */
  deployID?: string;
  /**
   * Rollout 提示信息，批次出错时为出错原因
   */
  message?: string;
  /**
   * 新镜像标签
   */
  newImageTag?: string;
  /**
   * 旧镜像标签
   */
  oldImageTag?: string;
  /**
   * 当前批次暂停开始时间，未暂停时为 null
   */
  pauseStartTime?: string;
  /**
   * 总实例数
   */
  replicas?: number;
  /**
   * Rollout 原始 phase
   */
  rolloutPhase?: string;
  /**
   * 部署记录状态：deploying 分批发布中 / deployed 已部署 / rolled-back 已回滚 / failed 等
   */
  status?: string;
  /**
   * 发起时的策略 ID（快照，策略之后被修改或删除不影响）
   */
  strategyID?: string;
  /**
   * 发起时的策略名称（快照）
   */
  strategyName?: string;
  /**
   * 已更新为新版本的实例数
   */
  updatedReplicas?: number;
}

export interface BatchOutputObj {
  /**
   * 本批完成后的暂停方式：none / duration / permanent
   */
  pauseMode?: string;
  /**
   * 定时暂停时长（秒），pauseMode 不为 duration 时为 0
   */
  pauseSeconds?: number;
  /**
   * 本批"累计更新到"的实例数：正整数或百分比
   */
  target?: string;
}
