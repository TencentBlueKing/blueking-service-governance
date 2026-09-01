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
import { defineConfig, devices } from '@playwright/test';
/**
 * Read environment variables from file.
 * https://github.com/motdotla/dotenv
 */
import dotenv from 'dotenv';
import fs from 'fs';
import path from 'path';
import { defineBddConfig } from 'playwright-bdd';

const PROJECT_ROOT = path.resolve(__dirname, '../../..');
const ENABLE_PARALLEL = process.env.E2E_PARALLEL === 'true';
const WORKERS = ENABLE_PARALLEL ? Number(process.env.E2E_WORKERS || '2') : 1;

/** 解析 BKMS_TEST_REPORT_DIR → 绝对路径；与 base.page.ts screenshot() 对齐。 */
function resolveReportDir() {
  const dir = process.env.BKMS_TEST_REPORT_DIR;
  if (!dir) return path.resolve(PROJECT_ROOT, 'test-reports', 'default');
  return path.isAbsolute(dir) ? dir : path.resolve(PROJECT_ROOT, dir);
}

function validateEnv(names: string[]) {
  for (const name of names) {
    if (!process.env[name]) {
      throw new Error(`环境变量 ${name} 未设置，请在 .env.test 或 .env.development 中配置`);
    }
  }
}

// 测试需要的环境变量
if (fs.existsSync(path.resolve(__dirname, '.env.test'))) {
  dotenv.config({ path: path.resolve(__dirname, '.env.test') });
  process.env.BK_CI === 'true' ? validateEnv(['BKMS_TEST_ACCESS_TOKEN']) : validateEnv([]);
}
if (fs.existsSync(path.resolve(__dirname, '../.env.development'))) {
  dotenv.config({ path: path.resolve(__dirname, '../.env.development') });
  process.env.BK_CI === 'true' ? validateEnv([]) : validateEnv(['BK_APP_HOST', 'BK_APP_PORT']);
}

console.log('====================');
console.log('测试环境', process.env.BKMS_TEST_SITE);
console.log('默认空间', process.env.BKMS_TEST_DEFAULT_SPACE);
console.log('默认环境', process.env.BKMS_TEST_DEFAULT_ENV);
console.log('默认应用', process.env.BKMS_TEST_DEFAULT_APP);
console.log('tRPC 应用', process.env.BKMS_TEST_TRPC_APP);
console.log('Helm 应用', process.env.BKMS_TEST_HELM_APP);
console.log('====================');

// playwright-bdd 配置：从 features + steps 生成测试文件
const testDir = defineBddConfig({
  features: 'features/*.feature',
  steps: ['steps/**/*.steps.ts', 'fixtures/**/*.ts'],
  outputDir: '.features-gen',
  /* 使用英文，因为 bkms-bdd-gen 生成的测试用例是英文的 */
  language: 'en',
});

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
  testDir,
  timeout: 180_000,
  expect: {
    timeout: 10_000,
  },
  /* Stateful tests run serially by default. Use E2E_PARALLEL=true for readonly or isolated flows. */
  fullyParallel: ENABLE_PARALLEL,
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: process.env.BK_CI === 'true',
  /* Retry on CI only */
  retries: process.env.BK_CI === 'true' ? 1 : 0,
  workers: WORKERS,
  /* Reporter to use. See https://playwright.dev/docs/test-reporters */
  /**
   * 统一产物目录：所有 reporter 和 page.screenshot 都写到同一个 reportDir。
   * - 相对路径时相对仓库根目录；
   * - 绝对路径直用；
   * - 未设置环境变量时回退到 <repo>/test-reports/default。
   */
  reporter: [
    ['list'],
    ['./scripts/screenshot-reporter.js', { outputDir: resolveReportDir() }],
    // 原生 HTML 报告，失败用例 trace 直接可 show-report 查看；放独立子目录避免和自研 report.html 重名
    ['html', { outputFolder: path.join(resolveReportDir(), 'playwright-html'), open: 'never' }],
  ],
  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    headless: process.env.BK_CI === 'true' ? true : false,
    viewport: { width: 1366, height: 768 },
    ignoreHTTPSErrors: true,
    actionTimeout: 15_000,
    navigationTimeout: 30_000,
    screenshot: 'off',
    /* Base URL to use in actions like `await page.goto('')`. */
    // baseURL: 'http://localhost:3000',
    baseURL:
      process.env.BK_CI === 'true'
        ? process.env.BKMS_TEST_SITE
        : `http://${process.env.BK_APP_HOST}:${process.env.BK_APP_PORT}`,

    /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
    trace: 'retain-on-failure',
  },

  /* Configure projects for major browsers */
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
      },
    },
  ],

  /**
   * Run your local dev server before starting the tests.
   * CI 模式下目标是远程站点（BKMS_TEST_SITE），不需要启动本地 dev server；
   * 本地模式才拉起 pnpm run dev。
   */
  webServer:
    process.env.BK_CI === 'true'
      ? undefined
      : {
          command: 'cd ../ && pnpm run dev',
          url: `http://${process.env.BK_APP_HOST}:${process.env.BK_APP_PORT}`,
          reuseExistingServer: true,
        },
});
