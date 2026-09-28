/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：polaris-config

export interface ListAppPolarisConfigsRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export type CreateAppPolarisConfigRequest = CreateAppPolarisConfigInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export type UpdateImportedPolarisRequest = UpdateImportedPolarisInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export type GetImportedPolarisServiceRequest = GetImportedPolarisServiceInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export type ValidateAppPolarisConfigRequest = CreateAppPolarisConfigInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface DeleteAppPolarisConfigRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 配置名称
   */
  configName: string;
}

export type PatchAppPolarisConfigRequest = PatchAppPolarisConfigInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 配置名称
   */
  configName: string;
};

export interface GetEnvInstanceStatsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 配置名称
   */
  configName: string;
}

export type PutEnvWeightRequest = PutEnvWeightInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 配置名称
   */
  configName: string;
  /**
   * 环境名称
   */
  envName: string;
};

export interface ListAppPolarisConfigVarsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 配置名称
   */
  configName: string;
}

export interface ListAppPolarisConfigsOutput {
  /**
   * 北极星配置列表
   */
  data?: PolarisConfigOutputObj[];
}

export interface CreateAppPolarisConfigInput {
  /**
   * 是否由平台创建新的北极星服务
   */
  createNewService?: boolean;
  /**
   * 是否为直连模式，默认 false
   */
  direct?: boolean;
  /**
   * 是否启用健康检查，默认 false
   */
  enableHealthCheck?: boolean;
  /**
   * 是否启用权重因子。仅 createNewService 为 true 时写入新建的北极星服务，不落本地。
   * 从现有引入时忽略该字段，当前值向北极星读取。
   */
  enableWeightFactor?: boolean;
  /**
   * 组件实例标识，用于环境变量拼接，只能包含字母、数字、下划线
   */
  instanceKey: string;
  /**
   * 是否保留未就绪的 Pod 在北极星，默认 true
   */
  keepNotReadyPod?: boolean;
  /**
   * 操作人(即北极星负责人, 仅 createNewService 为 true 时有效)
   */
  operator?: string;
  /**
   * 北极星实例名称
   */
  polarisName: string;
  /**
   * 北极星环境（命名空间）
   */
  polarisNamespace: "Test" | "Production" | "Development" | "Pre-release";
  /**
   * 北极星 Token（当 createNewService 为 false 时必填，为 true 时由平台创建后回填）
   */
  polarisToken?: string;
  /**
   * 注册模式，默认 on_deploy（等部署后注册）。
   * immediate 表示绑定环境后立即下发 PolarisConfig CR 与配套 Service 完成注册，
   * 该配置不再注入环境变量和 tRPC 框架配置。创建后不可修改
   */
  registerMode?: "immediate" | "on_deploy";
  /**
   * 生效的环境列表
   */
  scopeEnvNames?: string[];
  /**
   * 服务标签
   */
  serviceLabels?: Record<string, string>;
  /**
   * 服务端口
   */
  servicePort: number;
}

export interface CreateAppPolarisConfigOutput {
  /**
   * 配置名称
   */
  data?: PolarisNameOutputObj;
}

export interface UpdateImportedPolarisInput {
  /**
   * 是否开启权重因子
   */
  enableWeightFactor: boolean;
  /**
   * 北极星实例名称
   */
  polarisName: string;
  /**
   * 北极星环境（命名空间）
   */
  polarisNamespace: "Test" | "Production" | "Development" | "Pre-release";
  /**
   * 北极星 Token
   */
  polarisToken: string;
}

export interface GetImportedPolarisServiceInput {
  /**
   * 北极星实例名称
   */
  polarisName: string;
  /**
   * 北极星环境（命名空间）
   */
  polarisNamespace: "Test" | "Production" | "Development" | "Pre-release";
  /**
   * 北极星 Token
   */
  polarisToken: string;
}

