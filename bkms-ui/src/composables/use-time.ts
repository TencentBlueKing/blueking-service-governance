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

import { computed } from 'vue';

import dayjs from 'dayjs';
import duration from 'dayjs/plugin/duration';
import relativeTime from 'dayjs/plugin/relativeTime';
import timezone from 'dayjs/plugin/timezone';
// 如果需要支持时区，请导入相关插件
import utc from 'dayjs/plugin/utc';
import { i18n } from '~/modules/i18n';

import 'dayjs/locale/zh-cn';

// 扩展 dayjs 以使用插件
dayjs.extend(utc);
dayjs.extend(timezone);
dayjs.extend(duration);
dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

/**
 * 格式化时间：默认半个月内显示相对时间，超出显示绝对时间；alwaysRelative 可强制显示相对时间
 * @returns { text, tooltip } text 为展示文本，tooltip 为 hover 绝对时间（仅相对时间时有值）
 */
export function formatRelativeTimeWithTooltip(
  date: Date | null | string | undefined,
  options: { alwaysRelative?: boolean } = {},
): {
  text: string;
  tooltip: string;
} {
  if (!date) return { text: '--', tooltip: '' };

  const target = dayjs(date);
  if (!target.isValid()) return { text: '--', tooltip: '' };

  const now = dayjs();
  const absText = target.format('YYYY-MM-DD HH:mm:ss');
  const diffDays = now.diff(target, 'day');
  // 超过 15 天直接显示绝对时间
  if (!options.alwaysRelative && diffDays >= 15) return { text: absText, tooltip: '' };
  // 相对时间
  const relText = target.from(now);

  return { text: relText, tooltip: absText };
}

export function useGreeting() {
  const greetingText = computed(() => {
    const hour = new Date().getHours();
    if (hour < 12) return i18n.global.t('上午好');
    if (hour < 18) return i18n.global.t('下午好');
    return i18n.global.t('晚上好');
  });
  return { greetingText };
}

export default function useTime() {
  // 计算时间差，返回格式：xx小时xx分钟xx秒
  function calculateTimeDifference(startTime: number | string, endTime: number | string): string {
    // 计算时间差
    const start = dayjs(startTime);
    const end = dayjs(endTime);
    const diffInSeconds = end.diff(start, 'second');

    // 使用 dayjs 的 duration 获取时分秒
    const duration = dayjs.duration(diffInSeconds, 'seconds');

    // 格式化输出
    const hours = duration.hours();
    const minutes = duration.minutes();
    const seconds = duration.seconds();

    // 构建结果字符串
    const parts = [];
    if (hours > 0) parts.push(`${hours}小时`);
    if (minutes > 0) parts.push(`${minutes}分`);
    if (seconds > 0) parts.push(`${seconds}秒`);

    return parts.join('');
  }

  function parseTimeToSeconds(timeStr: string) {
    if (!timeStr || timeStr === '--') return 0;

    let totalSeconds = 0;
    // 匹配小时部分
    const hoursMatch = timeStr.match(/(\d+)小时/);
    if (hoursMatch) totalSeconds += parseInt(hoursMatch[1], 10) * 3600;

    // 匹配分钟部分
    const minutesMatch = timeStr.match(/(\d+)分/);
    if (minutesMatch) totalSeconds += parseInt(minutesMatch[1], 10) * 60;

    // 匹配秒部分
    const secondsMatch = timeStr.match(/(\d+)秒/);
    if (secondsMatch) totalSeconds += parseInt(secondsMatch[1], 10);

    return totalSeconds;
  }

  // 比较时间先后顺序
  function compareTime(startTime: number | string, endTime: number | string): boolean {
    const start = dayjs(startTime);
    const end = dayjs(endTime);
    return start.isBefore(end);
  }

  // 转换日期的函数
  function formatDateString(dateString: Date | number | string) {
    return dayjs(dateString).format('YYYY-MM-DD HH:mm:ss');
  }

  // 转换为相对时间
  function formatRelativeTime(dateString: Date | number | string) {
    return dayjs(dateString).locale('zh-cn').fromNow();
  }

  return {
    calculateTimeDifference,
    parseTimeToSeconds,
    compareTime,
    formatDateString,
    formatRelativeTime,
    formatRelativeTimeWithTooltip,
  };
}
