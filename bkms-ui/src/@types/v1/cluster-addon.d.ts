/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：cluster-addon

export interface ListClusterAddonsRequest {
  /**
   * 环境 ID
   */
  envID: string;
  /**
   * 命名空间，默认为插件定义中的 defaultNamespace
   */
  namespace?: string;
}

export type UpsertClusterAddonRequest = UpsertClusterAddonInput & {
  /**
   * 环境 ID
   */
  envID: string;
  /**
   * 插件名称
   */
  addonName: string;
};

export interface DeleteClusterAddonRequest {
  /**
   * 环境 ID
   */
  envID: string;
  /**
   * 插件名称
   */
  addonName: string;
  /**
   * 命名空间，默认为插件定义中的 defaultNamespace
   */
  namespace?: string;
}

export interface ListClusterAddonsOutput {
  /**
   * 插件列表
   */
  addons?: ClusterAddonInfoOutput[];
}

export interface UpsertClusterAddonInput {
  /**
   * Chart 版本
   */
  chartVersion: string;
  /**
   * 命名空间（可选，默认为插件定义中的 defaultNamespace）
   */
  namespace?: string;
  /**
   * Helm values 参数（JSON 格式）
   */
  values?: Record<string, unknown>;
}

export interface DeleteClusterAddonOutput {
  /**
   * 状态描述
   */
  message?: string;
  /**
   * 卸载状态
   */
  status?: string;
}

export interface ClusterAddonInfoOutput {
  /**
   * HelmChart 信息
   */
  chartInfo?: HelmChartInfoOutput;
  /**
   * 插件描述
   */
  description?: string;
  /**
   * 展示名称
   */
  displayName?: string;
  /**
   * 集群安装信息
   */
  installInfo?: ClusterInstallInfoOutput;
  /**
   * 插件名称
   */
  name?: string;
  /**
   * 可选安装该插件的应用类型列表
   */
  optionalForAppTypes?: string[];
  /**
   * 必装该插件的应用类型列表
   */
  requiredForAppTypes?: string[];
  /**
   * 支持的操作列表（如 install, upgrade, uninstall）
   */
  supportedActions?: string[];
}

export interface HelmChartInfoOutput {
  /**
   * 仓库中可用的 Chart 版本列表
   */
  availableVersions?: string[];
  /**
   * Chart 名称
   */
  chartName?: string;
  /**
   * 默认安装时使用的 Chart 版本
   */
  defaultChartVersion?: string;
  /**
   * 安装示例参数（YAML 字符串，可包含注释）
   */
  exampleValues?: string;
}

export interface ClusterInstallInfoOutput {
  /**
   * 当前已安装的 Chart 版本
   */
  currentChartVersion?: string;
  /**
   * 当前已安装的 values 参数（JSON 字符串，未安装时为空）
   */
  currentValues?: string;
  /**
   * 状态信息（安装失败时给出提示信息）
   */
  message?: string;
  /**
   * 插件安装的命名空间
   */
  namespace?: string;
  /**
   * 当前安装状态（空字符串表示未安装）
   */
  status?: string;
}
