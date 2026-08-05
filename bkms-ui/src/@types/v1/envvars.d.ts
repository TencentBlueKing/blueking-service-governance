/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：envvars

export interface GetAppEnvVarsRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export type CreateAppEnvVarsRequest = CreateAppDefinedEnvVarInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export interface ListDetailedAppEnvVarsRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export interface ExportAppEnvVarsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 导出范围：appDefined 或 effectiveByEnv
   */
  scope: string;
  /**
   * 环境名称；scope=effectiveByEnv 时必填
   */
  envName?: string;
}

export interface ImportAppDefinedEnvVarRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 应用环境变量导入请求文件
   */
  file: unknown;
}

export interface PreviewAppDefinedEnvVarRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 应用环境变量导入预览请求文件
   */
  file: unknown;
}

export type UpdateAppEnvVarsRequest = UpdateAppDefinedEnvVarInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 旧环境变量 Key
   */
  key: string;
};

export interface DeleteAppEnvVarsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境变量 Key
   */
  key: string;
}

export interface ListAppBgEnvVarsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface ListAppEnvVarsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 环境名称
   */
  envName: string;
}

export interface ListEnvAvailableEnvVarsRequest {
  /**
   * 环境 ID
   */
  envID: string;
}

export interface ListEnvBgEnvVarsRequest {
  /**
   * 环境 ID
   */
  envID: string;
}

export interface ListDetailedEnvScopedEnvVarsRequest {
  /**
   * 环境 ID
   */
  envID: string;
}

export interface ExportEnvScopedEnvVarsRequest {
  /**
   * 环境 ID
   */
  envID: string;
}

export interface ImportEnvScopedEnvVarRequest {
  /**
   * 环境 ID
   */
  envID: string;
  /**
   * 单环境环境变量导入请求文件
   */
  file: unknown;
}

export interface PreviewEnvScopedEnvVarRequest {
  /**
   * 环境 ID
   */
  envID: string;
  /**
   * 单环境环境变量导入预览请求文件
   */
  file: unknown;
}

export type CreateScopedEnvVarRequest = CreateScopedEnvVarInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export interface ListPublicScopedEnvVarsRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export interface ExportPublicScopedEnvVarsRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export interface ImportPublicScopedEnvVarRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 公共环境变量导入请求文件
   */
  file: unknown;
}

export interface PreviewPublicScopedEnvVarRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 公共环境变量导入预览请求文件
   */
  file: unknown;
}

export type UpdateScopedEnvVarRequest = UpdateScopedEnvVarInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * Scoped EnvVar ID
   */
  scopedEnvVarID: string;
};

export interface DeleteScopedEnvVarRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * Scoped EnvVar ID
   */
  scopedEnvVarID: string;
}

export interface ListAppDefinedEnvVarsOutput {
  /**
   * 应用直接定义的环境变量列表
   */
  data?: AppDefinedEnvVarOutputObj[];
}

export interface CreateAppDefinedEnvVarInput {
  /**
   * 描述
   */
  description?: string;
  /**
   * 是否敏感
   */
  isSensitive?: boolean;
  /**
   * 环境变量 Key
   */
  key: string;
  /**
   * 环境变量值，允许为空
   */
  value?: string;
}

export interface CreateAppDefinedEnvVarOutput {
  /**
   * 应用直接定义的环境变量
   */
  data?: AppDefinedEnvVarOutputObj;
}

export interface ListDetailedAppEnvVarsOutput {
  /**
   * 应用环境变量详情列表
   */
  data?: AppEnvVarDetailedOutputObj[];
}

export interface ImportEnvVarOutput {
  /**
   * 导入结果汇总
   */
  data?: EnvVarImportPreviewSummaryOutputObj;
}

export interface PreviewEnvVarOutput {
  /**
   * 预览结果
   */
  data?: EnvVarImportPreviewOutputObj;
}

