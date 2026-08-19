/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：app

export interface GetAppIDAutoSuffixRequest {
}

export interface GetAppRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export interface DeleteAppRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export interface GetAppDeployOverviewRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export interface GetAppDeployStatusesRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export type UpdateAppDisplayNameRequest = UpdateAppDisplayNameInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export type UpdateHelmSpecRequest = UpdateHelmSpecInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export type UpdateAppTafSpecRequest = UpdateAppModelSpecInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export type UpdateAppTrpcSpecRequest = UpdateAppModelSpecInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface ListAppsRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 应用名过滤
   */
  appName?: string;
}

export type CreateAppRequest = CreateAppInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export interface GetAppIDAutoSuffixOutput {
  /**
   * 后缀字符串
   */
  suffix?: string;
}

export interface GetAppOutput {
  data?: AppDetailOutputObj;
}

export interface EmptyOutput {
}

export interface GetAppDeployOverviewOutput {
  /**
   * 已关联（AppIDs）环境行
   */
  data?: AppDeployOverviewEnvObj[];
}

export interface GetAppDeployStatusesOutput {
  /**
   * 应用部署的环境列表，包含标准环境和当前应用拥有的特性环境
   */
  data?: AppDeployedEnvOutputObj[];
}

export interface UpdateAppDisplayNameInput {
  /**
   * 待更新的应用显示名
   */
  displayName: string;
}

export interface UpdateHelmSpecInput {
  /**
   * 待更新的 Helm Chart 配置
   */
  helmSpec: HelmSpecInput;
}

export interface UpdateAppModelSpecInput {
  /**
   * 应用模型规范
   */
  appModelSpec: AppModelSpecInput;
}

export interface ListAppsOutput {
  data?: AppInfoOutputObj[];
}

export interface CreateAppInput {
  /**
   * 应用模型规范（type 为 trpc/taf 时需要）
   */
  appModelSpec?: AppModelSpecInput;
  /**
   * 构建配置
   */
  buildConfig: BuildConfigInput;
  /**
   * Helm 应用描述规范（type 为 helm/agones 时需要）
   */
  helmSpec?: HelmSpecInput;
  /**
   * 应用 ID. 应用在全局范围内的唯一标识，默认情况下基于 name 自动生成，在 name 后追加后缀。
   * 长度范围 1~63 个字符，以小写字母开头，可包含小写字母、数字、中划线，不能以中划线结尾
   */
  id: string;
  /**
   * 应用名称. 长度范围 1~63 个字符，以小写字母开头，可包含小写字母、数字、中划线，不能以中划线结尾
   */
  name: string;
  /**
   * 应用类型
   */
  type: "trpc" | "taf" | "helm" | "agones";
}

export interface CreateAppOutput {
  data?: AppOutputObj;
}

export interface AppOutputObj {
  /**
   * 应用显示名称
   */
  displayName?: string;
  /**
   * 应用 ID
   */
  id?: string;
  /**
   * 应用名称
   */
  name?: string;
  /**
   * 应用类型
   */
  type?: string;
  /**
   * 工作空间 ID
   */
  workspaceID?: string;
}

export interface AppModelSpecInput {
  /**
   * 容器启动参数
   */
  args?: string[];
  /**
   * 容器启动命令
   */
  command?: string[];
  /**
   * 容器环境变量，仅创建应用时生效；更新 tRPC/TAF Spec 时为兼容旧客户端接收但忽略
   */
  envVars?: VariableInput[];
  /**
   * TAF 框架配置
   */
  tafSpec?: TafSpecInput;
  /**
   * tRPC 框架配置
   */
  trpcSpec?: TrpcSpecInput;
}

export interface BuildConfigInput {
  /**
   * 镜像仓库配置
   */
  imageBuildConfig?: ImageBuildConfigInput;
  /**
   * 流水线配置
   */
  pipelineBuildConfig?: PipelineBuildConfigInput;
  /**
   * 代码仓库配置
   */
  repoBuildConfig?: RepoBuildConfigInput;
  /**
   * 来源类型
   */
  sourceType: "imageRegistry" | "codeRepository" | "pipeline";
}

export interface HelmSpecInput {
  /**
   * Helm 源配置
   */
  helmSource: HelmSourceInput;
}

