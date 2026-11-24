/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：frontend-features

export interface GetFrontendFeaturesRequest {
}

export interface FrontendFeaturesResponse {
  data?: FrontendFeaturesOutput;
}

export interface FrontendFeaturesOutput {
  /**
   * 是否启用 BSCP（服务配置中心）对接，为 false 时前端隐藏 BSCP 配置相关入口
   */
  enableBSCP?: boolean;
  /**
   * 是否启用 Go 模块代理（GOPROXY）构建参数指引，为 false 时前端隐藏相关提示与快捷添加入口
   */
  enableGoProxy?: boolean;
  /**
   * 制品库是否为独立部署（不接入权限中心，改用制品库账号认证）
   */
  enableIndependentBkRepo?: boolean;
  /**
   * 是否启用 KubeInsight（集群诊断）对接，为 false 时前端隐藏集群诊断相关入口
   */
  enableKubeInsight?: boolean;
  /**
   * 是否为多租户模式。为 true 时前端需要展示租户信息、按用户所属租户选择和展示用户
   */
  enableMultiTenantMode?: boolean;
  /**
   * 是否配置了 WeTerm 环境，为 false 时前端隐藏 WeTerm 入口
   */
  enableWeTerm?: boolean;
  /**
   * 是否允许空间自建北极星接入点并按集群绑定。为 false 时前端隐藏接入点管理、集群绑定、组件安装等入口，
   * 所有集群都使用平台托管接入点
   */
  polarisWorkspaceAccessPoint?: boolean;
}
