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

type InstanceRequestError = Record<string, unknown> & {
  code?: number;
  error?: {
    message?: string;
    traceId?: string;
  };
  status?: number;
  traceId?: string;
};

/** 按通用请求层的固定 error.message 结构展示 List/Watch 接口错误。 */
export function notifyInstanceWatchError(error: unknown) {
  // SSE 协议校验等前端异常不含接口响应体，直接展示原生错误信息。
  if (error instanceof Error) {
    Message({
      theme: 'error',
      message: error.message || window.i18n.t('请求异常'),
    });
    return;
  }

  const responseError = error as InstanceRequestError | undefined;
  const traceId = responseError?.traceId || responseError?.error?.traceId;
  Message({
    theme: 'error',
    actions: [
      {
        id: 'assistant',
        disabled: true,
      },
    ],
    message: {
      code: responseError?.status ?? responseError?.code,
      overview: appendTraceId(responseError?.error?.message || window.i18n.t('请求异常'), traceId),
      suggestion: '',
      type: 'json',
      details: appendTraceIdToDetails(responseError?.error || {}, traceId),
    },
  });
}
