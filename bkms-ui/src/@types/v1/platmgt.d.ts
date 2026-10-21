/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：platmgt

export interface ListRoleBindingsRequest {
  /**
   * 用户名关键字
   */
  keyword?: string;
}

export type AssignRolesRequest = AssignRolesInput;

export interface ListRolesRequest {
}

export interface RevokeRoleRequest {
  /**
   * 平台管理员用户名
   */
  username: string;
}

export interface ListPlatWorkspacesRequest {
  /**
   * 搜索关键词，匹配空间 ID / 空间名称
   */
  keyword?: string;
  /**
   * 空间状态过滤：Ready / Processing / Disabled
   */
  state?: string;
  /**
   * 排序字段：id / displayName / updatedAt
   */
  sortBy?: string;
  /**
   * 排序方向：asc / desc
   */
  sortOrder?: string;
  /**
   * 页码，从 1 开始
   */
  page: number;
  /**
   * 每页数量，支持 5/10/20/50/100
   */
  pageSize: number;
}

export interface GetPlatWorkspaceStatsRequest {
}

export interface GetPlatWorkspaceRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export interface GetWorkspaceRoleStatusRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 角色 Code
   */
  roleCode: string;
  /**
   * 用户名
   */
  username: string;
}

export type GrantWorkspaceAdminRequest = GrantAdminInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export interface RevokeWorkspaceAdminRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export interface GetPlatWorkspaceImageRegistryRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export type UpdatePlatWorkspaceImageRegistryRequest = UpdateImageRegistryInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export interface ListRoleBindingsResponse {
  data?: RoleBindingOutput[];
}

export interface AssignRolesInput {
  /**
   * 平台管理员角色编码
   */
  roleCode: RoleCode;
  /**
   * 平台管理员用户名列表
   */
  usernames: string[];
}

export interface ListRolesResponse {
  data?: RoleOutput[];
}

export interface ListWorkspacesResponse {
  data?: PaginatedWorkspaceOutput;
}

export interface WorkspaceStatsResponse {
  data?: WorkspaceStatsOutput;
}

export interface GetWorkspaceResponse {
  data?: WorkspaceInfoOutput;
}

export interface GetRoleStatusResponse {
  data?: RoleStatusOutput;
}

export interface GrantAdminInput {
  /**
   * 是否授予临时管理员。true 表示临时管理员，false 表示永久管理员
   * 使用指针类型以区分字段缺失（nil）和显式传递 false 两种情况；
   * 由于 false 具有明确语义，当前 IsTemporary 字段强制必传， 避免默认值误解
   */
  isTemporary: boolean;
}

export interface ImageRegistryResponse {
  data?: ImageRegistryOutput;
}

export interface UpdateImageRegistryInput {
  /**
   * 外部镜像仓库密码；留空表示沿用该空间已有外部仓库的密码
   */
  password?: string;
  /**
   * 外部镜像仓库地址，形如 mirrors.tencent.com/my-namespace，不带协议前缀
   */
  registry?: string;
  /**
   * 目标镜像仓库类型：builtin（切回平台镜像仓库，其余字段不生效）/ external（外部镜像仓库）
   */
  type: "builtin" | "external";
  /**
   * 外部镜像仓库账号
   */
  username?: string;
}

export interface ImageRegistryOutput {
  /**
   * 镜像密码，配置缺失时为空
   */
  password?: string;
  /**
   * 镜像仓库地址（含命名空间），配置缺失时为空
   */
  registry?: string;
  /**
   * 镜像仓库类型：builtin（平台镜像仓库）/ external（外部镜像仓库）
   */
  type?: string;
  /**
   * 镜像账号，配置缺失时为空
   */
  username?: string;
}

export interface RoleStatusOutput {
  /**
   * 当前目标用户是否拥有指定工作空间角色
   */
  hasRole?: boolean;
}

export interface WorkspaceInfoOutput {
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 创建人
   */
  creator?: string;
  /**
   * 工作空间描述
   */
  description?: string;
  /**
   * 工作空间展示名称
   */
  displayName?: string;
  /**
   * 工作空间 ID
   */
  id?: string;
  /**
   * 工作空间状态
   */
  state?: string;
  /**
   * 更新时间
   */
  updatedAt?: string;
  /**
   * 更新人
   */
  updater?: string;
}

export interface WorkspaceStatsOutput {
  /**
   * Disabled 状态工作空间数量
   */
  disabledCount?: string;
  /**
   * Processing 状态工作空间数量
   */
  processingCount?: string;
  /**
   * Ready 状态工作空间数量
   */
  readyCount?: string;
  /**
   * TotalCount 工作空间总数
   */
  totalCount?: string;
}

export interface PaginatedWorkspaceOutput {
  /**
   * 总数
   */
  count?: string;
  /**
   * 当前页码
   */
  page?: number;
  /**
   * 每页数量
   */
  pageSize?: number;
  /**
   * 当前页结果
   */
  results?: WorkspaceWithStatsOutput[];
  /**
   * 按当前筛选条件命中的工作空间状态统计，基于未分页的完整结果集计算，不受 page / pageSize 影响
   */
  statistics?: WorkspaceStatsOutput;
}

export interface WorkspaceWithStatsOutput {
  /**
   * 应用数量
   */
  appCount?: number;
  /**
   * 创建人
   */
  creator?: string;
  /**
   * 工作空间描述
   */
  description?: string;
  /**
   * 工作空间展示名称
   */
  displayName?: string;
  /**
   * 环境数量
   */
  envCount?: number;
  /**
   * 工作空间 ID
   */
  id?: string;
  /**
   * 工作空间状态
   */
  state?: string;
  /**
   * 更新时间
   */
  updatedAt?: string;
  /**
   * 更新人
   */
  updater?: string;
}

export interface RoleOutput {
  /**
   * 角色名称
   */
  name?: string;
  /**
   * 角色编码
   */
  roleCode?: RoleCode;
}

export type RoleCode = "admin";

export interface RoleBindingOutput {
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 添加人
   */
  creator?: string;
  /**
   * 平台管理员角色编码
   */
  roleCode?: RoleCode;
  /**
   * 更新时间
   */
  updatedAt?: string;
  /**
   * 更新人
   */
  updater?: string;
  /**
   * 平台管理员用户名
   */
  username?: string;
}
