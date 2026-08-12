/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：builds

export type UpdateBuildConfigRequest = UpdateBuildConfigInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface ListBuildRecordsRequest {
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

export type CreateBuildRequest = CreateBuildInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface DownloadBuildLogsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 蓝盾构建 ID
   */
  buildID: string;
}

export interface StreamBuildLogsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 蓝盾构建 ID
   */
  buildID: string;
}

export interface GetRecommendedImageTagRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 分支/Tag（仅 custom 类型使用）
   */
  branch?: string;
}

export interface UpdateBuildConfigInput {
  /**
   * 代码仓库构建配置
   */
  codeRepo?: RepositoryBuildConfigInput;
  /**
   * 镜像仓库配置
   */
  image?: ImageBuildConfigInput;
  /**
   * 蓝盾流水线配置
   */
  pipeline?: PipelineBuildConfigInput;
  /**
   * 构建来源：codeRepository / imageRegistry / pipeline
   */
  sourceType: "codeRepository" | "imageRegistry" | "pipeline";
  /**
   * 镜像 Tag 配置
   */
  tagConfig?: TagConfigInput;
}

export interface UpdateBuildConfigOutput {
  /**
   * 构建配置详情
   */
  data?: BuildConfigOutputObj;
}

export interface ListBuildRecordsOutput {
  /**
   * 构建记录分页结果
   */
  data?: PaginatedBuildRecordOutputObjs;
}

export interface CreateBuildInput {
  /**
   * 构建分支或标签
   */
  branch?: string;
  /**
   * 本次构建使用的镜像 Tag
   */
  imageTag?: string;
}

export interface CreateBuildOutput {
  /**
   * 新创建的构建记录
   */
  data?: BuildRecordOutputObj;
}

export interface GetRecommendedImageTagOutput {
  /**
   * 推荐的镜像 Tag
   */
  data?: string;
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
   * 开始时间
   */
  startedAt?: string;
  /**
   * 构建状态
   */
  status?: string;
  /**
   * 自动触发时关联的触发策略 ID，手动触发为空
   */
  triggerPolicyID?: string;
  /**
   * 触发方式：manual 手动，auto 自动
   */
  triggerType?: string;
}

export interface PaginatedBuildRecordOutputObjs {
  /**
   * 记录总数
   */
  count?: string;
  /**
   * 当前页结果
   */
  results?: BuildRecordOutputObj[];
}

export interface BuildConfigOutputObj {
  /**
   * 应用 ID
   */
  appID?: string;
  /**
   * 代码仓库构建配置
   */
  codeRepo?: RepositoryBuildConfigOutputObj;
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 镜像仓库配置
   */
  image?: ImageBuildConfigOutputObj;
  /**
   * 实际执行的流水线类型
   */
  pipelineType?: string;
  /**
   * 构建来源
   */
  sourceType?: string;
  /**
   * 镜像 Tag 配置
   */
  tagConfig?: TagConfigOutput;
  /**
   * 更新时间
   */
  updatedAt?: string;
}

export interface RepositoryBuildConfigOutputObj {
  /**
   * 默认构建分支
   */
  defaultBranch?: string;
  /**
   * DockerBuildArgs Docker 构建参数
   */
  dockerBuildArgs?: Record<string, string>;
  /**
   * Dockerfile 路径，仅 imageBuildMode=repositoryDockerfile 时有效
   */
  dockerfile?: string;
  /**
   * ImageBuildMode 镜像构建方式：repositoryDockerfile 表示使用仓库 Dockerfile，platform 表示平台通用构建
   */
  imageBuildMode?: string;
  /**
   * PlatformBuildConfig 平台通用构建配置，仅 imageBuildMode=platform 时返回
   */
  platformBuildConfig?: PlatformBuildConfigOutputObj;
  /**
   * 蓝盾侧仓库别名
   */
  repoAlias?: string;
  /**
   * 代码仓库地址
   */
  repoURL?: string;
  /**
   * 构建目录
   */
  sourceDir?: string;
  /**
   * 代码仓库类型
   */
  type?: string;
}

