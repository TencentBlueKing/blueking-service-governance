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

import { effectScope, nextTick, ref } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const instanceServiceMocks = vi.hoisted(() => ({
  listAppInstances: vi.fn(),
  watchAppInstances: vi.fn(),
}));
const notifyFailureMock = vi.hoisted(() => vi.fn());

vi.mock('~/api/modules/v1', () => ({
  InstanceService: instanceServiceMocks,
}));
vi.mock('~/pages/application/detail/deploy/instance-list/composables/instance-watch-notifier', () => ({
  notifyInstanceWatchError: notifyFailureMock,
}));

import type { EffectScope } from 'vue';

import {
  extractSseEventBlocks,
  paginateInstances,
  parseInstanceSseBlock,
  reduceInstanceWatchEvent,
  sortInstancesByRestart,
} from '~/pages/application/detail/deploy/instance-list/composables/instance-watch-utils';
import { useInstanceListWatch } from '~/pages/application/detail/deploy/instance-list/composables/use-instance-list-watch';

import type { AppInstanceOutputObj } from '~/@types/v1/instance';
import type { InstanceListSourceMode } from '~/pages/application/detail/deploy/instance-list/types';

class VisibilityDocument extends EventTarget {
  visibilityState: DocumentVisibilityState = 'visible';
}

function createControlledStream() {
  const encoder = new TextEncoder();
  let controller!: ReadableStreamDefaultController<Uint8Array>;
  const response = new Response(
    new ReadableStream<Uint8Array>({
      start(streamController) {
        controller = streamController;
      },
    }),
    { headers: { 'Content-Type': 'text/event-stream' } },
  );

  return {
    close: () => controller.close(),
    response,
    send: (value: string) => controller.enqueue(encoder.encode(value)),
  };
}

async function flushPromises(rounds = 12) {
  for (let index = 0; index < rounds; index += 1) {
    await Promise.resolve();
  }
}

describe('instance watch SSE', () => {
  it('keeps fragmented data and extracts CRLF/LF event blocks', () => {
    const first = extractSseEventBlocks('event: message\ndata: {"type":"ADDED"}\n');
    expect(first.blocks).toEqual([]);

    const second = extractSseEventBlocks(`${first.rest}\n: heartbeat\r\n\r\n`);
    expect(second.blocks).toEqual(['event: message\ndata: {"type":"ADDED"}', ': heartbeat']);
    expect(second.rest).toBe('');
  });

  it('ignores event names and parses JSON type from multi-line data', () => {
    const parsed = parseInstanceSseBlock('event: message\ndata: {"type":\ndata: "ENDED", "reason": "watch timeout"}');
    expect(parsed.event).toEqual({ type: 'ENDED', reason: 'watch timeout' });
  });

  it('recognizes comment-only heartbeats', () => {
    expect(parseInstanceSseBlock(': keep-alive')).toEqual({ heartbeat: true });
  });

  it('rejects malformed JSON', () => {
    expect(() => parseInstanceSseBlock('data: {invalid-json}')).toThrow();
  });
});

