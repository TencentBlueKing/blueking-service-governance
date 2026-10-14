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

import { type RecommendSource, useRecommendTag } from '~/composables/use-recommend-tag';

const getRecommendedImageTag = vi.hoisted(() => vi.fn());

vi.mock('~/api/modules/bkmsserver', () => ({
  ApiServerService: {
    GetRecommendedImageTag: getRecommendedImageTag,
  },
}));

vi.mock('~/stores/app-detail', () => ({
  useAppDetail: () => ({ appID: 'app-1' }),
}));

const commitA = 'a1b2c3d';
const commitB = 'b2c3d4e';

describe('useRecommendTag', () => {
  const activeScopes: ReturnType<typeof effectScope>[] = [];

  function mountTag(source: ReturnType<typeof ref<RecommendSource>>, manualFetchOnly = false) {
    const tags: string[] = [];
    const scope = effectScope();
    let fetchRecommendTag: () => Promise<string> = async () => '';
    scope.run(() => {
      ({ fetchRecommendTag } = useRecommendTag(() => source.value, {
        manualFetchOnly,
        onRecommend: tag => {
          tags.push(tag);
        },
      }));
    });
    activeScopes.push(scope);
    return { tags, fetchRecommendTag };
  }

  beforeEach(() => {
    vi.useFakeTimers();
    getRecommendedImageTag.mockReset();
  });

  afterEach(() => {
    for (const scope of activeScopes.splice(0)) scope.stop();
    vi.useRealTimers();
  });

  it('drops a stale recommendation when the revision changes before the response', async () => {
    let releaseA: (tag: string) => void = () => {};
    const pendingA = new Promise<string>(resolve => {
      releaseA = resolve;
    });
    getRecommendedImageTag.mockImplementation((query: { revision?: string }) => {
      if (query.revision === commitA) return pendingA;
      return Promise.resolve('tag-b');
    });

    const source = ref<RecommendSource>({ revisionType: 'commit', revision: commitA });
    const { tags, fetchRecommendTag } = mountTag(source);

    void fetchRecommendTag();
    await Promise.resolve();

    source.value = { revisionType: 'commit', revision: commitB };
    await nextTick();
    releaseA('tag-a');
    await Promise.resolve();
    await Promise.resolve();

    expect(tags).not.toContain('tag-a');

    await vi.advanceTimersByTimeAsync(500);
    await Promise.resolve();
    await Promise.resolve();

    expect(tags).toEqual(['tag-b']);
  });

  it('drops a stale recommendation under manual fetch when the commit id becomes incomplete', async () => {
    let releaseA: (tag: string) => void = () => {};
    const pendingA = new Promise<string>(resolve => {
      releaseA = resolve;
    });
    getRecommendedImageTag.mockImplementation(() => pendingA);

    const source = ref<RecommendSource>({ revisionType: 'commit', revision: commitA });
    const { tags, fetchRecommendTag } = mountTag(source, true);

    void fetchRecommendTag();
    await Promise.resolve();

    source.value = { revisionType: 'commit', revision: 'abc' };
    await nextTick();
    releaseA('tag-a');
    await vi.advanceTimersByTimeAsync(500);
    await Promise.resolve();

    expect(tags).toEqual([]);
    expect(getRecommendedImageTag).toHaveBeenCalledTimes(1);
  });
});
