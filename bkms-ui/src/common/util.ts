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

import { Message } from 'bkui-vue';
import yaml from 'js-yaml';
import { isArray, isObject } from 'lodash-es';
import * as monaco from 'monaco-editor';
import { i18n } from '~/modules/i18n';

type DateSortOrder = 'asc' | 'desc';
type DateValue = Date | null | number | string | undefined;

export function calculateTimePassed(timeStr: Date | string) {
  try {
    const givenDt = new Date(filterTimeFormat(timeStr));
    const nowDt = new Date();

    // 输入时间晚于当前时间，返回0
    if (givenDt > nowDt) return '0';

    // 计算年份差
    let years = nowDt.getFullYear() - givenDt.getFullYear();
    const tempDate = new Date(givenDt); // 复制输入时间

    // 调整年份到 givenDt.year + years，检查是否超过当前时间
    tempDate.setFullYear(givenDt.getFullYear() + years);
    if (tempDate > nowDt) {
      years -= 1;
      tempDate.setFullYear(givenDt.getFullYear() + years);
    }

    // 计算剩余时间差（毫秒）
    const deltaMs = nowDt.getTime() - tempDate.getTime();

    // 分解为天、小时、分钟、秒
    const days = Math.floor(deltaMs / (1000 * 60 * 60 * 24)) ? `${Math.floor(deltaMs / (1000 * 60 * 60 * 24))}d` : '';
    const remainingMs = deltaMs % (1000 * 60 * 60 * 24);
    const hours = Math.floor(remainingMs / (1000 * 60 * 60)) ? `${Math.floor(remainingMs / (1000 * 60 * 60))}h` : '';
    const minutes = Math.floor((remainingMs % (1000 * 60 * 60)) / (1000 * 60))
      ? `${Math.floor((remainingMs % (1000 * 60 * 60)) / (1000 * 60))}m`
      : '';
    const seconds = Math.floor((remainingMs % (1000 * 60)) / 1000)
      ? `${Math.floor((remainingMs % (1000 * 60)) / 1000)}s`
      : '';

    return `${years ? `${years}y ` : ''}${days} ${hours} ${minutes} ${seconds}`;
  } catch (_) {
    return '0';
  }
}
// 格式化时间戳 dateFormat(date, "YYYY-mm-dd HH:MM:SS")
export function filterTimeFormat(date: Date | string, fmt = 'YYYY-mm-dd HH:MM:SS'): string {
  if (!date) return '';
  const newDate = new Date(date);
  const fmtArr = ['Y+', 'm+', 'd+', 'H+', 'M+', 'S+'];
  const opt: { [key: string]: string } = {
    'Y+': newDate.getFullYear().toString(), // 年
    'm+': (newDate.getMonth() + 1).toString(), // 月
    'd+': newDate.getDate().toString(), // 日
    'H+': newDate.getHours().toString(), // 时
    'M+': newDate.getMinutes().toString(), // 分
    'S+': newDate.getSeconds().toString(), // 秒
  };
  let res;
  let time = fmt;
  fmtArr.forEach(key => {
    res = new RegExp(`(${key})`).exec(fmt);
    if (res) {
      time = time.replace(res[1], res[1].length === 1 ? opt[key] : opt[key].padStart(res[1].length, '0'));
    }
  });
  return time;
}
/**
 *  @param {number}  timezone -12 - 12
 */
export function formatTimeByTimezone(date: Date | string, timezone?: number, fmt = 'YYYY-mm-dd HH:MM:SS') {
  let formatTime = '--';
  if (date) {
    try {
      const currentTimezone = new Date().getTimezoneOffset() / -60;
      const offsetTimezone = timezone || timezone === 0 ? currentTimezone - timezone : 0;
      const timeString = new Date(date).getTime();
      formatTime = filterTimeFormat(new Date(timeString - offsetTimezone * 60 * 60 * 1000), fmt);
    } catch (_) {
      // ignore timezone format error
    }
  }
  return formatTime;
}

/**
 * 计算两个时间戳之间的持续时间（项目专用）
 * @param startTime 开始时间
 * @param endTime 结束时间
 * @returns 格式化的持续时间字符串
 * @example
 * getDurationTime(start, end) // "1 day 2 hours 30 minutes 45 seconds" 或 "1天 2小时 30分钟 45秒"
 */
