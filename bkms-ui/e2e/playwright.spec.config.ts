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
import dotenv from 'dotenv';
import fs from 'fs';
import path from 'path';

const PROJECT_ROOT = path.resolve(__dirname, '../../..');
const ENABLE_PARALLEL = process.env.E2E_PARALLEL === 'true';
const WORKERS = ENABLE_PARALLEL ? Number(process.env.E2E_WORKERS || '2') : 1;

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

if (fs.existsSync(path.resolve(__dirname, '.env.test'))) {
  dotenv.config({ path: path.resolve(__dirname, '.env.test') });
  process.env.BK_CI === 'true' ? validateEnv(['BKMS_TEST_ACCESS_TOKEN']) : validateEnv([]);
}
if (fs.existsSync(path.resolve(__dirname, '../.env.development'))) {
  dotenv.config({ path: path.resolve(__dirname, '../.env.development') });
  process.env.BK_CI === 'true' ? validateEnv([]) : validateEnv(['BK_APP_HOST', 'BK_APP_PORT']);
}

export default defineConfig({
  testDir: './tests',
  timeout: 180_000,
  expect: {
    timeout: 10_000,
  },
  fullyParallel: ENABLE_PARALLEL,
  forbidOnly: process.env.BK_CI === 'true',
  retries: process.env.BK_CI === 'true' ? 1 : 0,
  workers: WORKERS,
  reporter: [
    ['list'],
    ['./scripts/screenshot-reporter.js', { outputDir: resolveReportDir() }],
    ['html', { outputFolder: path.join(resolveReportDir(), 'playwright-html'), open: 'never' }],
  ],
  use: {
    headless: process.env.BK_CI === 'true' ? true : false,
    viewport: { width: 1366, height: 768 },
    ignoreHTTPSErrors: true,
    actionTimeout: 15_000,
    navigationTimeout: 30_000,
    screenshot: 'off',
    baseURL:
      process.env.BK_CI === 'true'
        ? process.env.BKMS_TEST_SITE
        : `http://${process.env.BK_APP_HOST}:${process.env.BK_APP_PORT}`,
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
      },
    },
  ],
  webServer:
    process.env.BK_CI === 'true'
      ? undefined
      : {
          command: 'cd ../ && pnpm run dev',
          url: `http://${process.env.BK_APP_HOST}:${process.env.BK_APP_PORT}`,
          reuseExistingServer: true,
        },
});
