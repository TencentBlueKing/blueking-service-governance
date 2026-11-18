/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListWorkspaceFrameworkConfigTemplatesRequest, TemplateOutputObj, CreateWorkspaceFrameworkConfigTemplateRequest, GetWorkspaceFrameworkConfigTemplateRequest, UpdateWorkspaceFrameworkConfigTemplateRequest, DeleteWorkspaceFrameworkConfigTemplateRequest, EmptyOutput, SetWorkspaceFrameworkConfigTemplateDefaultRequest } from '~/@types/v1/framework-config-templates';

export const FrameworkConfigTemplatesService = {
  /**
   * 查询工作空间框架配置模板列表
   *
   * 返回平台内置模板与空间自定义模板，每个框架、语言先列内置模板（TAF 没有内置模板）；不返回模板内容
   *
   * @method GET
   * @path /workspaces/{workspaceID}/framework-config-templates
   * @tag framework-config-templates
   * @param workspaceID path string required 工作空间 ID
   * @param framework query string 框架：trpc、taf
   * @param language query string 语言：go、cpp
   * @response 200 ListTemplatesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listWorkspaceFrameworkConfigTemplates: async <Request extends ListWorkspaceFrameworkConfigTemplatesRequest = ListWorkspaceFrameworkConfigTemplatesRequest, ResponseData = TemplateOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/framework-config-templates')(params, config),
  /**
   * 新建工作空间框架配置模板
   *
   * @method POST
   * @path /workspaces/{workspaceID}/framework-config-templates
   * @tag framework-config-templates
   * @param workspaceID path string required 工作空间 ID
   * @param body body CreateTemplateInput required 框架配置模板
   * @response 200 TemplateOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 409 GinErrorOutput 同一框架、语言下模板名称已存在，或并发设置了其它默认模板
   */
  createWorkspaceFrameworkConfigTemplate: async <Request extends CreateWorkspaceFrameworkConfigTemplateRequest = CreateWorkspaceFrameworkConfigTemplateRequest, ResponseData = TemplateOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/workspaces/{workspaceID}/framework-config-templates')(params, config),
  /**
   * 查询工作空间框架配置模板详情
   *
   * @method GET
   * @path /workspaces/{workspaceID}/framework-config-templates/{templateID}
   * @tag framework-config-templates
   * @param workspaceID path string required 工作空间 ID
   * @param templateID path string required 模板 ID，内置模板为 builtin-{framework}-{language}
   * @response 200 TemplateOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getWorkspaceFrameworkConfigTemplate: async <Request extends GetWorkspaceFrameworkConfigTemplateRequest = GetWorkspaceFrameworkConfigTemplateRequest, ResponseData = TemplateOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/workspaces/{workspaceID}/framework-config-templates/{templateID}')(params, config),
  /**
   * 编辑工作空间框架配置模板
   *
   * 框架与语言不可修改；平台内置模板只读
   *
   * @method PUT
   * @path /workspaces/{workspaceID}/framework-config-templates/{templateID}
   * @tag framework-config-templates
   * @param workspaceID path string required 工作空间 ID
   * @param templateID path string required 模板 ID
   * @param body body UpdateTemplateInput required 框架配置模板
   * @response 200 TemplateOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 409 GinErrorOutput 同一框架、语言下模板名称已存在，或并发设置了其它默认模板
   */
  updateWorkspaceFrameworkConfigTemplate: async <Request extends UpdateWorkspaceFrameworkConfigTemplateRequest = UpdateWorkspaceFrameworkConfigTemplateRequest, ResponseData = TemplateOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/workspaces/{workspaceID}/framework-config-templates/{templateID}')(params, config),
  /**
   * 删除工作空间框架配置模板
   *
   * 删除默认模板后，该框架、语言回落到平台内置模板（TAF 没有内置模板，则不再有默认模板）；平台内置模板不能删除
   *
   * @method DELETE
   * @path /workspaces/{workspaceID}/framework-config-templates/{templateID}
   * @tag framework-config-templates
   * @param workspaceID path string required 工作空间 ID
   * @param templateID path string required 模板 ID
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteWorkspaceFrameworkConfigTemplate: async <Request extends DeleteWorkspaceFrameworkConfigTemplateRequest = DeleteWorkspaceFrameworkConfigTemplateRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/workspaces/{workspaceID}/framework-config-templates/{templateID}')(params, config),
  /**
   * 设为默认框架配置模板
   *
   * 创建应用时默认使用该模板；对平台内置模板调用表示恢复使用平台默认模板
   *
   * @method PUT
   * @path /workspaces/{workspaceID}/framework-config-templates/{templateID}/default
   * @tag framework-config-templates
   * @param workspaceID path string required 工作空间 ID
   * @param templateID path string required 模板 ID，内置模板为 builtin-{framework}-{language}
   * @response 200 TemplateOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 409 GinErrorOutput 并发设置了其它默认模板
   */
  setWorkspaceFrameworkConfigTemplateDefault: async <Request extends SetWorkspaceFrameworkConfigTemplateDefaultRequest = SetWorkspaceFrameworkConfigTemplateDefaultRequest, ResponseData = TemplateOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/workspaces/{workspaceID}/framework-config-templates/{templateID}/default')(params, config),
};
