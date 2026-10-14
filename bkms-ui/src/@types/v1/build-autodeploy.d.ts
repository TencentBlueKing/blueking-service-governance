/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：build-autodeploy

export type CreateTafBuildDeployRequest = CreateAppModelBuildDeployInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
};

export type CreateTrpcBuildDeployRequest = CreateAppModelBuildDeployInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
};

export interface CreateAppModelBuildDeployInput {
  /**
   * 兼容字段：按分支检出，仅在 revision 为空且 revisionType 为空或 branch 时生效
   */
  branch?: string;
  /**
   * 本次构建使用的镜像 Tag
   */
  imageTag: string;
  /**
   * 自动部署时的副本数
   */
  replicas: number;
  /**
   * 代码修订：分支名、Tag 名或 CommitID（7–40 位十六进制）
   */
  revision?: string;
  /**
   * 代码来源类型（branch / tag / commit），缺省为 branch
   */
  revisionType?: "branch" | "tag" | "commit";
  /**
   * 泳道名称，空字符串表示默认泳道
   */
  trafficLaneName?: string;
}

export interface CreateBuildOutput {
  /**
   * 构建记录详情
   */
  data?: BuildRecordOutputObj;
}

export interface BuildRecordOutputObj {
  /**
   * 产物地址
   */
  artifact?: string;
  /**
   * 蓝盾构建 ID
   */
  buildID?: string;
  /**
   * 提交哈希
   */
  commitID?: string;
  /**
   * 结束时间
   */
  endedAt?: string;
  /**
   * 额外元数据
   */
  extras?: Record<string, string>;
  /**
   * 构建序号
   */
  num?: string;
  /**
   * 操作人
   */
  operator?: string;
  /**
   * 构建参数
   */
  params?: Record<string, string>;
  /**
   * 蓝盾流水线 ID
   */
  pipelineID?: string;
  /**
   * 代码仓库地址
   */
  repoURL?: string;
  /**
   * 代码版本
   */
  revision?: string;
  /**
   * 代码来源类型（branch / tag / commit），与构建记录列表同一推导口径
   */
  revisionType?: string;
  /**
   * 开始时间
   */
  startedAt?: string;
  /**
   * 构建状态
   */
  status?: string;
}
