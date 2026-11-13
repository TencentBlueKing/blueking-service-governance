import { describe, expect, it } from 'vitest';

import enUS from '../locales/en-US.yml';
import zhCN from '../locales/zh-CN.yml';

// 临时校验：locale yml 必须能被 unplugin-vue-i18n 正常解析（重复 key 会在此抛 Map keys must be unique）
describe('locale yml 解析自检', () => {
  it('两个语言文件均解析为非空对象', () => {
    expect(Object.keys(zhCN).length).toBeGreaterThan(0);
    expect(Object.keys(enUS).length).toBeGreaterThan(0);
  });
});
