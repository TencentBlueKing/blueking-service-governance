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
import fs from 'fs';
import path from 'path';
// [fixtures guide](https://vitalets.github.io/playwright-bdd/#/getting-started/add-fixtures)
import { test as base, createBdd } from 'playwright-bdd';

import AppDetailPage from '../pages/app-detail.page';
import AppPage from '../pages/app.page';
import BasePage from '../pages/base.page';
import HomePage from '../pages/home.page';

import type { APIRequestContext, BrowserContext } from '@playwright/test';

export type BkmsFixtures = {
  pages: {
    appDetailPage: AppDetailPage;
    appPage: AppPage;
    basePage: BasePage;
    homePage: HomePage;
  };
  testConfig: {
    app: string;
    appType: AppType;
    env: string;
    reportDir: string;
    space: string;
    token: string;
  };
  // 步骤自己传递数据
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  userData: Record<string, any>;
};

type AppInfo = {
  name?: string;
  type?: string;
};

type AppType = 'helm' | 'taf' | 'trpc';

const APP_TYPES = new Set<AppType>(['helm', 'taf', 'trpc']);
const E2E_APP_PREFIX = 'e2e-';

async function discoverE2eAppName({
  appType,
  request,
  space,
  token,
}: {
  appType: AppType;
  request: APIRequestContext;
  space: string;
  token: string;
}) {
  if (!space) return '';

  const response = await request.get(joinUrl(getApiV1Prefix(), `/workspaces/${encodeURIComponent(space)}/apps`), {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });

  if (!response.ok()) {
    throw new Error(`自动发现 ${appType} 测试应用失败：应用列表接口返回 ${response.status()} ${response.url()}`);
  }

  const apps = extractAppList(await response.json());
  const candidates = apps
    .filter(app => app.name?.startsWith(E2E_APP_PREFIX) && app.type === appType)
    .sort((a, b) => {
      const preferredName = `${E2E_APP_PREFIX}${appType}`;
      if (a.name === preferredName) return -1;
      if (b.name === preferredName) return 1;
      return (a.name || '').localeCompare(b.name || '');
    });

  return candidates[0]?.name || '';
}

function extractAppList(payload: unknown): AppInfo[] {
  if (Array.isArray(payload)) return payload as AppInfo[];
  if (payload && typeof payload === 'object' && Array.isArray((payload as { data?: unknown }).data)) {
    return (payload as { data: AppInfo[] }).data;
  }
  return [];
}

function getApiV1Prefix() {
  if (process.env.BK_API_V1_PREFIX) return process.env.BK_API_V1_PREFIX;
  const apiPrefix = process.env.BK_API_PREFIX || '';
  return `${apiPrefix.replace(/\/$/, '')}/bkms/v1/bkms-server`;
}

function getConfiguredAppName(tag: string | undefined, appType: AppType) {
  const isSemanticTag = tag && APP_TYPES.has(tag as AppType);
  if (tag && tag !== 'default' && !isSemanticTag) return tag;

  if (tag === 'trpc' || appType === 'trpc') {
    return process.env.BKMS_TEST_TRPC_APP || process.env.BKMS_TEST_DEFAULT_APP || '';
  }

  if (tag === 'helm' || appType === 'helm') {
    return process.env.BKMS_TEST_HELM_APP || '';
  }

  if (tag === 'taf' || appType === 'taf') {
    return process.env.BKMS_TEST_TAF_APP || '';
  }

  return process.env.BKMS_TEST_DEFAULT_APP || '';
}

/** Gherkin 标签形如 @space:default，来自 bddData，不是 Playwright testInfo.annotations */
function getTagValue(kind: string, $tags: string[]) {
  const prefix = `@${kind}:`;
  const raw = $tags.find(t => t.startsWith(prefix));
  return raw ? raw.slice(prefix.length) : undefined;
}

function joinUrl(prefix: string, pathName: string) {
  return `${prefix.replace(/\/$/, '')}/${pathName.replace(/^\//, '')}`;
}

async function resolveAppName({
  appType,
  appTypeTag,
  request,
  space,
  tag,
  token,
}: {
  appType: AppType;
  appTypeTag: string | undefined;
  request: APIRequestContext;
  space: string;
  tag: string | undefined;
  token: string;
}) {
  const configuredAppName = getConfiguredAppName(tag, appType);
  if (configuredAppName) return configuredAppName;
  if (!tag && !appTypeTag) return '';

  return discoverE2eAppName({ appType, request, space, token });
}

function resolveAppType(tag: string | undefined): AppType {
  if (tag === 'helm' || tag === 'taf' || tag === 'trpc') {
    return tag;
  }

  return 'trpc';
}