export function getDurationTime(startTime: Date | string, endTime: Date | string): string {
  try {
    const start = new Date(startTime).getTime();
    const end = new Date(endTime).getTime();
    const deltaMs = Math.abs(end - start);

    const days = Math.floor(deltaMs / (1000 * 60 * 60 * 24));
    const hours = Math.floor((deltaMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const minutes = Math.floor((deltaMs % (1000 * 60 * 60)) / (1000 * 60));
    const seconds = Math.floor((deltaMs % (1000 * 60)) / 1000);

    const parts = [];
    if (days > 0) parts.push(i18n.global.t('{count}天', days));
    if (hours > 0) parts.push(i18n.global.t('{count}小时', hours));
    if (minutes > 0) parts.push(i18n.global.t('{count}分钟', minutes));
    if (seconds > 0) parts.push(i18n.global.t('{count}秒', seconds));

    return parts.length > 0 ? parts.join('') : i18n.global.t('{count}秒', 0);
  } catch (_) {
    return i18n.global.t('{count}秒', 0);
  }
}

/**
 * 按日期排序，日期为空或无效的数据排在末尾
 * @param data 待排序的数据列表（原地排序）
 * @param getDate 获取每项日期值的方法
 * @param order 排序方向，asc 为正序，desc 为逆序，默认为 desc
 * @returns 按指定日期顺序排序后的原数据列表
 * @example
 * sortByDate(list, item => item.createdAt)
 * sortByDate(list, item => item.createdAt, 'asc')
 */
export function sortByDate<T>(data: T[], getDate: (item: T) => DateValue, order: DateSortOrder = 'desc'): T[] {
  const getTimestamp = (value: DateValue): null | number => {
    if (value === null || value === undefined || value === '') return null;

    let timestamp: number;
    if (value instanceof Date) {
      timestamp = value.getTime();
    } else if (typeof value === 'number') {
      timestamp = value;
    } else {
      timestamp = new Date(value).getTime();
    }

    return Number.isNaN(timestamp) ? null : timestamp;
  };

  return data.sort((a, b) => {
    const timeA = getTimestamp(getDate(a));
    const timeB = getTimestamp(getDate(b));
    if (timeA === null) return timeB === null ? 0 : 1;
    if (timeB === null) return -1;
    return order === 'asc' ? timeA - timeB : timeB - timeA;
  });
}

// 判断两个对象是否相等
export const deepEqual = (x: unknown, y: unknown) => {
  // 指向同一内存时
  if (x === y) {
    return true;
  }
  if (typeof x === 'object' && x != null && typeof y === 'object' && y != null) {
    const xRecord = x as Record<string, unknown>;
    const yRecord = y as Record<string, unknown>;
    if (Object.keys(xRecord).length !== Object.keys(yRecord).length) {
      return false;
    }

    for (const prop in xRecord) {
      if (Object.prototype.hasOwnProperty.call(yRecord, prop)) {
        if (!deepEqual(xRecord[prop], yRecord[prop])) return false;
      } else {
        return false;
      }
    }
    return true;
  }
  return false;
};

// base64解码
export function safeAtob(base64: string) {
  // 1. 解码 Base64 为二进制字符串
  let binaryString: string;
  try {
    binaryString = atob(base64);
  } catch (error) {
    if (error instanceof DOMException) {
      throw new Error(`Invalid base64 string: ${error.message}`);
    }
    throw error; // 重新抛出非DOMException的错误
  }

  // 2. 将二进制字符串转为 UTF-8 字节数组
  const bytes = Uint8Array.from(
    binaryString,
    char => char.charCodeAt(0), // 将每个字符转为 Unicode 码点
  );

  // 3. 将字节数组解码为原始字符串（UTF-8）
  const decoder = new TextDecoder('utf-8');
  return decoder.decode(bytes);
}

// base64编码
export function safeBtoa(str: string) {
  // 1. 将字符串转为 UTF-8 字节数组
  const encoder = new TextEncoder();
  const bytes = encoder.encode(str); // 例如：中文 "你" → [0xE4, 0xBD, 0xA0]

  // 2. 将字节数组转为二进制字符串（btoa 需要的格式）
  const binaryString = bytes.reduce((acc, byte) => acc + String.fromCharCode(byte), '');

  // 3. 编码为 Base64
  return btoa(binaryString);
}

export const copyText = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text);
    Message({
      theme: 'success',
      message: '复制成功',
    });
    return true;
  } catch (_) {
    // 回退方案：使用旧方法
    const textarea = document.createElement('textarea');
    document.body.appendChild(textarea);
    textarea.value = text;
    textarea.select();
    try {
      document.execCommand('copy');
      Message({
        theme: 'success',
        message: '复制成功',
      });
      return true;
    } catch (_) {
      return false;
    } finally {
      document.body.removeChild(textarea);
    }
  }
};

