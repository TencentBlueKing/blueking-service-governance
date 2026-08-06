/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { GetUserStatisticsRequest, UserStatisticsOutputObj, ListWorkspacesRequest, WorkspaceInfoOutputObj, CreateWorkspaceRequest, WorkspaceDetailOutputObj, ListWorkspacesOverviewRequest, WorkspaceWithAppsOutputObj, GetWorkspaceRequest, DeleteWorkspaceRequest, EmptyOutput, UpdateWorkspaceInfoRequest, ListWorkspaceRoleMemberGroupsRequest, RoleMemberGroupOutputObj, AddWorkspaceUserRequest, SetWorkspaceStateRequest, RemoveWorkspaceUserRequest } from '~/@types/v1/workspace';

export const WorkspaceService = {
  /**
   * 获取用户统计信息
   *
   * @method GET
   * @path /user-statistics
   * @tag workspace
   * @response 200 GetUserStatisticsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getUserStatistics: async <Request extends GetUserStatisticsRequest = GetUserStatisticsRequest, ResponseData = UserStatisticsOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/user-statistics')(params, config),
  /**
   * 获取工作空间列表
   *
   * [bkms-cli 使用] 避免破坏性修改
   *
   * @method GET
   * @path /workspaces
   * @tag workspace
   * @param keyword query string 搜索关键词
   * @response 200 ListWorkspacesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listWorkspaces: async <Request extends ListWorkspacesRequest = ListWorkspacesRequest, ResponseData = WorkspaceInfoOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces')(params, config),
  /**
   * 创建工作空间
   *
   * 1. 级联创建/绑定蓝盾、BCS、镜像仓库、监控、日志等\n2. 初始化用户权限\n3. 写入 DB workspace\n4. 创建默认环境
   *
   * @method POST
   * @path /workspaces
   * @tag workspace
   * @param body body CreateWorkspaceInput required 创建工作空间请求
   * @response 200 CreateWorkspaceOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createWorkspace: async <Request extends CreateWorkspaceRequest = CreateWorkspaceRequest, ResponseData = WorkspaceDetailOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces')(params, config),
  /**
   * 获取工作空间概览列表
   *
   * 含应用信息，按最近操作时间排序
   *
   * @method GET
   * @path /workspaces-overview
   * @tag workspace
   * @param limit query number required 返回的工作空间数量上限
   * @response 200 ListWorkspacesOverviewOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listWorkspacesOverview: async <Request extends ListWorkspacesOverviewRequest = ListWorkspacesOverviewRequest, ResponseData = WorkspaceWithAppsOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces-overview')(params, config),
  /**
   * 获取指定工作空间的详细信息，包括基本信息、镜像仓库、蓝鲸关联系统信息等
   *
   * [bkms-cli 使用] 避免破坏性修改
   *
   * @method GET
   * @path /workspaces/{workspaceID}
   * @tag workspace
   * @param workspaceID path string required 工作空间 ID
   * @response 200 GetWorkspaceOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getWorkspace: async <Request extends GetWorkspaceRequest = GetWorkspaceRequest, ResponseData = WorkspaceDetailOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}')(params, config),
  /**
   * 删除工作空间
   *
   * @method DELETE
   * @path /workspaces/{workspaceID}
   * @tag workspace
   * @param workspaceID path string required 工作空间 ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteWorkspace: async <Request extends DeleteWorkspaceRequest = DeleteWorkspaceRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/workspaces/{workspaceID}')(params, config),
  /**
   * 更新工作空间信息
   *
   * @method PUT
   * @path /workspaces/{workspaceID}/info
   * @tag workspace
   * @param workspaceID path string required 工作空间 ID
   * @param body body UpdateWorkspaceInfoInput required 更新工作空间信息请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateWorkspaceInfo: async <Request extends UpdateWorkspaceInfoRequest = UpdateWorkspaceInfoRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/workspaces/{workspaceID}/info')(params, config),
  /**
   * 获取工作空间下角色成员组列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/role-member-groups
   * @tag workspace
   * @param workspaceID path string required 工作空间 ID
   * @response 200 ListWorkspaceRoleMemberGroupsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listWorkspaceRoleMemberGroups: async <Request extends ListWorkspaceRoleMemberGroupsRequest = ListWorkspaceRoleMemberGroupsRequest, ResponseData = RoleMemberGroupOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/role-member-groups')(params, config),
  /**
   * 授予用户工作空间下角色身份
   *
   * @method POST
   * @path /workspaces/{workspaceID}/roles/{roleCode}/users
   * @tag workspace
   * @param workspaceID path string required 工作空间 ID
   * @param roleCode path string required 角色 Code
   * @param body body AddWorkspaceUserInput required 添加用户请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  addWorkspaceUser: async <Request extends AddWorkspaceUserRequest = AddWorkspaceUserRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/roles/{roleCode}/users')(params, config),
  /**
   * 设置工作空间状态
   *
   * @method PATCH
   * @path /workspaces/{workspaceID}/state
   * @tag workspace
   * @param workspaceID path string required 工作空间 ID
   * @param body body SetWorkspaceStateInput required 设置工作空间状态请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  setWorkspaceState: async <Request extends SetWorkspaceStateRequest = SetWorkspaceStateRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.patch<Request, ResponseData>('/workspaces/{workspaceID}/state')(params, config),
  /**
   * 移除用户工作空间下角色身份
   *
   * @method DELETE
   * @path /workspaces/{workspaceID}/users/{userID}
   * @tag workspace
   * @param workspaceID path string required 工作空间 ID
   * @param userID path string required 用户 ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  removeWorkspaceUser: async <Request extends RemoveWorkspaceUserRequest = RemoveWorkspaceUserRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/workspaces/{workspaceID}/users/{userID}')(params, config),
};