describe('instance watch event reducer', () => {
  const polarisInfos = [
    {
      ip: '127.0.0.1',
      port: 8080,
      serviceName: 'demo',
      serviceNamespace: 'default',
      weight: '100',
    },
  ];

  function createInstances(): AppInstanceOutputObj[] {
    return [{ id: 'pod-a', status: 'Pending', polarisInfos }];
  }

  it('adds a new instance with empty plugin data', () => {
    const result = reduceInstanceWatchEvent(createInstances(), {
      type: 'ADDED',
      object: { id: 'pod-b', status: 'Running', polarisInfos },
    });

    expect(result).toHaveLength(2);
    expect(result[1]).toMatchObject({ id: 'pod-b', status: 'Running', polarisInfos: [] });
  });

  it('preserves Polaris data on duplicate ADDED and MODIFIED events', () => {
    const added = reduceInstanceWatchEvent(createInstances(), {
      type: 'ADDED',
      object: { id: 'pod-a', status: 'Running', polarisInfos: [] },
    });
    const modified = reduceInstanceWatchEvent(added, {
      type: 'MODIFIED',
      object: { id: 'pod-a', restartCount: '2', status: 'Running', polarisInfos: [] },
    });

    expect(modified[0]).toMatchObject({ restartCount: '2', status: 'Running', polarisInfos });
  });

  it('applies Polaris PLUGIN data including a real empty array', () => {
    const result = reduceInstanceWatchEvent(createInstances(), {
      type: 'PLUGIN',
      plugin: 'polaris',
      object: { id: 'pod-a', data: [] },
    });

    expect(result[0].polarisInfos).toEqual([]);
  });

  it('ignores unknown plugins and plugin events for missing instances', () => {
    const instances = createInstances();
    const unknownPlugin = reduceInstanceWatchEvent(instances, {
      type: 'PLUGIN',
      plugin: 'future-plugin',
      object: { id: 'pod-a', data: [] },
    });
    const missingInstance = reduceInstanceWatchEvent(instances, {
      type: 'PLUGIN',
      plugin: 'polaris',
      object: { id: 'pod-b', data: [] },
    });

    expect(unknownPlugin).toBe(instances);
    expect(missingInstance).toBe(instances);
  });

  it('deletes only by object id and ignores ENDED as a data mutation', () => {
    const instances = createInstances();
    const ended = reduceInstanceWatchEvent(instances, {
      type: 'ENDED',
      object: null,
      reason: 'watch timeout',
    });
    const deleted = reduceInstanceWatchEvent(ended, {
      type: 'DELETED',
      object: { id: 'pod-a' },
    });

    expect(ended).toBe(instances);
    expect(deleted).toEqual([]);
  });
});

describe('instance local sorting and pagination', () => {
  it('sorts the full collection before slicing the requested page', () => {
    const instances = [
      { id: 'pod-a', restartCount: '5' },
      { id: 'pod-b', restartCount: '1' },
      { id: 'pod-c' },
      { id: 'pod-d', restartCount: '3' },
    ];

    const sorted = sortInstancesByRestart(instances, 'desc');
    expect(paginateInstances(sorted, 2, 2).map(item => item.id)).toEqual(['pod-b', 'pod-c']);
  });
});

