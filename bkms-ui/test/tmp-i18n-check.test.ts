import { describe, expect, it } from 'vitest';
import { createI18n } from 'vue-i18n';

import zhCN from '../locales/zh-CN.yml';

describe('i18n 插值自检', () => {
  it('打印带占位符的译文', () => {
    const i18n = createI18n({ legacy: false, locale: 'zh-CN', messages: { 'zh-CN': zhCN } });
    const cast = i18n.global as unknown as { t: (key: string, params?: unknown[]) => string };
    console.log('RESULT_LABEL=' + cast.t('第 {0} 批', [1]));
    console.log('RESULT_ACC=' + cast.t('累计 {0}', ['1 个']));
    console.log('RESULT_PAUSE=' + cast.t('暂停 {0} 秒后自动继续', [600]));
    console.log('RESULT_TYPE=' + typeof (zhCN as Record<string, unknown>)['第 {0} 批']);
    expect(1).toBe(1);
  });
});
