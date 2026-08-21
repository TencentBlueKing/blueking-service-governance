/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：app-spec

export interface GetAppDefaultAppSpecAnnotationsRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export type SetAppDefaultAppSpecAnnotationsRequest = AppSpecAnnotationsInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface GetAppDefaultAppSpecLabelsRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export type SetAppDefaultAppSpecLabelsRequest = AppSpecLabelsInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface GetAppDefaultAppSpecLifecycleRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export type SetAppDefaultAppSpecLifecycleRequest = SetAppDefaultAppSpecLifecycleInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface GetAppDefaultAppSpecProbeRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export type SetAppDefaultAppSpecProbeRequest = SetAppDefaultAppSpecProbeInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface GetAppDefaultAppSpecResourcesRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export type SetAppDefaultAppSpecResourcesRequest = SetAppDefaultAppSpecResourcesInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface GetAppDefaultAppSpecTkeRouteEniRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export type SetAppDefaultAppSpecTkeRouteEniRequest = AppSpecTkeRouteEniInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface GetAppDefaultAppSpecUpdateStrategyRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export type SetAppDefaultAppSpecUpdateStrategyRequest = SetAppDefaultAppSpecUpdateStrategyInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface GetAppSpecOverviewRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export interface GetEnvAppSpecAnnotationsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export type SetEnvAppSpecAnnotationsRequest = AppSpecAnnotationsInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
};

export interface DeleteEnvAppSpecAnnotationsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetEnvEffectiveAppSpecAnnotationsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetEnvAppSpecDevModeRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export type SetEnvAppSpecDevModeRequest = SetEnvAppSpecDevModeInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
};

export interface DeleteEnvAppSpecDevModeRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetEnvEffectiveAppSpecDevModeRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetEnvAppSpecLabelsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export type SetEnvAppSpecLabelsRequest = AppSpecLabelsInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
};

export interface DeleteEnvAppSpecLabelsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetEnvEffectiveAppSpecLabelsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetEnvAppSpecLifecycleRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export type SetEnvAppSpecLifecycleRequest = SetEnvAppSpecLifecycleInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
};

export interface DeleteEnvAppSpecLifecycleRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetEnvEffectiveAppSpecLifecycleRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetEnvAppSpecProbeRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export type SetEnvAppSpecProbeRequest = SetEnvAppSpecProbeInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
};

export interface DeleteEnvAppSpecProbeRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetEnvEffectiveAppSpecProbeRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface DeleteEnvAppSpecProbeByTypeRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
  /**
   * 探针类型，可选 liveness、readiness、startup
   */
  probeType: string;
}

export interface GetEnvAppSpecResourcesRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export type SetEnvAppSpecResourcesRequest = SetEnvAppSpecResourcesInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
};

export interface DeleteEnvAppSpecResourcesRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetEnvEffectiveAppSpecResourcesRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetEnvAppSpecTkeRouteEniRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export type SetEnvAppSpecTkeRouteEniRequest = AppSpecTkeRouteEniInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
};

export interface DeleteEnvAppSpecTkeRouteEniRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetEnvEffectiveAppSpecTkeRouteEniRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetEnvAppSpecUpdateStrategyRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export type SetEnvAppSpecUpdateStrategyRequest = SetEnvAppSpecUpdateStrategyInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
};

export interface DeleteEnvAppSpecUpdateStrategyRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface GetEnvEffectiveAppSpecUpdateStrategyRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface ListWorkspaceAppSpecDevModeRulesRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export type CreateWorkspaceAppSpecDevModeRuleRequest = DevModeRuleInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export type UpdateWorkspaceAppSpecDevModeRuleRequest = DevModeRuleInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 规则 ID
   */
  ruleID: string;
};

export interface DeleteWorkspaceAppSpecDevModeRuleRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 规则 ID
   */
  ruleID: string;
}

export interface ListWorkspaceAppSpecResourcesRulesRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export type CreateWorkspaceAppSpecResourcesRuleRequest = ResourcesRuleInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export type UpdateWorkspaceAppSpecResourcesRuleRequest = ResourcesRuleInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 规则 ID
   */
  ruleID: string;
};

export interface DeleteWorkspaceAppSpecResourcesRuleRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 规则 ID
   */
  ruleID: string;
}

export interface AppSpecAnnotationsSectionOutput {
  data?: AppSpecAnnotationsOutput;
}

export interface AppSpecAnnotationsInput {
  /**
   * 自定义注解，key 需为合法的 Kubernetes annotation key（qualified name），
   * value 无格式与长度限制。
   */
  annotations?: Record<string, string>;
}

export interface EmptyOutput {
}

export interface AppSpecLabelsSectionOutput {
  data?: AppSpecLabelsOutput;
}

