/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListRoleBindingsRequest, RoleBindingOutput, AssignRolesRequest, ListRolesRequest, RoleOutput, RevokeRoleRequest, ListPlatWorkspacesRequest, PaginatedWorkspaceOutput, GetPlatWorkspaceStatsRequest, WorkspaceStatsOutput, GetPlatWorkspaceRequest, WorkspaceInfoOutput, GetWorkspaceRoleStatusRequest, RoleStatusOutput, GrantWorkspaceAdminRequest, RevokeWorkspaceAdminRequest } from '~/@types/v1/platmgt';

export const PlatmgtService = {
  /**
   * 查询平台管理员角色绑定列表
   *
   * @method GET
   * @path /plat-mgt/admins
   * @tag platmgt
   * @param keyword query string 用户名关键字
   * @response 200 ListRoleBindingsResponse OK
   * @response 400 GinErrorOutput Bad Request
   * @response 403 GinErrorOutput Forbidden
   * @response 500 GinErrorOutput Internal Server Error
   */
  listRoleBindings: async <Request extends ListRoleBindingsRequest = ListRoleBindingsRequest, ResponseData = RoleBindingOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/plat-mgt/admins')(params, config),
  /**
   * 批量授予平台管理员角色（已存在则跳过）
   *
   * @method POST
   * @path /plat-mgt/admins
   * @tag platmgt
   * @param input body AssignRolesInput required 平台管理员角色批量授权参数
   * @response 204 unknown No Content
   * @response 400 GinErrorOutput Bad Request
   * @response 403 GinErrorOutput Forbidden
   * @response 500 GinErrorOutput Internal Server Error
   */
  assignRoles: async <Request extends AssignRolesRequest = AssignRolesRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/plat-mgt/admins')(params, config),
  /**
   * 查询可分配的平台管理员角色列表
   *
   * @method GET
   * @path /plat-mgt/admins/roles
   * @tag platmgt
   * @response 200 ListRolesResponse OK
   * @response 403 GinErrorOutput Forbidden
   * @response 500 GinErrorOutput Internal Server Error
   */
  listRoles: async <Request extends ListRolesRequest = ListRolesRequest, ResponseData = RoleOutput[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/plat-mgt/admins/roles')(params, config),
  /**
   * 撤销平台管理员角色
   *
   * @method DELETE
   * @path /plat-mgt/admins/{username}
   * @tag platmgt
   * @param username path string required 平台管理员用户名
   * @response 204 unknown No Content
   * @response 400 GinErrorOutput Bad Request
   * @response 403 GinErrorOutput Forbidden
   * @response 500 GinErrorOutput Internal Server Error
   */
  revokeRole: async <Request extends RevokeRoleRequest = RevokeRoleRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/plat-mgt/admins/{username}')(params, config),
  /**
   * 查询平台空间列表
   *
   * @method GET
   * @path /plat-mgt/workspaces
   * @tag platmgt
   * @param keyword query string 搜索关键词，匹配空间 ID / 空间名称
   * @param state query string 空间状态过滤：Ready / Processing / Disabled
   * @param sortBy query string 排序字段：id / displayName / updatedAt
   * @param sortOrder query string 排序方向：asc / desc
   * @param page query number required 页码，从 1 开始
   * @param pageSize query number required 每页数量，支持 5/10/20/50/100
   * @response 200 ListWorkspacesResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  listPlatWorkspaces: async <Request extends ListPlatWorkspacesRequest = ListPlatWorkspacesRequest, ResponseData = PaginatedWorkspaceOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/plat-mgt/workspaces')(params, config),
  /**
   * 查询平台工作空间数据统计
   *
   * @method GET
   * @path /plat-mgt/workspaces/statistics
   * @tag platmgt
   * @response 200 WorkspaceStatsResponse OK
   * @response 403 GinErrorOutput Forbidden
   */
  getPlatWorkspaceStats: async <Request extends GetPlatWorkspaceStatsRequest = GetPlatWorkspaceStatsRequest, ResponseData = WorkspaceStatsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/plat-mgt/workspaces/statistics')(params, config),
  /**
   * 查询平台空间详情
   *
   * @method GET
   * @path /plat-mgt/workspaces/{workspaceID}
   * @tag platmgt
   * @param workspaceID path string required 工作空间 ID
   * @response 200 GetWorkspaceResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  getPlatWorkspace: async <Request extends GetPlatWorkspaceRequest = GetPlatWorkspaceRequest, ResponseData = WorkspaceInfoOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/plat-mgt/workspaces/{workspaceID}')(params, config),
  /**
   * 查询指定用户在目标空间是否拥有指定角色
   *
   * @method GET
   * @path /plat-mgt/workspaces/{workspaceID}/admins
   * @tag platmgt
   * @param workspaceID path string required 工作空间 ID
   * @param roleCode query string required 角色 Code
   * @param username query string required 用户名
   * @response 200 GetRoleStatusResponse OK
   * @response 400 GinErrorOutput Bad Request
   */
  getWorkspaceRoleStatus: async <Request extends GetWorkspaceRoleStatusRequest = GetWorkspaceRoleStatusRequest, ResponseData = RoleStatusOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/plat-mgt/workspaces/{workspaceID}/admins')(params, config),
  /**
   * 为当前用户授予目标空间管理员身份
   *
   * @method POST
   * @path /plat-mgt/workspaces/{workspaceID}/admins
   * @tag platmgt
   * @param workspaceID path string required 工作空间 ID
   * @param body body GrantAdminInput required 管理员授权参数
   * @response 204 unknown No Content
   * @response 400 GinErrorOutput Bad Request
   */
  grantWorkspaceAdmin: async <Request extends GrantWorkspaceAdminRequest = GrantWorkspaceAdminRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/plat-mgt/workspaces/{workspaceID}/admins')(params, config),
  /**
   * 退出目标空间管理员身份
   *
   * @method DELETE
   * @path /plat-mgt/workspaces/{workspaceID}/admins
   * @tag platmgt
   * @param workspaceID path string required 工作空间 ID
   * @response 204 unknown No Content
   * @response 400 GinErrorOutput Bad Request
   * @response 403 GinErrorOutput Forbidden
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  revokeWorkspaceAdmin: async <Request extends RevokeWorkspaceAdminRequest = RevokeWorkspaceAdminRequest, ResponseData = unknown>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/plat-mgt/workspaces/{workspaceID}/admins')(params, config),
};
