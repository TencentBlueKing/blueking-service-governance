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

import { merge, uniqueId } from 'lodash-es';
import { addQueue, removeQueue } from '~/api/request-queue';

export type Config = {
  id?: string; // 唯一ID
  interceptorErr?: boolean; // 是否自动拦截http异常弹出message
  irrevocable?: boolean; // 请求不能取消
  isBodyParam?: boolean; // 是否将请求参数放在body中
  multipart?: boolean; // 是否将非路径参数组装为 multipart/form-data
  needRes?: boolean;
  needStatus?: boolean; // 是否需要返回 status 等响应信息
  originalResponse?: boolean; // 返回原始res对象
  prefix?: string;
  responseType?: 'blob' | 'json' | 'text';
  validateCode?: boolean; // 校验code是否正确，默认true
} & RequestInit;

export type FetchReturnType<T, P extends Config> = P['originalResponse'] extends true ? Response : T;

export type ResCallback = (res: Response, config: Partial<Config>) => unknown;

/** 请求拦截器 */
type ReqInterceptor = (config: Partial<Config>) => Partial<Config>;
/** 响应错误拦截器 */
type ResErrorInterceptor = (error: unknown, config: Partial<Config>) => unknown;

const interceptorsReq: ReqInterceptor[] = [];
const interceptorsRes: Array<ResCallback> = [];
const interceptorsResError: ResErrorInterceptor[] = [];

const OriginFetch = window.fetch;

// fetch
function fetch<T, C extends Config>(input: RequestInfo | URL, init: Partial<C>) {
  interceptorsReq.forEach(fn => {
    init = fn(init) as Partial<C>;
  });

  return new Promise<FetchReturnType<T, C>>((resolve, reject) => {
    const controller = new AbortController();
    const requestID = uniqueId();
    const defaultHeaders = {
      'Access-Control-Allow-Origin': '*',
      'X-Bkapi-Request-Id': requestID,
    };

    const request = OriginFetch(
      input,
      merge(
        {
          headers: defaultHeaders,
          signal: controller.signal,
        },
        init,
      ),
    )
      .then(res => {
        interceptorsRes.forEach(fn => {
          res = fn(res, init) as Response;
        });
        resolve(res as FetchReturnType<T, C>);
        removeQueue(requestID);
      })
      .catch(err => {
        interceptorsResError.forEach(fn => {
          err = fn(err, init);
        });
        reject(err);
        removeQueue(requestID);
      });

    // const route = useRoute();
    addQueue({
      id: init?.id || requestID,
      controller,
      request,
      config: init,
      // routeName: route?.name,
    });
  });
}

const interceptors = {
  request: {
    use(callback: ReqInterceptor) {
      interceptorsReq.push(callback);
    },
  },
  response: {
    use(callback: ResCallback, errorCallback?: ResErrorInterceptor) {
      interceptorsRes.push(callback);
      errorCallback && interceptorsResError.push(errorCallback);
    },
  },
};

export { fetch, interceptors };
