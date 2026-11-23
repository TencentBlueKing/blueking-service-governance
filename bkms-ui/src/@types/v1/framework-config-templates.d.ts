/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：framework-config-templates

export interface ListWorkspaceFrameworkConfigTemplatesRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 框架：trpc、taf
   */
  framework?: string;
  /**
   * 语言：go、cpp
   */
  language?: string;
}

export type CreateWorkspaceFrameworkConfigTemplateRequest = CreateTemplateInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
};

export interface GetWorkspaceFrameworkConfigTemplateRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 模板 ID，内置模板为 builtin-{framework}-{language}
   */
  templateID: string;
}

export type UpdateWorkspaceFrameworkConfigTemplateRequest = UpdateTemplateInput & {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 模板 ID
   */
  templateID: string;
};

export interface DeleteWorkspaceFrameworkConfigTemplateRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 模板 ID
   */
  templateID: string;
}

export interface SetWorkspaceFrameworkConfigTemplateDefaultRequest {
  /**
   * 工作空间 ID
   */
  workspaceID: string;
  /**
   * 模板 ID，内置模板为 builtin-{framework}-{language}
   */
  templateID: string;
}

export interface ListTemplatesOutput {
  data?: TemplateOutputObj[];
}

export interface CreateTemplateInput {
  /**
   * 框架配置文件内容
   */
  content: string;
  /**
   * 框架配置文件名，规则与应用配置文件名一致
   */
  fileName: string;
  /**
   * 框架：trpc、taf
   */
  framework: string;
  /**
   * 是否设为创建应用时默认使用的模板
   */
  isDefault?: boolean;
  /**
   * 语言：go、cpp
   */
  language: string;
  /**
   * 模板名称，同一框架、同一语言下不能重复，不能为「平台默认」
   */
  name: string;
}

export interface TemplateOutput {
  data?: TemplateOutputObj;
}

export interface UpdateTemplateInput {
  /**
   * 框架配置文件内容
   */
  content: string;
  /**
   * 框架配置文件名，规则与应用配置文件名一致
   */
  fileName: string;
  /**
   * 是否设为创建应用时默认使用的模板；不传不修改，取消默认后回落到平台默认模板，TAF 没有平台默认模板
   */
  isDefault?: boolean;
  /**
   * 模板名称，同一框架、同一语言下不能重复，不能为「平台默认」
   */
  name: string;
}

export interface EmptyOutput {
}

export interface TemplateOutputObj {
  /**
   * 是否为平台内置模板，内置模板只读
   */
  builtin?: boolean;
  /**
   * 框架配置文件内容，列表接口不返回
   */
  content?: string;
  /**
   * 创建时间，内置模板为空
   */
  createdAt?: string;
  /**
   * 创建人，内置模板为空
   */
  creator?: string;
  /**
   * 框架配置文件名
   */
  fileName?: string;
  /**
   * 框架
   */
  framework?: string;
  /**
   * 模板 ID，内置模板为 builtin-{framework}-{language}
   */
  id?: string;
  /**
   * 是否为所在框架、语言的默认模板
   */
  isDefault?: boolean;
  /**
   * 语言
   */
  language?: string;
  /**
   * 模板名称
   */
  name?: string;
  /**
   * 更新时间，内置模板为空
   */
  updatedAt?: string;
  /**
   * 最近修改人，内置模板为空
   */
  updater?: string;
}
