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
 * `iconcool.js` 由 iconcool 平台导出，会把整段 `<symbol>` sprite 直接插到 `<body>` 最前面。
 * 新导出里混进了 Figma 直出的裸 id（`id="a"` / `id="b"` / `id="clip0_3483_86"` ...），
 * 这些 id 会进入全局命名空间，与页面内其它 svg 的 `url(#a)` 抢同一个 id：
 * 例如 `src/assets/coffee.svg` 的 `fill="url(#coffee-grad)"` 被 svgo 压缩成 `url(#a)` 后，
 * 会解析到 sprite 里的 `<mask id="a">`，路径取不到 paint server，整块图标渲染为空白。
 *
 * 这里在 sprite 插入后把它内部的非 `bkms-icon-*` id 迁到 `bkms-sprite-*` 私有命名空间，
 * 并同步改写 sprite 内部的 `url(#...)` / `href="#..."` 引用。
 * 只处理 sprite 内部，`<symbol id="bkms-icon-xxx">` 保持原样，组件里的
 * `<use xlink:href="#bkms-icon-xxx">` 不受影响。
 */

/** iconcool 导出的 sprite 根节点特征：`<svg data-name="bkms" style="position:absolute;width:0;height:0">` */
export const SPRITE_SELECTOR = 'svg[data-name="bkms"]';
/** sprite 内部私有 id 前缀 */
export const SPRITE_ID_PREFIX = 'bkms-sprite-';
/** `<use>` 依赖的 symbol id 前缀，必须保持原样 */
export const SYMBOL_ID_PREFIX = 'bkms-icon-';

/** 把 `url(#x)` / `href="#x"` 里的 id 换成新名字，其它内容原样保留 */
const rewriteReferences = (value: string, rename: Map<string, string>): string => {
  const withUrl = value.replace(/url\(\s*#([^)\s'"]+)\s*\)/g, (raw: string, id: string) => {
    const next = rename.get(id);
    return next ? `url(#${next})` : raw;
  });
  if (!withUrl.startsWith('#')) return withUrl;
  const next = rename.get(withUrl.slice(1));
  return next ? `#${next}` : withUrl;
};

/**
 * 收敛 iconcool sprite 内部的 id 命名空间。
 * @param doc 目标文档，默认取全局 document
 * @returns sprite 是否存在（false 表示 iconcool.js 还没插入，可延后重试）
 */
export const scopeIconSpriteIds = (doc?: Document): boolean => {
  const root = doc ?? (typeof document === 'undefined' ? null : document);
  if (!root) return false;

  const sprite = root.querySelector(SPRITE_SELECTOR);
  if (!sprite) return false;

  const rename = new Map<string, string>();
  const taken = new Set<string>();
  for (const el of sprite.querySelectorAll('[id]')) {
    const id = el.getAttribute('id') ?? '';
    if (!id || id.startsWith(SYMBOL_ID_PREFIX)) continue;
    let next = `${SPRITE_ID_PREFIX}${id}`;
    // sprite 内部理论上不会重复，真出现重复时逐个加后缀，避免改名后互相覆盖
    while (taken.has(next)) next = `${next}_`;
    taken.add(next);
    rename.set(id, next);
  }
  if (rename.size === 0) return true;

  for (const el of sprite.querySelectorAll('[id]')) {
    const next = rename.get(el.getAttribute('id') ?? '');
    if (next) el.setAttribute('id', next);
  }

  for (const el of sprite.querySelectorAll('*')) {
    for (const name of el.getAttributeNames()) {
      if (name === 'id') continue;
      const value = el.getAttribute(name) ?? '';
      if (!value.includes('#')) continue;
      const next = rewriteReferences(value, rename);
      if (next !== value) el.setAttribute(name, next);
    }
  }

  return true;
};