export interface UpdateAppDefinedEnvVarInput {
  /**
   * 描述
   */
  description?: string;
  /**
   * 是否敏感，未传时保持原值不变
   */
  isSensitive?: boolean;
  /**
   * 更新后的环境变量 Key
   */
  updatedKey: string;
  /**
   * 环境变量值，未传时保持原值，允许显式传空字符串
   */
  value?: string;
}

export interface UpdateAppDefinedEnvVarOutput {
  /**
   * 应用直接定义的环境变量
   */
  data?: AppDefinedEnvVarOutputObj;
}

export interface EmptyOutput {
}

export interface ListAppBgEnvVarsOutput {
  /**
   * 应用在某个环境下的背景环境变量列表
   */
  data?: BgEnvVarOutputObj[];
}

export interface ListAppEnvVarsOutput {
  /**
   * 应用部署到某个环境后可用的环境变量列表
   */
  data?: EnvVarOutputObj[];
}

export interface ListEnvAvailableEnvVarsOutput {
  /**
   * 环境下所有可用的环境变量列表
   */
  data?: EnvVarOutputObj[];
}

export interface ListEnvBgEnvVarsOutput {
  /**
   * 指定环境的背景环境变量列表
   */
  data?: BgEnvVarOutputObj[];
}

export interface ListDetailedEnvScopedEnvVarsOutput {
  /**
   * 作用域为当前环境的环境变量详情列表
   */
  data?: ScopedEnvVarDetailedOutputObj[];
}

export interface CreateScopedEnvVarInput {
  /**
   * 描述
   */
  description?: string;
  /**
   * 是否敏感
   */
  isSensitive?: boolean;
  /**
   * 环境变量 Key
   */
  key: string;
  /**
   * 作用域类型，目前支持 workspace、envType、env
   */
  scopeType: "workspace" | "envType" | "env";
  /**
   * 作用域值
   */
  scopeValue?: string;
  /**
   * 环境变量值，允许为空
   */
  value?: string;
}

export interface CreateScopedEnvVarOutput {
  /**
   * 作用域级别环境变量
   */
  data?: ScopedEnvVarOutputObj;
}

export interface ListPublicScopedEnvVarsOutput {
  /**
   * 作用域为 workspace 和 envType 的环境变量列表
   */
  data?: ScopedEnvVarOutputObj[];
}

export interface UpdateScopedEnvVarInput {
  /**
   * 描述
   */
  description?: string;
  /**
   * 是否敏感，未传时保持原值不变
   */
  isSensitive?: boolean;
  /**
   * 环境变量 Key
   */
  key: string;
  /**
   * 环境变量值，未传时保持原值，允许显式传空字符串
   */
  value?: string;
}

export interface UpdateScopedEnvVarOutput {
  /**
   * 作用域级别环境变量
   */
  data?: ScopedEnvVarOutputObj;
}

export interface ScopedEnvVarOutputObj {
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 描述
   */
  description?: string;
  /**
   * 环境变量 ID
   */
  id?: string;
  /**
   * 是否敏感
   */
  isSensitive?: boolean;
  /**
   * 环境变量 Key
   */
  key?: string;
  /**
   * 作用域类型，目前支持 workspace、envType、env
   */
  scopeType?: string;
  /**
   * 作用域值
   * - 当 scopeType 为 workspace 时，固定为空字符串
   * - 当 scopeType 为 envType 时，可选值为 development、test、staging、production
   * - 当 scopeType 为 env 时，值为具体环境名称
   */
  scopeValue?: string;
  /**
   * 更新时间
   */
  updatedAt?: string;
  /**
   * 环境变量值
   */
  value?: string;
  /**
   * 工作空间 ID
   */
  workspaceID?: string;
}

export interface ScopedEnvVarDetailedOutputObj {
  /**
   * 冲突信息
   */
  conflictedInfo?: EnvVarConflictedInfoOutputObj;
  /**
   * 环境变量基础信息
   */
  scopedEnvVar?: ScopedEnvVarOutputObj;
}

