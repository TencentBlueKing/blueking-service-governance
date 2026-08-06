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

export default function useTableSort<T>() {
  interface SortParams {
    data: T[];
    sortList: {
      field: string;
      order: 'asc' | 'desc';
    }[];
  }
  type CustomLogic = (itemA: T[keyof T], itemB: T[keyof T]) => any;

  function sortMethod({ data, sortList }: SortParams, customLogic?: CustomLogic) {
    const curField = sortList[0].field as keyof T;
    const curOrder: 'asc' | 'desc' = sortList[0].order;
    return data.sort((a: T, b: T) => {
      let compareResult;
      // 日期字段比较
      if ((['createAt', 'updatedAt'] as (keyof T)[]).includes(curField)) {
        const timeA =
          a[curField] && !isNaN(new Date(a[curField] as string).getTime())
            ? new Date(a[curField] as string).getTime()
            : -Infinity;
        const timeB =
          b[curField] && !isNaN(new Date(b[curField] as string).getTime())
            ? new Date(b[curField] as string).getTime()
            : -Infinity;

        compareResult = timeA - timeB;
      } else if (customLogic) {
        compareResult = customLogic(a[curField], b[curField]);
      } else {
        // 其他字段按字符串比较
        const strA = String(a[curField] || '');
        const strB = String(b[curField] || '');
        compareResult = strA.localeCompare(strB);
      }
      return curOrder === 'asc' ? compareResult : -compareResult;
    });
  }

  return {
    sortMethod,
  };
}