export interface AppSpecLabelsInput {
  /**
   * 自定义标签，key 需为合法的 Kubernetes label key（qualified name），
   * value 需符合 Kubernetes label value 规范（≤63 字符，不允许特殊字符）。
   */
  labels?: Record<string, string>;
}

export interface AppSpecLifecycleSectionOutput {
  data?: AppSpecLifecycleOutput;
}

export interface SetAppDefaultAppSpecLifecycleInput {
  /**
   * 待设置的 lifecycle section 值
   */
  appSpecLifecycle: AppSpecLifecycleInput;
}

export interface AppSpecProbeSectionOutput {
  data?: AppSpecProbeOutput;
}

export interface SetAppDefaultAppSpecProbeInput {
  /**
   * 待设置的 probe section 值
   */
  appSpecProbe: AppSpecProbeInput;
}

export interface AppSpecResourcesSectionOutput {
  data?: AppSpecResourcesOutput;
}

export interface SetAppDefaultAppSpecResourcesInput {
  /**
   * 待设置的 resources section 值
   */
  appSpecResources: AppSpecResourcesInput;
}

export interface AppSpecTkeRouteEniSectionOutput {
  data?: AppSpecTkeRouteEniOutput;
}

export interface AppSpecTkeRouteEniInput {
  /**
   * 是否启用 TKE Route ENI (VPC-CNI) 网络模式
   */
  enabled?: boolean;
}

export interface AppSpecUpdateStrategySectionOutput {
  data?: AppSpecUpdateStrategyOutput;
}

export interface SetAppDefaultAppSpecUpdateStrategyInput {
  /**
   * 待设置的 updateStrategy section 值
   */
  appSpecUpdateStrategy: AppSpecUpdateStrategyInput;
}

export interface GetAppSpecOverviewOutput {
  data?: AppSpecOverviewOutput;
}

export interface AppSpecDevModeSectionOutput {
  data?: AppSpecDevModeOutput;
}

export interface SetEnvAppSpecDevModeInput {
  /**
   * 待设置的 devMode section 值
   */
  appSpecDevMode: AppSpecDevModeInput;
}

export interface SetEnvAppSpecLifecycleInput {
  /**
   * 待设置的 lifecycle section 值
   */
  appSpecLifecycle: EnvAppSpecLifecycleInput;
}

export interface SetEnvAppSpecProbeInput {
  /**
   * 待设置的 probe section 值
   */
  appSpecProbe: EnvAppSpecProbeInput;
}

export interface SetEnvAppSpecResourcesInput {
  /**
   * 待设置的 resources section 值
   */
  appSpecResources: EnvAppSpecResourcesInput;
}

export interface SetEnvAppSpecUpdateStrategyInput {
  /**
   * 待设置的 updateStrategy section 值
   */
  appSpecUpdateStrategy: EnvAppSpecUpdateStrategyInput;
}

export interface ListDevModeRulesOutput {
  data?: DevModeRuleOutputObj[];
}

export interface DevModeRuleInput {
  envTypes: string[];
  spec: DevModeSpecInput;
}

export interface DevModeRuleOutput {
  data?: DevModeRuleOutputObj;
}

export interface EmptyOutput {
}

export interface ListResourcesRulesOutput {
  data?: ResourcesRuleOutputObj[];
}

export interface ResourcesRuleInput {
  envTypes: string[];
  spec: ResourcesSpecInput;
}

export interface ResourcesRuleOutput {
  data?: ResourcesRuleOutputObj;
}

export interface ResourcesRuleOutputObj {
  createdAt?: string;
  envTypes?: string[];
  id?: string;
  spec?: AppSpecResourcesOutput;
  updatedAt?: string;
}

export interface AppSpecResourcesOutput {
  /**
   * CPU limits
   */
  cpuLimits?: string;
  /**
   * CPU requests
   */
  cpuRequests?: string;
  /**
   * Memory limits
   */
  memoryLimits?: string;
  /**
   * Memory requests
   */
  memoryRequests?: string;
  /**
   * 副本数量
   */
  replicas?: number;
}

export interface ResourcesSpecInput {
  cpuLimits: string;
  cpuRequests: string;
  memoryLimits: string;
  memoryRequests: string;
  replicas: number;
}

export interface DevModeRuleOutputObj {
  createdAt?: string;
  envTypes?: string[];
  id?: string;
  spec?: DevModeSpecOutput;
  updatedAt?: string;
}

export interface DevModeSpecOutput {
  enabled?: boolean;
}

export interface DevModeSpecInput {
  enabled: boolean;
}

export interface EnvAppSpecUpdateStrategyInput {
  /**
   * 滚动更新时最大可增加实例数，支持大于等于 0 的整数或百分比。
   */
  maxSurge?: string;
  /**
   * 滚动更新时最大不可用实例数，支持大于等于 0 的整数或百分比。
   */
  maxUnavailable?: string;
}

