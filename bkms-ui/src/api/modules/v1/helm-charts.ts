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
import type { ListAppHelmChartsRequest, PaginatedAppHelmChartsOutputObjs, ListHelmChartBuildRecordsRequest, PaginatedHelmChartBuildRecordOutputObjs, CreateHelmChartBuildRequest, CreateHelmChartBuildOutputObj, DownloadHelmChartBuildLogsRequest, StreamHelmChartBuildLogsRequest, GetHelmChartSemverRequest, GetHelmChartSemverOutputObj, ListChartVersionsRequest, ChartVersionOutputObj, GetHelmChartFilesRequest, GetHelmChartFilesOutputObj, GetValuesFileRequest } from '~/@types/v1/helm-charts';

export const HelmChartsService = {
  /**
   * 获取 Helm Chart 制品列表
   *
   * @method GET
   * @path /apps/{appID}/charts
   * @tag helm-charts
   * @param appID path string required 应用 ID
   * @param keyword query string 搜索关键字，按版本号模糊匹配
   * @param page query number required 分页页码（从 1 开始）
   * @param pageSize query number required 分页大小，可选值：5/10/20/50/100
   * @response 200 ListAppHelmChartsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listAppHelmCharts: async <Request extends ListAppHelmChartsRequest = ListAppHelmChartsRequest, ResponseData = PaginatedAppHelmChartsOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/charts')(params, config),
  /**
   * 获取 Helm Chart 构建记录列表
   *
   * @method GET
   * @path /apps/{appID}/charts/builds
   * @tag helm-charts
   * @param appID path string required 应用 ID
   * @param keyword query string 搜索关键字，按版本号 / 构建号 / 操作人模糊匹配
   * @param page query number required 分页页码（从 1 开始）
   * @param pageSize query number required 分页大小，可选值：5/10/20/50/100
   * @response 200 ListHelmChartBuildRecordsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listHelmChartBuildRecords: async <Request extends ListHelmChartBuildRecordsRequest = ListHelmChartBuildRecordsRequest, ResponseData = PaginatedHelmChartBuildRecordOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/charts/builds')(params, config),
  /**
   * 触发 Helm Chart 构建
   *
   * @method POST
   * @path /apps/{appID}/charts/builds
   * @tag helm-charts
   * @param appID path string required 应用 ID
   * @param body body CreateHelmChartBuildInput required 触发 Helm Chart 构建请求
   * @response 200 CreateHelmChartBuildOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createHelmChartBuild: async <Request extends CreateHelmChartBuildRequest = CreateHelmChartBuildRequest, ResponseData = CreateHelmChartBuildOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/charts/builds')(params, config),
  /**
   * 下载 Helm Chart 构建日志
   *
   * @method GET
   * @path /apps/{appID}/charts/builds/{buildID}/logs/download
   * @tag helm-charts
   * @param appID path string required 应用 ID
   * @param buildID path string required 蓝盾构建 ID
   * @response 200 string binary log stream
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  downloadHelmChartBuildLogs: async <Request extends DownloadHelmChartBuildLogsRequest = DownloadHelmChartBuildLogsRequest, ResponseData = string>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/charts/builds/{buildID}/logs/download')(params, config),
  /**
   * 流式推送 Helm Chart 构建日志
   *
   * @method GET
   * @path /apps/{appID}/charts/builds/{buildID}/logs/stream
   * @tag helm-charts
   * @param appID path string required 应用 ID
   * @param buildID path string required 蓝盾构建 ID
   * @response 200 string SSE event stream
   * @response 400 GinErrorOutput Bad Request
   * @response 404 GinErrorOutput Not Found
   * @response 500 GinErrorOutput Internal Server Error
   */
  streamHelmChartBuildLogs: async <Request extends StreamHelmChartBuildLogsRequest = StreamHelmChartBuildLogsRequest, ResponseData = string>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/charts/builds/{buildID}/logs/stream')(params, config),
  /**
   * 查询 Helm Chart semver counter 当前值
   *
   * @method GET
   * @path /apps/{appID}/charts/semver
   * @tag helm-charts
   * @param appID path string required 应用 ID
   * @param bumpType query string semver 递增段类型，可选值：patch/minor/major
   * @response 200 GetHelmChartSemverOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getHelmChartSemver: async <Request extends GetHelmChartSemverRequest = GetHelmChartSemverRequest, ResponseData = GetHelmChartSemverOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/charts/semver')(params, config),
  /**
   * 获取 Helm Chart 版本列表
   *
   * @method GET
   * @path /apps/{appID}/charts/versions
   * @tag helm-charts
   * @param appID path string required 应用 ID
   * @response 200 ListChartVersionsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listChartVersions: async <Request extends ListChartVersionsRequest = ListChartVersionsRequest, ResponseData = ChartVersionOutputObj[]>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/charts/versions')(params, config),
  /**
   * 获取指定 Helm Chart 版本的文件树
   *
   * @method GET
   * @path /apps/{appID}/charts/{chartVersion}/files
   * @tag helm-charts
   * @param appID path string required 应用 ID
   * @param chartVersion path string required Chart 版本号
   * @response 200 GetHelmChartFilesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getHelmChartFiles: async <Request extends GetHelmChartFilesRequest = GetHelmChartFilesRequest, ResponseData = GetHelmChartFilesOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/charts/{chartVersion}/files')(params, config),
  /**
   * 获取 values 文件
   *
   * @method GET
   * @path /apps/{appID}/charts/{chartVersion}/valuesfile
   * @tag helm-charts
   * @param appID path string required 应用 ID
   * @param chartVersion path string required Chart 版本号
   * @response 200 GetValuesFileOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getValuesFile: async <Request extends GetValuesFileRequest = GetValuesFileRequest, ResponseData = string>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/charts/{chartVersion}/valuesfile')(params, config),
};
