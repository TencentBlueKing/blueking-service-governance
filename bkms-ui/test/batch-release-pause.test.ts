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
import { describe, expect, it } from 'vitest';
import { interpolate, resolvePauseText, resolvePauseTextDescriptor } from '~/common/batch-release';

/** 模拟 vue-i18n：模板原样返回，由调用方插值（dev 下运行时编译不可用时的行为） */
const rawT = (key: string) => key;

/** 模拟 vue-i18n：译文已插值 */
const interpolatingT = (key: string, params: (number | string)[] = []) => interpolate(key, params);

describe('暂停方式文案：三种展示形态', () => {
  it('card（批次卡片第二行）用逗号分隔', () => {
    expect(resolvePauseText('permanent', undefined, rawT, 'card')).toBe('永久暂停，手动继续');
    expect(resolvePauseText('duration', 600, rawT, 'card')).toBe('暂停 600 秒后自动继续');
    expect(resolvePauseText('none', undefined, rawT, 'card')).toBe('不停');
  });

  it('detail（列表 / 预览 Popover）用中点分隔', () => {
    expect(resolvePauseText('permanent', undefined, rawT, 'detail')).toBe('永久暂停 · 手动继续');
    expect(resolvePauseText('duration', 600, rawT, 'detail')).toBe('暂停 600 秒后自动继续');
    expect(resolvePauseText('none', undefined, rawT, 'detail')).toBe('不停');
  });

  it('step（进度步骤条副文本）补充「自动继续」语义', () => {
    expect(resolvePauseText('permanent', undefined, rawT, 'step')).toBe('永久暂停 · 手动继续');
    expect(resolvePauseText('duration', 600, rawT, 'step')).toBe('暂停 600 秒 · 自动继续');
    expect(resolvePauseText('none', undefined, rawT, 'step')).toBe('不停 · 自动继续');
  });

  it('同一 pauseMode 在三种形态下只有标点 / 后缀差异，主语一致', () => {
    for (const variant of ['card', 'detail', 'step'] as const) {
      expect(resolvePauseText('permanent', undefined, rawT, variant)).toContain('永久暂停');
      expect(resolvePauseText('none', undefined, rawT, variant)).toContain('不停');
    }
  });
});

describe('暂停方式文案：兜底与插值', () => {
  it('duration 未配置 / 非法秒数按 30 秒兜底，不出现「暂停 0 秒」', () => {
    expect(resolvePauseText('duration', undefined, rawT, 'detail')).toBe('暂停 30 秒后自动继续');
    expect(resolvePauseText('duration', 0, rawT, 'detail')).toBe('暂停 30 秒后自动继续');
    expect(resolvePauseText('duration', Number.NaN, rawT, 'detail')).toBe('暂停 30 秒后自动继续');
  });

  it('非法 pauseMode 降级为「不停」', () => {
    expect(resolvePauseText(undefined, 600, rawT, 'card')).toBe('不停');
    expect(resolvePauseText('unknown', 600, rawT, 'card')).toBe('不停');
  });

  it('key 缺参时仍完成插值（两种 t 实现结果一致）', () => {
    expect(resolvePauseText('duration', 600, rawT, 'step')).toBe(
      resolvePauseText('duration', 600, interpolatingT, 'step'),
    );
  });

  it('描述符形式供组件层自行翻译', () => {
    expect(resolvePauseTextDescriptor('duration', 600, 'step')).toEqual({
      key: '暂停 {0} 秒 · 自动继续',
      params: [600],
    });
    expect(resolvePauseTextDescriptor('permanent', 600, 'step')).toEqual({
      key: '永久暂停 · 手动继续',
      params: [],
    });
  });
});
