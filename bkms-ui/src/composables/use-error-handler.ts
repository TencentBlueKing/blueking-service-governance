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

import { Message } from 'bkui-vue';
import { appendTraceId, appendTraceIdToDetails } from '~/api/trace-id';

interface BackendError {
  [key: string]: unknown;
  code?: number | string;
  datas?: Record<string, unknown>;
  details?: Array<Record<string, unknown>> | Record<string, unknown> | string;
  message?: string;
  status?: number; // HTTP 状态码
  traceId?: string;
}

interface MessageAction {
  disabled?: boolean;
  id: string;
  render?: () => unknown;
}

interface MessageConfig {
  actions?: MessageAction[];
  delay?: number;
  extCls?: string;
  theme?: 'error' | 'primary' | 'success' | 'warning';
  message?:
    | string
    | {
        code?: number | string;
        details?: string;
        overview?: string;
        suggestion?: string;
        type?: string;
      };
}

/**
 * 自定义错误处理 Hook
 * @param error 错误对象
 * @param customCode 自定义错误代码
 * @param customMessage 错误消息配置
 */
export function useErrorHandler() {
  const handleError = (error: BackendError, customCode?: number, customMessageConfig?: MessageConfig) => {
    const httpStatus = error?.status || error?.code;
    const traceId = error?.traceId;
    // 自定义code处理
    if (customCode && customMessageConfig && httpStatus === customCode) {
      // 保留自定义 Message 的 actions、样式和详情，仅增强用户可见的错误概览。
      Message({
        ...customMessageConfig,
        message: buildTraceMessage(customMessageConfig.message, traceId),
      });
    } else {
      const message = (error?.msg || error?.datas?.msg || error?.message || error?.datas?.message || '') as string;
      Message({
        theme: 'error',
        message: {
          code: httpStatus,
          details: traceId
            ? appendTraceIdToDetails(error.details || message || {}, traceId)
            : error.details || message || '',
          overview: appendTraceId(message, traceId),
          suggestion: '',
          type: 'json',
        },
      });
    }
  };

  return { handleError };
}

/**
 * 构建包含 Trace ID 的自定义错误消息。
 * 无 Trace ID 时原样返回；字符串消息会转换为结构化消息，确保 Trace ID 同时展示在概览和详情中。
 * 已经是结构化的消息会保留原配置，仅增强 overview 和 details。
 */
function buildTraceMessage(message: MessageConfig['message'], traceId?: string): MessageConfig['message'] {
  if (!traceId) {
    return message;
  }

  if (typeof message === 'string') {
    return {
      overview: appendTraceId(message, traceId),
      details: appendTraceIdToDetails({}, traceId),
      type: 'json',
    };
  }

  return {
    ...message,
    overview: appendTraceId(message?.overview || '', traceId),
    details: appendTraceIdToDetails(message?.details || {}, traceId),
  };
}
