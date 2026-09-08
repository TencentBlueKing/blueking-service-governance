/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：bkmonitor-dashboard

export interface ListAppDashboardsRequest {
  /**
   * 应用 ID
   */
  appID: string;
}

export type CreateAppDashboardRequest = AppDashboardCreateInput & {
  /**
   * 应用 ID
   */
  appID: string;
};

export type UpdateAppDashboardRequest = AppDashboardUpdateInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 仪表盘 uid
   */
  uid: string;
};

export interface DeleteAppDashboardRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 仪表盘 uid
   */
  uid: string;
}

export interface ListDashboardsResp {
  /**
   * Data 应用绑定的仪表盘列表
   */
  data?: DashboardOutput[];
}

export interface AppDashboardCreateInput {
  /**
   * Title 仪表盘标题
   */
  title: string;
  /**
   * UID 仪表盘 uid
   */
  uid: string;
}

export interface EmptyOutput {
}

export interface AppDashboardUpdateInput {
  /**
   * Title 仪表盘标题（可选）
   */
  title?: string;
  /**
   * UID 仪表盘 uid（可选，变更绑定的仪表盘）
   */
  uid?: string;
}

export interface DashboardOutput {
  /**
   * Title 仪表盘标题
   */
  title?: string;
  /**
   * UID 仪表盘 uid
   */
  uid?: string;
  /**
   * URL 仪表盘访问 URL
   */
  url?: string;
}