describe('instance List + Watch lifecycle', () => {
  const activeScopes: EffectScope[] = [];
  let visibilityDocument: VisibilityDocument;

  function startWatch(envName = ref('prod'), getMode?: () => InstanceListSourceMode) {
    let state!: ReturnType<typeof useInstanceListWatch>;
    const scope = effectScope();
    scope.run(() => {
      state = useInstanceListWatch({
        getMode,
        getScope: () => ({ appID: 'app-1', envName: envName.value }),
      });
    });
    activeScopes.push(scope);
    return state;
  }

  beforeEach(() => {
    vi.useFakeTimers();
    instanceServiceMocks.listAppInstances.mockReset();
    instanceServiceMocks.watchAppInstances.mockReset();
    notifyFailureMock.mockReset();
    visibilityDocument = new VisibilityDocument();
    vi.stubGlobal('document', visibilityDocument);
  });

  afterEach(() => {
    for (const scope of activeScopes.splice(0)) scope.stop();
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('re-lists immediately with a new resourceVersion after watch timeout', async () => {
    const firstStream = createControlledStream();
    const secondStream = createControlledStream();
    instanceServiceMocks.listAppInstances
      .mockResolvedValueOnce({ results: [{ id: 'pod-a' }], resourceVersion: 'rv-1' })
      .mockResolvedValueOnce({ results: [{ id: 'pod-b' }], resourceVersion: 'rv-2' });
    instanceServiceMocks.watchAppInstances
      .mockResolvedValueOnce(firstStream.response)
      .mockResolvedValueOnce(secondStream.response);

    const state = startWatch();
    await flushPromises();
    firstStream.send('data: {"type":"ENDED","reason":"watch timeout"}\n\n');
    await flushPromises();

    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(2);
    expect(instanceServiceMocks.watchAppInstances.mock.calls[1][0]).toMatchObject({ resourceVersion: 'rv-2' });
    expect(state.instances.value.map(instance => instance.id)).toEqual(['pod-b']);
    expect(notifyFailureMock).not.toHaveBeenCalled();
  });

  it('stops initial loading as soon as List succeeds without waiting for Watch headers', async () => {
    let resolveWatch!: (response: Response) => void;
    const watchResponse = new Promise<Response>(resolve => {
      resolveWatch = resolve;
    });
    instanceServiceMocks.listAppInstances.mockResolvedValue({
      results: [{ id: 'pod-a' }],
      resourceVersion: 'rv-1',
    });
    instanceServiceMocks.watchAppInstances.mockReturnValue(watchResponse);

    const state = startWatch();
    await flushPromises();

    expect(state.instances.value.map(instance => instance.id)).toEqual(['pod-a']);
    expect(state.isInitialLoading.value).toBe(false);
    expect(state.isWatching.value).toBe(false);

    resolveWatch(createControlledStream().response);
    await flushPromises();
    expect(state.isWatching.value).toBe(true);
  });

  it('polls List(all=true) every 10 seconds in polling mode without starting Watch', async () => {
    instanceServiceMocks.listAppInstances
      .mockResolvedValueOnce({ results: [{ id: 'pod-a' }], resourceVersion: 'rv-1' })
      .mockResolvedValueOnce({ results: [{ id: 'pod-b' }], resourceVersion: 'rv-2' });

    const state = startWatch(ref('fed'), () => 'polling');
    await flushPromises();

    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(1);
    expect(instanceServiceMocks.listAppInstances.mock.calls[0][0]).toMatchObject({
      all: true,
      appID: 'app-1',
      envName: 'fed',
    });
    expect(instanceServiceMocks.listAppInstances.mock.calls[0][0]).not.toHaveProperty('page');
    expect(instanceServiceMocks.listAppInstances.mock.calls[0][0]).not.toHaveProperty('pageSize');
    expect(instanceServiceMocks.watchAppInstances).not.toHaveBeenCalled();
    expect(state.instances.value.map(instance => instance.id)).toEqual(['pod-a']);
    expect(state.isInitialLoading.value).toBe(false);

    await vi.advanceTimersByTimeAsync(9999);
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(1);
    await flushPromises();

    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(2);
    expect(instanceServiceMocks.watchAppInstances).not.toHaveBeenCalled();
    expect(state.instances.value.map(instance => instance.id)).toEqual(['pod-b']);
  });

  it('backs off by 5s for consecutive slow Lists and gradually recovers after fast responses', async () => {
    let call = 0;
    instanceServiceMocks.listAppInstances.mockImplementation((_request: unknown, _config: { signal: AbortSignal }) => {
      call += 1;
      const result = { results: [{ id: `pod-${call}` }], resourceVersion: `rv-${call}` };
      // 前五轮 List 耗时 14s（慢于一个基准周期），后续各轮立即返回。
      if (call <= 5) {
        return new Promise(resolve => {
          setTimeout(() => resolve(result), 14000);
        });
      }
      return Promise.resolve(result);
    });

    const state = startWatch(ref('fed'), () => 'polling');
    // 推进到首轮 List（14s）返回；耗时 ≥ 10s 后，间隔从 10s 退避到 15s。
    await vi.advanceTimersByTimeAsync(14000);
    await flushPromises();
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(1);
    expect(state.pollingIntervalMs.value).toBe(15000);

    // 连续慢响应时每轮继续增加 5s，允许退避至 30s 及更长。
    await vi.advanceTimersByTimeAsync(15000);
    await vi.advanceTimersByTimeAsync(14000);
    await flushPromises();
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(2);
    expect(state.pollingIntervalMs.value).toBe(20000);

    await vi.advanceTimersByTimeAsync(20000);
    await vi.advanceTimersByTimeAsync(14000);
    await flushPromises();
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(3);
    expect(state.pollingIntervalMs.value).toBe(25000);

    await vi.advanceTimersByTimeAsync(25000);
    await vi.advanceTimersByTimeAsync(14000);
    await flushPromises();
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(4);
    expect(state.pollingIntervalMs.value).toBe(30000);

    // 达到上限后继续慢响应也不再增长。
    await vi.advanceTimersByTimeAsync(30000);
    await vi.advanceTimersByTimeAsync(14000);
    await flushPromises();
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(5);
    expect(state.pollingIntervalMs.value).toBe(30000);

    // 快响应每轮只恢复一个步长，避免从 30s 直接跳回 10s。
    await vi.advanceTimersByTimeAsync(30000);
    await flushPromises();
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(6);
    expect(state.pollingIntervalMs.value).toBe(25000);

    await vi.advanceTimersByTimeAsync(25000);
    await flushPromises();
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(7);
    expect(state.pollingIntervalMs.value).toBe(20000);

    await vi.advanceTimersByTimeAsync(20000);
    await flushPromises();
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(8);
    expect(state.pollingIntervalMs.value).toBe(15000);

    await vi.advanceTimersByTimeAsync(15000);
    await flushPromises();
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(9);
    expect(state.pollingIntervalMs.value).toBe(10000);
  });

  it('shows the List error for every polling failure and clears it after a successful List', async () => {
    const firstError = { error: { message: 'cluster disconnected' }, status: 503 };
    const secondError = { error: { message: 'cluster still disconnected' }, status: 503 };
    const thirdError = { error: { message: 'cluster disconnected again' }, status: 503 };
    instanceServiceMocks.listAppInstances
      .mockRejectedValueOnce(firstError)
      .mockRejectedValueOnce(secondError)
      .mockResolvedValueOnce({ results: [{ id: 'pod-a' }], resourceVersion: 'rv-1' })
      .mockRejectedValueOnce(thirdError);

    const state = startWatch(ref('fed'), () => 'polling');
    await flushPromises();

    expect(state.lastError.value).toBe(firstError);
    expect(notifyFailureMock).toHaveBeenCalledTimes(1);
    expect(notifyFailureMock).toHaveBeenLastCalledWith(firstError);
    expect(state.isInitialLoading.value).toBe(false);

    await vi.advanceTimersByTimeAsync(10000);
    await flushPromises();
    expect(state.lastError.value).toBe(secondError);
    expect(notifyFailureMock).toHaveBeenCalledTimes(2);
    expect(notifyFailureMock).toHaveBeenLastCalledWith(secondError);

    await vi.advanceTimersByTimeAsync(10000);
    await flushPromises();
    expect(state.lastError.value).toBeUndefined();
    expect(state.instances.value.map(instance => instance.id)).toEqual(['pod-a']);

    await vi.advanceTimersByTimeAsync(10000);
    await flushPromises();
    expect(state.lastError.value).toBe(thirdError);
    expect(notifyFailureMock).toHaveBeenCalledTimes(3);
    expect(notifyFailureMock).toHaveBeenLastCalledWith(thirdError);
    expect(state.instances.value.map(instance => instance.id)).toEqual(['pod-a']);
  });

  it('stops polling while hidden and resumes with a fresh List when visible again', async () => {
    const signals: AbortSignal[] = [];
    let resolveFirstList!: (value: { resourceVersion: string; results: AppInstanceOutputObj[] }) => void;
    const firstList = new Promise<{ resourceVersion: string; results: AppInstanceOutputObj[] }>(resolve => {
      resolveFirstList = resolve;
    });
    instanceServiceMocks.listAppInstances.mockImplementation((_request: unknown, config: { signal: AbortSignal }) => {
      signals.push(config.signal);
      if (signals.length === 1) return firstList;
      return Promise.resolve({ results: [{ id: `pod-${signals.length}` }], resourceVersion: `rv-${signals.length}` });
    });

    const state = startWatch(ref('fed'), () => 'polling');
    await flushPromises();
    visibilityDocument.visibilityState = 'hidden';
    visibilityDocument.dispatchEvent(new Event('visibilitychange'));

    expect(signals[0].aborted).toBe(true);
    resolveFirstList({ results: [{ id: 'pod-stale' }], resourceVersion: 'rv-stale' });
    await flushPromises();
    await vi.advanceTimersByTimeAsync(10000);
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(1);
    expect(state.instances.value).toEqual([]);

    visibilityDocument.visibilityState = 'visible';
    visibilityDocument.dispatchEvent(new Event('visibilitychange'));
    await flushPromises();

    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(2);
    expect(state.instances.value.map(instance => instance.id)).toEqual(['pod-2']);
  });

  it('re-lists with a new resourceVersion after the abnormal reconnect cooldown for an unexpected EOF', async () => {
    const firstStream = createControlledStream();
    const secondStream = createControlledStream();
    instanceServiceMocks.listAppInstances
      .mockResolvedValueOnce({ results: [{ id: 'pod-a' }], resourceVersion: 'rv-1' })
      .mockResolvedValueOnce({ results: [{ id: 'pod-b' }], resourceVersion: 'rv-2' });
    instanceServiceMocks.watchAppInstances
      .mockResolvedValueOnce(firstStream.response)
      .mockResolvedValueOnce(secondStream.response);

    const state = startWatch();
    await flushPromises();
    firstStream.close();
    await flushPromises();

    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(4999);
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(1);
    await flushPromises();

    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(2);
    expect(instanceServiceMocks.watchAppInstances.mock.calls[1][0]).toMatchObject({ resourceVersion: 'rv-2' });
    expect(state.instances.value.map(instance => instance.id)).toEqual(['pod-b']);
    expect(notifyFailureMock).not.toHaveBeenCalled();
  });

  it('re-lists with a new resourceVersion after the abnormal reconnect cooldown for a Watch 409', async () => {
    const stream = createControlledStream();
    instanceServiceMocks.listAppInstances
      .mockResolvedValueOnce({ results: [{ id: 'pod-a' }], resourceVersion: 'rv-1' })
      .mockResolvedValueOnce({ results: [{ id: 'pod-b' }], resourceVersion: 'rv-2' });
    const responseError = { error: { message: 'resource version expired' }, status: 409 };
    instanceServiceMocks.watchAppInstances.mockRejectedValueOnce(responseError).mockResolvedValueOnce(stream.response);

    const state = startWatch();
    await flushPromises();

    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(1);
    expect(instanceServiceMocks.watchAppInstances).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(4999);
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(1);
    await flushPromises();

    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(2);
    expect(instanceServiceMocks.watchAppInstances).toHaveBeenCalledTimes(2);
    expect(instanceServiceMocks.watchAppInstances.mock.calls[1][0]).toMatchObject({ resourceVersion: 'rv-2' });
    expect(state.instances.value.map(instance => instance.id)).toEqual(['pod-b']);
    expect(notifyFailureMock).not.toHaveBeenCalled();
  });

  it('cancels a delayed abnormal reconnect while hidden and starts fresh when visible again', async () => {
    const firstStream = createControlledStream();
    const secondStream = createControlledStream();
    instanceServiceMocks.listAppInstances
      .mockResolvedValueOnce({ results: [{ id: 'pod-a' }], resourceVersion: 'rv-1' })
      .mockResolvedValueOnce({ results: [{ id: 'pod-b' }], resourceVersion: 'rv-2' });
    instanceServiceMocks.watchAppInstances
      .mockResolvedValueOnce(firstStream.response)
      .mockResolvedValueOnce(secondStream.response);

    const state = startWatch();
    await flushPromises();
    firstStream.close();
    await flushPromises();
    visibilityDocument.visibilityState = 'hidden';
    visibilityDocument.dispatchEvent(new Event('visibilitychange'));

    await vi.advanceTimersByTimeAsync(5000);
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(1);

    visibilityDocument.visibilityState = 'visible';
    visibilityDocument.dispatchEvent(new Event('visibilitychange'));
    await flushPromises();

    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(2);
    expect(instanceServiceMocks.watchAppInstances.mock.calls[1][0]).toMatchObject({ resourceVersion: 'rv-2' });
    expect(state.instances.value.map(instance => instance.id)).toEqual(['pod-b']);
  });

  it('reports a non-409 Watch connection failure without reconnecting', async () => {
    instanceServiceMocks.listAppInstances.mockResolvedValue({
      results: [{ id: 'pod-a' }],
      resourceVersion: 'rv-1',
    });
    const responseError = { error: { message: 'cluster unavailable' }, status: 503 };
    instanceServiceMocks.watchAppInstances.mockRejectedValue(responseError);

    const state = startWatch();
    await flushPromises();

    expect(state.lastError.value).toBe(responseError);
    expect(notifyFailureMock).toHaveBeenCalledWith(responseError);
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(1);
    expect(instanceServiceMocks.watchAppInstances).toHaveBeenCalledTimes(1);
  });

  it('reports a List failure without starting Watch or retrying', async () => {
    const responseError = { error: { message: 'cluster unavailable' } };
    instanceServiceMocks.listAppInstances.mockRejectedValue(responseError);

    const state = startWatch();
    await flushPromises();

    expect(state.lastError.value).toBe(responseError);
    expect(notifyFailureMock).toHaveBeenCalledWith(responseError);
    expect(instanceServiceMocks.watchAppInstances).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(15000);
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(1);
  });

  it('treats an ENDED event with an unknown reason as a terminal Watch failure', async () => {
    const stream = createControlledStream();
    instanceServiceMocks.listAppInstances.mockResolvedValue({ results: [], resourceVersion: 'rv-1' });
    instanceServiceMocks.watchAppInstances.mockResolvedValue(stream.response);

    const state = startWatch();
    await flushPromises();
    stream.send('data: {"type":"ENDED","object":null}\n\n');
    await flushPromises();

    expect(state.lastError.value).toBeInstanceOf(Error);
    expect(notifyFailureMock).toHaveBeenCalledWith(expect.any(Error));
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(1);
  });

  it('re-lists after cluster watch interrupted', async () => {
    const firstStream = createControlledStream();
    const secondStream = createControlledStream();
    instanceServiceMocks.listAppInstances
      .mockResolvedValueOnce({ results: [{ id: 'pod-a' }], resourceVersion: 'rv-1' })
      .mockResolvedValueOnce({ results: [{ id: 'pod-b' }], resourceVersion: 'rv-2' });
    instanceServiceMocks.watchAppInstances
      .mockResolvedValueOnce(firstStream.response)
      .mockResolvedValueOnce(secondStream.response);

    const state = startWatch();
    await flushPromises();
    firstStream.send('data: {"type":"ENDED","reason":"cluster watch interrupted"}\n\n');
    await flushPromises();

    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(2);
    expect(instanceServiceMocks.watchAppInstances.mock.calls[1][0]).toMatchObject({ resourceVersion: 'rv-2' });
    expect(state.instances.value.map(instance => instance.id)).toEqual(['pod-b']);
    expect(notifyFailureMock).not.toHaveBeenCalled();
  });

  it('aborts while hidden and starts from a fresh List when visible again', async () => {
    const firstStream = createControlledStream();
    const secondStream = createControlledStream();
    const signals: AbortSignal[] = [];
    instanceServiceMocks.listAppInstances.mockResolvedValue({ results: [], resourceVersion: 'rv-visible' });
    instanceServiceMocks.watchAppInstances.mockImplementation(
      async (_request: unknown, config: { signal: AbortSignal }) => {
        signals.push(config.signal);
        return signals.length === 1 ? firstStream.response : secondStream.response;
      },
    );

    startWatch();
    await flushPromises();
    visibilityDocument.visibilityState = 'hidden';
    visibilityDocument.dispatchEvent(new Event('visibilitychange'));

    expect(signals[0].aborted).toBe(true);
    await vi.advanceTimersByTimeAsync(10000);
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(1);

    visibilityDocument.visibilityState = 'visible';
    visibilityDocument.dispatchEvent(new Event('visibilitychange'));
    await flushPromises();
    expect(instanceServiceMocks.listAppInstances).toHaveBeenCalledTimes(2);
    expect(signals).toHaveLength(2);
  });

  it('drops a stale List result after the environment changes', async () => {
    const envName = ref('prod');
    const stream = createControlledStream();
    let resolveOldList!: (value: { resourceVersion: string; results: AppInstanceOutputObj[] }) => void;
    const oldList = new Promise<{ resourceVersion: string; results: AppInstanceOutputObj[] }>(resolve => {
      resolveOldList = resolve;
    });
    instanceServiceMocks.listAppInstances.mockImplementation((request: { envName: string }) =>
      request.envName === 'prod'
        ? oldList
        : Promise.resolve({ results: [{ id: 'pod-new' }], resourceVersion: 'rv-new' }),
    );
    instanceServiceMocks.watchAppInstances.mockResolvedValue(stream.response);

    const state = startWatch(envName);
    await flushPromises();
    envName.value = 'staging';
    await nextTick();
    await flushPromises();
    resolveOldList({ results: [{ id: 'pod-old' }], resourceVersion: 'rv-old' });
    await flushPromises();

    expect(state.instances.value.map(instance => instance.id)).toEqual(['pod-new']);
    expect(instanceServiceMocks.watchAppInstances).toHaveBeenCalledTimes(1);
    expect(instanceServiceMocks.watchAppInstances.mock.calls[0][0]).toMatchObject({
      envName: 'staging',
      resourceVersion: 'rv-new',
    });
  });
});