export interface EnvVarConflictedInfoOutputObj {
  /**
   * 冲突详情
   */
  conflictedDetail?: string;
  /**
   * 冲突来源列表
   */
  conflictedSources?: EnvVarConflictedSourceOutputObj[];
  /**
   * 当前变量是否覆盖冲突变量并生效
   */
  overrideConflicted?: boolean;
}

export interface EnvVarConflictedSourceOutputObj {
  /**
   * 冲突来源
   */
  source?: string;
  /**
   * 冲突来源值
   */
  sourceValue?: string;
}

export interface BgEnvVarOutputObj {
  /**
   * 描述
   */
  description?: string;
  /**
   * 环境变量 Key
   */
  key?: string;
  /**
   * 来源，如 builtin、scopedWorkspace、scopedEnvType、scopedEnv、app
   */
  source?: string;
  /**
   * 环境变量值
   */
  value?: string;
}

export interface EnvVarOutputObj {
  /**
   * 环境变量描述
   */
  description?: string;
  /**
   * 是否是内置变量
   */
  isBuiltin?: boolean;
  /**
   * 是否是敏感变量
   */
  isSensitive?: boolean;
  /**
   * 环境变量 Key
   */
  key?: string;
  /**
   * 环境变量值
   */
  value?: string;
}

export interface AppDefinedEnvVarOutputObj {
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 描述
   */
  description?: string;
  /**
   * 是否敏感
   */
  isSensitive?: boolean;
  /**
   * 环境变量 Key
   */
  key?: string;
  /**
   * 更新时间
   */
  updatedAt?: string;
  /**
   * 环境变量值
   */
  value?: string;
}

export interface EnvVarImportPreviewOutputObj {
  /**
   * 逐条预览结果
   */
  items?: EnvVarImportPreviewItemOutputObj[];
  /**
   * 汇总统计
   */
  summary?: EnvVarImportPreviewSummaryOutputObj;
}

export interface EnvVarImportPreviewItemOutputObj {
  /**
   * 导入动作：new（新增）/ overwrite（覆盖）
   */
  action?: string;
  /**
   * 输入中显式声明的原始 scope 信息；未声明时省略该字段
   */
  declaredScope?: EnvVarImportPreviewScopeOutputObj;
  /**
   * 描述信息
   */
  description?: string;
  /**
   * scope 生效状态：none / applied
   */
  effectScope?: string;
  /**
   * 预览后实际生效的 scope 信息；不适用时省略该字段
   */
  effectiveScope?: EnvVarImportPreviewScopeOutputObj;
  /**
   * 环境变量 Key
   */
  key?: string;
  /**
   * 额外提示信息；无提示时省略该字段
   */
  messages?: string[];
  /**
   * 被覆盖变量的原值，仅当 action 为 overwrite 时返回；其他场景省略该字段
   */
  originalValue?: string;
  /**
   * 环境变量 Value（导入值）
   */
  value?: string;
}

export interface EnvVarImportPreviewSummaryOutputObj {
  /**
   * 新增条数
   */
  new?: number;
  /**
   * 覆盖条数
   */
  overwrite?: number;
  /**
   * 导入条目总数
   */
  total?: number;
}

export interface EnvVarImportPreviewScopeOutputObj {
  /**
   * scope 类型（workspace / envType / env）
   */
  type?: string;
  /**
   * scope 值；workspace 时省略
   */
  value?: string;
}

export interface AppEnvVarDetailedOutputObj {
  /**
   * 应用环境变量基础信息
   */
  appEnvVar?: DetailedAppEnvVarOutputObj;
  /**
   * 冲突信息
   */
  conflictedInfo?: EnvVarConflictedInfoOutputObj;
}

export interface DetailedAppEnvVarOutputObj {
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 描述
   */
  description?: string;
  /**
   * 是否敏感
   */
  isSensitive?: boolean;
  /**
   * 环境变量 Key
   */
  key?: string;
  /**
   * 更新时间
   */
  updatedAt?: string;
  /**
   * 环境变量值
   */
  value?: string;
}
