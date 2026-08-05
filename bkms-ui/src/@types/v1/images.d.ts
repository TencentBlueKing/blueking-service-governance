/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：images

export interface ListDeployableImageTagsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
  /**
   * 搜索关键字（按 TAG 名称模糊搜索）
   */
  keyword?: string;
  /**
   * 分页参数：页码，从 1 开始
   */
  page: number;
  /**
   * 分页参数：每页数量，支持 5/10/20/50/100
   */
  pageSize: number;
}

export interface ListAppImagesRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 搜索关键字
   */
  keyword?: string;
  /**
   * 分页参数：页码，从 1 开始
   */
  page: number;
  /**
   * 分页参数：每页数量，支持 5/10/20/50/100
   */
  pageSize: number;
}

export interface RefreshAppImagesRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export interface DeleteAppImageRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 镜像标签
   */
  tag: string;
}

export interface ListImageTagDeployRecordsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 镜像标签
   */
  tag: string;
  /**
   * 分页参数：页码，从 1 开始
   */
  page: number;
  /**
   * 分页参数：每页数量，支持 5/10/20/50/100
   */
  pageSize: number;
}

export interface PromoteAppImageRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 镜像标签
   */
  tag: string;
}

export interface ListAppImageUsagesRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 镜像标签
   */
  tag: string;
}

export interface ListPlatformBuildImagesRequest {
  /**
   * 镜像类型：builder / runner
   */
  type: string;
  /**
   * 搜索关键字
   */
  keyword?: string;
}

export interface ListPlatformBuildImageTagsRequest {
  /**
   * 平台通用构建镜像记录 ID
   */
  imageID: string;
  /**
   * 搜索关键字
   */
  keyword?: string;
  /**
   * 分页参数：页码，从 1 开始
   */
  page: number;
  /**
   * 分页参数：每页数量，支持 5/10/20/50/100
   */
  pageSize: number;
}

export interface ListDeployableImageTagsOutput {
  data?: PaginatedDeployableImageTagOutputObjs;
}

export interface ListAppImagesOutput {
  data?: PaginatedAppImagesOutputObjs;
}

export interface RefreshAppImagesOutput {
  data?: RefreshResultInfoOutputObj;
}

export interface ImageEmptyOutput {
}

export interface ListImageTagDeployRecordsOutput {
  data?: PaginatedImageTagDeployRecordOutputObjs;
}

export interface ListAppImageUsagesOutput {
  data?: ImageTagUsagesOutputObj;
}

export interface ListRuntimeImagesOutput {
  data?: RuntimeImagesOutputObjs;
}

export interface ListRuntimeImageTagsOutput {
  data?: PaginatedRuntimeImageTagOutputObjs;
}

export interface PaginatedRuntimeImageTagOutputObjs {
  /**
   * 满足条件的总记录数
   */
  count?: string;
  /**
   * 当前页的镜像 TAG 列表
   */
  results?: RuntimeImageTagOutputObj[];
  /**
   * 快照状态信息
   */
  snapshotStatus?: SnapshotStatusInfoOutputObj;
}

export interface RuntimeImageTagOutputObj {
  /**
   * 镜像构建时间
   */
  builtAt?: string;
  /**
   * 摘要
   */
  digest?: string;
  /**
   * 镜像大小
   */
  size?: string;
  /**
   * 镜像标签名
   */
  tag?: string;
}

export interface SnapshotStatusInfoOutputObj {
  /**
   * 最后成功详情同步时间
   */
  lastDetailSyncedAt?: string;
  /**
   * 最后失败信息（为空表示无失败）
   */
  lastError?: string;
  /**
   * 最后成功刷新时间
   */
  lastRefreshedAt?: string;
  /**
   * 当前刷新状态：idle / refreshing / detail_syncing
   */
  refreshStatus?: string;
  /**
   * 仓库实例唯一标识，便于排查问题
   */
  repoKey?: string;
}

export interface RuntimeImagesOutputObjs {
  results?: RuntimeImageOutputObj[];
}

export interface RuntimeImageOutputObj {
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 描述
   */
  description?: string;
  /**
   * 记录 ID
   */
  id?: string;
  /**
   * 镜像仓库名称，不包含 tag
   */
  name?: string;
  /**
   * 镜像类型
   */
  type?: string;
  /**
   * 更新时间
   */
  updatedAt?: string;
}

export interface ImageTagUsagesOutputObj {
  /**
   * 当前镜像 tag 是否仍可能被使用
   */
  inUse?: boolean;
  /**
   * 命中的 env/lane/workload 列表
   */
  usages?: ImageTagUsageOutputObj[];
}

export interface ImageTagUsageOutputObj {
  /**
   * 环境名称
   */
  envName?: string;
  /**
   * 泳道名称，基线泳道为空字符串
   */
  laneName?: string;
  /**
   * 当前命中的部署记录原始状态值
   */
  status?: string;
  /**
   * 当前命中的部署记录对应的 workload 名称
   */
  workloadName?: string;
}

export interface PaginatedImageTagDeployRecordOutputObjs {
  count?: string;
  /**
   * 当前页部署记录结果
   */
  results?: ImageTagDeployRecordOutputObj[];
}

export interface ImageTagDeployRecordOutputObj {
  /**
   * 记录创建时间
   */
  createdAt?: string;
  /**
   * 环境名称
   */
  envName?: string;
  /**
   * 部署人
   */
  operator?: string;
  /**
   * 部署状态
   */
  status?: string;
}

export interface RefreshResultInfoOutputObj {
  /**
   * 本次新增标签数量
   */
  addedTagCnt?: string;
  /**
   * 提示信息
   */
  message?: string;
  /**
   * 本次删除标签数量
   */
  removedTagCnt?: string;
  /**
   * 刷新状态：success / refreshing（已有刷新在进行中）/ failed
   */
  status?: string;
}

export interface PaginatedAppImagesOutputObjs {
  count?: string;
  /**
   * 当前工作空间中所有生产类型环境的名称列表
   */
  productionEnvNames?: string[];
  /**
   * 当前页镜像结果
   */
  results?: AppImageOutputObj[];
  /**
   * 快照状态信息
   */
  snapshotStatus?: SnapshotStatusInfoOutputObj;
}

export interface AppImageOutputObj {
  /**
   * 镜像构建时间
   */
  builtAt?: string;
  /**
   * 已部署环境列表
   */
  deployedEnvs?: DeployedEnvInfoOutputObj[];
  /**
   * 摘要
   */
  digest?: string;
  /**
   * 是否已晋级
   */
  isPromoted?: boolean;
  /**
   * 晋级时间
   */
  promotedAt?: string;
  /**
   * 晋级操作人
   */
  promotedBy?: string;
  /**
   * 镜像仓库
   */
  repository?: string;
  /**
   * 镜像大小
   */
  size?: string;
  /**
   * 镜像 TAG
   */
  tag?: string;
}

export interface DeployedEnvInfoOutputObj {
  /**
   * 环境名称
   */
  envName?: string;
  /**
   * 环境类型（development/test/staging/production）
   */
  envType?: string;
}

export interface PaginatedDeployableImageTagOutputObjs {
  /**
   * 满足条件的总记录数
   */
  count?: string;
  /**
   * 当前页的镜像 TAG 列表
   */
  results?: DeployableImageTagOutputObj[];
}

export interface DeployableImageTagOutputObj {
  /**
   * 镜像构建时间
   */
  builtAt?: string;
  /**
   * 镜像标签名
   */
  tag?: string;
}
