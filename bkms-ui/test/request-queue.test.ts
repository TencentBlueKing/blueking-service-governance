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
/**
 * 请求队列取消行为测试（M3 回归，基于真实队列而非 mock cancelRequest）
 *
 * 背景：fetch 拦截器此前完成时固定 removeQueue(自动生成的 requestID)，自定义 init.id 的
 * 队列项完成后残留 → 后续同 id 请求（列表与详情共用 requestTag）无法入队，
 * AbortController 不被跟踪，cancelRequest 取消到的是已结束的请求。
 *
 * 1. 共用自定义 id 的顺序请求：前一个完成后队列被清理，后一个仍可被取消；
 * 2. 并发共用 id：未入队请求完成时不清除在途请求的队列项；
 * 3. addQueue 重复 id 不入队并返回 false。
 *
 * stub/mock：仅替换 window.fetch（拦截器在模块加载时捕获其引用，必须在求值前替换，
 * 故用 vi.hoisted）；队列与取消全部走真实实现。
 */
import { describe, expect, it, vi } from 'vitest';

// 拦截器模块加载时捕获 window.fetch 引用，vi.hoisted 先于模块求值执行
const fetchMock = vi.hoisted(() => {
  const mock = vi.fn();
  vi.stubGlobal('fetch', mock);
  return mock;
});

import { fetch as fetchApi } from '~/api/interceptors';
import { type IQueue, addQueue, cancelRequest, clearQueue } from '~/api/request-queue';

/** 发起一个挂起的 fetch（仅在被 abort 时以 AbortError 结束） */
function pendingFetch(url: string, id: string) {
  fetchMock.mockImplementationOnce((_input: RequestInfo | URL, init: RequestInit) => {
    return new Promise((_resolve, reject) => {
      init.signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')));
    });
  });
  return fetchApi(url, { id });
}

describe('请求队列：自定义 id 的取消行为', () => {
  it('共用自定义 id 的顺序请求：前一个完成后，后一个仍可被取消', async () => {
    // 第一个请求（如模板列表）立即完成 → 队列按实际入队 ID 清理，不残留
    fetchMock.mockResolvedValueOnce({ ok: true });
    await fetchApi('/list', { id: 'tag-shared' });

    // 第二个请求（如模板详情）共用同一 id，挂起等待
    const detail = pendingFetch('/detail', 'tag-shared');

    // 取消基于真实队列行为：应中断第二个（在途）请求而非已完成的第一个
    await cancelRequest('tag-shared');
    await expect(detail).rejects.toMatchObject({ name: 'AbortError' });
  });

  it('并发共用 id：未入队请求完成时不清除在途请求的队列项', async () => {
    const first = pendingFetch('/first', 'tag-concurrent');

    // 第二个同 id 请求未入队（id 已被在途请求占用），立即完成
    fetchMock.mockResolvedValueOnce({ ok: true });
    await fetchApi('/second', { id: 'tag-concurrent' });

    // 第一个请求仍可被取消
    await cancelRequest('tag-concurrent');
    await expect(first).rejects.toMatchObject({ name: 'AbortError' });
  });

  it('addQueue 对重复 id 不入队并返回 false', () => {
    const entry: IQueue = {
      id: 'tag-duplicate',
      controller: new AbortController(),
      request: Promise.resolve(),
    };
    expect(addQueue(entry)).toBe(true);
    expect(addQueue({ ...entry, controller: new AbortController() })).toBe(false);
    clearQueue('tag-duplicate');
  });
});
