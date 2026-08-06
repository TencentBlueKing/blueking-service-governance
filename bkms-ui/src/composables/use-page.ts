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

import type { ComputedRef, Ref } from 'vue';
import { computed, reactive } from 'vue';

export interface IOptions extends IPageConf {
  onPageChange?: (current: number) => any;
  onPageSizeChange?: (limit: number) => any;
}

export interface IPageConf {
  count?: number;
  current: number;
  limit: number;
  remote: boolean; // 是否远程分页，触发table刷新
}

export interface IPageConfResult {
  curPageData: ComputedRef<any[]>;
  handleResetPage: Function;
  pageConf: IPageConf;
  pagination: ComputedRef<IPagination>;
  pageChange: (current: number) => void;
  pageSizeChange: (size: number) => void;
}

export interface IPagination extends IPageConf {
  count?: number;
  showTotalCount: boolean;
}

/**
 * 前端分页，支持全量数据或单页数据
 * @param data 全量数据
 * @param options 配置数据
 */
export default function usePageConf<T>(
  data: Ref<T[]>,
  options: IOptions = {
    current: 1,
    limit: 10,
    count: 0,
    remote: false,
  },
  count?: Ref<number>,
): IPageConfResult {
  const pageConf = reactive<IPageConf>({
    current: options.current,
    limit: options.limit,
    remote: options.remote,
  });

  const curPageData = computed<T[]>(() => {
    const { current, limit } = pageConf;
    return data.value.slice(limit * (current - 1), limit * current);
  });

  const pageChange = (current = 1) => {
    pageConf.current = current;
    const { onPageChange = null } = options;
    onPageChange && typeof onPageChange === 'function' && onPageChange(current);
  };

  const pageSizeChange = (limit = 10) => {
    pageConf.limit = limit;
    pageConf.current = 1;
    const { onPageSizeChange = null } = options;
    onPageSizeChange && typeof onPageSizeChange === 'function' && onPageSizeChange(limit);
  };

  const pagination = computed<IPagination>(() => {
    if (!count?.value) {
      return {
        ...pageConf,
        count: data.value.length,
        showTotalCount: false,
      };
    }
    return {
      ...pageConf,
      count: count.value,
      showTotalCount: true,
    };
  });

  const handleResetPage = () => {
    pageConf.current = 1;
  };

  return {
    pageConf,
    pagination,
    curPageData,
    pageChange,
    pageSizeChange,
    handleResetPage,
  };
}
