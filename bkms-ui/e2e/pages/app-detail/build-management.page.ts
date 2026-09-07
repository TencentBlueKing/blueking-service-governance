/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */
import { type Locator, expect } from '@playwright/test';

import AppDetailBase from './app-detail-base.page';

/**
 * 构建管理域 Page Object。
 *
 * 覆盖：构建记录列表（搜索/分页/异常空态）、构建日志侧栏，
 * 以及构建记录 / 日志 SSE 的 route mock。
 */
export default class BuildManagementPage extends AppDetailBase {
  private buildRecordSearchKeyword = '';

  private buildRecord(overrides: Record<string, unknown> = {}) {
    return {
      artifact: 'registry.example.com/bkms/e2e-service:v1.0.0',
      buildID: 'build-e2e-001',
      commitID: 'abcdef1234567890',
      endedAt: '2026-01-01T10:03:30Z',
      extras: {
        BK_CI_GIT_REPO_HEAD_COMMIT_ID: 'abcdef1234567890',
        BK_CI_GIT_REPO_URL: 'https://example.com/bkms/e2e-service.git',
      },
      num: '101',
      operator: 'e2e-user',
      params: {
        BKMS_IMAGE_TAG: 'v1.0.0',
      },
      pipelineID: 'pipeline-e2e',
      revision: 'main',
      startedAt: '2026-01-01T10:00:00Z',
      status: 'success',
      ...overrides,
    };
  }

  private async routeBuildLogStream(message = 'e2e build log line') {
    await this.page.route('**/apps/*/builds/*/logs/stream', async route => {
      await route.fulfill({
        body: [
          'event: message',
          `data: {"Logs":[{"Timestamp":"2026-01-01T10:00:10Z","Message":"${message}"}]}`,
          '',
          'event: done',
          'data: {}',
          '',
        ].join('\n'),
        contentType: 'text/event-stream',
        status: 200,
      });
    });
  }

  private async routeBuildRecords(records: Array<Record<string, unknown>>, options: { failFirst?: boolean } = {}) {
    let failed = false;
    await this.page.route('**/apps/*/builds**', async route => {
      const request = route.request();
      const url = new URL(request.url());
      if (request.method() !== 'GET' || !url.pathname.endsWith('/builds')) {
        await route.fallback();
        return;
      }

      if (options.failFirst && !failed) {
        failed = true;
        await this.fulfillJson(
          route,
          {
            error: {
              message: 'build records list failed',
            },
            status: 500,
          },
          500,
        );
        return;
      }

      const keyword = (url.searchParams.get('keyword') || '').trim().toLowerCase();
      const page = Number(url.searchParams.get('page') || 1);
      const pageSize = Number(url.searchParams.get('pageSize') || 10);
      const filtered = keyword
        ? records.filter(record => JSON.stringify(record).toLowerCase().includes(keyword))
        : records;
      const results = filtered.slice((page - 1) * pageSize, page * pageSize);
      await this.fulfillJson(route, {
        data: {
          count: String(filtered.length),
          results,
        },
        status: 0,
      });
    });
  }

  /** 构建记录搜索输入框 */
  buildRecordSearchInput(): Locator {
    return this.page.locator('input[type="search"]').first();
  }

  /** 获取构建管理主表行，排除固定列复制出的 VXE 行 */
  buildTableMainRows(): Locator {
    return this.page.locator('.build-table .vxe-table--main-wrapper .vxe-table--body tr');
  }

  /** 切换构建记录每页条数为 20 */
  async changeBuildRecordsPageSizeToTwenty() {
    const pagination = this.page.locator('.bk-pagination').last();
    await pagination.locator('.bk-select').first().click();
    const option = this.page.getByText('20', { exact: true }).last();
    await option.click();
    await expect(this.buildTableMainRows()).toHaveCount(18, { timeout: 10000 });
  }

