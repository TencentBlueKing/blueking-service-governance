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
import { isObject, merge } from 'lodash-es';
import { objectToQueryParams } from '~/common/util';

import { type Config, fetch, interceptors } from './interceptors';
import { appendTraceId, appendTraceIdToDetails, attachTraceId, getTraceId } from './trace-id';

type HttpMethods = 'DELETE' | 'GET' | 'PATCH' | 'POST' | 'PUT';
type RequestParams = Record<string, unknown>;
type ResponseData = Record<string, unknown> & {
  code?: number;
  data?: unknown;
  error?: {
    message?: string;
    traceId?: string;
  };
  status?: number;
  traceId?: string;
};

interceptors.response.use(
  async (response: Response, config: Config) => {
    // 流式响应和文件下载需要立即返回原始 Response，不能预先读取成功响应体。
    // 错误响应仍按原逻辑解析，以便后续展示后端返回的错误详情。
    const res =
      config.originalResponse && response.ok
        ? {}
        : await resultReduction(response.clone(), config).catch((): ResponseData => ({}));
    // Trace ID 仅用于失败提示和错误对象透传，不写入成功响应数据。
    const traceId = getTraceId(response);
    let resolveRes;
    if (config.originalResponse) {
      // 返回原生response
      resolveRes = response;
    } else if (config.needRes) {
      // 返回后端完整response
      resolveRes = res;
    } else {
      // 返回data字段
      resolveRes = res.data;
    }

    // Cookie 过期
    if (response.status === 401) {
      window.location.href = `${window.BK_LOGIN_URL}?c_url=${window.location.href}`;
      return Promise.reject(new Error('Unauthorized'));
    }

    // 无权限
    if (response.status === 403) {
      // TODO: 权限弹窗
      Message({
        theme: 'error',
        message: traceId
          ? {
              overview: appendTraceId('无权限', traceId),
              details: appendTraceIdToDetails({}, traceId),
              type: 'json',
            }
          : '无权限',
      });
      return Promise.reject(attachTraceId(new Error('Forbidden'), traceId));
    }

    // 默认使用 Message 弹窗，特殊错误使用 UI 展示异常
    if (response.status === 400) {
      config.interceptorErr &&
        Message({
          theme: 'error',
          actions: [
            {
              id: 'assistant',
              disabled: true,
            },
          ],
          message: {
            code: response.status,
            overview: appendTraceId(res?.error?.message || window.i18n.t('请求异常'), traceId),
            suggestion: '',
            type: 'json',
            details: appendTraceIdToDetails(res?.error || {}, traceId),
          },
        });
      // 同时透传到拒绝对象，供关闭默认拦截的页面级错误处理复用。
      attachTraceId(res, traceId);
      return Promise.reject(res);
    }

    // 其他异常状态
    if (response.status < 200 || response.status >= 300) {
      config.interceptorErr &&
        Message({
          theme: 'error',
          actions: [
            {
              id: 'assistant',
              disabled: true, // 不显示助手
            },
          ],
          message: {
            code: response.status,
            overview: appendTraceId(res?.error?.message || window.i18n.t('请求异常'), traceId),
            suggestion: '',
            type: 'json',
            details: appendTraceIdToDetails(res?.error || {}, traceId),
          },
        });
      // 关闭默认拦截时，调用方仍可从拒绝对象中取得 Trace ID。
      attachTraceId(res, traceId);
      return Promise.reject(
        config?.needStatus
          ? {
              ...res,
              status: response.status,
              statusText: response.statusText,
            }
          : res,
      );
    }

    // API状态码不正确
    if (config.validateCode && res.status !== 0 && res.code !== 0) {
      // 优化后的Messagea
      config.interceptorErr &&
        Message({
          theme: 'error',
          message: {
            code: res.status ?? res.code,
            overview: appendTraceId(window.i18n.t('请求失败'), traceId),
            suggestion: '',
            type: 'key-value',
            details: traceId ? appendTraceIdToDetails({}, traceId) : undefined,
          },
        });
      // 业务码失败同样透传 Trace ID，供页面级错误处理复用。
      attachTraceId(res, traceId);
      return Promise.reject(res);
    }

    return Promise.resolve(resolveRes);
  },
  (error: unknown, config: Config) => {
    // 请求被主动取消（路由切换等），静默处理
    if (error instanceof DOMException && error.name === 'AbortError') {
      return Promise.reject(error);
    }

    // 网络层异常（断网、DNS 解析失败、CORS 错误等），弹出提示
    if (error instanceof TypeError) {
      config.interceptorErr &&
        Message({
          theme: 'error',
          message: window.i18n.t('网络异常，请检查网络连接后重试'),
        });
      return Promise.reject(error);
    }

    // 其他未知异常，弹出通用提示
    config.interceptorErr &&
      Message({
        theme: 'error',
        message: error instanceof Error ? error.message : window.i18n.t('请求异常'),
      });
    return Promise.reject(error);
  },
);