export interface HelmSourceInput {
  /**
   * BCS 仓库配置。如果 repoType 为 BCSRepo，bcsRepoConfig 需要有有效值
   */
  bcsRepoConfig?: BCSRepoConfigInput;
  /**
   * Git 仓库配置。如果 repoType 为 GitRepo，gitRepoConfig 需要有有效值
   */
  gitRepoConfig?: HelmGitRepoConfigInput;
  /**
   * Helm 仓库配置。如果 repoType 为 HelmRepo，helmRepoConfig 需要有有效值
   */
  helmRepoConfig?: HelmRepoConfigInput;
  /**
   * 仓库类型
   */
  repoType: "GitRepo" | "HelmRepo" | "BCSRepo";
  /**
   * Value 文件列表
   */
  valueFiles?: string[];
}

export interface BCSRepoConfigInput {
  /**
   * Chart 名称
   */
  chartName: string;
  /**
   * 项目 Code
   */
  projectCode: string;
  /**
   * 仓库名称
   */
  repoName: string;
}

export interface HelmGitRepoConfigInput {
  /**
   * 代码库别名
   */
  repoAlias: string;
  /**
   * 代码库地址
   */
  repoURL: string;
  /**
   * 代码库分支
   */
  revision: string;
  /**
   * Helm Chart 目录
   */
  sourceDir: string;
  /**
   * 代码库类型
   */
  type: "TGit" | "GitHub";
}

