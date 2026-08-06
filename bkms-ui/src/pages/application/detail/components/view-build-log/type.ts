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

export type BuildAlertTheme = 'error' | 'info' | 'success' | 'warning';
/** 构建提示和流水线跳转所需的构建信息。 */
export interface BuildInfo {
  /** 蓝盾构建 ID */
  buildID: string;
  /** 镜像 Tag */
  imageTag: string;
  /** 构建操作人 */
  operator: string;
  /** 蓝盾流水线 ID */
  pipelineID: string;
  /** 代码分支 */
  revision: string;
  /** 构建状态 */
  status: BuildStatus;
}

/** SSE error 事件数据，兼容后端旧版字符串错误格式。 */
export interface BuildLogError {
  error?:
    | string
    | {
        details?: BuildLogErrorDetail[];
        message?: string;
      };
}

/** SSE error 事件中的业务错误详情。 */
export interface BuildLogErrorDetail {
  code?: string;
  message?: string;
}

/** 蓝盾返回的单行构建日志。 */
export interface BuildLogLine {
  /** 实际接口返回 PascalCase，camelCase 用于兼容接口文档。 */
  LineNo?: number;
  lineNo?: number;
  Message?: string;
  message?: string;
  Timestamp?: number;
  timestamp?: number;
}

/** SSE message 事件数据。 */
export interface BuildLogMessage {
  /** 实际接口返回 PascalCase，camelCase 用于兼容接口文档。 */
  Finished?: boolean;
  finished?: boolean;
  HasMore?: boolean;
  hasMore?: boolean;
  Logs?: BuildLogLine[];
  logs?: BuildLogLine[];
}

/** 无侧滑外壳的构建日志面板参数。 */
export interface BuildLogPanelProps extends BuildTipsProps {
  active: boolean;
}

/** 构建日志查询和下载接口的公共参数。 */
export interface BuildLogRequest {
  appID: string;
  buildID: string;
}

export type BuildStatus = 'failed' | 'pollingBroken' | 'running' | 'success' | 'warning';

/** 构建状态提示组件参数。 */
export interface BuildTipsProps {
  buildInfo: BuildInfo;
  needClose?: boolean;
}

/** 构建日志侧滑组件参数。 */
export interface ViewBuildLogProps {
  buildInfo: BuildInfo;
}
