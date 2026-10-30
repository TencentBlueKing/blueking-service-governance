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
 * 把组件库样式里的旧 Brand / Neutral，以及 2px 圆角，换成 AIdev 规范。
 * 只处理编译结果，不改 node_modules。
 */

import { readFileSync } from 'node:fs';

const colorPairs = [
  ['#2C77F4', '#274ECF'],
  ['#3A84FF', '#3B6EF5'],
  ['#699DF4', '#6691FF'],
  ['#5594FA', '#6691FF'],
  ['#A3C5FD', '#99B9FF'],
  ['#CDDFFE', '#C7D8FF'],
  ['#E1ECFF', '#E5EDFF'],
  ['#F0F5FF', '#F0F4FF'],
  ['#313238', '#111214'],
  ['#4D4F56', '#4A5973'],
  ['#63656E', '#4A5973'],
  ['#979BA5', '#8F9FBD'],
  ['#C4C6CC', '#C1CDE5'],
  ['#DCDEE5', '#E4E9F2'],
  ['#F0F1F5', '#E9EDF5'],
  ['#EAEBF0', '#E9EDF5'],
  ['#F5F7FA', '#F0F4FA'],
  ['#FAFBFD', '#F7F9FD'],
  // 组件库里的焦点蓝、面包屑悬停蓝，不在源码映射表，但和 Brand 是同一套交互色。
  ['#3C96FF', '#6691FF'],
  ['#0082FF', '#3B6EF5'],
];

const colorLookup = new Map(colorPairs.map(([from, to]) => [from.toLowerCase(), to]));
const colorPattern = new RegExp(colorPairs.map(([from]) => from).join('|'), 'gi');

const radiusPattern =
  /((?:--|@)[\w-]*radius[\w-]*\s*:\s*|border(?:-(?:top|right|bottom|left)(?:-(?:left|right))?)?-radius\s*:\s*)([^;}\n]+)/gi;

export function rewriteAidevCss(code) {
  if (!code) return code;
  const colored = code.replace(colorPattern, match => colorLookup.get(match.toLowerCase()) ?? match);
  return colored.replace(radiusPattern, (full, property, value) => {
    return property + value.replace(/(?<![\d.])2px\b/g, '6px');
  });
}

function expandHex(hex) {
  const raw = hex.slice(1);
  if (raw.length === 3) {
    return `#${raw
      .split('')
      .map(char => char + char)
      .join('')}`;
  }
  return `#${raw}`;
}

function hslLightness(hex) {
  const value = expandHex(hex).slice(1);
  const red = parseInt(value.slice(0, 2), 16) / 255;
  const green = parseInt(value.slice(2, 4), 16) / 255;
  const blue = parseInt(value.slice(4, 6), 16) / 255;
  return (Math.max(red, green, blue) + Math.min(red, green, blue)) / 2;
}

function isGray(hex) {
  const value = expandHex(hex).slice(1);
  const red = parseInt(value.slice(0, 2), 16) / 255;
  const green = parseInt(value.slice(2, 4), 16) / 255;
  const blue = parseInt(value.slice(4, 6), 16) / 255;
  const max = Math.max(red, green, blue);
  const min = Math.min(red, green, blue);
  // 近白时饱和度会被放大，用通道差判断灰。
  if (max - min < 0.09) return true;
  const lightness = (max + min) / 2;
  const saturation = lightness > 0.5 ? (max - min) / (2 - max - min) : (max - min) / (max + min);
  return saturation < 0.2;
}

// bk-aidev 空态插图换色：灰阶按明度进品牌蓝，纯白纸面和已有彩色不动。
export function mapExceptionGray(hex) {
  if (!isGray(hex)) return hex;
  const lightness = hslLightness(hex);
  if (lightness >= 0.985) return '#FFFFFF';
  if (lightness >= 0.94) return '#F7F9FD';
  if (lightness >= 0.91) return '#E5EDFF';
  if (lightness >= 0.8) return '#C7D8FF';
  if (lightness <= 0.25) return '#111214';
  return '#99B9FF';
}

export function recolorSvg(svg) {
  return svg.replace(/#(?:[0-9a-fA-F]{6}|[0-9a-fA-F]{3})\b/g, hex => mapExceptionGray(hex));
}

export function recolorExceptionIllustrations(code) {
  if (!code || !code.includes('data:image/svg+xml;base64,')) return code;
  return code.replace(/data:image\/svg\+xml;base64,([A-Za-z0-9+/=]+)/g, (full, encoded) => {
    const svg = Buffer.from(encoded, 'base64').toString('utf8');
    const next = recolorSvg(svg);
    if (next === svg) return full;
    return `data:image/svg+xml;base64,${Buffer.from(next, 'utf8').toString('base64')}`;
  });
}

export function isComponentStyle(id) {
  const file = id.split('?')[0];
  return (
    /[\\/]node_modules[\\/](?:bkui-vue|@blueking)[\\/]/.test(file) && /\.(css|less|scss)$/.test(file)
  );
}

export function aidevExceptionEsbuildPlugin() {
  return {
    name: 'aidev-exception-illustration',
    setup(build) {
      build.onLoad({ filter: /[\\/]bkui-vue[\\/]lib[\\/]exception[\\/]index\.js$/ }, args => {
        const code = readFileSync(args.path, 'utf8');
        return { contents: recolorExceptionIllustrations(code), loader: 'js' };
      });
    },
  };
}

export function aidevComponentStylePlugin() {
  return {
    name: 'aidev-component-style',
    enforce: 'post',
    transform(code, id) {
      if (!code) return null;
      if (/[\\/]bkui-vue[\\/]lib[\\/]exception[\\/]index\.js/.test(id.split('?')[0])) {
        const next = recolorExceptionIllustrations(code);
        if (next === code) return null;
        return { code: next, map: null };
      }
      if (!isComponentStyle(id)) return null;
      const next = rewriteAidevCss(code);
      if (next === code) return null;
      return { code: next, map: null };
    },
  };
}

export function aidevLessPlugin() {
  return {
    install(lessInstance, pluginManager) {
      pluginManager.addPostProcessor({
        process(css) {
          return rewriteAidevCss(css);
        },
      });
    },
  };
}
