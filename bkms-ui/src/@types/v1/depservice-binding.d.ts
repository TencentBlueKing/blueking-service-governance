/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// 模块：depservice-binding

export interface ListServiceBindingsRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 依赖服务名，目前仅支持 redis
   */
  serviceName: "redis";
}

export type CreateServiceBindingRequest = CreateBindingInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 依赖服务名，目前仅支持 redis
   */
  serviceName: "redis";
};

export interface GetServiceBindingRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 依赖服务名，目前仅支持 redis
   */
  serviceName: "redis";
  /**
   * 绑定名称
   */
  name: string;
}

export type UpdateServiceBindingRequest = UpdateBindingInput & {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 依赖服务名，目前仅支持 redis
   */
  serviceName: "redis";
  /**
   * 绑定名称
   */
  name: string;
};

export interface DeleteServiceBindingRequest {
  /**
   * 应用 ID
   */
  appID: string;
  /**
   * 依赖服务名，目前仅支持 redis
   */
  serviceName: "redis";
  /**
   * 绑定名称
   */
  name: string;
}

export interface ListBindingsOutput {
  data?: BindingOutputObj[];
}

export interface CreateBindingInput {
  /**
   * 描述
   */
  description?: string;
  /**
   * 环境名 → 实例 ID；允许为空
   */
  envInstanceMap?: Record<string, string>;
  /**
   * 环境变量模板；允许为空
   */
  envVars?: Record<string, string>;
  /**
   * 绑定名称（应用内同服务下唯一）
   */
  name: string;
}

export interface BindingOutput {
  data?: BindingOutputObj;
}

export interface UpdateBindingInput {
  /**
   * 描述
   */
  description?: string;
  /**
   * 环境名 → 实例 ID；省略或空表示清空映射
   */
  envInstanceMap?: Record<string, string>;
  /**
   * 环境变量模板；省略或空表示清空
   */
  envVars?: Record<string, string>;
}

export interface EmptyOutput {
}

export interface BindingOutputObj {
  appID?: string;
  createdAt?: string;
  description?: string;
  envInstanceMap?: Record<string, string>;
  envVars?: Record<string, string>;
  id?: string;
  name?: string;
  serviceName?: string;
  updatedAt?: string;
  workspaceID?: string;
}
