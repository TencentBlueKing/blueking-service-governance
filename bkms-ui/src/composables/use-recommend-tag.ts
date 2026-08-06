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

import { computed, watch } from 'vue';

import { useDebounce } from '@vueuse/core';
import { get } from 'lodash-es';
import { ApiServerService } from '~/api/modules/bkmsserver';
import { filterTimeFormat } from '~/common/util';
import { useAppDetail } from '~/stores/app-detail';

/**
 * 推荐镜像 Tag 相关逻辑
 * - onRecommend: 获取到推荐 Tag 后的回调
 * - getDefaultBranch: 获取应用默认构建分支
 *
 * @param branchGetter 获取当前分支的函数
 * @param options.onRecommend 获取到推荐 Tag 后的回调
 */
export function useRecommendTag(branchGetter: () => string, options?: { onRecommend?: (tag: string) => void }) {
  const appDetailStore = useAppDetail();
  let skipNextWatch = false;

  /** 纯请求逻辑，不碰标志位 */
  async function doFetch(branch: string): Promise<string> {
    const rawTag = await ApiServerService.GetRecommendedImageTag({
      appID: appDetailStore.appID,
      branch,
    }).catch(() => '');
    const tag = rawTag || filterTimeFormat(new Date(), 'YYYYmmddHHMM');
    options?.onRecommend?.(tag);
    return tag;
  }

  /** 手动调用：设 skipNextWatch 跳过接下来的 watch 触发，避免重复请求 */
  async function fetchRecommendTag(branch: string): Promise<string> {
    skipNextWatch = true;
    return doFetch(branch);
  }

  /** 获取应用默认构建分支（来自 buildConfig.repoBuildConfig.defaultBranch） */
  function getDefaultBranch(): string {
    return get(appDetailStore, 'appDetail.buildConfig.repoBuildConfig.defaultBranch') || '';
  }

  // debounce watch：分支变化时自动获取推荐 Tag
  const debounceBranch = useDebounce(computed(branchGetter), 500);
  watch(debounceBranch, async newBranch => {
    if (skipNextWatch) {
      skipNextWatch = false;
      return;
    }
    if (newBranch) {
      await doFetch(newBranch);
    }
  });

  return { getDefaultBranch, fetchRecommendTag };
}
