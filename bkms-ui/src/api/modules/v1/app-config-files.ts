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
import type { ListAppConfigFileVersionsRequest, PaginatedAppConfigFileVersionOutputObjs, CompareAppConfigFileVersionsRequest, CompareAppConfigFileVersionsOutput, GetAppConfigFileVersionRequest, AppConfigFileVersionOutputObj, DeleteAppConfigFileVersionRequest, AppConfigFileEmptyOutput, RollbackAppConfigFileVersionRequest, AppConfigFileOutputObj, ListAppConfigFilesRequest, ListAppConfigFilesOutput, CreateAppConfigFileRequest, CreateAppConfigFileOutput, UpdateAppConfigFileRequest, UpdateAppConfigFileOutput, DeleteAppConfigFileRequest, UpdateAppConfigFileContentRequest, UpdateAppConfigFileContentOutput, GetAppConfigFileDetailsRequest, GetAppConfigFileDetailsOutput, UpdateAppConfigFileOverlayContentRequest, PreviewOverlayMergeRequest } from '~/@types/v1/app-config-files';

export const AppConfigFilesService = {
  /**
   * 查询应用配置文件版本列表
   *
   * @method GET
   * @path /apps/{appID}/app-config-file/versions
   * @tag app-config-files
   * @param appID path string required 应用 ID
   * @param appConfigFileID query string 应用配置文件 ID
   * @param envName query string 环境名
   * @param name query string 文件名
   * @param version query number 版本号
   * @param creator query string 创建人
   * @param description query string 版本描述
   * @param page query number required 页码，从 1 开始
   * @param pageSize query number required 每页数量，仅支持 5/10/20/50/100
   * @response 200 ListAppConfigFileVersionsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listAppConfigFileVersions: async <Request extends ListAppConfigFileVersionsRequest = ListAppConfigFileVersionsRequest, ResponseData = PaginatedAppConfigFileVersionOutputObjs>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/app-config-file/versions')(params, config),
  /**
   * 对比两个应用配置文件版本
   *
   * @method POST
   * @path /apps/{appID}/app-config-file/versions/compare
   * @tag app-config-files
   * @param appID path string required 应用 ID
   * @param body body CompareAppConfigFileVersionsInput required 对比应用配置文件版本请求
   * @response 200 CompareAppConfigFileVersionsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  compareAppConfigFileVersions: async <Request extends CompareAppConfigFileVersionsRequest = CompareAppConfigFileVersionsRequest, ResponseData = CompareAppConfigFileVersionsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/app-config-file/versions/compare')(params, config),
  /**
   * 查询应用配置文件某个版本详情
   *
   * @method GET
   * @path /apps/{appID}/app-config-file/versions/{id}
   * @tag app-config-files
   * @param appID path string required 应用 ID
   * @param id path string required 版本记录 ID
   * @response 200 GetAppConfigFileVersionOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getAppConfigFileVersion: async <Request extends GetAppConfigFileVersionRequest = GetAppConfigFileVersionRequest, ResponseData = AppConfigFileVersionOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/app-config-file/versions/{id}')(params, config),
  /**
   * 删除应用配置文件历史版本
   *
   * @method DELETE
   * @path /apps/{appID}/app-config-file/versions/{id}
   * @tag app-config-files
   * @param appID path string required 应用 ID
   * @param id path string required 版本记录 ID
   * @response 200 AppConfigFileEmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteAppConfigFileVersion: async <Request extends DeleteAppConfigFileVersionRequest = DeleteAppConfigFileVersionRequest, ResponseData = AppConfigFileEmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/app-config-file/versions/{id}')(params, config),
  /**
   * 回滚到指定应用配置文件版本
   *
   * @method POST
   * @path /apps/{appID}/app-config-file/versions/{id}/rollback
   * @tag app-config-files
   * @param appID path string required 应用 ID
   * @param id path string required 版本记录 ID
   * @param body body RollbackAppConfigFileVersionInput required 回滚应用配置文件版本请求
   * @response 200 RollbackAppConfigFileVersionOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  rollbackAppConfigFileVersion: async <Request extends RollbackAppConfigFileVersionRequest = RollbackAppConfigFileVersionRequest, ResponseData = AppConfigFileOutputObj>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/app-config-file/versions/{id}/rollback')(params, config),
  /**
   * 查看一个应用所有的应用配置文件列表
   *
   * @method GET
   * @path /apps/{appID}/app-config-files
   * @tag app-config-files
   * @param appID path string required 应用 ID
   * @param type query string 按文件类型过滤，仅展示指定类型（normal/overlay）
   * @param envName query string 按环境名称过滤，可选。为空表示不过滤
   * @response 200 ListAppConfigFilesOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  listAppConfigFiles: async <Request extends ListAppConfigFilesRequest = ListAppConfigFilesRequest, ResponseData = ListAppConfigFilesOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/app-config-files')(params, config),
  /**
   * 创建一个应用配置文件
   *
   * @method POST
   * @path /apps/{appID}/app-config-files
   * @tag app-config-files
   * @param appID path string required 应用 ID
   * @param body body CreateAppConfigFileInput required 创建应用配置文件请求
   * @response 200 CreateAppConfigFileOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  createAppConfigFile: async <Request extends CreateAppConfigFileRequest = CreateAppConfigFileRequest, ResponseData = CreateAppConfigFileOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/app-config-files')(params, config),
  /**
   * 修改一个应用配置文件的基础属性
   *
   * @method PUT
   * @path /apps/{appID}/app-config-files/{id}
   * @tag app-config-files
   * @param appID path string required 应用 ID
   * @param id path string required 应用配置文件 ID
   * @param body body UpdateAppConfigFileInput required 更新应用配置文件基础属性请求
   * @response 200 UpdateAppConfigFileOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateAppConfigFile: async <Request extends UpdateAppConfigFileRequest = UpdateAppConfigFileRequest, ResponseData = UpdateAppConfigFileOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/app-config-files/{id}')(params, config),
  /**
   * 通过 ID 删除应用的一个应用配置文件
   *
   * @method DELETE
   * @path /apps/{appID}/app-config-files/{id}
   * @tag app-config-files
   * @param appID path string required 应用 ID
   * @param id path string required 应用配置文件 ID
   * @response 200 AppConfigFileEmptyOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  deleteAppConfigFile: async <Request extends DeleteAppConfigFileRequest = DeleteAppConfigFileRequest, ResponseData = AppConfigFileEmptyOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.delete<Request, ResponseData>('/apps/{appID}/app-config-files/{id}')(params, config),
  /**
   * 修改一个应用配置文件的 Content
   *
   * @method PUT
   * @path /apps/{appID}/app-config-files/{id}/content
   * @tag app-config-files
   * @param appID path string required 应用 ID
   * @param id path string required 应用配置文件 ID
   * @param body body UpdateAppConfigFileContentInput required 修改应用配置文件 Content 请求
   * @response 200 UpdateAppConfigFileContentOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateAppConfigFileContent: async <Request extends UpdateAppConfigFileContentRequest = UpdateAppConfigFileContentRequest, ResponseData = UpdateAppConfigFileContentOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/app-config-files/{id}/content')(params, config),
  /**
   * 查看一个应用配置文件详情
   *
   * @method GET
   * @path /apps/{appID}/app-config-files/{id}/details
   * @tag app-config-files
   * @param appID path string required 应用 ID
   * @param id path string required 应用配置文件 ID
   * @response 200 GetAppConfigFileDetailsOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  getAppConfigFileDetails: async <Request extends GetAppConfigFileDetailsRequest = GetAppConfigFileDetailsRequest, ResponseData = GetAppConfigFileDetailsOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.get<Request, ResponseData>('/apps/{appID}/app-config-files/{id}/details')(params, config),
  /**
   * 修改一个应用配置文件的 overlayContent
   *
   * @method PUT
   * @path /apps/{appID}/app-config-files/{id}/overlay-content
   * @tag app-config-files
   * @param appID path string required 应用 ID
   * @param id path string required 应用配置文件 ID
   * @param body body UpdateAppConfigFileOverlayContentInput required overlayContent 请求
   * @response 200 UpdateAppConfigFileContentOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  updateAppConfigFileOverlayContent: async <Request extends UpdateAppConfigFileOverlayContentRequest = UpdateAppConfigFileOverlayContentRequest, ResponseData = UpdateAppConfigFileContentOutput>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.put<Request, ResponseData>('/apps/{appID}/app-config-files/{id}/overlay-content')(params, config),
  /**
   * 预览覆盖内容与基础配置文件合并的结果
   *
   * @method POST
   * @path /apps/{appID}/app-config-files/{id}/preview-overlay-merge
   * @tag app-config-files
   * @param appID path string required 应用 ID
   * @param id path string required 基础应用配置文件 ID
   * @param body body PreviewOverlayMergeInput required 预览 overlay 合并请求
   * @response 200 PreviewOverlayMergeOutput OK
   * @response 400 GinErrorOutput Bad Request
   */
  previewOverlayMerge: async <Request extends PreviewOverlayMergeRequest = PreviewOverlayMergeRequest, ResponseData = string>(
    params?: NoInfer<Request>,
    config?: Config,
  ) => await v1Fetch.post<Request, ResponseData>('/apps/{appID}/app-config-files/{id}/preview-overlay-merge')(params, config),
};
