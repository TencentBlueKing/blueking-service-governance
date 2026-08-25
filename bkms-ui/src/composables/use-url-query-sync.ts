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

/** 挂载时对单个字段的处理结果 */
type MountFieldResult = { type: 'remove' } | { type: 'skip' } | { type: 'write'; value: UrlQueryValue };

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
  /**
   * 自定义取值（可选，仅单值模式生效）：覆盖 URL 上的值，用于特殊场景（如多环境模式强制某 Tab）
   * 返回空字符串 '' 表示该字段在当前场景不参与 URL 同步（挂载时不写回、并从 URL 移除）
   */
  override?: (valueFromQuery: string | undefined) => string;
}

type UrlQueryValue = string | string[];

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
          return resolveArrayValue(raw, allowed, defaultValue as string[]);
        }
        // 单值模式：URL → override 转换 → allowed 校验 → 默认值回退
        return resolveSingleValue(raw, allowed, defaultValue as string, override);
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

  // 挂载时校验：缺失字段补默认值，非法值收敛为实际渲染值，与 URL 现值有差异才合并为一次 replace
  onMounted(() => {
    const query = { ...route.query };
    let needReplace = false;
    // 逐个字段校验：缺失补默认 / override 禁用则移除 / 有差异才写回
    for (const [name, config] of Object.entries(configs)) {
      const { queryKey, data } = config;
      // 当前渲染值 = get 逻辑（URL 优先 → 默认值，含 override / allowed 校验）
      const currentValue = (fields[name as keyof F] as WritableComputedRef<string | string[]>).value;
      // 决定该字段挂载时的处理方式：skip（不改）/ remove（移除）/ write（写回）
      const result = resolveFieldOnMount(query[queryKey], currentValue, data.default);
      if (result.type === 'skip') {
        continue;
      }
      needReplace = true;
      if (result.type === 'remove') {
        delete query[queryKey];
        continue;
      }
      query[queryKey] = result.value;
    }
    // 有差异才 replace。故意不在 onBeforeUnmount 清 query：
    // 切应用时 detail 会先卸载子页再 await push，卸载 replace 会打断 push；
    // 跨菜单已由 detail 传 query: undefined；同菜单靠快照 + 本处写回。
    if (needReplace) {
      router.replace({ query });
    }
  });

  return { fields };
}

/** 默认值为空（单值 '' / 数组 []）时，URL 缺失场景无需补写 */
function isDefaultValueEmpty(defaultValue: string | string[]): boolean {
  if (Array.isArray(defaultValue)) {
    return defaultValue.length === 0;
  }
  return defaultValue === '';
}

/** 规范化比较字段值：单值直接比较，数组按序逐项比较 */
function isSameFieldValue(expected: string | string[], raw: string | string[] | undefined): boolean {
  if (Array.isArray(expected)) {
    const rawArray = toArray(raw);
    if (expected.length !== rawArray.length) {
      return false;
    }
    return expected.every((value, index) => value === rawArray[index]);
  }
  return expected === raw;
}

/** 规范化 URL 原值：过滤 null / 非字符串项，单值模式收敛为 string | undefined */
function normalizeRawQueryValue(raw: unknown): string | string[] | undefined {
  if (Array.isArray(raw)) {
    return raw.filter((value): value is string => typeof value === 'string');
  }
  if (typeof raw === 'string') {
    return raw;
  }
  return undefined;
}

/** 数组模式取值：URL 值过滤非法项（未配置 allowed 时透传），空数组回退默认值 */
function resolveArrayValue(raw: unknown, allowed: readonly string[] | undefined, defaultValue: string[]): string[] {
  const values = toArray(normalizeRawQueryValue(raw));
  let filtered = values;
  if (allowed?.length) {
    filtered = values.filter(value => allowed.includes(value));
  }
  if (filtered.length > 0) {
    return filtered;
  }
  return defaultValue;
}

/**
 * 挂载时决定单个字段的处理方式（按顺序判断）：
 * 1. URL 缺失且默认值为空 → skip（无需补写）
 * 2. 当前渲染值为空字符串（override 禁用）→ remove（该字段不参与 URL 同步，从 URL 移除）
 * 3. 当前渲染值与 URL 现值一致 → skip（无需写回）
 * 4. 其余情况 → write（写回期望值；URL 缺失时默认值必然不等，会被写回）
 */
function resolveFieldOnMount(
  rawFromUrl: unknown,
  currentValue: string | string[],
  defaultValue: string | string[],
): MountFieldResult {
  if (rawFromUrl === undefined && isDefaultValueEmpty(defaultValue)) {
    return { type: 'skip' };
  }
  if (currentValue === '') {
    return { type: 'remove' };
  }
  const normalizedRaw = normalizeRawQueryValue(rawFromUrl);
  if (isSameFieldValue(currentValue, normalizedRaw)) {
    return { type: 'skip' };
  }
  return { type: 'write', value: currentValue };
}

/** 单值模式取值：URL → override 转换 → allowed 校验，非法/缺失回退默认值 */
function resolveSingleValue(
  raw: unknown,
  allowed: readonly string[] | undefined,
  defaultValue: string,
  override: ((valueFromQuery: string | undefined) => string) | undefined,
): string {
  const valueFromQuery = typeof raw === 'string' ? raw : undefined;
  const candidate = override ? override(valueFromQuery) : valueFromQuery;
  // override 返回空字符串：该场景字段不参与 URL 同步
  if (candidate === '') {
    return '';
  }
  // 未配置 allowed 时不校验，直接透传 URL 值（含 override 结果）
  if (!allowed?.length) {
    return candidate ?? defaultValue;
  }
  // 校验 override 结果或 URL 值，非法回退默认值
  if (candidate && allowed.includes(candidate)) {
    return candidate;
  }
  return defaultValue;
}

/** 将规范化后的 URL 值转为数组：undefined 视为空数组，单值包装为数组 */
function toArray(value: string | string[] | undefined): string[] {
  if (Array.isArray(value)) {
    return value;
  }
  if (value === undefined) {
    return [];
  }
  return [value];
}
