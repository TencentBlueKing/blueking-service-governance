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
import { type WritableComputedRef, computed, onMounted } from 'vue';

import { useRoute, useRouter } from 'vue-router';

type FieldsMap<F extends Record<string, UrlFieldConfig>> = {
  [K in keyof F]: WritableComputedRef<FieldValue<F[K]>>;
};

/** 根据 data.default 类型推断单值/数组模式 */
type FieldValue<C extends UrlFieldConfig> = C['data']['default'] extends readonly string[] ? string[] : string;

interface UrlFieldConfig {
  /** 字段取值策略：默认值、合法值列表、自定义覆盖 */
  data: UrlFieldData;
  /** URL query 参数名（如 activeTab、envNames） */
  queryKey: string;
}

interface UrlFieldData {
  /** 合法的值列表（可选）：单值模式用于校验，数组模式用于过滤非法项；不传则不校验直接透传 */
  allowed?: readonly string[];
  /** 默认值：单值模式传 string，数组模式传 string[]（由类型自动推断）；空默认值（'' 或 []）不补写 URL */
  default: string | string[];
  /** 自定义取值（可选，仅单值模式生效）：覆盖 URL 上的值，用于特殊场景（如多环境模式强制某 Tab） */
  override?: (valueFromQuery: string | undefined) => string;
}

type UrlQueryValue = string | string[];

/**
 * 页面状态与 URL query 字段的双向同步锚定（通用，不限于 Tab）：
 * - 支持任意 query key（queryKey），可同时管理多个字段（如 activeTab + envNames）
 * - 支持单值（string）与数组（string[]，重复参数 ?key=a&key=b）两种模式，由 data.default 类型自动推断
 * - get：从 route.query[queryKey] 读取并校验，非法值/空回退 data.default
 * - set：状态变化时通过 router.replace 写回 URL（数组由 vue-router 自动序列化为重复参数）
 * - onMounted：挂载时校验各字段，缺失补默认值、非法值收敛为实际渲染值，合并为一次 router.replace
 */
export function useUrlQuerySync<F extends Record<string, UrlFieldConfig>>(configs: F): { fields: FieldsMap<F> } {
  const route = useRoute();
  const router = useRouter();

  const fields = {} as unknown as FieldsMap<F>;

  for (const [name, config] of Object.entries(configs)) {
    const { queryKey, data } = config;
    const { allowed, default: defaultValue, override } = data;
    const isArrayMode = Array.isArray(defaultValue);

    fields[name as keyof F] = computed({
      get: () => {
        const raw = route.query[queryKey];
        // 数组模式：重复参数解析为数组，过滤非法值（未配置 allowed 时透传），空回退默认值
        if (isArrayMode) {
          const values = (Array.isArray(raw) ? raw : raw === undefined ? [] : [raw]).filter(
            (v): v is string => typeof v === 'string' && (!allowed?.length || allowed.includes(v)),
          );
          return values.length > 0 ? values : (defaultValue as string[]);
        }
        // 单值模式
        const valueFromQuery = typeof raw === 'string' ? raw : undefined;
        const candidate = override ? override(valueFromQuery) : valueFromQuery;
        // 未配置 allowed 时不校验，直接透传 URL 值（含 override 结果）
        if (!allowed?.length) return candidate ?? (defaultValue as string);
        // 校验 override 结果或 URL 值，非法回退默认值
        return candidate && allowed.includes(candidate) ? candidate : (defaultValue as string);
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

  // 挂载时校验：缺失字段补默认值，非法值收敛为当前渲染值（走 get 逻辑，含 override/allowed 校验），合并为一次 replace
  onMounted(() => {
    const query = { ...route.query };
    let needUpdate = false;
    for (const [name, config] of Object.entries(configs)) {
      const { queryKey, data } = config;
      const { default: defaultValue } = data;
      const raw = query[queryKey];
      // 空默认值且字段缺失时不补写（数组空 / 单值空字符串）
      if (raw === undefined && (Array.isArray(defaultValue) ? defaultValue.length === 0 : defaultValue === ''))
        continue;
      // vue-router 的 query 值类型含 null，规范化为 UrlQueryValue | undefined 再比较
      const rawValue: undefined | UrlQueryValue = Array.isArray(raw)
        ? raw.filter((v): v is string => typeof v === 'string')
        : (raw ?? undefined);
      // 期望值 = get 逻辑的结果（含 override / allowed 校验 / 数组过滤），与 URL 现值不一致才更新
      const expected = (fields[name as keyof F] as WritableComputedRef<string | string[]>).value;
      if (!isSameFieldValue(expected, rawValue)) {
        query[queryKey] = expected;
        needUpdate = true;
      }
    }
    if (needUpdate) {
      router.replace({ query });
    }
  });

  return { fields };
}

/** 规范化比较字段值：单值直接比较，数组按序逐项比较（忽略非字符串项） */
function isSameFieldValue(expected: string | string[], raw: undefined | UrlQueryValue): boolean {
  if (Array.isArray(expected)) {
    const rawArray = (Array.isArray(raw) ? raw : raw === undefined ? [] : [raw]).filter(
      (v): v is string => typeof v === 'string',
    );
    return expected.length === rawArray.length && expected.every((v, i) => v === rawArray[i]);
  }
  return expected === (typeof raw === 'string' ? raw : undefined);
}
