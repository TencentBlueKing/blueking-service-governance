/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：workspace

export interface GetUserStatisticsRequest {
}

export interface ListWorkspacesRequest {
  /**
   * 搜索关键词
   */
  keyword?: string;
}

export type CreateWorkspaceRequest = CreateWorkspaceInput;

export interface ListWorkspacesOverviewRequest {
  /**
   * 返回的工作空间数量上限
   */
  limit: number;
}

export interface GetWorkspaceRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export interface DeleteWorkspaceRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export type UpdateWorkspaceInfoRequest = UpdateWorkspaceInfoInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export interface ListWorkspaceRoleMemberGroupsRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
}

export type AddWorkspaceUserRequest = AddWorkspaceUserInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 角色 Code
   */
  roleCode: string;
};

export type SetWorkspaceStateRequest = SetWorkspaceStateInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export interface RemoveWorkspaceUserRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 用户 ID
   */
  userID: string;
}

export interface GetUserStatisticsOutput {
  data?: UserStatisticsOutputObj;
}

export interface ListWorkspacesOutput {
  data?: WorkspaceInfoOutputObj[];
}

export interface CreateWorkspaceInput {
  /**
   * bkccID 业务 ID
   * 创建时，新建容器项目，必填。
   * 创建时，绑定已有容器项目，无需填写。
   */
  bkCCBizID?: number;
  /**
   * 蓝盾项目 ID;
   * 创建时，新建容器项目，无需填写
   * 创建时，绑定已有容器项目，必填.
   */
  bkCIProjectID?: string;
  /**
   * 描述信息（0-512 字符）
   */
  description?: string;
  /**
   * 展示用名称，一般为中文名（1-32 字符）
   */
  displayName: string;
  /**
   * 工作空间 ID, 1-27 字符的空间 ID，由小写字母、数字、中划线组成，以小写字母开头，不能以中划线结尾
   */
  id: string;
  /**
   * 镜像仓库信息, 传入时代表绑定已有镜像仓库, 不传入则在容器项目的制品库中创建默认镜像仓库
   */
  imageRegistry?: ImageRegistryInput;
  /**
   * 空间管理员用户, 创建者即使不在列表中也会默认加入管理员
   */
  managers?: string[];
}

export interface CreateWorkspaceOutput {
  data?: WorkspaceDetailOutputObj;
}

export interface ListWorkspacesOverviewOutput {
  data?: WorkspaceWithAppsOutputObj[];
}

export interface GetWorkspaceOutput {
  data?: WorkspaceDetailOutputObj;
}

export interface EmptyOutput {
}

export interface UpdateWorkspaceInfoInput {
  /**
   * 描述信息（0-512 字符）
   */
  description?: string;
  /**
   * 展示用名称，一般为中文名（1-32 字符）
   */
  displayName: string;
}

export interface ListWorkspaceRoleMemberGroupsOutput {
  data?: RoleMemberGroupOutputObj[];
}

export interface AddWorkspaceUserInput {
  /**
   * 用户列表
   */
  userIDs: string[];
}

export interface SetWorkspaceStateInput {
  /**
   * 工作空间状态
   */
  state: "Ready" | "Disabled";
}

export interface RoleMemberGroupOutputObj {
  /**
   * 用户组成员
   */
  members?: string[];
  /**
   * 角色 Code
   */
  roleCode?: string;
  /**
   * 角色 ID
   */
  roleID?: string;
  /**
   * 角色名称
   */
  roleName?: string;
  /**
   * 用户组 ID
   */
  userGroupID?: string;
}