// 校验yaml返回错误行和错误信息
export interface IMonacoEditorErrorMarkerItem {
  endColumn: number;
  endLineNumber: number;
  message: string;
  severity: monaco.MarkerSeverity.Error;
  startColumn: number;
  startLineNumber: number;
}
export const validateYAML = (yamlString: string): IMonacoEditorErrorMarkerItem[] => {
  const markers: IMonacoEditorErrorMarkerItem[] = [];
  try {
    const docs: unknown[] = [];
    yaml.loadAll(yamlString, doc => docs.push(doc));
    // 校验每个 YAML 文档解析后是否为合法的 key-value 结构（object/map）
    // 如果解析结果是纯字符串、数字等非对象类型，说明格式不符合 values.yaml 的要求
    for (const doc of docs) {
      if (doc !== null && doc !== undefined && typeof doc !== 'object') {
        markers.push({
          severity: monaco.MarkerSeverity.Error,
          message: `cannot unmarshal !!${typeof doc} \`${String(doc).substring(0, 50)}\` into map[string]interface {}`,
          startLineNumber: 1,
          startColumn: 1,
          endLineNumber: 1,
          endColumn: 1,
        });
      }
    }
  } catch (e: unknown) {
    if (e instanceof Error && 'mark' in e) {
      const yamlError = e as unknown as { mark: { column: number; line: number }; reason: string };
      markers.push({
        severity: monaco.MarkerSeverity.Error,
        message: yamlError.reason,
        startLineNumber: yamlError.mark.line,
        startColumn: yamlError.mark.column,
        endLineNumber: yamlError.mark.line,
        endColumn: yamlError.mark.column,
      });
    }
  }
  return markers;
};

/** 支持的 query 参数值类型 */
type QueryParams = Record<string, QueryValue | QueryValue[]>;

type QueryValue = boolean | null | number | string | undefined;
export function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#x27;');
}

export function objectToQueryParams(params: Record<string, unknown>): string | undefined {
  if (!isObject(params) || !Object.keys(params)?.length) {
    return;
  }

  const encodeValue = (value: unknown) => encodeURIComponent(String(value));

  function buildParams(obj: unknown, prefix = ''): string[] {
    const result: string[] = [];
    if (!isObject(obj)) return result;
    const objRecord = obj as Record<string, unknown>;
    Object.keys(objRecord).forEach(key => {
      const value = objRecord[key];
      const paramKey = prefix ? `${prefix}.${key}` : key;
      if (isArray(value)) {
        value.forEach(item => {
          if (isObject(item)) {
            result.push(...buildParams(item, paramKey));
          } else {
            result.push(`${encodeURIComponent(paramKey)}=${encodeValue(item)}`);
          }
        });
      } else if (isObject(value)) {
        result.push(...buildParams(value, paramKey));
      } else {
        result.push(`${encodeURIComponent(paramKey)}=${encodeValue(value)}`);
      }
    });
    return result;
  }

  return buildParams(params).join('&');
}

/**
 * 将对象转换为 URL query 参数字符串
 * @param obj 参数对象，支持基本类型和数组
 * @returns 编码后的 query 字符串（不含 `?` 前缀）
 *
 * @example
 * objectToUrlParams({ name: 'test', ids: [1, 2] })
 * // => 'name=test&ids=1&ids=2'
 */
export function objectToUrlParams(obj: QueryParams): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(obj)) {
    if (value === null || value === undefined) continue;
    if (Array.isArray(value)) {
      for (const v of value) {
        if (v !== null && v !== undefined) {
          params.append(key, String(v));
        }
      }
    } else {
      params.append(key, String(value));
    }
  }
  return params.toString();
}

export const parseCss = (v: number | string) => (typeof v === 'string' ? v : `${v}px`);

/**
 * 将 JSON 转换为 YAML
 * @param data JSON 数据或 JSON 字符串
 * @returns YAML 字符串
 */
