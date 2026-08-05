/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：component-defs

export interface ListComponentDefsRequest {
  /**
   * 按可使用该组件定义的工作空间 ID 过滤
   */
  scopeWorkspaceID?: string;
  /**
   * 按可管理该组件定义的工作空间 ID 过滤
   */
  managedByWorkspaceID?: string;
  /**
   * 搜索关键词
   */
  keyword?: string;
}

export type CreateComponentDefRequest = CreateComponentDefInput;

export interface GetComponentDefsBuiltinVarsRequest {
}

export type PreviewComponentDefRequest = PreviewComponentDefInput;

export interface DeleteComponentDefRequest {
  /**
   * 组件定义名称
   */
  compDefName: string;
}

export type PatchComponentDefRequest = PatchComponentDefInput & {
  /**
   * 组件定义名称
   */
  compDefName: string;
};

export interface ListComponentDefsOutput {
  data?: ComponentDefOutputObj[];
}

export interface CreateComponentDefInput {
  /**
   * 组件名称
   */
  compDefName: string;
  /**
   * 组件描述
   */
  description?: string;
  /**
   * 组件展示名称
   */
  displayName?: string;
  /**
   * 标记在哪些工作空间下可以管理该组件定义
   */
  managedByWorkspaceIDs?: string[];
  /**
   * 根节点 YAML Patch 模板列表
   */
  patchers?: string[];
  /**
   * 属性定义列表
   */
  properties?: PropertyDefInput[];
  /**
   * 生效范围类型: global / workspace
   */
  scopeType: "global" | "workspace";
  /**
   * 生效的工作空间列表
   */
  scopeWorkspaceIDs?: string[];
  /**
   * 额外 Kubernetes 资源 YAML 模板列表
   */
  specs?: string[];
}

export interface EmptyOutput {
}

export interface ListBuiltinVarsOutput {
  data?: BuiltinVarOutputObj[];
}

export interface PreviewComponentDefInput {
  /**
   * 组件名称，用于预览 name 等内置变量
   */
  compDefName: string;
  /**
   * 根节点 YAML Patch 模板列表
   */
  patchers?: string[];
  /**
   * 属性定义列表
   */
  properties?: PropertyDefInput[];
  /**
   * 额外 Kubernetes 资源 YAML 模板列表
   */
  specs?: string[];
}

export interface PreviewOutput {
  /**
   * patch 预览列表
   */
  patchPreview?: PreviewPatchOutput[];
  /**
   * 渲染后的附加资源列表
   */
  resources?: PreviewResourceOutput[];
}

export interface PatchComponentDefInput {
  /**
   * 组件描述
   */
  description?: string;
  /**
   * 组件展示名称
   */
  displayName?: string;
  /**
   * 标记在哪些工作空间下可以管理该组件定义
   */
  managedByWorkspaceIDs?: string[];
  /**
   * 根节点 YAML Patch 模板列表（传入时全量替换）
   */
  patchers?: string[];
  /**
   * 属性定义列表（传入时全量替换）
   */
  propertiesInput?: ComponentDefPropertiesInput;
  /**
   * 生效范围类型
   */
  scopeType?: "global" | "workspace";
  /**
   * 生效的工作空间列表
   */
  scopeWorkspaceIDs?: string[];
  /**
   * 额外 Kubernetes 资源 YAML 模板列表（传入时全量替换）
   */
  specs?: string[];
}

export interface ComponentDefPropertiesInput {
  properties?: PropertyDefInput[];
}

export interface PropertyDefInput {
  /**
   * 属性默认值
   */
  defaultValue?: unknown;
  /**
   * 属性描述
   */
  description?: string;
  /**
   * 属性名称
   */
  name: string;
  /**
   * SELECT 类型可配置的候选项
   */
  options?: PropertyOptionInput[];
  /**
   * 属性类型
   */
  type: "STRING" | "INT" | "TEXT" | "SELECT" | "BOOL" | "MAP";
}

export interface PropertyOptionInput {
  label: string;
  value: string;
}

export interface PreviewPatchOutput {
  /**
   * 预置底稿 YAML
   */
  baseManifest?: string;
  /**
   * 应用全部 patcher 后的 YAML
   */
  patchedManifest?: string;
  /**
   * 被 patch 的目标资源类型；当前固定 GameDeployment
   */
  targetKind?: string;
}

export interface PreviewResourceOutput {
  apiVersion?: string;
  kind?: string;
  /**
   * 渲染后的完整资源 YAML
   */
  manifest?: string;
  name?: string;
}

export interface BuiltinVarOutputObj {
  description?: string;
  key?: string;
}

export interface ComponentDefOutputObj {
  appCompInstanceCount?: number;
  createdAt?: string;
  creator?: string;
  description?: string;
  displayName?: string;
  isBuiltin?: boolean;
  managedByWorkspaceIDs?: string[];
  name?: string;
  patchers?: string[];
  properties?: PropertyDefOutputObj[];
  scopeType?: string;
  scopeWorkspaceIDs?: string[];
  specs?: string[];
  updatedAt?: string;
  updater?: string;
  version?: string;
  workspaceCompInstanceCount?: number;
}

export interface PropertyDefOutputObj {
  defaultValue?: unknown;
  description?: string;
  name?: string;
  options?: PropertyOptionOutputObj[];
  type?: string;
}

export interface PropertyOptionOutputObj {
  label?: string;
  value?: string;
}