/** 仅对 XHR / fetch / document 注入 Bearer，避免静态资源等请求携带 Authorization */
const BEARER_RESOURCE_TYPES = new Set(['xhr', 'fetch', 'document']);
/** 静态资源类型：统一走 http，避免本地/内网环境无 https 证书时加载失败 */
const STATIC_RESOURCE_TYPES = new Set(['script', 'stylesheet', 'image', 'font', 'media']);
/** 仅在蓝盾流水线（BK_CI=true）中，把静态资源 https 重写为 http */
const REWRITE_STATIC_HTTPS_TO_HTTP = process.env.BK_CI === 'true';

async function routeRequests(context: BrowserContext, token: string | undefined) {
  await context.route('**/*', async route => {
    const request = route.request();
    const type = request.resourceType();

    if (REWRITE_STATIC_HTTPS_TO_HTTP && STATIC_RESOURCE_TYPES.has(type) && request.url().startsWith('https://')) {
      // route.continue 不允许换协议（New URL must have same protocol as overridden URL），
      // 改为后端发起 http 请求，再把响应原样回填给浏览器
      const httpUrl = `http://${request.url().slice('https://'.length)}`;
      try {
        const response = await route.fetch({ url: httpUrl });
        await route.fulfill({ response });
      } catch (_) {
        // console.warn(`[fixtures] rewrite https→http failed: ${httpUrl}`, error);
        await route.continue();
      }
      return;
    }

    if (token && BEARER_RESOURCE_TYPES.has(type)) {
      await route.continue({ headers: { Authorization: `Bearer ${token}` } });
      return;
    }

    await route.continue();
  });
}

// 自定义 fixtures
export const test = base.extend<BkmsFixtures>({
  context: async ({ browser, contextOptions }, use) => {
    const context = await browser.newContext(contextOptions);
    const token = process.env.BKMS_TEST_ACCESS_TOKEN;
    await routeRequests(context, token);
    await use(context);
    await context.close();
  },
  // playwright-bdd 会解析 fixture 首参：必须是对象解构，且不允许 ...rest（见 fixtureParameterNames）
  pages: [
    async ({ page, request, testConfig }, use) => {
      await use({
        homePage: new HomePage({ page, request, testConfig }),
        basePage: new BasePage({ page, request, testConfig }),
        appPage: new AppPage({ page, request, testConfig }),
        appDetailPage: new AppDetailPage({ page, request, testConfig }),
      });
    },
    { scope: 'test' },
  ],
  testConfig: [
    async ({ $tags, request }, use) => {
      /**
       * 解析 @<kind>:xxx 标签到运行时值：
       * - 未打标签 或 打了 `@<kind>:default` → 回退到 BKMS_TEST_DEFAULT_<KIND> 环境变量
       * - 打了显式值（如 @space:demo）→ 直接使用该值
       * 环境变量本身也等于 'default' 的极端情况下保留字面量 'default'。
       */
      const resolve = (kind: 'app' | 'env' | 'space', envVar: string) => {
        const tag = getTagValue(kind, $tags);
        const fallback = process.env[envVar] ?? '';
        return !tag || tag === 'default' ? fallback : tag;
      };
      const appTag = getTagValue('app', $tags);
      const appTypeTag = getTagValue('appType', $tags);
      const appType = resolveAppType(appTypeTag || (APP_TYPES.has(appTag as AppType) ? appTag : undefined));
      const space = resolve('space', 'BKMS_TEST_DEFAULT_SPACE');
      const token = process.env.BKMS_TEST_ACCESS_TOKEN || '';

      await use({
        reportDir: process.env.BKMS_TEST_REPORT_DIR || 'test-reports/default',
        token,
        space,
        env: resolve('env', 'BKMS_TEST_DEFAULT_ENV'),
        app: await resolveAppName({
          appType,
          appTypeTag,
          request,
          space,
          tag: appTag,
          token,
        }),
        appType,
      });
    },
    { scope: 'test' },
  ],
  userData: [
    // eslint-disable-next-line no-empty-pattern
    async ({}, use) => {
      await use({});
    },
    { scope: 'test' },
  ],
});

export const { Given, When, Then, Before } = createBdd(test);

Before(async ({ page }) => {
  if (process.env.BKMS_TEST_USE_MOCK) {
    const _path = path.resolve(__dirname, '../mocks.json');
    if (fs.existsSync(_path)) {
      try {
        const mocks = JSON.parse(fs.readFileSync(_path, 'utf-8'));
        for (const [url, data] of Object.entries(mocks)) {
          page.route(url, async route => {
            return route.fulfill({
              status: 200,
              contentType: 'application/json',
              body: JSON.stringify(data),
            });
          });
        }
      } catch (error) {
        console.error('Failed to parse mocks.json', error);
      }
    }
  }
});
