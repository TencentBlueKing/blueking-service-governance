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
import type { CreateAppComponentRequest, AppComponentNameOutputObj, DeleteAppComponentRequest, EmptyOutput, PatchAppComponentRequest } from '~/@types/v1/app-components';

export const AppComponentsService = {
  /**
   * 添加应用组件
   *
   * @method POST
   * @path /apps/{appID}/components
   * @tag app-components
   * @param appID path string required 应用 ID
   * @param body body CreateAppComponentInput required 添加应用组件请求
   * @response 200 CreateAppComponentOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createAppComponent: async <Request extends CreateAppComponentRequest = CreateAppComponentRequest, ResponseData = AppComponentNameOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/components')(params, config),
  /**
   * 删除应用组件
   *
   * @method DELETE
   * @path /apps/{appID}/components/{compName}
   * @tag app-components
   * @param appID path string required 应用 ID
   * @param compName path string required 组件名称
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteAppComponent: async <Request extends DeleteAppComponentRequest = DeleteAppComponentRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/components/{compName}')(params, config),
  /**
   * 更新应用组件
   *
   * @method PATCH
   * @path /apps/{appID}/components/{compName}
   * @tag app-components
   * @param appID path string required 应用 ID
   * @param compName path string required 组件名称
   * @param body body PatchAppComponentInput required 更新应用组件请求
   * @response 200 EmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  patchAppComponent: async <Request extends PatchAppComponentRequest = PatchAppComponentRequest, ResponseData = EmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.patch<Request, ResponseData>('/apps/{appID}/components/{compName}')(params, config),
};
