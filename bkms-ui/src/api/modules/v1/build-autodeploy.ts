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
import type { CreateTafBuildDeployRequest, BuildRecordOutputObj, CreateTrpcBuildDeployRequest } from '~/@types/v1/build-autodeploy';

export const BuildAutodeployService = {
  /**
   * 触发 TAF 应用构建并自动部署
   *
   * @method POST
   * @path /apps/{appID}/envs/{envName}/taf-build-deploys
   * @tag build-autodeploy
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param body body CreateAppModelBuildDeployInput required 构建自动部署请求
   * @response 200 CreateBuildOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  createTafBuildDeploy: async <Request extends CreateTafBuildDeployRequest = CreateTafBuildDeployRequest, ResponseData = BuildRecordOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/envs/{envName}/taf-build-deploys')(params, config),
  /**
   * 触发 TRPC 应用构建并自动部署
   *
   * @method POST
   * @path /apps/{appID}/envs/{envName}/trpc-build-deploys
   * @tag build-autodeploy
   * @param appID path string required 应用 ID
   * @param envName path string required 环境名称
   * @param body body CreateAppModelBuildDeployInput required 构建自动部署请求
   * @response 200 CreateBuildOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  createTrpcBuildDeploy: async <Request extends CreateTrpcBuildDeployRequest = CreateTrpcBuildDeployRequest, ResponseData = BuildRecordOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/envs/{envName}/trpc-build-deploys')(params, config),
};