  /** 切换构建记录分页到第二页 */
  async changeBuildRecordsToSecondPage() {
    const pagination = this.page.locator('.bk-pagination').last();
    await pagination.getByText('2', { exact: true }).click();
    await expect(this.getBuildTable().getByText('#211', { exact: true }).first()).toBeVisible({ timeout: 10000 });
  }

  /** 清空构建记录搜索 */
  async clearBuildRecordSearch() {
    const searchInput = this.buildRecordSearchInput();
    await searchInput.fill('');
    await expect(this.getBuildTable().getByText('#101', { exact: true }).first()).toBeVisible({ timeout: 10000 });
  }

  /** 断言构建失败状态可打开日志 */
  async expectBuildFailedRecordsVisible() {
    const table = this.getBuildTable();
    await expect(table.getByText('#301', { exact: true })).toBeVisible();
    await expect(table.getByText('#302', { exact: true })).toBeVisible();
  }

  /** 断言构建日志侧栏内容已加载 */
  async expectBuildLogPanelVisible(statusText = '构建成功') {
    await expect(this.page.getByText('构建日志', { exact: true }).last()).toBeVisible({ timeout: 10000 });
    await expect(this.page.getByText(statusText, { exact: false }).last()).toBeVisible();
    await expect(this.page.getByText('e2e build log line', { exact: false })).toBeVisible({ timeout: 10000 });
  }

  /** 断言构建管理页的记录列表与查询入口已加载 */
  async expectBuildManagementVisible() {
    const table = this.getBuildTable();
    await expect(this.page.getByText('构建管理', { exact: true }).last()).toBeVisible();
    await expect(this.page.getByRole('button', { name: '执行构建' }).first()).toBeVisible();
    await expect(this.buildRecordSearchInput()).toBeVisible();
    await expect(table).toBeVisible();
    for (const column of ['构建号', '源材料', '触发人', '构建开始时间', '构建结束时间', '构建耗时', '制品']) {
      await expect(table.getByText(column, { exact: true }).first()).toBeVisible();
    }
    await expect(this.buildTableMainRows().first()).toBeVisible({ timeout: 10000 });
  }

  /** 断言构建记录搜索结果与关键字匹配 */
  async expectBuildRecordSearchResultMatches() {
    if (!this.buildRecordSearchKeyword) {
      throw new Error('未设置构建记录查询关键字');
    }
    await expect(this.buildTableMainRows()).toHaveCount(1, { timeout: 10000 });
    await expect(this.getBuildTable().getByText(this.buildRecordSearchKeyword, { exact: true }).first()).toBeVisible();
  }

  /** 断言构建记录错误空态可见 */
  async expectBuildRecordsErrorVisible() {
    const exception = this.getBuildTable().locator('.bk-exception:visible').filter({ hasText: '数据获取异常' }).first();
    await expect(exception).toBeVisible({ timeout: 10000 });
    await expect(exception.getByRole('button', { name: '刷新' })).toBeVisible();
  }

  /** 断言构建记录每页 20 条展示 */
  async expectBuildRecordsPageSizeTwentyVisible() {
    await expect(this.buildTableMainRows()).toHaveCount(18);
    await expect(this.getBuildTable().getByText('#218', { exact: true }).first()).toBeVisible();
  }

  /** 断言构建记录已恢复到完整列表 */
  async expectBuildRecordsRestored() {
    await expect(this.getBuildTable().getByText('#101', { exact: true }).first()).toBeVisible({ timeout: 10000 });
    await expect(this.getBuildTable().getByText('#102', { exact: true }).first()).toBeVisible({ timeout: 10000 });
  }

  /** 断言构建记录分页第二页展示 */
  async expectBuildRecordsSecondPageVisible() {
    await expect(this.getBuildTable().getByText('#211', { exact: true }).first()).toBeVisible();
  }

  /** 获取构建日志侧栏 */
  getBuildLogSideslider(): Locator {
    return this.page.locator('.build-log-sideslider:visible').first();
  }

  /** 获取构建管理表格 */
  getBuildTable(): Locator {
    return this.page.locator('.build-table').first();
  }