export interface ImageBuildConfigOutputObj {
  /**
   * 镜像名称
   */
  name?: string;
  /**
   * 镜像仓库用户名
   */
  username?: string;
}

export interface TagConfigOutput {
  /**
   * 自定义 Tag 配置
   */
  customOpts?: CustomTagOptsOutput;
  /**
   * Tag 生成策略
   */
  type?: string;
}

export interface CustomTagOptsOutput {
  /**
   * 自定义前缀
   */
  prefix?: string;
  /**
   * 是否拼接构建时间
   */
  withBuildTime?: boolean;
  /**
   * 是否拼接代码版本
   */
  withRevision?: boolean;
}

export interface PlatformBuildConfigOutputObj {
  /**
   * 构建阶段基础镜像
   */
  builderImage?: string;
  /**
   * 命令配置
   */
  commands?: BuildCommandsOutputObj;
  /**
   * 运行阶段基础镜像
   */
  runnerImage?: string;
}

export interface BuildCommandsOutputObj {
  /**
   * 编译命令列表
   */
  build?: string[];
  /**
   * 编译前置命令列表
   */
  preBuild?: string[];
  /**
   * 运行环境命令列表
   */
  runtimeEnv?: string[];
  /**
   * 启动命令
   */
  start?: string;
}

export interface RepositoryBuildConfigInput {
  /**
   * 默认构建分支
   */
  defaultBranch: string;
  /**
   * DockerBuildArgs Docker 构建参数
   */
  dockerBuildArgs?: Record<string, string>;
  /**
   * Dockerfile 路径，空表示默认路径。仅 imageBuildMode=repositoryDockerfile 时有效
   */
  dockerfile?: string;
  /**
   * ImageBuildMode 镜像构建方式：repositoryDockerfile 表示使用仓库 Dockerfile，platform 表示平台通用构建
   */
  imageBuildMode?: "platform" | "repositoryDockerfile";
  /**
   * PlatformBuildConfig 平台通用构建配置，仅 imageBuildMode=platform 时有效；Dockerfile 是流水线内部中间产物
   */
  platformBuildConfig?: PlatformBuildConfigInput;
  /**
   * 蓝盾侧仓库别名
   */
  repoAlias: string;
  /**
   * 代码仓库地址
   */
  repoURL: string;
  /**
   * 构建目录，空表示仓库根目录；非空时必须是仓库内相对路径，不能以 / 开头，也不能包含 .. 路径段
   */
  sourceDir?: string;
  /**
   * 代码仓库类型：TGit 或 GitHub
   */
  type: "TGit" | "GitHub";
}

export interface ImageBuildConfigInput {
  /**
   * 镜像名称，不包含 Tag
   */
  name: string;
  /**
   * 镜像仓库密码
   */
  password?: string;
  /**
   * 镜像仓库用户名
   */
  username?: string;
}

export interface PipelineBuildConfigInput {
  /**
   * 流水线额外参数
   */
  params?: Record<string, string>;
  /**
   * 蓝盾流水线 ID
   */
  pipelineID: string;
}

export interface TagConfigInput {
  /**
   * 自定义 Tag 配置，仅当 type=custom 时生效
   */
  customOpts?: CustomTagOptsInput;
  /**
   * Tag 生成策略：semver 或 custom
   */
  type: "semver" | "custom";
}

export interface CustomTagOptsInput {
  /**
   * 自定义前缀
   */
  prefix?: string;
  /**
   * 是否拼接构建时间
   */
  withBuildTime?: boolean;
  /**
   * 是否拼接代码版本
   */
  withRevision?: boolean;
}

export interface PlatformBuildConfigInput {
  /**
   * 构建阶段基础镜像
   */
  builderImage?: string;
  /**
   * 命令配置
   */
  commands?: BuildCommandsInput;
  /**
   * 运行阶段基础镜像
   */
  runnerImage?: string;
}

export interface BuildCommandsInput {
  /**
   * 编译命令列表
   */
  build?: string[];
  /**
   * 编译前置命令列表
   */
  preBuild?: string[];
  /**
   * 运行环境命令列表
   */
  runtimeEnv?: string[];
  /**
   * 启动命令
   */
  start?: string;
}
