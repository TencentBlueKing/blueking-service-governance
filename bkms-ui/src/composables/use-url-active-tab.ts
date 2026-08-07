/*
 * Tencent is pleased to support the open source community by making
 * 蓝鲸智云PaaS平台 (BlueKing PaaS) available.
 *
 * Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
 *
 * 蓝鲸智云PaaS平台 (BlueKing PaaS) is licensed under the MIT License.
 *
 * License for 蓝鲸智云PaaS平台 (BlueKing PaaS):
 *
 * ---------------------------------------------------
 * Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
 * documentation files (the "Software"), to deal in the Software without restriction, including without limitation
 * the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and
 * to permit persons to whom the Software is furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all copies or substantial portions of
 * the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO
 * THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF
 * CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
 * IN THE SOFTWARE.
 */
import { type WritableComputedRef, computed, onBeforeUnmount, onMounted } from 'vue';

import { useRoute, useRouter } from 'vue-router';

type FieldsMap<F extends Record<string, UrlFieldConfig>> = {
  [K in keyof F]: WritableComputedRef<FieldValue<F[K]>>;
};

/** 根据 defaultTab 类型推断单值/数组模式 */
type FieldValue<C extends UrlFieldConfig> = C['defaultTab'] extends readonly string[] ? string[] : string;

interface UrlFieldConfig {
  /** 默认值：单值模式传 string，数组模式传 string[]（由类型自动推断） */
  defaultTab: UrlQueryValue;
  /** URL query 参数名（如 activeTab、envNames） */
  queryKey: string;
  /** 合法的值列表（可选）：单值模式用于校验，数组模式用于过滤非法项；不传则不校验直接透传 */
  tabValues?: readonly string[];
  /** 自定义 get 逻辑（可选，仅单值模式生效），用于特殊场景（如多环境模式强制某 Tab） */
  getTab?: (tabFromQuery: string | undefined) => string;
}

type UrlQueryValue = string | string[];

/**
 * 页面状态与 URL query 字段的双向同步锚定（通用，不限于 Tab）：
 * - 支持任意 query key（queryKey），可同时管理多个字段（如 activeTab + envNames）
 * - 支持单值（string）与数组（string[]，重复参数 ?key=a&key=b）两种模式，由 defaultTab 类型自动推断
 * - get：从 route.query[queryKey] 读取并校验，非法值/空回退 defaultTab
 * - set：状态变化时通过 router.replace 写回 URL（数组由 vue-router 自动序列化为重复参数）
 * - onMounted：首次进入时把所有缺失字段合并为一次 router.replace 补全默认值
 * - onBeforeUnmount：卸载时统一清理各字段（若父级菜单切换已完成清理则跳过）
 */
export function useUrlActiveTab<F extends Record<string, UrlFieldConfig>>(configs: F): { fields: FieldsMap<F> } {
  const route = useRoute();
  const router = useRouter();

  const fields = {} as unknown as FieldsMap<F>;

  for (const [name, config] of Object.entries(configs)) {
    const { queryKey, tabValues, defaultTab, getTab } = config;
    const isArrayMode = Array.isArray(defaultTab);

    fields[name as keyof F] = computed({
      get: () => {
        const raw = route.query[queryKey];
        // 数组模式：重复参数解析为数组，过滤非法值（未配置 tabValues 时透传），空回退默认值
        if (isArrayMode) {
          const values = (Array.isArray(raw) ? raw : raw === undefined ? [] : [raw]).filter(
            (v): v is string => typeof v === 'string' && (!tabValues?.length || tabValues.includes(v)),
          );
          return values.length > 0 ? values : (defaultTab as string[]);
        }
        // 单值模式
        const tabFromQuery = typeof raw === 'string' ? raw : undefined;
        const candidate = getTab ? getTab(tabFromQuery) : tabFromQuery;
        // 未配置 tabValues 时不校验，直接透传 URL 值（含 getTab 结果）
        if (!tabValues?.length) return candidate ?? (defaultTab as string);
        // 校验 getTab 结果或 URL 值，非法回退默认值
        return candidate && tabValues.includes(candidate) ? candidate : (defaultTab as string);
      },
      set: (value: UrlQueryValue) => {
        router.replace({
          query: {
            ...route.query,
            [queryKey]: value,
          },
        });
      },
    }) as unknown as FieldsMap<F>[keyof F];
  }

  // 首次进入：合并所有缺失字段，一次 replace 补全默认值（空默认值不补写：数组空 / 单值空字符串）
  onMounted(() => {
    const query = { ...route.query };
    let needUpdate = false;
    for (const config of Object.values(configs)) {
      const { queryKey, defaultTab } = config;
      if (query[queryKey] === undefined) {
        if (Array.isArray(defaultTab) ? defaultTab.length === 0 : defaultTab === '') continue;
        query[queryKey] = defaultTab;
        needUpdate = true;
      }
    }
    if (needUpdate) {
      router.replace({ query });
    }
  });

  // 卸载时统一清理各字段（若父级菜单切换已完成清理则跳过）
  onBeforeUnmount(() => {
    const unloadPath = route.path;
    if (Object.values(configs).some(config => route.query[config.queryKey])) {
      setTimeout(() => {
        const current = router.currentRoute.value;
        // 仅当仍停留在同一路径时才清理，避免覆盖跳转后新页面的 query
        if (current.path !== unloadPath) return;
        if (Object.values(configs).some(config => current.query[config.queryKey])) {
          const restQuery = { ...current.query };
          for (const config of Object.values(configs)) {
            delete restQuery[config.queryKey];
          }
          router.replace({ query: restQuery });
        }
      }, 0);
    }
  });

  return { fields };
}
