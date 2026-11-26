/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：account

export interface GetCurrentUserRequest {
}

export interface RefreshTokenRequest {
}

export interface CreateTokenRequest {
}

export interface ValidateTokenRequest {
}

export interface GetRoleRequest {
}

export interface CurrentUserOutput {
  /**
   * 用户管理个人中心地址，未配置用户管理地址时为空
   */
  personalCenterURL?: string;
  /**
   * 用户所属租户，认证上游未返回租户时为空（如单租户环境）
   */
  tenantID?: string;
  /**
   * 用户时区（IANA 时区名），认证上游未返回时为 Asia/Shanghai
   */
  timeZone?: string;
  /**
   * 用户 ID
   */
  userID?: string;
}

export interface GetRoleResponse {
  data?: RoleInfo;
}

export interface RoleInfo {
  /**
   * 当前用户的平台角色编码，没有平台角色时返回 null
   */
  platRoleCode?: RoleCode;
  /**
   * 当前登录用户名
   */
  username?: string;
}

export type RoleCode = "admin";