export function convertToYaml(data: unknown): string {
  if (!data) return '';
  try {
    const jsonData = typeof data === 'string' ? JSON.parse(data) : data;
    return yaml.dump(jsonData as Record<string, unknown>, { indent: 2 });
  } catch (_) {
    return typeof data === 'string' ? data : JSON.stringify(data, null, 2);
  }
}

/**
 *
 * @param oldArr
 * @param newArr
 * @param key
 * @returns
 */

export function diffArrayFast<T extends Record<string, unknown>>(oldArr: T[], newArr: T[], key = 'id') {
  const oldMap = new Map();
  const newMap = new Map();

  const added: T[] = [];
  const removed: T[] = [];
  const common: T[] = [];

  // 建立索引
  for (const item of oldArr) {
    oldMap.set(item[key], item);
  }

  for (const item of newArr) {
    newMap.set(item[key], item);
  }

  // 找新增 & common
  for (const [k, newItem] of newMap) {
    if (!oldMap.has(k)) {
      added.push(newItem);
    } else {
      common.push(newItem);
    }
  }

  // 找删除
  for (const [k, oldItem] of oldMap) {
    if (!newMap.has(k)) {
      removed.push(oldItem);
    }
  }

  return { added, removed, common };
}

export function downloadBase64File(base64: string, filename: string, mimeType: string) {
  // 解码 Base64
  const binaryString = atob(base64);
  const bytes = new Uint8Array(binaryString.length);
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }

  // 创建 Blob
  const blob = new Blob([bytes], { type: mimeType });

  // 触发下载
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

/**
 * 通过创建 a 标签触发文件下载（适用于大文件，浏览器原生处理下载流）
 * @param url 完整的下载 URL（应包含认证信息或依赖 Cookie）
 * @param filename 下载文件名
 */
export function downloadByLink(url: string, filename: string) {
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.target = '_blank';
  a.rel = 'noopener noreferrer';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
}

/*
 * 下载原始文件响应，优先使用 Content-Disposition 中的服务端文件名。
 */
export async function downloadResponseFile(response: Response, fallbackFilename: string) {
  const blob = await response.blob();
  const encodedFilename =
    response.headers.get('Content-Disposition')?.match(/filename\*?\s*=\s*(?:UTF-8'')?"?([^";]+)"?/i)?.[1] ||
    fallbackFilename;
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = decodeURIComponent(encodedFilename);
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

/**
 * 格式化文件大小
 * @param size 文件大小（字节）
 * @param placeholder 为空时的占位符，默认 '--'
 * @returns 格式化后的字符串，如 1.23 MB
 */
