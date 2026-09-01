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
export type Schema<T> = Partial<Record<keyof T, SchemaItem<T>>>;

export type SchemaItem<T> = {
  default?: boolean | number | string | string[];
  requiredIf?: (data: Partial<T>) => boolean;
  selector: string;
  type: 'array' | 'checkbox' | 'input' | 'radio' | 'select';
};

export function createValidForm<T>(defaults: T, overrides?: Partial<T>): T {
  return {
    ...defaults,
    ...overrides,
  };
}

/**
 * 将字符串数组转换为字符串数组，如果字符串包含逗号，则转换为字符串数组
 * @param data 数据
 * @returns 转换后的数据
 */
export function transformFormData<T extends Record<string, string | string[]>>(data: T): T {
  return Object.keys(data).reduce(
    (acc, key) => {
      const value = data[key];
      if (value && typeof value === 'string' && value.split(',').length > 1) {
        acc[key as keyof T] = value.split(',').map(item => item.trim()) as T[keyof T];
      } else {
        acc[key as keyof T] = (value as string).trim() as T[keyof T];
      }
      return acc;
    },
    {} as unknown as T,
  );
}
