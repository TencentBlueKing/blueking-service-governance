/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：helm-charts

export interface ListAppHelmChartsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 搜索关键字，按版本号模糊匹配
   */
  keyword?: string;
  /**
   * 分页页码（从 1 开始）
   */
  page: number;
  /**
   * 分页大小，可选值：5/10/20/50/100
   */
  pageSize: number;
}

export interface ListHelmChartBuildRecordsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 搜索关键字，按版本号 / 构建号 / 操作人模糊匹配
   */
  keyword?: string;
  /**
   * 分页页码（从 1 开始）
   */
  page: number;
  /**
   * 分页大小，可选值：5/10/20/50/100
   */
  pageSize: number;
}

export type CreateHelmChartBuildRequest = CreateHelmChartBuildInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface DownloadHelmChartBuildLogsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 蓝盾构建 ID
   */
  buildID: string;
}

export interface StreamHelmChartBuildLogsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 蓝盾构建 ID
   */
  buildID: string;
}

export interface GetHelmChartSemverRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * semver 递增段类型，可选值：patch/minor/major
   */
  bumpType?: string;
}

export interface ListChartVersionsRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export interface GetHelmChartFilesRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * Chart 版本号
   */
  chartVersion: string;
}

export interface GetValuesFileRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * Chart 版本号
   */
  chartVersion: string;
}

export interface ListAppHelmChartsOutput {
  /**
   * 分页 Helm Chart 制品列表
   */
  data?: PaginatedAppHelmChartsOutputObjs;
}

export interface ListHelmChartBuildRecordsOutput {
  /**
   * 分页 Helm Chart 构建记录列表
   */
  data?: PaginatedHelmChartBuildRecordOutputObjs;
}

export interface CreateHelmChartBuildInput {
  /**
   * 兼容字段：按分支检出，仅在 revision 为空且 revisionType 为空或 branch 时生效
   */
  branch?: string;
  /**
   * semver 递增段类型（默认 patch）
   * 经典归零语义：递增 major 时 minor+patch 归零，递增 minor 时 patch 归零
   */
  bumpType: "patch" | "minor" | "major";
  /**
   * 代码修订：分支名、Tag 名或 CommitID（7–40 位十六进制），与 branch 至少填一个
   */
  revision?: string;
  /**
   * 代码来源类型（branch / tag / commit），缺省为 branch
   */
  revisionType?: "branch" | "tag" | "commit";
}

export interface CreateHelmChartBuildOutput {
  /**
   * 触发 Helm Chart 构建 - 输出对象
   */
  data?: CreateHelmChartBuildOutputObj;
}

export interface GetHelmChartSemverOutput {
  /**
   * 查询 Helm Chart semver counter 当前值 - 输出对象
   */
  data?: GetHelmChartSemverOutputObj;
}

export interface ListChartVersionsOutput {
  /**
   * Helm Chart 版本列表
   */
  data?: ChartVersionOutputObj[];
}

export interface GetHelmChartFilesOutput {
  /**
   * Chart 文件输出对象
   */
  data?: GetHelmChartFilesOutputObj;
}

export interface GetValuesFileOutput {
  /**
   * 应用配置文件内容
   */
  data?: string;
}

export interface GetHelmChartFilesOutputObj {
  /**
   * Chart 名称
   */
  chartName?: string;
  /**
   * Chart 版本号
   */
  chartVersion?: string;
  /**
   * Chart 根目录节点
   */
  root?: HelmChartFileNode;
}

export interface HelmChartFileNode {
  /**
   * 子节点（仅目录有效）
   */
  children?: HelmChartFileNode[];
  /**
   * 文件内容（仅文本文件且大小未超限时返回 UTF-8 文本）
   */
  content?: string;
  /**
   * 是否为二进制文件（true 时不返回 content）
   */
  isBinary?: boolean;
  /**
   * 是否为目录
   */
  isDir?: boolean;
  /**
   * 节点名称（不含父级路径）
   */
  name?: string;
  /**
   * 节点相对路径（相对于 chart 根目录）
   */
  path?: string;
  /**
   * 文件大小（字节，仅文件有效）
   */
  size?: string;
}

export interface ChartVersionOutputObj {
  /**
   * 版本创建时间
   */
  createdAt?: string;
  /**
   * Helm Chart 版本名
   */
  name?: string;
  /**
   * 版本更新时间
   */
  updatedAt?: string;
}

export interface GetHelmChartSemverOutputObj {
  /**
   * 当前最新的 semver 值
   */
  latest?: SemverOutputObj;
  /**
   * 按 bumpType 递增后的下一个 semver 值（仅当请求中 bumpType 非空时返回）
   */
  next?: SemverOutputObj;
}

export interface SemverOutputObj {
  /**
   * 主版本号
   */
  major?: string;
  /**
   * 次版本号
   */
  minor?: string;
  /**
   * 修订版本号
   */
  patch?: string;
  /**
   * 格式化版本字符串（格式：major.minor.patch）
   */
  version?: string;
}

export interface CreateHelmChartBuildOutputObj {
  /**
   * 蓝盾构建 ID（预留查询能力）
   */
  buildID?: string;
  /**
   * 本次构建的 Chart 版本号（semver 格式：major.minor.patch）
   */
  chartVersion?: string;
}

export interface PaginatedHelmChartBuildRecordOutputObjs {
  /**
   * 总记录数
   */
  count?: string;
  /**
   * 当前页构建记录列表
   */
  results?: HelmChartBuildRecordOutputObj[];
}

export interface HelmChartBuildRecordOutputObj {
  /**
   * 蓝盾构建 ID
   */
  buildID?: string;
  /**
   * 本次构建产出的 Chart 版本号
   */
  chartVersion?: string;
  /**
   * 构建结束时间
   */
  endedAt?: string;
  /**
   * 构建额外信息（包含 commit ID 等，由轮询任务回写）
   */
  extras?: Record<string, string>;
  /**
   * 构建序号（每个 AppID 独立自增）
   */
  num?: string;
  /**
   * 触发人
   */
  operator?: string;
  /**
   * 构建参数（包含代码库、分支等信息）
   */
  params?: Record<string, string>;
  /**
   * 蓝盾流水线 ID
   */
  pipelineID?: string;
  /**
   * 构建开始时间
   */
  startedAt?: string;
  /**
   * 构建状态
   */
  status?: string;
}

export interface PaginatedAppHelmChartsOutputObjs {
  /**
   * 总记录数（去重后的版本数）
   */
  count?: string;
  /**
   * 当前页 Chart 制品列表
   */
  results?: AppHelmChartOutputObj[];
}

export interface AppHelmChartOutputObj {
  /**
   * 版本号（semver）
   */
  chartVersion?: string;
  /**
   * 制品产生时间（来自 Helm Repo index entry 的 created 字段）
   */
  createdAt?: string;
  /**
   * 已部署到的环境列表
   */
  deployedEnvs?: DeployedEnvInfo[];
  /**
   * Chart 产物摘要（来自 Helm Repo index entry 的 digest 字段）
   */
  digest?: string;
}

export interface DeployedEnvInfo {
  /**
   * envName 环境名称
   */
  envName?: string;
  /**
   * envType 环境类型
   */
  envType?: string;
}
