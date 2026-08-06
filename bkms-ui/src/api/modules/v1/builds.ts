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
import type { UpdateBuildConfigRequest, BuildConfigOutputObj, ListBuildRecordsRequest, PaginatedBuildRecordOutputObjs, CreateBuildRequest, BuildRecordOutputObj, DownloadBuildLogsRequest, StreamBuildLogsRequest, GetRecommendedImageTagRequest } from '~/@types/v1/builds';

export const BuildsService = {
  /**
   * 更新应用构建配置
   *
   * @method PUT
   * @path /apps/{appID}/build-configs
   * @tag builds
   * @param appID path string required 应用 ID
   * @param body body UpdateBuildConfigInput required 更新构建配置请求
   * @response 200 UpdateBuildConfigOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  updateBuildConfig: async <Request extends UpdateBuildConfigRequest = UpdateBuildConfigRequest, ResponseData = BuildConfigOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/build-configs')(params, config),
  /**
   * 获取应用构建记录列表
   *
   * @method GET
   * @path /apps/{appID}/builds
   * @tag builds
   * @param appID path string required 应用 ID
   * @param keyword query string 搜索关键字
   * @param page query number required 分页参数：页码，从 1 开始
   * @param pageSize query number required 分页参数：每页数量，支持 5/10/20/50/100
   * @response 200 ListBuildRecordsOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  listBuildRecords: async <Request extends ListBuildRecordsRequest = ListBuildRecordsRequest, ResponseData = PaginatedBuildRecordOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/builds')(params, config),
  /**
   * 开始构建应用
   *
   * @method POST
   * @path /apps/{appID}/builds
   * @tag builds
   * @param appID path string required 应用 ID
   * @param body body CreateBuildInput required 创建构建请求
   * @response 200 CreateBuildOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  createBuild: async <Request extends CreateBuildRequest = CreateBuildRequest, ResponseData = BuildRecordOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/builds')(params, config),
  /**
   * 下载应用构建日志
   *
   * @method GET
   * @path /apps/{appID}/builds/{buildID}/logs/download
   * @tag builds
   * @param appID path string required 应用 ID
   * @param buildID path string required 蓝盾构建 ID
   * @response 200 string binary log stream
   * @response 400 GinErrorOutput Bad Request
   */
  downloadBuildLogs: async <Request extends DownloadBuildLogsRequest = DownloadBuildLogsRequest, ResponseData = string>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/builds/{buildID}/logs/download')(params, config),
  /**
   * 流式推送应用构建日志
   *
   * @method GET
   * @path /apps/{appID}/builds/{buildID}/logs/stream
   * @tag builds
   * @param appID path string required 应用 ID
   * @param buildID path string required 蓝盾构建 ID
   * @response 200 string SSE event stream
   * @response 400 GinErrorOutput Bad Request
   */
  streamBuildLogs: async <Request extends StreamBuildLogsRequest = StreamBuildLogsRequest, ResponseData = string>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/builds/{buildID}/logs/stream')(params, config),
  /**
   * 获取应用推荐的镜像 Tag
   *
   * @method GET
   * @path /apps/{appID}/recommended-image-tag
   * @tag builds
   * @param appID path string required 应用 ID
   * @param branch query string 分支/Tag（仅 custom 类型使用）
   * @response 200 GetRecommendedImageTagOutput OK
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  getRecommendedImageTag: async <Request extends GetRecommendedImageTagRequest = GetRecommendedImageTagRequest, ResponseData = string>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/recommended-image-tag')(params, config),
};
