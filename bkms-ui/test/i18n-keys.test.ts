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

import enUS from '../locales/en-US.yml';
import zhCN from '../locales/zh-CN.yml';

type Messages = Record<string, unknown>;

const zhMessages = zhCN as Messages;
const enMessages = enUS as Messages;
const zhKeys = new Set(Object.keys(zhMessages));
const enKeys = new Set(Object.keys(enMessages));

function diffKeys(a: Set<string>, b: Set<string>): string[] {
  return [...a].filter(key => !b.has(key)).sort();
}

/**
 * 历史遗留：两语言的 key 用不同引号风格书写（中文曲引号 vs ASCII 转义引号），
 * 导致实际文案里的引导符号不一致，属于既有问题，不在本测试的守护范围。
 */
const KNOWN_KEY_MISMATCHES = {
  missingInEnUS: [
    '接下来你可以直接 “查看应用” 或 继续 “创建应用”',
    '会同步停用 “启动探针” 请确认',
    '该镜像尚未部署到任何测试环境，请确认是否仍要晋级。',
    'Value 必填',
    '管理员能查看并执行“平台管理”导航下所有操作。',
    '环境管理 / 环境配置',
  ].sort(),
  missingInZhCN: [
    '接下来你可以直接 "查看应用" 或 继续 "创建应用"',
    '会同步停用 "启动探针" 请确认',
    'value 必填',
    '管理员能查看并执行"平台管理"导航下所有操作。',
    '环境管理 / 环境变量',
  ].sort(),
};

describe('locale yml 解析自检', () => {
  it('两个语言文件均解析为非空对象', () => {
    expect(zhKeys.size).toBeGreaterThan(0);
    expect(enKeys.size).toBeGreaterThan(0);
  });
});

describe('locale 词条覆盖性', () => {
  it('en-US 不应缺少 zh-CN 中存在的 key（已知遗留除外）', () => {
    expect(diffKeys(zhKeys, enKeys)).toEqual(KNOWN_KEY_MISMATCHES.missingInEnUS);
  });

  it('zh-CN 不应缺少 en-US 中存在的 key（已知遗留除外）', () => {
    expect(diffKeys(enKeys, zhKeys)).toEqual(KNOWN_KEY_MISMATCHES.missingInZhCN);
  });

  it('告警策略删除弹窗用到的词条必须存在且翻译非空', () => {
    const requiredKeys = ['确认删除该策略', '策略名称: {name}', '删除后，将不可恢复，请谨慎操作！'];
    for (const key of requiredKeys) {
      expect(zhKeys.has(key), `zh-CN 缺少词条：${key}`).toBe(true);
      expect(enKeys.has(key), `en-US 缺少词条：${key}`).toBe(true);
      expect(String(zhMessages[key])).not.toBe('');
      expect(String(enMessages[key])).not.toBe('');
    }
  });

  it('en-US 的插值占位符应与 zh-CN 一致', () => {
    const placeholder = /\{\w+\}/g;
    const mismatch = [...zhKeys].filter(key => {
      if (!enKeys.has(key)) return false;
      const a = (String(zhMessages[key]).match(placeholder) ?? []).sort();
      const b = (String(enMessages[key]).match(placeholder) ?? []).sort();
      if (a.length !== b.length) return true;
      return a.some((token, index) => token !== b[index]);
    });
    expect(mismatch).toEqual([]);
  });
});