export interface GetImportedPolarisServiceOutput {
  /**
   * 北极星线上服务信息（不含 token）
   */
  service?: ImportedPolarisServiceOutput;
}

export interface ValidateAppPolarisConfigOutput {
  /**
   * 校验警告信息
   */
  warnings?: string[];
}

export interface PatchAppPolarisConfigInput {
  /**
   * 是否为直连模式（可选更新）
   */
  direct?: boolean;
  /**
   * 是否启用健康检查（可选更新）
   */
  enableHealthCheck?: boolean;
  /**
   * 是否启用权重因子。传入时写回北极星，不落本地；未传表示不改。
   * 平台创建的服务走依赖服务实例，从现有引入的服务用 Token 写回。
   */
  enableWeightFactor?: boolean;
  /**
   * 组件实例标识（可选更新）
   */
  instanceKey?: string;
  /**
   * 是否保留未就绪的 Pod 在北极星（可选更新）
   */
  keepNotReadyPod?: boolean;
  /**
   * 操作人/负责人（可选更新；未出现表示不改；空字符串非法）
   */
  operator?: string;
  /**
   * 北极星 Token（可选更新）
   */
  polarisToken?: string;
  /**
   * 生效的环境列表（可选更新；传入时全量替换，空数组表示清空，nil 表示不更新）
   */
  scopeEnvNames?: string[];
  /**
   * 服务标签（可选更新，传入时全量替换）
   */
  serviceLabels?: Record<string, string>;
  /**
   * 服务端口（可选更新）
   */
  servicePort?: number;
}

export interface PatchAppPolarisConfigOutput {
  /**
   * 更新后的北极星配置
   */
  data?: PolarisConfigOutputObj;
}

export interface GetEnvInstanceStatsOutput {
  data?: GetEnvInstanceStatsOutputObj;
}

export interface PutEnvWeightInput {
  /**
   * 该环境是否开启动态权重，不传表示保持原值。
   * 开启后上面的权重作为动态调权的基准权重
   */
  dynamicWeight?: boolean;
  /**
   * 单实例权重，取值范围 0-10000
   */
  weight: number;
}

export interface PutEnvWeightOutput {
  /**
   * 更新后的北极星配置
   */
  data?: PolarisConfigOutputObj;
}

export interface ListAppPolarisConfigVarsOutput {
  /**
   * 变量列表
   */
  data?: PolarisConfigVarOutput[];
}

export interface PolarisConfigVarOutput {
  /**
   * 变量名
   */
  key?: string;
  /**
   * 变量值
   */
  value?: string;
}

export interface PolarisConfigOutputObj {
  /**
   * 所属应用 ID
   */
  appID?: string;
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 关联的依赖服务实例 ID（平台创建时有值，用于后续管理）
   */
  depSvcInstID?: string;
  /**
   * 是否为直连模式（注册 PodIP 到北极星）
   */
  direct?: boolean;
  /**
   * 是否启用健康检查
   */
  enableHealthCheck?: boolean;
  /**
   * 是否启用权重因子。向北极星实时读取，读不到时为 null。
   * 开启后才能为单个环境开启动态权重；该值不参与 CR 组装。
   */
  enableWeightFactor?: boolean;
  /**
   * 各环境是否开启动态权重，key 为环境名称；未出现的环境表示未开启
   */
  envDynamicWeights?: Record<string, boolean>;
  /**
   * 各环境中已经生效的关键字段、下发错误和部署状态
   */
  envStates?: Record<string, PolarisEnvStateOutput>;
  /**
   * 各环境的单实例权重，key 为环境名称
   */
  envWeights?: Record<string, number>;
  /**
   * 组件实例标识，用于环境变量拼接
   */
  instanceKey?: string;
  /**
   * 是否保留未就绪的 Pod 在北极星
   */
  keepNotReadyPod?: boolean;
  /**
   * 组件名称
   */
  name?: string;
  /**
   * 负责人
   */
  operator?: string;
  /**
   * 北极星实例名称
   */
  polarisName?: string;
  /**
   * 北极星环境（命名空间）
   */
  polarisNamespace?: string;
  /**
   * 北极星 Token（敏感信息，返回时脱敏）
   */
  polarisToken?: string;
  /**
   * 注册模式：immediate（绑定后立即注册）| on_deploy（等部署后注册）
   */
  registerMode?: string;
  /**
   * 生效的环境列表
   */
  scopeEnvNames?: string[];
  /**
   * 服务标签
   */
  serviceLabels?: Record<string, string>;
  /**
   * 服务端口
   */
  servicePort?: number;
  /**
   * 更新时间
   */
  updatedAt?: string;
  /**
   * 校验警告信息
   */
  warnings?: string[];
}

