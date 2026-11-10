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

/**
 * 图标三套载体的一致性守门：
 * A. 字体图标：`src/fonts/style.css` 的 `.bkms-icon-x:before{content:"\eXXX"}` + `fonts/iconcool.*`（码点契约）
 * B. SVG sprite：`src/fonts/iconcool.js` 插入的 `<symbol id="bkms-icon-x">`（id 契约）
 * C. 资源 svg：`src/assets/*.svg`，经 vite-svg-loader 内联（内部 id 契约）
 *
 * 这几类问题都只会在运行期以“空白/豆腐块”的形式暴露，静态断言最划算。
 */
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { beforeAll, describe, expect, it } from 'vitest';
import { scopeIconSpriteIds, SPRITE_ID_PREFIX, SPRITE_SELECTOR, SYMBOL_ID_PREFIX } from '~/fonts/icon-sprite';

// 副作用导入：iconcool.js 会把 sprite 插到 <body> 开头，等价于 application 启动时的行为
import '~/fonts/iconcool';

/** vitest 的 root 即项目根目录 */
const ROOT = process.cwd();
const readFile = (relPath: string) => readFileSync(join(ROOT, relPath), 'utf8');

/** 仅出现在注释/文档里、并非真实图标的类名 */
const DOC_ONLY_ICON_NAMES = new Set(['exmpale']);

const CSS_SOURCE = readFile('src/fonts/style.css');
const SPRITE_SOURCE = readFile('src/fonts/iconcool.js');

/** 字体侧：class 名 -> 码点 */
const fontIcons = new Map(
  [...CSS_SOURCE.matchAll(/\.bkms-icon-([\w-]+):before\s*\{\s*content:\s*['"]\\([\da-f]{4,6})['"]/gi)].map(
    matched => [matched[1], Number.parseInt(matched[2], 16)] as const,
  ),
);

/** sprite 侧：symbol 名 */
const spriteIcons = new Set([...SPRITE_SOURCE.matchAll(/<symbol id="bkms-icon-([\w-]+)"/g)].map(matched => matched[1]));

const walk = (dir: string, exts: RegExp, out: string[] = []): string[] => {
  for (const entry of readdirSync(join(ROOT, dir), { withFileTypes: true })) {
    const rel = `${dir}/${entry.name}`;
    if (entry.isDirectory()) walk(rel, exts, out);
    else if (exts.test(entry.name)) out.push(rel);
  }
  return out;
};

describe('字体图标 / sprite / 资源 svg 一致性', () => {
  it('style.css 的图标码点不重复（重复会让两个图标渲染成同一个字形）', () => {
    const duplicated = [...fontIcons.entries()].reduce<Record<string, string[]>>((acc, [name, code]) => {
      const key = code.toString(16);
      acc[key] = [...(acc[key] ?? []), name];
      return acc;
    }, {});
    const conflicts = Object.entries(duplicated).filter(([, names]) => names.length > 1);
    expect(conflicts).toEqual([]);
  });

  it('代码里引用的 bkms-icon-* 都能在字体与 sprite 中找到', () => {
    const used = new Set<string>();
    for (const file of walk('src', /\.(vue|tsx?)$/)) {
      // src/fonts 是图标定义/说明本身，不是“使用方”
      if (file.startsWith('src/fonts/')) continue;
      for (const matched of readFile(file).matchAll(/bkms-icon-([\w-]+)/g)) {
        if (!DOC_ONLY_ICON_NAMES.has(matched[1])) used.add(matched[1]);
      }
    }

    expect([...used].filter(name => !fontIcons.has(name))).toEqual([]);
    expect([...used].filter(name => !spriteIcons.has(name))).toEqual([]);
  });

  it('字体与 sprite 的图标集合一一对应（导出漏勾选会在这里暴露）', () => {
    expect([...fontIcons.keys()].filter(name => !spriteIcons.has(name))).toEqual([]);
    expect([...spriteIcons].filter(name => !fontIcons.has(name))).toEqual([]);
  });

  it('资源 svg 里的 url(#id) 引用能在同一文件内找到定义', () => {
    const broken: string[] = [];
    for (const file of walk('src/assets', /\.svg$/)) {
      const source = readFile(file);
      const defined = new Set([...source.matchAll(/\sid="([^"]+)"/g)].map(matched => matched[1]));
      for (const matched of source.matchAll(/url\(#([^)\s'"]+)\)/g)) {
        if (!defined.has(matched[1])) broken.push(`${file} -> #${matched[1]}`);
      }
    }
    expect(broken).toEqual([]);
  });
});

describe('iconcool sprite 的 id 作用域', () => {
  beforeAll(() => {
    expect(scopeIconSpriteIds()).toBe(true);
  });

  it('sprite 内的裸 id 全部收敛到私有命名空间', () => {
    const sprite = document.querySelector(SPRITE_SELECTOR);
    expect(sprite).toBeTruthy();

    const strayIds = [...(sprite?.querySelectorAll('[id]') ?? [])]
      .map(el => el.getAttribute('id') ?? '')
      .filter(id => !id.startsWith(SYMBOL_ID_PREFIX) && !id.startsWith(SPRITE_ID_PREFIX));
    expect(strayIds).toEqual([]);
  });

  it('<use xlink:href="#bkms-icon-x"> 依赖的 symbol id 保持不变', () => {
    const symbolIds = [...document.querySelectorAll(`${SPRITE_SELECTOR} symbol[id]`)].map(
      el => el.getAttribute('id') ?? '',
    );
    expect(symbolIds).toHaveLength(spriteIcons.size);
    expect(symbolIds.filter(id => id.startsWith(SYMBOL_ID_PREFIX))).toHaveLength(spriteIcons.size);
    expect(symbolIds.filter(id => document.getElementById(id) === null)).toEqual([]);
  });

  it('sprite 内部的 url(#...) 引用仍指向 sprite 自身', () => {
    const sprite = document.querySelector(SPRITE_SELECTOR);
    expect(sprite).toBeTruthy();

    const dangling: string[] = [];
    for (const el of sprite?.querySelectorAll('*') ?? []) {
      for (const attr of el.getAttributeNames()) {
        if (attr === 'id') continue;
        const value = el.getAttribute(attr) ?? '';
        for (const matched of value.matchAll(/url\(#([^)\s'"]+)\)/g)) {
          const target = document.getElementById(matched[1]);
          if (!target || !sprite?.contains(target)) dangling.push(`${attr}=url(#${matched[1]})`);
        }
      }
    }
    expect(dangling).toEqual([]);
  });

  it('页面内 svg 的短 id 不再被 sprite 抢占（coffee.svg 渐变失效的回归用例）', () => {
    // svgo 压缩后 coffee.svg 的 coffee-grad 会变成 id="a" + url(#a)
    document.body.insertAdjacentHTML(
      'beforeend',
      '<svg id="coffee"><defs><linearGradient id="a"></linearGradient></defs><path fill="url(#a)"/></svg>',
    );

    const target = document.getElementById('a');
    expect(target?.tagName.toLowerCase()).toBe('lineargradient');
    expect(target?.closest('svg')?.id).toBe('coffee');
  });
});
