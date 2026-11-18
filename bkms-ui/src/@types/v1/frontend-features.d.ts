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
   * 是否允许空间自建北极星接入点并按集群绑定。为 false 时前端隐藏接入点管理、集群绑定、组件安装等入口，
   * 所有集群都使用平台托管接入点
   */
  polarisWorkspaceAccessPoint?: boolean;
}
