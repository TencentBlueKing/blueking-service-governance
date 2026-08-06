/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

/* eslint-disable */
// gen-api-v1.js 自动生成，请勿手动修改
// 来源：apps/bkms-server/docs/apis/swagger.json
// Swagger：bkms-server Gin API 1.0
// BasePath：/v1
import type { Config } from '~/api/interceptors';
import type { NoInfer } from '~/api/ts-helpers';
import { v1Fetch } from '~/api/clients';
import type { ListComponentDefsRequest, ComponentDefOutputObj, CreateComponentDefRequest, EmptyOutput, GetComponentDefsBuiltinVarsRequest, BuiltinVarOutputObj, PreviewComponentDefRequest, PreviewOutput, DeleteComponentDefRequest, PatchComponentDefRequest } from '~/@types/v1/component-defs';

export const ComponentDefsService = {
  /**
   * 获取组件定义
   *
   * @method GET
   * @path /component-defs
   * @tag component-defs
   * @param scopeWorkspaceID query string 按可使用该组件定义的工作空间 ID 过滤
   * @param managedByWorkspaceID query string 按可管理该组件定义的工作空间 ID 过滤
   * @param keyword query string 搜索关键词
   * @response 200 ListComponentDefsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listComponentDefs: async <Request extends ListComponentDefsRequest = ListComponentDefsRequest, ResponseData = ComponentDefOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/component-defs')(params, config),
  /**
   * 创建组件定义
   *
   * @method POST
   * @path /component-defs
   * @tag component-defs
   * @param body body CreateComponentDefInput required 创建组件定义请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createComponentDef: async <Request extends CreateComponentDefRequest = CreateComponentDefRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/component-defs')(params, config),
  /**
   * 获取组件输出模板系统变量列表
   *
   * @method GET
   * @path /component-defs/builtin-vars
   * @tag component-defs
   * @response 200 ListBuiltinVarsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getComponentDefsBuiltinVars: async <Request extends GetComponentDefsBuiltinVarsRequest = GetComponentDefsBuiltinVarsRequest, ResponseData = BuiltinVarOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/component-defs/builtin-vars')(params, config),
  /**
   * 预览组件定义（试运行）
   *
   * @method POST
   * @path /component-defs/preview
   * @tag component-defs
   * @param body body PreviewComponentDefInput required 预览组件定义请求
   * @response 200 PreviewOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  previewComponentDef: async <Request extends PreviewComponentDefRequest = PreviewComponentDefRequest, ResponseData = PreviewOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/component-defs/preview')(params, config),
  /**
   * 删除组件定义
   *
   * @method DELETE
   * @path /component-defs/{compDefName}
   * @tag component-defs
   * @param compDefName path string required 组件定义名称
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteComponentDef: async <Request extends DeleteComponentDefRequest = DeleteComponentDefRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/component-defs/{compDefName}')(params, config),
  /**
   * 更新组件定义
   *
   * @method PATCH
   * @path /component-defs/{compDefName}
   * @tag component-defs
   * @param compDefName path string required 组件定义名称
   * @param body body PatchComponentDefInput required 更新组件定义请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  patchComponentDef: async <Request extends PatchComponentDefRequest = PatchComponentDefRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.patch<Request, ResponseData>('/component-defs/{compDefName}')(params, config),
};