export interface PolarisEnvStateOutput {
  /**
   * 集群中已经生效的关键字段；nil 表示尚未完成首次应用部署
   */
  appliedFields?: RedeployRequiredFieldsOutput;
  /**
   * 最近一次记录的动态下发错误；应用部署完成或正常下发成功后清空
   */
  lastError?: string;
  /**
   * 生成响应时，当前配置期望的 Polaris Token 是否不同于该环境最近一次应用部署记录的 Token；
   * 环境不在当前生效范围或尚无部署快照时为 false，Token 值本身始终脱敏
   */
  polarisTokenChanged?: boolean;
  /**
   * 部署状态: deployed / pendingCreate / pendingModify / pendingDelete
   */
  status?: string;
  /**
   * 环境信息最后更新时间；尚无环境记录时为空
   */
  updatedAt?: string;
}

export interface RedeployRequiredFieldsOutput {
  /**
   * InstanceKey 北极星实例标识，用于生成环境变量
   */
  instanceKey?: string;
  /**
   * PolarisToken 北极星访问令牌
   */
  polarisToken?: string;
  /**
   * ServicePort 北极星服务端口
   */
  servicePort?: number;
}

export interface GetEnvInstanceStatsOutputObj {
  /**
   * 各环境匹配到的北极星实例统计，key 为环境名
   */
  envStats?: Record<string, EnvInstanceStatsOutput>;
  /**
   * 北极星服务下全部健康实例数（含非平台注册，例如迁移业务）
   */
  totalHealthyInstanceCount?: number;
  /**
   * 北极星服务下全部健康实例的权重总和
   */
  totalHealthyInstanceWeight?: number;
}

export interface EnvInstanceStatsOutput {
  /**
   * 匹配实例中健康的数量（isHealthy && !isIsolated && weight > 0）
   */
  healthyInstanceCount?: number;
  /**
   * 匹配健康实例的权重总和
   */
  healthyInstanceWeight?: number;
  /**
   * 匹配实例中隔离的数量（isIsolated == true）
   */
  isolatedInstanceCount?: number;
  /**
   * 匹配到本环境 Pod 的实例总数
   */
  totalInstanceCount?: number;
  /**
   * 本环境被单独设置过权重的实例数，其实际权重可能与配置的单实例权重不一致
   */
  weightOverriddenInstanceCount?: number;
}

export interface ImportedPolarisServiceOutput {
  /**
   * 创建时间
   */
  ctime?: string;
  /**
   * 北极星服务当前是否开启权重因子
   */
  enableWeightFactor?: boolean;
  /**
   * 服务 metadata
   */
  metadata?: Record<string, string>;
  /**
   * 修改时间
   */
  mtime?: string;
  /**
   * 服务名
   */
  name?: string;
  /**
   * 命名空间
   */
  namespace?: string;
  /**
   * 负责人，逗号分隔
   */
  owners?: string;
  /**
   * 所属平台 ID
   */
  platformId?: string;
  /**
   * 版本号
   */
  revision?: string;
}

export interface PolarisNameOutputObj {
  /**
   * 配置名称
   */
  name?: string;
}