export function formatSize(size: number, placeholder = '--'): string {
  if (!size) return placeholder;
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(2)} KB`;
  if (size < 1024 * 1024 * 1024) return `${(size / (1024 * 1024)).toFixed(2)} MB`;
  return `${(size / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

/**
 * 根据数组字段生成选项列表（带统计数量）
 *
 * @example
 * // 基础用法：生成分类选项
 * generateFieldOptions(abnormalItems, 'category', {
 *   allItem: { label: '全部问题', value: 0 }
 * })
 * // => [{ label: '全部问题', value: 0, count: 10 }, { label: '节点异常', value: 1, count: 5 }]
 *
 * @example
 * // 使用原始字段值作为 value（适合筛选场景）
 * generateFieldOptions(items, 'status', {
 *   valueStrategy: 'field'  // value 为 'running'、'stopped' 等原始值
 * })
 */
export function generateFieldOptions<T extends object>(
  items: T[],
  key: keyof T,
  options?: {
    allItem?: { label: string; value: number | string };
    labelFormatter?: (fieldValue: unknown) => string;
    valueStrategy?: 'field' | 'index';
  },
) {
  const { allItem, labelFormatter, valueStrategy = 'index' } = options || {};

  // 统计各分类数量
  const fieldCount: Record<string, number> = {};
  items.forEach(item => {
    const fieldValue = item[key];
    if (fieldValue) {
      const strValue = String(fieldValue);
      fieldCount[strValue] = (fieldCount[strValue] || 0) + 1;
    }
  });

  const totalCount = items.length;

  // 动态生成分类列表
  const result: Array<{ count?: number; label: string; value: number | string }> = [];

  // 如果传入了allItem，添加"全部"选项
  if (allItem) {
    result.push({
      label: allItem.label,
      value: allItem.value,
      count: totalCount,
    });
  }

  // 根据实际字段值动态添加
  let index = allItem ? 1 : 0;
  Object.keys(fieldCount).forEach((fieldValue: string) => {
    const label = labelFormatter ? labelFormatter(fieldValue) : fieldValue;
    const value = valueStrategy === 'field' ? fieldValue : index++;
    result.push({
      label,
      value,
      count: fieldCount[fieldValue],
    });
  });

  return result;
}

/**
 * 跳转到蓝盾流水线详情页
 */
export function gotoPipelineDetail(bkCIProjectID: string, pipelineID: string, pipelineBuildID: string) {
  if (!bkCIProjectID || !pipelineID || !pipelineBuildID) return;
  const [project, pipeline, build] = [bkCIProjectID, pipelineID, pipelineBuildID].map(encodeURIComponent);
  window.open(`${import.meta.env.BK_DEVOPS}/console/pipeline/${project}/${pipeline}/detail/${build}`);
}

/**
 * 检查 API 错误响应中是否包含指定的错误码
 * 错误结构: err.error.details[].code
 * @param err 错误对象
 * @param code 要匹配的错误码
 * @returns 是否包含该错误码
 */
export function hasErrorCode(err: unknown, code: string): boolean {
  const details = (err as { error?: { details?: { code: string }[] } })?.error?.details;
  return Array.isArray(details) && details.some(detail => detail.code === code);
}

/**
 * 映射数组对象的字段名
 * @param array 源数组
 * @param fieldMap 字段映射关系 { 原字段名: 新字段名 }
 * @returns 映射后的新数组
 *
 * @example
 * mapKeys([{ name: 'foo', id: 1 }], { name: 'label', id: 'value' })
 * // => [{ label: 'foo', value: 1 }]
 */
export function mapKeys<T extends object>(array: T[], fieldMap: Record<string, string>) {
  return array.map(item => replaceKey(item, fieldMap));
}

/**
 * 将对象映射转换为选项数组
 * @param map 源对象映射
 * @param fieldMap 字段映射关系，例如 { name: 'label', id: 'value' }
 * @returns 转换后的选项数组
 *
 * @example
 * mapToOptions(statusMap, { name: 'label', id: 'value' })
 * // => [{ label: '成功', value: 'success' }, { label: '失败', value: 'fail' }]
 */
export function mapToOptions<T extends Record<string, unknown>>(
  map: Record<string, T>,
  fieldMap: Record<string, string>,
) {
  return Object.keys(map).map(key => replaceKey(map[key], fieldMap));
}

/**
 * 解析 YAML 格式的字符串为对象
 * @param raw - YAML 字符串
 * @returns 解析后的对象，解析失败返回空对象
 */
export function parseYamlValues(raw?: string): Record<string, unknown> {
  if (!raw) return {};
  try {
    const parsed = yaml.load(raw);
    return typeof parsed === 'object' && parsed !== null ? (parsed as Record<string, unknown>) : {};
  } catch {
    return {};
  }
}

/**
 * 字段映射核心逻辑
 * @param source 源对象
 * @param fieldMap 字段映射关系 { 新字段名: 原字段名 }
 * @returns 映射后的新对象
 */
export function replaceKey<T extends object>(source: T, fieldMap: Record<string, string>): Record<string, unknown> {
  const result: Record<string, unknown> = {};
  const record = source as Record<string, unknown>;
  Object.entries(fieldMap).forEach(([newKey, oldKey]) => {
    if (oldKey in record) {
      result[newKey] = record[oldKey];
    }
  });
  return result;
}

/**
 * 接口错误提示。
 * 适用于请求关闭 interceptorErr 后，需要复用 fetch.ts 中 HTTP 异常 Message 展示结构的场景。
 * @param error 错误对象
 */
export function showApiErrorMessage(error: unknown) {
  const apiError = error as {
    code?: number | string;
    error?: {
      message?: string;
    };
    message?: string;
    status?: number | string;
  };
  Message({
    theme: 'error',
    actions: [
      {
        id: 'assistant',
        disabled: true,
      },
    ],
    message: {
      code: apiError?.status ?? apiError?.code,
      overview: apiError?.error?.message || apiError?.message || i18n.global.t('请求异常'),
      suggestion: '',
      type: 'json',
      details: `${JSON.stringify(apiError?.error || {}, null, 2)}`,
    },
  });
}