export interface EnvAppSpecResourcesInput {
  /**
   * CPU limits
   */
  cpuLimits?: string;
  /**
   * CPU requests
   */
  cpuRequests?: string;
  /**
   * Memory limits
   */
  memoryLimits?: string;
  /**
   * Memory requests
   */
  memoryRequests?: string;
  /**
   * 副本数量，必须为正整数。
   */
  replicas?: number;
}

export interface EnvAppSpecProbeInput {
  /**
   * 存活探针配置
   */
  liveness?: ProbeInput;
  /**
   * 就绪探针配置
   */
  readiness?: ProbeInput;
  /**
   * 启动探针配置
   */
  startup?: ProbeInput;
}

export interface ProbeInput {
  /**
   * 连续失败次数阈值
   */
  failureThreshold?: number;
  /**
   * 容器启动后延迟探测的秒数
   */
  initialDelaySeconds?: number;
  /**
   * 探测频率（秒）
   */
  periodSeconds?: number;
  /**
   * 探针处理器
   */
  probeHandler: ProbeHandlerInput;
  /**
   * 连续成功次数阈值
   */
  successThreshold?: number;
  /**
   * 探针超时秒数
   */
  timeoutSeconds?: number;
}

export interface ProbeHandlerInput {
  /**
   * 执行命令（仅当 type=EXEC 时有效；与 shCommand 二选一，表示命令使用 exec 模式调用）
   */
  command?: string[];
  /**
   * HTTP 请求头（仅当 type=HTTP 时有效）
   */
  headers?: Record<string, string>;
  /**
   * 检查端口（type=HTTP 或 type=TCP 时有效，取值 1~65535）
   */
  port?: number;
  /**
   * shCommand 与 command 二选一的脚本正文（仅当 type=EXEC 时有效；以 /bin/sh -c 方式执行）
   */
  shCommand?: string;
  /**
   * 处理器类型：EXEC、HTTP 或 TCP
   */
  type: "EXEC" | "HTTP" | "TCP";
  /**
   * HTTP 请求 URL（仅当 type=HTTP 时有效）
   */
  url?: string;
}

export interface EnvAppSpecLifecycleInput {
  /**
   * 容器启动后执行的钩子
   */
  postStart?: LifecycleHandlerInput;
  /**
   * 容器终止前执行的钩子
   */
  preStop?: LifecycleHandlerInput;
  /**
   * Pod 优雅终止超时时间（秒），必须 >= 0
   */
  terminationGracePeriodSeconds?: number;
}

export interface LifecycleHandlerInput {
  /**
   * Exec 命令配置，type 为 "EXEC" 时需要提供
   */
  exec?: LifecycleExecActionInput;
  /**
   * HTTP 请求配置，type 为 "HTTP" 时需要提供
   */
  http?: LifecycleHTTPGetActionInput;
  /**
   * 处理器类型：EXEC 或 HTTP
   */
  type: "EXEC" | "HTTP";
}

export interface LifecycleExecActionInput {
  /**
   * 在容器内执行的命令，可选；与 shCommand 二选一，表示命令使用 exec 模式调用
   */
  command?: string[];
  /**
   * shCommand 与 command 二选一的脚本正文，以 /bin/sh -c 方式执行
   */
  shCommand?: string;
  /**
   * 睡眠等待时间（秒），可选，必须 >= 0。应用到 workload 时转换为 sleep 命令
   */
  sleepSeconds?: number;
}

export interface LifecycleHTTPGetActionInput {
  /**
   * 自定义请求头，可选
   */
  headers?: Record<string, string>;
  /**
   * 发送 HTTP GET 请求的完整 URL
   */
  url?: string;
}

export interface AppSpecDevModeInput {
  /**
   * 是否启用开发模式
   */
  enabled?: boolean;
  /**
   * 脚本挂载路径，由应用类型决定，不接受自定义修改
   */
  mountPath?: string;
  /**
   * 开发模式根目录，由应用类型决定，不接受自定义修改
   */
  workPath?: string;
}

export interface AppSpecDevModeOutput {
  /**
   * 是否启用开发模式
   */
  enabled?: boolean;
  /**
   * 脚本挂载路径
   */
  mountPath?: string;
  /**
   * 开发模式根目录
   */
  workPath?: string;
}

export interface AppSpecOverviewOutput {
  /**
   * 所有已修改过环境下应用配置的环境名列表
   */
  configuredEnvs?: string[];
}

export interface AppSpecUpdateStrategyInput {
  /**
   * 滚动更新时最大可增加实例数，支持大于等于 0 的整数或百分比。
   */
  maxSurge?: string;
  /**
   * 滚动更新时最大不可用实例数，支持大于等于 0 的整数或百分比。
   */
  maxUnavailable?: string;
}

