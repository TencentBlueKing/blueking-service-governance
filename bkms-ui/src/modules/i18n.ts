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

import { createI18n } from 'vue-i18n';

import type { Locale } from 'vue-i18n';
import type { UserModule } from '~/types.ts';

const i18n = createI18n({
  legacy: false,
  locale: '',
  messages: {},
});

// locales目录下语言map
const localesMap = Object.fromEntries(
  Object.entries(import.meta.glob('../../locales/*.yml')).map(([path, loadLocale]) => [
    path.match(/([\w-]*)\.yml$/)?.[1],
    loadLocale,
  ]),
) as Record<Locale, () => Promise<{ default: Record<string, string> }>>;

const availableLocales = Object.keys(localesMap);
// 已加载语言
const loadedLanguages: string[] = [];

// 异步加载语言
async function loadLanguageAsync(lang: string): Promise<Locale> {
  if (i18n.global.locale.value === lang) return setI18nLanguage(lang);

  if (loadedLanguages.includes(lang)) return setI18nLanguage(lang);

  const messages = await localesMap[lang]();
  i18n.global.setLocaleMessage(lang, messages.default);
  loadedLanguages.push(lang);
  return setI18nLanguage(lang);
}

// 设置语言
function setI18nLanguage(lang: Locale) {
  i18n.global.locale.value = lang;
  if (typeof document !== 'undefined') document.querySelector('html')?.setAttribute('lang', lang);
  return lang;
}

// Setup i18n
const install: UserModule = async ({ app }) => {
  app.use(i18n);
  // 加载默认语言
  await loadLanguageAsync('zh-CN');
};
window.i18n = i18n.global;
export { availableLocales, i18n, install, loadLanguageAsync };