export interface WorkspaceDetailOutputObj {
  /**
   * 关联的蓝鲸体系系统的信息
   */
  bkSystems?: BkSystemsOutputObj;
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 创建人
   */
  creator?: string;
  /**
   * 描述信息
   */
  description?: string;
  /**
   * 展示用名称，一般为中文名
   */
  displayName?: string;
  /**
   * 工作空间唯一标识
   */
  id?: string;
  /**
   * 镜像仓库信息
   */
  imageRegistry?: ImageRegistryOutputObj;
  /**
   * 使用的镜像仓库类型
   */
  imageRegistryType?: string;
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

export interface BkSystemsOutputObj {
  /**
   * 蓝鲸容器服务（BCS）项目 Code, 可读字符串, 如 bkce
   */
  bkBCSProjectCode?: string;
  /**
   * 蓝鲸容器服务（BCS）项目 ID
   */
  bkBCSProjectID?: string;
  /**
   * bkcc 业务 ID
   */
  bkCCBizID?: string;
  /**
   * 蓝盾项目 ID; 创建时不传代表创建默认蓝盾项目
   */
  bkCIProjectID?: string;
  /**
   * 蓝盾项目 UID, 32 位字符串
   */
  bkCIProjectUID?: string;
  /**
   * 蓝鲸日志平台项目 ID
   */
  bkLogProjectID?: string;
  /**
   * 蓝鲸监控平台项目 ID
   */
  bkMonitorProjectID?: string;
  /**
   * 蓝盾制品库项目 ID
   */
  bkRepoProjectID?: string;
  /**
   * 是否绑定已有蓝盾项目
   */
  isBoundExistedBKCIProject?: boolean;
  /**
   * 二级业务 ID
   */
  level2BizID?: string;
  /**
   * 运营产品 ID
   */
  obsProductID?: string;
  /**
   * 运营产品名称
   */
  obsProductName?: string;
}

export interface ImageRegistryOutputObj {
  /**
   * 镜像仓库密码
   */
  password?: string;
  /**
   * 镜像仓库地址
   */
  registry?: string;
  /**
   * 镜像仓库用户名
   */
  username?: string;
}

export interface WorkspaceWithAppsOutputObj {
  /**
   * 该空间下的应用列表（按当前用户操作时间倒序排序）
   */
  apps?: AppInfoOutputObj[];
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 创建人
   */
  creator?: string;
  /**
   * 描述信息
   */
  description?: string;
  /**
   * 展示用名称，一般为中文名
   */
  displayName?: string;
  /**
   * 工作空间唯一标识
   */
  id?: string;
  /**
   * 工作空间最近操作时间（来自当前用户的审计日志）
   */
  lastOperatedAt?: string;
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

export interface AppInfoOutputObj {
  /**
   * 创建人
   */
  creator?: string;
  /**
   * 应用部署的环境列表
   */
  deployedEnvs?: AppDeployedEnvOutputObj[];
  /**
   * 展示用名称
   */
  displayName?: string;
  /**
   * 应用 ID
   */
  id?: string;
  /**
   * 应用使用的编程语言（如 go、cpp），仅 trpc 类型应用有值
   */
  language?: string;
  /**
   * 应用最近操作时间（来自审计日志）
   */
  lastOperatedAt?: string;
  /**
   * 应用名称
   */
  name?: string;
  /**
   * 应用类型
   */
  type?: string;
  /**
   * 工作空间 ID
   */
  workspaceID?: string;
}

export interface AppDeployedEnvOutputObj {
  /**
   * 部署状态
   */
  deployStatus?: string;
  /**
   * 环境展示名称
   */
  displayName?: string;
  /**
   * 环境 ID
   */
  id?: string;
  /**
   * 部署的镜像 Tag
   */
  imageTag?: string;
  /**
   * 环境类别（standard / feature）
   */
  kind?: string;
  /**
   * 环境名称（英文标识）
   */
  name?: string;
  /**
   * 泳道名称，空字符串表示默认泳道
   */
  trafficLaneName?: string;
  /**
   * 环境类型（development / test / staging / production）
   */
  type?: string;
}

export interface ImageRegistryInput {
  /**
   * 镜像仓库密码
   */
  password: string;
  /**
   * 镜像仓库地址
   */
  registry: string;
  /**
   * 镜像仓库用户名
   */
  username: string;
}

export interface WorkspaceInfoOutputObj {
  /**
   * 关联的蓝鲸体系系统的信息
   */
  bkSystems?: BkSystemsOutputObj;
  /**
   * 创建时间
   */
  createdAt?: string;
  /**
   * 创建人
   */
  creator?: string;
  /**
   * 描述信息
   */
  description?: string;
  /**
   * 展示用名称，一般为中文名
   */
  displayName?: string;
  /**
   * 工作空间唯一标识
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

export interface UserStatisticsOutputObj {
  /**
   * 用户总应用数量
   */
  appCount?: string;
  /**
   * 用户总环境数量
   */
  envCount?: string;
  /**
   * 用户工作空间数量
   */
  workspaceCount?: string;
  /**
   * 用户工作空间统计
   */
  workspaceStatistics?: UserWorkspaceStatisticsOutputObj[];
}

export interface UserWorkspaceStatisticsOutputObj {
  /**
   * 用户某工作空间下的应用数量
   */
  appCount?: string;
  /**
   * 用户某工作空间下的环境数量
   */
  envCount?: string;
  /**
   * 工作空间 ID
   */
  workspaceID?: string;
}
