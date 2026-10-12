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
import { type Page, expect } from '@playwright/test';

import { type BkmsFixtures, test } from '../fixtures/fixtures';

const INSTANCE_TABLE = '.instance-table';
const INSTANCE_ROWS = `${INSTANCE_TABLE} .vxe-table--main-wrapper .vxe-table--body tr`;

async function openInstanceList(page: Page, testConfig: BkmsFixtures['testConfig']) {
  const app = process.env.BKMS_TEST_TAF_APP || testConfig.app;
  if (!app) {
    throw new Error('实例列表 E2E 前置条件不满足：请配置 BKMS_TEST_TAF_APP。');
  }

  const query = new URLSearchParams({ activeTab: 'instance', envName: testConfig.env });
  await page.goto(`/#/${testConfig.space}/app/${app}/taf/deployment?${query.toString()}`, {
    waitUntil: 'domcontentloaded',
  });
  await expect(page.getByText('实例列表', { exact: true }).first()).toBeVisible({ timeout: 15000 });

  const envSelector = page.getByText('请选择', { exact: true }).first();
  if (await envSelector.isVisible().catch(() => false)) {
    await envSelector.click();
    const envOption = page.locator('.env-list-scroll > div').filter({ hasText: testConfig.env }).first();
    await expect(envOption).toBeVisible({ timeout: 15000 });
    await envOption.click();
  }

  // 未部署空态与实例行均为异步渲染（initLoading 期间先出 Skeleton，完成后才出 Exception 或表格行）。
  // 用非阻塞 isVisible() 检测会漏判未渲染的空态，导致后续等表格行空等 30s。
  // 这里 race 等任一出现：空态先出 → 抛前置错误；表格行先出 → 继续。
  const uninstalled = page.getByText('该环境尚未部署应用', { exact: true }).first();
  const firstRow = page.locator(INSTANCE_ROWS).first();
  await Promise.race([
    uninstalled.waitFor({ state: 'visible', timeout: 30000 }),
    firstRow.waitFor({ state: 'visible', timeout: 30000 }),
  ]);

  if (await uninstalled.isVisible().catch(() => false)) {
    throw new Error(`实例列表 E2E 前置条件不满足：应用 ${app} 在环境 ${testConfig.env} 中尚未部署测试应用。`);
  }
}

test.describe('实例列表回归 @QG-instance-list', () => {
  test.describe.configure({ mode: 'serial' });

  test('状态列展示实例与北极星，就绪列展示就绪结果 @QG-instance-status', async ({ page, testConfig }) => {
    await openInstanceList(page, testConfig);

    const rows = page.locator(INSTANCE_ROWS);
    const rowCount = await rows.count();
    expect(rowCount).toBeGreaterThan(0);
    await expect(page.getByRole('columnheader', { name: /就绪/ }).first()).toBeVisible();

    for (let index = 0; index < rowCount; index += 1) {
      const row = rows.nth(index);
      const statusCell = row.locator('.instance-status-col');
      const labels = statusCell.locator('.multi-line-cell > .cell-label');
      await expect(labels).toHaveCount(2);
      await expect(labels.nth(0)).toHaveText('实例');
      await expect(labels.nth(1)).toHaveText('北极星');
    }
  });

  test('当前页实例行按内容撑开且状态表头支持分组筛选 @QG-instance-layout-filter', async ({ page, testConfig }) => {
    await openInstanceList(page, testConfig);

    const table = page.locator(INSTANCE_TABLE).first();
    const rows = page.locator(INSTANCE_ROWS);
    const bodyWrapper = table.locator('.vxe-table--body-wrapper').first();
    const maxHeight = await bodyWrapper.evaluate(element => getComputedStyle(element).maxHeight);
    expect(maxHeight).not.toBe('320px');

    const rowCount = await rows.count();
    if (rowCount > 5) {
      const scrollMetrics = await bodyWrapper.evaluate(element => ({
        clientHeight: element.clientHeight,
        overflowY: getComputedStyle(element).overflowY,
        scrollHeight: element.scrollHeight,
      }));
      expect(scrollMetrics.overflowY).not.toBe('scroll');
      expect(scrollMetrics.scrollHeight).toBeLessThanOrEqual(scrollMetrics.clientHeight + 1);
    }

    const statusHeader = table.getByRole('columnheader', { name: /状态/ }).first();
    await expect(statusHeader).toBeVisible();
    await statusHeader.getByText('状态', { exact: true }).click();
    const filterGroups = page.locator('.group-section:visible');
    await expect(filterGroups).toHaveCount(2);
    await expect(filterGroups.nth(0)).toContainText('实例状态');
    await expect(filterGroups.nth(1)).toContainText('北极星状态');
    const statusOption = filterGroups.nth(0).locator('.bk-checkbox').first();
    await expect(statusOption).toBeVisible();
    await statusOption.click();
    await expect(statusOption).toHaveClass(/is-checked/);
    await page.getByRole('button', { name: '重置' }).click();
    await expect(statusOption).not.toHaveClass(/is-checked/);

    const readinessHeader = table.getByRole('columnheader', { name: /就绪/ }).first();
    const readinessLabel = readinessHeader.getByText('就绪', { exact: true });
    await readinessLabel.hover();
    await expect(page.getByText('实例是否可以接收流量', { exact: true }).last()).toBeVisible();
    await readinessLabel.click();
    const readinessFilterGroups = page.locator('.group-section:visible');
    await expect(readinessFilterGroups).toHaveCount(1);
    await expect(readinessFilterGroups.first()).toContainText(/是|否/);
    const readinessOption = readinessFilterGroups.first().locator('.bk-checkbox').first();
    await expect(readinessOption).toBeVisible();
    await readinessOption.click();
    await expect(readinessOption).toHaveClass(/is-checked/);
    await page.getByRole('button', { name: '重置' }).click();
    await expect(readinessOption).not.toHaveClass(/is-checked/);
  });
});