export interface AppSpecUpdateStrategyOutput {
  /**
   * 滚动更新时最大可增加实例数
   */
  maxSurge?: string;
  /**
   * 滚动更新时最大不可用实例数
   */
  maxUnavailable?: string;
}

export interface AppSpecTkeRouteEniOutput {
  /**
   * 是否启用 TKE Route ENI (VPC-CNI) 网络模式
   */
  enabled?: boolean;
}

export interface AppSpecResourcesInput {
  /**
   * CPU limits
   */
  cpuLimits: string;
  /**
   * CPU requests
   */
  cpuRequests: string;
  /**
   * Memory limits
   */
  memoryLimits: string;
  /**
   * Memory requests
   */
  memoryRequests: string;
  /**
   * 副本数量，必须为正整数。
   */
  replicas?: number;
}

export interface AppSpecProbeInput {
  /**
   * 存活探针配置
   */
  liveness?: ProbeInput;
  /**
   * 就绪探针配置
   */
  readiness?: ProbeInput;
  /**
   * 启动探针配置
   */
  startup?: ProbeInput;
}

export interface AppSpecProbeOutput {
  /**
   * 存活探针配置
   */
  liveness?: ProbeOutput;
  /**
   * 就绪探针配置
   */
  readiness?: ProbeOutput;
  /**
   * 启动探针配置
   */
  startup?: ProbeOutput;
}

export interface ProbeOutput {
  /**
   * 连续失败次数阈值
   */
  failureThreshold?: number;
  /**
   * 容器启动后延迟探测的秒数
   */
  initialDelaySeconds?: number;
  /**
   * 探测频率（秒）
   */
  periodSeconds?: number;
  /**
   * 探针处理器
   */
  probeHandler?: ProbeHandlerOutput;
  /**
   * 连续成功次数阈值
   */
  successThreshold?: number;
  /**
   * 探针超时秒数
   */
  timeoutSeconds?: number;
}

export interface ProbeHandlerOutput {
  /**
   * 执行命令（仅当 type=EXEC 时有效；与 shCommand 二选一，表示命令使用 exec 模式调用）
   */
  command?: string[];
  /**
   * HTTP 请求头（仅当 type=HTTP 时有效）
   */
  headers?: Record<string, string>;
  /**
   * 检查端口（type=HTTP 或 type=TCP 时有效，取值 1~65535）
   */
  port?: number;
  /**
   * shCommand 与 command 二选一的脚本正文（仅当 type=EXEC 时有效；以 /bin/sh -c 方式执行）
   */
  shCommand?: string;
  /**
   * 处理器类型：EXEC、HTTP 或 TCP
   */
  type?: string;
  /**
   * HTTP 请求 URL（仅当 type=HTTP 时有效）
   */
  url?: string;
}

export interface AppSpecLifecycleInput {
  /**
   * 容器启动后执行的钩子
   */
  postStart?: LifecycleHandlerInput;
  /**
   * 容器终止前执行的钩子
   */
  preStop?: LifecycleHandlerInput;
  /**
   * Pod 优雅终止超时时间（秒），必须 >= 0
   */
  terminationGracePeriodSeconds?: number;
}

export interface AppSpecLifecycleOutput {
  /**
   * 容器启动后执行的钩子
   */
  postStart?: LifecycleHandlerOutput;
  /**
   * 容器终止前执行的钩子
   */
  preStop?: LifecycleHandlerOutput;
  /**
   * Pod 优雅终止超时时间（秒）
   */
  terminationGracePeriodSeconds?: string;
}

export interface LifecycleHandlerOutput {
  /**
   * Exec 命令配置，type 为 "EXEC" 时存在
   */
  exec?: LifecycleExecActionOutput;
  /**
   * HTTP 请求配置，type 为 "HTTP" 时存在
   */
  http?: LifecycleHTTPGetActionOutput;
  /**
   * 处理器类型：EXEC 或 HTTP
   */
  type?: string;
}

export interface LifecycleExecActionOutput {
  /**
   * 在容器内执行的命令，与 shCommand 二选一，表示命令使用 exec 模式调用
   */
  command?: string[];
  /**
   * shCommand 与 command 二选一的脚本正文，以 /bin/sh -c 方式执行
   */
  shCommand?: string;
  /**
   * 睡眠等待时间（秒），可选。应用到 workload 时转换为 sleep 命令
   */
  sleepSeconds?: string;
}

export interface LifecycleHTTPGetActionOutput {
  /**
   * 自定义请求头
   */
  headers?: Record<string, string>;
  /**
   * 发送 HTTP GET 请求的完整 URL
   */
  url?: string;
}

export interface AppSpecLabelsOutput {
  /**
   * 自定义标签 key/value 映射
   */
  labels?: Record<string, string>;
}

export interface AppSpecAnnotationsOutput {
  /**
   * 自定义注解 key/value 映射
   */
  annotations?: Record<string, string>;
}