export default class ConsoleFetch {
  config: Config; // 全局配置

  constructor(config: Config) {
    this.config = merge(
      {
        mode: 'cors',
        cache: 'default',
        credentials: 'include',
        headers: {
          'X-Requested-With': 'fetch',
          'Content-Type': 'application/json',
        },
        redirect: 'follow',
        referrerPolicy: 'no-referrer-when-downgrade',
        responseType: 'json',
        validateCode: false,
        interceptorErr: true,
      },
      config,
    );
  }
  delete<P, T>(url: string) {
    return <C extends Config>(params?: P, config?: C) => this.request<T, C>('DELETE', url, params, config);
  }
  // P 请求参数类型 T 返回类型
  get<P, T>(url: string) {
    return <C extends Config>(params?: P, config?: C) => this.request<T, C>('GET', url, params, config);
  }

  // 替换URL上的变量和删除params上的变量参数
  parseUrlAndParams(url: string, params: RequestParams = {}) {
    const variableData: RequestParams = {};
    let newUrl = url;

    // 查找URL中的所有 {var} 格式的参数
    const urlParams = url.match(/\{([^}]+)\}/g) || [];

    urlParams.forEach(param => {
      const key = param.slice(1, -1); // 移除 { } 获取参数名
      if (params[key] !== undefined) {
        variableData[key] = params[key];
        // 替换URL中的参数
        newUrl = newUrl.replace(param, String(params[key]));
        // 删除已处理的参数
        delete params[key];
      }
    });

    return {
      url: newUrl,
      params,
    };
  }

  patch<P, T>(url: string) {
    return <C extends Config>(params?: P, config?: C) => this.request<T, C>('PATCH', url, params, config);
  }

  post<P, T>(url: string) {
    return <C extends Config>(params?: P, config?: C) => this.request<T, C>('POST', url, params, config);
  }

  put<P, T>(url: string) {
    return <C extends Config>(params?: P, config?: C) => this.request<T, C>('PUT', url, params, config);
  }

  async request<T, C extends Config>(method: HttpMethods, url: string, params?: unknown, config?: C) {
    const fetchConfig = merge(
      {},
      this.config,
      {
        headers: {},
      },
      config || {},
    );
    let body: BodyInit | null | undefined;
    const requestParams: RequestParams = isObject(params) ? (params as RequestParams) : {};
    const parseData = this.parseUrlAndParams(`${fetchConfig.prefix}${url}`, requestParams);
    // GET 和 DELETE 请求参数放URL，其余请求放在body里面
    if ((method === 'GET' || method === 'DELETE') && !fetchConfig.isBodyParam) {
      const query = objectToQueryParams(parseData.params);
      parseData.url += query ? `?${query}` : '';
    } else if (fetchConfig.multipart) {
      const formData = new FormData();
      Object.entries(parseData.params).forEach(([key, value]) => appendFormDataValue(formData, key, value));
      body = formData;

      // multipart boundary 必须由浏览器生成，不能沿用默认的 application/json。
      const headers = new Headers(fetchConfig.headers as HeadersInit);
      headers.delete('Content-Type');
      const multipartHeaders: Record<string, string> = {};
      headers.forEach((value, key) => {
        multipartHeaders[key] = value;
      });
      fetchConfig.headers = multipartHeaders;
    } else {
      body = isObject(params) ? JSON.stringify(parseData.params || {}) : (params as BodyInit | null | undefined);
    }

    const requestConfig: Partial<Config> = {
      method,
      ...fetchConfig,
      body,
    };
    const response = await fetch<T, C>(parseData.url, requestConfig as Partial<C>);

    return response;
  }
}

function appendFormDataValue(formData: FormData, key: string, value: unknown) {
  if (value === undefined || value === null) return;
  if (Array.isArray(value)) {
    value.forEach(item => appendFormDataValue(formData, key, item));
    return;
  }
  if (value instanceof Blob) {
    formData.append(key, value);
    return;
  }
  formData.append(key, typeof value === 'object' ? JSON.stringify(value) : String(value));
}

async function resultReduction(response: Response, config: Config): Promise<ResponseData> {
  let res: ResponseData;
  switch (config.responseType) {
    case 'json':
      res = (await response.json()) as ResponseData;
      break;
    case 'text':
      res = (await response.text()) as unknown as ResponseData;
      break;
    case 'blob':
      res = (await response.blob()) as unknown as ResponseData;
      break;
    default:
      res = (await response.json()) as ResponseData;
      break;
  }
  return res;
}