export interface HelmRepoConfigInput {
  /**
   * Chart 名称
   */
  chartName: string;
  /**
   * 密码（可选）
   */
  password?: string;
  /**
   * 仓库地址
   */
  repoURL: string;
  /**
   * 用户名（可选）
   */
  username?: string;
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

export interface RepoBuildConfigInput {
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

export interface VariableInput {
  /**
   * 变量描述
   */
  description?: string;
  /**
   * 是否敏感
   */
  isSensitive?: boolean;
  /**
   * 变量名
   */
  key: string;
  /**
   * 变量值
   */
  value?: string;
}

export interface TafSpecInput {
  /**
   * 配置文件内容
   */
  fileContent?: string;
  /**
   * 配置文件名
   */
  fileName: string;
  /**
   * 配置文件路径
   */
  filePath: string;
}

export interface TrpcSpecInput {
  /**
   * 配置文件内容
   */
  fileContent?: string;
  /**
   * 配置文件名
   */
  fileName: string;
  /**
   * 配置文件路径
   */
  filePath: string;
  /**
   * 编程语言
   */
  language: "go" | "cpp";
}

export interface AppInfoOutputObj {
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 创建人
   */
  creator?: string;
  /**
   * 应用部署的环境列表
   */
  deployedEnvs?: AppDeployedEnvOutputObj[];
  /**
   * 应用显示名称
   */
  displayName?: string;
  /**
   * 应用 ID
   */
  id?: string;
  /**
   * 应用使用的编程语言
   */
  language?: string;
  /**
   * 应用最近操作时间（当前调用者视角）
   */
  lastOperatedAt?: string;
  /**
   * 应用名称
   */
  name?: string;
  /**
   * 应用类型
   */
  type?: string;
  /**
   * 工作空间 ID
   */
  workspaceID?: string;
}

export interface AppDeployedEnvOutputObj {
  /**
   * 部署状态
   */
  deployStatus?: string;
  /**
   * 环境展示名称
   */
  displayName?: string;
  /**
   * 环境 ID
   */
  id?: string;
  /**
   * 部署的镜像 Tag
   */
  imageTag?: string;
  /**
   * 环境类别（standard / feature）
   */
  kind?: string;
  /**
   * 环境名称（英文标识）
   */
  name?: string;
  /**
   * 泳道名称，空字符串表示默认泳道
   */
  trafficLaneName?: string;
  /**
   * 环境类型（development / test / staging / production）
   */
  type?: string;
}

export interface AppDeployOverviewEnvObj {
  /**
   * 自动扩缩容配置摘要，可选：无 GPA 配置时为 null
   */
  autoscaling?: DeployOverviewAutoscalingObj;
  /**
   * 部署状态（原始枚举）
   */
  deployStatus?: string;
  /**
   * 环境展示名称
   */
  envDisplayName?: string;
  /**
   * 环境 ID
   */
  envID?: string;
  /**
   * 环境类别（standard / feature）
   */
  envKind?: string;
  /**
   * 环境名称（英文标识）
   */
  envName?: string;
  /**
   * 环境类型（development / test / staging / production）
   */
  envType?: string;
  /**
   * 实例数，可选：集群查询失败或缺少 workload 时为 null
   */
  instances?: DeployOverviewInstancesObj;
  /**
   * 最近一次部署开始时间，可选：无部署记录时不返回该字段
   */
  lastDeployStartedAt?: string;
  /**
   * 资源规格（集群 GameDeployment 主容器实际值）
   */
  resources?: DeployOverviewResourcesObj;
}

export interface DeployOverviewAutoscalingObj {
  /**
   * 利用率计算基准：true 以 limits 为基准，false 以 requests 为基准
   */
  computeByLimits?: boolean;
  /**
   * 是否启用
   */
  enabled?: boolean;
  /**
   * 最大副本数
   */
  maxReplicas?: number;
  /**
   * 指标模式扩缩容指标列表，仅配置定时扩缩容时为空数组
   */
  metrics?: DeployOverviewAutoscalingMetricObj[];
  /**
   * 最小副本数
   */
  minReplicas?: number;
  /**
   * 集群 GPA CR 运行状态，可选：未启用 / CR 缺失 / 查询失败时为 null
   */
  status?: DeployOverviewAutoscalingStatusObj;
}

export interface DeployOverviewInstancesObj {
  /**
   * 存在但未 Ready 的 Pod 数
   */
  abnormal?: number;
  /**
   * 期望副本数（workload spec.replicas）
   */
  expected?: number;
  /**
   * Ready Pod 数
   */
  running?: number;
}

export interface DeployOverviewResourcesObj {
  /**
   * CPU limits（Kubernetes quantity 字符串），可选：未配置时不返回该字段
   */
  cpuLimits?: string;
  /**
   * CPU requests，可选：未配置时不返回该字段
   */
  cpuRequests?: string;
  /**
   * Memory limits，可选：未配置时不返回该字段
   */
  memoryLimits?: string;
  /**
   * Memory requests，可选：未配置时不返回该字段
   */
  memoryRequests?: string;
}

export interface DeployOverviewAutoscalingMetricObj {
  /**
   * 平均使用率阈值（百分比）
   */
  averageUtilization?: number;
  /**
   * 指标资源类型：cpu / memory
   */
  resource?: string;
}

export interface DeployOverviewAutoscalingStatusObj {
  /**
   * 当前副本数
   */
  currentReplicas?: number;
  /**
   * 期望副本数
   */
  desiredReplicas?: number;
  /**
   * 上次扩缩容时间（RFC3339 字符串），可选：尚未发生扩缩容时为空字符串
   */
  lastScaleTime?: string;
  /**
   * Phase：Active / Paused / Limited / Failed / Initializing / Unknown
   */
  phase?: string;
  /**
   * 非 True condition 的汇总消息，可选：一切正常时为空字符串
   */
  statusMessage?: string;
}

export interface AppDetailOutputObj {
  /**
   * 应用模型规范
   */
  appModelSpec?: AppModelSpecOutputObj;
  /**
   * 构建配置
   */
  buildConfig?: BuildConfigOutputObj;
  /**
   * 创建人
   */
  creator?: string;
  /**
   * 应用显示名称
   */
  displayName?: string;
  /**
   * Helm 应用描述规范
   */
  helmSpec?: HelmSpecOutputObj;
  /**
   * 应用 ID
   */
  id?: string;
  /**
   * 应用名称
   */
  name?: string;
  /**
   * 应用类型
   */
  type?: string;
  /**
   * 工作空间 ID
   */
  workspaceID?: string;
}

export interface AppModelSpecOutputObj {
  /**
   * 容器启动参数
   */
  args?: string[];
  /**
   * 容器启动命令
   */
  command?: string[];
  /**
   * 应用组件
   */
  components?: ComponentOutputObj[];
  /**
   * 容器环境变量
   */
  envVars?: VariableOutputObj[];
  /**
   * TAF 框架配置
   */
  tafSpec?: TafSpecOutputObj;
  /**
   * tRPC 框架配置
   */
  trpcSpec?: TrpcSpecOutputObj;
}

export interface BuildConfigOutputObj {
  /**
   * 镜像仓库配置
   */
  imageBuildConfig?: ImageBuildConfigOutputObj;
  /**
   * 流水线配置
   */
  pipelineBuildConfig?: PipelineBuildConfigOutputObj;
  /**
   * 代码仓库配置
   */
  repoBuildConfig?: RepoBuildConfigOutputObj;
  /**
   * 来源类型
   */
  sourceType?: string;
  /**
   * 镜像 Tag 配置
   */
  tagConfig?: TagConfigOutputObj;
}

export interface HelmSpecOutputObj {
  /**
   * Helm 源配置
   */
  helmSource?: HelmSourceOutputObj;
}

export interface HelmSourceOutputObj {
  /**
   * BCS 仓库配置
   */
  bcsRepoConfig?: BCSRepoConfigOutputObj;
  /**
   * Git 仓库配置
   */
  gitRepoConfig?: HelmGitRepoConfigOutputObj;
  /**
   * Helm 仓库配置
   */
  helmRepoConfig?: HelmRepoConfigOutputObj;
  /**
   * 仓库类型
   */
  repoType?: string;
  /**
   * Value 文件列表
   */
  valueFiles?: string[];
}

export interface BCSRepoConfigOutputObj {
  /**
   * Chart 名称
   */
  chartName?: string;
  /**
   * 项目 Code
   */
  projectCode?: string;
  /**
   * 仓库名称
   */
  repoName?: string;
}

export interface HelmGitRepoConfigOutputObj {
  /**
   * 代码库别名
   */
  repoAlias?: string;
  /**
   * 代码库地址
   */
  repoURL?: string;
  /**
   * 代码库分支
   */
  revision?: string;
  /**
   * Helm Chart 目录
   */
  sourceDir?: string;
  /**
   * 代码库类型
   */
  type?: string;
}

export interface HelmRepoConfigOutputObj {
  /**
   * Chart 名称
   */
  chartName?: string;
  /**
   * 仓库地址
   */
  repoURL?: string;
  /**
   * 用户名
   */
  username?: string;
}

export interface ImageBuildConfigOutputObj {
  /**
   * 镜像名称
   */
  name?: string;
  /**
   * 用户名
   */
  username?: string;
}

export interface PipelineBuildConfigOutputObj {
  /**
   * 构建流水线参数
   */
  params?: Record<string, string>;
  /**
   * 流水线 ID
   */
  pipelineID?: string;
}

export interface RepoBuildConfigOutputObj {
  /**
   * 默认分支
   */
  defaultBranch?: string;
  /**
   * DockerBuildArgs Docker 构建参数
   */
  dockerBuildArgs?: Record<string, string>;
  /**
   * Dockerfile 文件路径，仅 imageBuildMode=repositoryDockerfile 时有效
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
   * 代码库别名
   */
  repoAlias?: string;
  /**
   * 代码库地址
   */
  repoURL?: string;
  /**
   * 源码目录
   */
  sourceDir?: string;
  /**
   * 代码库类型
   */
  type?: string;
}

export interface TagConfigOutputObj {
  /**
   * 自定义 Tag 选项
   */
  customOpts?: CustomTagOptsOutputObj;
  /**
   * 版本号类型
   */
  type?: string;
}

export interface CustomTagOptsOutputObj {
  /**
   * 自定义前缀
   */
  prefix?: string;
  /**
   * 是否包含构建时间
   */
  withBuildTime?: boolean;
  /**
   * 是否包含分支/Tag 名称
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

export interface ComponentOutputObj {
  /**
   * 组件名称
   */
  name?: string;
  /**
   * 组件属性
   */
  properties?: Record<string, unknown>;
  /**
   * 引用的空间组件名
   */
  refWorkspaceCompName?: string;
  /**
   * 组件生效的环境列表
   */
  scopeEnvNames?: string[];
  /**
   * 组件生效范围类型
   */
  scopeType?: string;
  /**
   * 组件类型
   */
  type?: string;
  /**
   * 组件版本
   */
  version?: string;
}

export interface VariableOutputObj {
  /**
   * 变量描述
   */
  description?: string;
  /**
   * 是否敏感
   */
  isSensitive?: boolean;
  /**
   * 变量名
   */
  key?: string;
  /**
   * 变量值
   */
  value?: string;
}

export interface TafSpecOutputObj {
  /**
   * 配置文件内容
   */
  fileContent?: string;
  /**
   * 配置文件名
   */
  fileName?: string;
  /**
   * 配置文件路径
   */
  filePath?: string;
}

export interface TrpcSpecOutputObj {
  /**
   * 配置文件内容
   */
  fileContent?: string;
  /**
   * 配置文件名
   */
  fileName?: string;
  /**
   * 配置文件路径
   */
  filePath?: string;
  /**
   * 编程语言
   */
  language?: string;
}