  /** 进入构建管理页（二级菜单 key: build） */
  async gotoBuildManagement() {
    await this.gotoMenu('build');
    await this.expectBuildManagementVisible();
  }

  /** 进入构建管理页并等待构建记录异常空态 */
  async gotoBuildManagementWithRecordError() {
    await this.gotoMenu('build');
    await this.expectBuildRecordsErrorVisible();
  }

  /** 打开首行构建日志侧栏 */
  async openFirstBuildLog() {
    const firstBuildLink = this.page
      .locator('.build-table .vxe-table--fixed-left-wrapper .vxe-body--row')
      .first()
      .getByText(/^#/);
    await firstBuildLink.click();
  }

  /** 点击构建记录错误空态中的刷新 */
  async refreshBuildRecordsFromError() {
    await this.getBuildTable().locator('.bk-exception:visible').getByRole('button', { name: '刷新' }).click();
    await expect(this.getBuildTable().getByText('#101', { exact: true }).first()).toBeVisible({ timeout: 10000 });
  }

  /** 按约定关键字搜索构建记录 */
  async searchBuildRecordsByKnownKeyword() {
    this.buildRecordSearchKeyword = 'e2e-filter-user';
    const searchInput = this.buildRecordSearchInput();
    await searchInput.fill(this.buildRecordSearchKeyword);
    await expect(this.getBuildTable().getByText(this.buildRecordSearchKeyword, { exact: true }).first()).toBeVisible({
      timeout: 10000,
    });
  }

  /** 配置构建日志 SSE mock */
  async setupBuildLogStreamMock() {
    await this.routeBuildLogStream();
  }

  /** 配置构建记录接口先失败再恢复的 mock */
  async setupBuildRecordsErrorRecoveryMock() {
    await this.routeBuildRecords(
      [
        this.buildRecord(),
        this.buildRecord({
          artifact: 'registry.example.com/bkms/e2e-service:v1.0.1',
          buildID: 'build-e2e-002',
          num: '102',
          params: { BKMS_IMAGE_TAG: 'v1.0.1' },
        }),
      ],
      { failFirst: true },
    );
  }

  /** 配置构建记录分页 mock */
  async setupBuildRecordsPaginationMock() {
    const records = Array.from({ length: 18 }, (_, index) => {
      const num = String(201 + index);
      return this.buildRecord({
        artifact: `registry.example.com/bkms/e2e-service:v2.0.${index + 1}`,
        buildID: `build-e2e-page-${num}`,
        num,
        operator: index === 10 ? 'e2e-second-page-user' : 'e2e-page-user',
        params: { BKMS_IMAGE_TAG: `v2.0.${index + 1}` },
      });
    });
    await this.routeBuildRecords(records);
  }

  /** 配置构建记录只读 mock */
  async setupBuildRecordsReadonlyMock() {
    await this.routeBuildRecords([
      this.buildRecord(),
      this.buildRecord({
        artifact: 'registry.example.com/bkms/e2e-service:v1.0.1',
        buildID: 'build-e2e-002',
        num: '102',
        operator: 'e2e-filter-user',
        params: { BKMS_IMAGE_TAG: 'v1.0.1' },
      }),
    ]);
  }

  /** 配置构建失败状态记录 mock */
  async setupBuildRecordsWithFailedStatusesMock() {
    await this.routeBuildRecords([
      this.buildRecord({
        artifact: 'registry.example.com/bkms/e2e-service:v3.0.1',
        buildID: 'build-e2e-failed',
        num: '301',
        params: { BKMS_IMAGE_TAG: 'v3.0.1' },
        status: 'failed',
      }),
      this.buildRecord({
        artifact: 'registry.example.com/bkms/e2e-service:v3.0.2',
        buildID: 'build-e2e-polling-broken',
        num: '302',
        params: { BKMS_IMAGE_TAG: 'v3.0.2' },
        status: 'pollingBroken',
      }),
    ]);
  }
}
