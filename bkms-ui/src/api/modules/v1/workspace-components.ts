/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListWorkspaceComponentsRequest, WorkspaceComponentOutputObj, CreateWorkspaceComponentRequest, WorkspaceComponentNameOutputObj, DeleteWorkspaceComponentRequest, EmptyOutput, PatchWorkspaceComponentRequest } from '~/@types/v1/workspace-components';

export const WorkspaceComponentsService = {
  /**
   * 获取工作空间组件列表
   *
   * @method GET
   * @path /workspaces/{workspaceID}/components
   * @tag workspace-components
   * @param workspaceID path string required 工作空间 ID
   * @response 200 ListWorkspaceComponentsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listWorkspaceComponents: async <Request extends ListWorkspaceComponentsRequest = ListWorkspaceComponentsRequest, ResponseData = WorkspaceComponentOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/components')(params, config),
  /**
   * 添加工作空间组件
   *
   * @method POST
   * @path /workspaces/{workspaceID}/components
   * @tag workspace-components
   * @param workspaceID path string required 工作空间 ID
   * @param body body CreateWorkspaceComponentInput required 添加工作空间组件请求
   * @response 200 CreateWorkspaceComponentOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createWorkspaceComponent: async <Request extends CreateWorkspaceComponentRequest = CreateWorkspaceComponentRequest, ResponseData = WorkspaceComponentNameOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/components')(params, config),
  /**
   * 删除工作空间组件
   *
   * @method DELETE
   * @path /workspaces/{workspaceID}/components/{compName}
   * @tag workspace-components
   * @param workspaceID path string required 工作空间 ID
   * @param compName path string required 组件名称
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteWorkspaceComponent: async <Request extends DeleteWorkspaceComponentRequest = DeleteWorkspaceComponentRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/workspaces/{workspaceID}/components/{compName}')(params, config),
  /**
   * 更新工作空间组件
   *
   * @method PATCH
   * @path /workspaces/{workspaceID}/components/{compName}
   * @tag workspace-components
   * @param workspaceID path string required 工作空间 ID
   * @param compName path string required 组件名称
   * @param body body PatchWorkspaceComponentInput required 更新工作空间组件请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  patchWorkspaceComponent: async <Request extends PatchWorkspaceComponentRequest = PatchWorkspaceComponentRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.patch<Request, ResponseData>('/workspaces/{workspaceID}/components/{compName}')(params, config),
};
