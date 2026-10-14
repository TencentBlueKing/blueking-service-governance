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

import { type MaybeRefOrGetter, computed, toValue, watch } from 'vue';

import { useDebounce } from '@vueuse/core';
import { get } from 'lodash-es';
import { ApiServerService } from '~/api/modules/bkmsserver';
import { filterTimeFormat } from '~/common/util';
import { type RevisionType, isCommitID } from '~/components/code-revision/revision';
import { useAppDetail } from '~/stores/app-detail';

export type RecommendSource = {
  revision: string;
  revisionType: RevisionType;
};

/**
 * 推荐镜像 Tag。
 * 查询带上代码来源；CommitID 未填完或不合法时不发请求，避免把「不含修订」的版本号刷掉。
 */
export function useRecommendTag(
  sourceGetter: () => RecommendSource,
  options?: { manualFetchOnly?: MaybeRefOrGetter<boolean>; onRecommend?: (tag: string) => void },
) {
  const appDetailStore = useAppDetail();
  /** 手动确认和防抖 watch 会碰到同一次修订，记下 key 吃掉紧接着的那一次 */
  let echoedKey = '';
  /** 后返回的旧请求不能覆盖更新的修订 */
  let requestSerial = 0;

  function sourceKey(source: RecommendSource) {
    return `${source.revisionType}\0${source.revision}`;
  }

  /** 半截 CommitID 不是一次有效修订 */
  function shouldSkip(source: RecommendSource) {
    return source.revisionType === 'commit' && source.revision !== '' && !isCommitID(source.revision);
  }

  /** 来源已不再有效时，丢掉还在飞的推荐请求，避免它事后覆盖版本号 */
  function dropInFlight() {
    requestSerial += 1;
  }

  async function doFetch(source: RecommendSource): Promise<string> {
    if (shouldSkip(source)) {
      dropInFlight();
      return '';
    }

    const serial = ++requestSerial;
    const requestedKey = sourceKey(source);
    const rawTag = await ApiServerService.GetRecommendedImageTag({
      appID: appDetailStore.appID,
      revisionType: source.revisionType,
      revision: source.revision,
      // 分支仍带上兼容字段，旧服务只认 branch
      ...(source.revisionType === 'branch' ? { branch: source.revision } : {}),
    }).catch(() => '');

    // 序号挡后发先至；再对一下当前修订，避免防抖窗口里旧响应写回新输入
    const current = sourceGetter();
    if (serial !== requestSerial || sourceKey(current) !== requestedKey) return '';

    const tag = rawTag || filterTimeFormat(new Date(), 'YYYYmmddHHMM');
    options?.onRecommend?.(tag);
    return tag;
  }

  /**
   * 手动调用：切换来源、打开弹层、选中下拉时立刻重算。
   * 不拿 echoedKey 短路，否则同一默认分支再次打开弹层会跳过请求。
   * echoedKey 只留给紧接着的防抖 watch，避免同一次修订打两次。
   */
  async function fetchRecommendTag(): Promise<string> {
    const source = sourceGetter();
    if (shouldSkip(source)) {
      dropInFlight();
      return '';
    }

    echoedKey = sourceKey(source);
    return doFetch(source);
  }

  /** 获取应用默认构建分支（来自 buildConfig.repoBuildConfig.defaultBranch） */
  function getDefaultBranch(): string {
    return get(appDetailStore, 'appDetail.buildConfig.repoBuildConfig.defaultBranch') || '';
  }

  const liveSource = computed(sourceGetter);
  // 修订一变就作废在飞请求，不等 500ms。防抖只推迟下一次发起，manualFetchOnly 和空分支也不会跳过这次作废
  watch(
    liveSource,
    (next, prev) => {
      if (!prev || sourceKey(next) === sourceKey(prev)) return;
      dropInFlight();
    },
    { flush: 'sync' },
  );

  const debouncedSource = useDebounce(liveSource, 500);
  watch(debouncedSource, async source => {
    if (toValue(options?.manualFetchOnly)) return;
    if (shouldSkip(source)) {
      dropInFlight();
      return;
    }
    // 分支为空时没有修订可拼，打开弹层的那次请求由调用方显式发起
    if (!source.revision && source.revisionType === 'branch') return;

    const key = sourceKey(source);
    if (key === echoedKey) {
      echoedKey = '';
      return;
    }

    echoedKey = key;
    await doFetch(source);
  });

  return { getDefaultBranch, fetchRecommendTag };
}
