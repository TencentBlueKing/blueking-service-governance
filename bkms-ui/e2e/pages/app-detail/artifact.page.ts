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
import { expect } from '@playwright/test';

import AppDetailBase from './app-detail-base.page';

/**
 * 制品管理域 Page Object。
 *
 * 覆盖：容器镜像列表（同步/晋级/删除/搜索/展开详情）、
 * Helm Chart 版本列表与生命周期，以及对应的 route mock。
 */
export default class ArtifactPage extends AppDetailBase {
  private artifactSearchTag = '';

  private helmChartSearchVersion = '';

  private containerImage(overrides: Record<string, unknown> = {}) {
    return {
      builtAt: '2026-01-01T10:00:00Z',
      deployedEnvs: [{ envName: 'test', envType: 'test' }],
      digest: 'sha256:abcdef1234567890',
      isPromoted: false,
      repository: 'registry.example.com/bkms/e2e-service',
      size: '1024',
      tag: 'e2e-lifecycle-1',
      ...overrides,
    };
  }

  private helmChart(overrides: Record<string, unknown> = {}) {
    return {
      chartVersion: '1.2.3',
      createdAt: '2026-01-01T10:00:00Z',
      deployedEnvs: [{ envName: 'test', envType: 'test' }],
      digest: 'sha256:fedcba9876543210',
      ...overrides,
    };
  }

  private async routeBkciRepoRefs() {
    await this.page.route('**/workspaces/*/bkci-repositories/**', async route => {
      const url = new URL(route.request().url());
      if (!url.pathname.includes('/bkci-repositories/')) {
        await route.fallback();
        return;
      }
      await this.fulfillJson(route, {
        data: [{ name: 'main' }],
        status: 0,
      });
    });
  }

  private async routeEnvList() {
    await this.page.route('**/workspaces/*/envs**', async route => {
      const url = new URL(route.request().url());
      if (route.request().method() !== 'GET' || !url.pathname.endsWith('/envs')) {
        await route.fallback();
        return;
      }
      await this.fulfillJson(route, {
        data: [
          { displayName: '测试环境', name: 'test', type: 'test' },
          { displayName: '生产环境', name: 'prod', type: 'production' },
        ],
        status: 0,
      });
    });
  }

  private async routeHelmChartLifecycle() {
    let buildListCalls = 0;
    await this.routeBkciRepoRefs();
    await this.page.route('**/apps/*/charts**', async route => {
      const request = route.request();
      const url = new URL(request.url());
      const { pathname } = url;
      const method = request.method();

      if (method === 'GET' && pathname.endsWith('/charts')) {
        const keyword = (url.searchParams.get('keyword') || '').trim();
        const charts = [this.helmChart(), this.helmChart({ chartVersion: '1.2.4' })];
        const results = keyword ? charts.filter(chart => String(chart.chartVersion).includes(keyword)) : charts;
        await this.fulfillJson(route, {
          data: {
            count: String(results.length),
            results,
          },
          status: 0,
        });
        return;
      }

      if (method === 'GET' && pathname.endsWith('/charts/semver')) {
        await this.fulfillJson(route, {
          data: {
            latest: { major: '1', minor: '2', patch: '3', version: '1.2.3' },
            next: { major: '1', minor: '2', patch: '4', version: '1.2.4' },
          },
          status: 0,
        });
        return;
      }

      if (method === 'POST' && pathname.endsWith('/charts/builds')) {
        await this.fulfillJson(route, {
          data: {
            buildID: 'helm-build-e2e-001',
            chartVersion: '1.2.4',
          },
          status: 0,
        });
        return;
      }

      if (method === 'GET' && pathname.endsWith('/charts/builds')) {
        buildListCalls += 1;
        await this.fulfillJson(route, {
          data: {
            count: '1',
            results: [
              {
                buildID: 'helm-build-e2e-001',
                chartVersion: '1.2.4',
                endedAt: buildListCalls > 1 ? '2026-01-01T10:02:00Z' : '0001-01-01T00:00:00Z',
                extras: {
                  BK_CI_GIT_REPO_HEAD_COMMIT_ID: '1234567890abcdef',
                  BK_CI_GIT_REPO_URL: 'https://example.com/bkms/e2e-helm.git',
                },
                num: '501',
                operator: 'e2e-user',
                params: {
                  BKMS_REPO_REVISION: 'main',
                },
                pipelineID: 'helm-pipeline-e2e',
                startedAt: '2026-01-01T10:00:00Z',
                status: buildListCalls > 1 ? 'success' : 'running',
              },
            ],
          },
          status: 0,
        });
        return;
      }

      if (method === 'GET' && pathname.endsWith('/files')) {
        await this.fulfillJson(route, {
          data: {
            chartName: 'e2e-chart',
            chartVersion: '1.2.3',
            root: {
              children: [
                {
                  content: 'apiVersion: v2\nname: e2e-chart\nversion: 1.2.3\n',
                  isBinary: false,
                  isDir: false,
                  name: 'Chart.yaml',
                  path: 'Chart.yaml',
                  size: '48',
                },
              ],
              isDir: true,
              name: 'e2e-chart',
              path: '',
            },
          },
          status: 0,
        });
        return;
      }

      if (method === 'GET' && pathname.endsWith('/valuesfile')) {
        await this.fulfillJson(route, { data: 'replicaCount: 1\n', status: 0 });
        return;
      }

      await route.fallback();
    });
  }

  private async routeImageLifecycle(deleteMode: 'permission-error' | 'success' = 'success') {
    const state = {
      images: [this.containerImage(), this.containerImage({ isPromoted: true, tag: 'e2e-promoted-1' })],
    };
    await this.routeEnvList();
    await this.page.route('**/apps/*/images**', async route => {
      const request = route.request();
      const url = new URL(request.url());
      const { pathname } = url;
      const method = request.method();

      if (method === 'GET' && pathname.endsWith('/images')) {
        const keyword = (url.searchParams.get('keyword') || '').trim();
        const results = keyword ? state.images.filter(image => String(image.tag).includes(keyword)) : state.images;
        await this.fulfillJson(route, {
          data: {
            count: String(results.length),
            productionEnvNames: ['prod'],
            results,
            snapshotStatus: { refreshStatus: 'idle' },
          },
          status: 0,
        });
        return;
      }

      if (method === 'POST' && pathname.endsWith('/images/refresh')) {
        state.images.unshift(this.containerImage({ tag: 'e2e-synced-1' }));
        await this.fulfillJson(route, {
          data: { addedTagCnt: '1', removedTagCnt: '0', status: 'success' },
          status: 0,
        });
        return;
      }

      if (method === 'GET' && pathname.endsWith('/deploy-records')) {
        await this.fulfillJson(route, {
          data: {
            count: '1',
            results: [{ createdAt: '2026-01-01T11:00:00Z', envName: 'test', operator: 'e2e-user', status: 'deployed' }],
          },
          status: 0,
        });
        return;
      }

      if (method === 'GET' && pathname.endsWith('/usages')) {
        await this.fulfillJson(route, {
          data: {
            inUse: true,
            usages: [{ envName: 'test', laneName: '', status: 'deployed', workloadName: 'e2e-workload' }],
          },
          status: 0,
        });
        return;
      }

      if (method === 'PATCH' && pathname.endsWith('/promote')) {
        state.images = state.images.map(image =>
          image.tag === 'e2e-lifecycle-1'
            ? { ...image, isPromoted: true, promotedAt: '2026-01-01T12:00:00Z', promotedBy: 'e2e-user' }
            : image,
        );
        await this.fulfillJson(route, { data: {}, status: 0 });
        return;
      }

      if (method === 'DELETE' && pathname.includes('/images/')) {
        if (deleteMode === 'permission-error') {
          await this.fulfillJson(
            route,
            {
              error: {
                details: [
                  {
                    code: 'IMAGE_REPOSITORY_AUTH_REQUIRED',
                    extras: {},
                    message: 'permission required',
                    module: 'images',
                    system: 'bkms',
                  },
                ],
                message: 'permission required',
              },
              status: 500,
            },
            500,
          );
          return;
        }

        state.images = state.images.filter(image => image.tag !== 'e2e-lifecycle-1');
        await this.fulfillJson(route, { data: {}, status: 0 });
        return;
      }

      await route.fallback();
    });
  }

  /** 收起制品列表首行详情 */
  async collapseFirstArtifactRow() {
    const firstRow = this.page.locator('.artifact-table .vxe-table--body .vxe-body--row').first();
    await firstRow.click();
  }

  /** 新建 Helm Chart 版本并打开构建记录 */
  async createHelmChartVersionBuild() {
    await this.closeVisibleSideslider('版本详情');
    await this.page.getByRole('button', { name: '新建版本' }).click();
    const dialog = this.getDialog();
    await expect(dialog.getByText('新建版本', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(dialog.getByRole('textbox').last()).toHaveValue(/1\.2\.4/, { timeout: 10000 });

    const branchFormItem = dialog.locator('.bk-form-item').filter({ hasText: '分支' }).first();
    const branchSelect = branchFormItem.locator('.repo-ref-select');
    if (await branchSelect.isVisible().catch(() => false)) {
      await branchSelect.click();
      await this.selectOption('main');
    } else {
      const branchInput = branchFormItem.getByRole('textbox').first();
      await branchInput.fill('main');
    }

    const responsePromise = this.page.waitForResponse(
      response => response.request().method() === 'POST' && response.url().includes('/charts/builds'),
      { timeout: 30000 },
    );
    await dialog.getByRole('button', { name: '确定' }).click();
    await this.assertApiResponseOk(await responsePromise, '新建 Helm Chart 版本');
    await this.expectHelmChartBuildRecordVisible();
  }

  /** 确认删除首个镜像 Tag */
  async deleteFirstImageTag() {
    await this.page.locator('.artifact-table').getByRole('button', { name: '删除' }).first().click();
    const dialog = this.getDialog();
    await expect(dialog.getByText('确定删除镜像 Tag ?', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(dialog.locator('.bk-loading')).toBeHidden({ timeout: 10000 });
    const input = dialog.getByPlaceholder('请输入镜像 Tag');
    await input.fill('e2e-lifecycle-1');
    await expect(dialog.getByRole('button', { name: '删除' })).toBeEnabled({ timeout: 10000 });
    await dialog.getByRole('button', { name: '删除' }).click();
  }

  /** 展开制品列表首行并确认详情内容已展示 */
  async expandFirstArtifactRow() {
    const firstRow = this.page.locator('.artifact-table .vxe-table--body .vxe-body--row').first();
    await expect(firstRow).toBeVisible();
    await firstRow.click();
  }

  /** 断言制品管理页的只读内容已加载 */
  async expectArtifactManagementVisible() {
    const artifactTable = this.page.locator('.artifact-table');
    // 左侧导航和内容区均有同名文本，取内容区标题对应的最后一个元素。
    await expect(this.page.getByText('制品管理', { exact: true }).last()).toBeVisible();
    await expect(this.page.getByRole('button', { name: '一键同步' })).toBeVisible();
    await expect(artifactTable).toBeVisible();
    for (const column of ['镜像 Tag', '大小', '构建时间', '摘要', '已部署环境', '操作']) {
      // VXE 固定列会复制表头 DOM，首个可见匹配即可说明该列已渲染。
      await expect(artifactTable.getByText(column, { exact: true }).first()).toBeVisible();
    }
  }

  /** 断言按 Tag 查询后结果与查询条件一致 */
  async expectArtifactSearchResultMatches() {
    if (!this.artifactSearchTag) {
      throw new Error('未设置制品 Tag 查询条件');
    }
    const mainRows = this.page.locator('.artifact-table .vxe-table--main-wrapper .vxe-body--row');
    await expect(mainRows).toHaveCount(1);
    await expect(
      this.page
        .locator('.artifact-table .vxe-table--fixed-left-wrapper')
        .getByText(this.artifactSearchTag, { exact: true }),
    ).toBeVisible();
  }

  /** 断言制品列表首行详情已收起 */
  async expectFirstArtifactRowCollapsed() {
    await expect(this.page.getByText('部署记录', { exact: true }).first()).toBeHidden();
  }

  /** 断言展开的制品行包含详情字段，且部署记录已完成加载 */
  async expectFirstArtifactRowDetailVisible() {
    const detailPanel = this.page.locator('.artifact-table .vxe-body--expanded-column').first();
    for (const label of ['镜像仓库：', 'tag：', '大小：', '构建时间：', '摘要：', '已部署环境：', '部署记录']) {
      await expect(detailPanel.getByText(label, { exact: true }).first()).toBeVisible();
    }

    await expect(detailPanel.locator('.bk-loading')).toBeHidden();
    await expect(detailPanel.locator('ul').or(detailPanel.locator('.bk-exception')).first()).toBeVisible();
  }

  /** 断言制品列表首行详情已展开 */
  async expectFirstArtifactRowExpanded() {
    // VXE 固定列会复制展开内容，验证其中一个详情面板可见即可。
    await expect(this.page.getByText('部署记录', { exact: true }).first()).toBeVisible();
  }

  /** 确认 Helm Chart 构建记录侧栏展示目标构建 */
  async expectHelmChartBuildRecordVisible() {
    await expect(this.page.getByText('版本构建记录', { exact: true }).last()).toBeVisible({ timeout: 10000 });
    const recordRow = this.page.locator('.vxe-body--row:visible').filter({ hasText: '#501' }).last();
    await expect(recordRow).toBeVisible({ timeout: 10000 });
    await expect(recordRow).toContainText('1.2.4');
  }

  /** 断言 Helm Chart 列表与详情已加载 */
  async expectHelmChartDetailVisible() {
    const slider = this.getSideslider();
    await expect(slider.getByText('版本详情', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(slider.getByText('Chart.yaml', { exact: true }).first()).toBeVisible({ timeout: 10000 });
    await expect(slider.getByText('e2e-chart', { exact: false }).first()).toBeVisible({ timeout: 10000 });
  }

  /** 断言 Helm Chart 搜索结果匹配 */
  async expectHelmChartSearchResultMatches() {
    if (!this.helmChartSearchVersion) {
      throw new Error('未设置 Helm Chart 查询版本');
    }
    await expect(this.page.locator('.helm-chart-table .vxe-table--main-wrapper .vxe-body--row')).toHaveCount(1, {
      timeout: 10000,
    });
    await expect(this.page.getByText(this.helmChartSearchVersion, { exact: true }).first()).toBeVisible();
  }

  /** 断言镜像删除成功并从列表移除 */
  async expectImageDeleted() {
    await expect(this.page.getByText('镜像 Tag 删除成功', { exact: false })).toBeVisible({ timeout: 10000 });
    await expect(
      this.page.locator('.artifact-table .vxe-table--fixed-left-wrapper').getByText('e2e-lifecycle-1', { exact: true }),
    ).toHaveCount(0, { timeout: 10000 });
  }

  /** 断言镜像删除权限错误弹窗可见 */
  async expectImageDeletePermissionErrorVisible() {
    await expect(this.page.getByText('镜像 Tag 删除失败', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(this.page.getByRole('button', { name: '构建管理 - 构建配置' })).toBeVisible();
  }

  /** 断言镜像已晋级 */
  async expectImagePromoted() {
    await expect(this.page.getByText('已晋级', { exact: true }).first()).toBeVisible({ timeout: 10000 });
  }

  /** 断言一键同步后新增镜像已展示 */
  async expectImageSynced() {
    await expect(this.page.getByText('镜像同步成功', { exact: false })).toBeVisible({ timeout: 10000 });
    await expect(this.page.getByText('e2e-synced-1', { exact: true }).first()).toBeVisible({ timeout: 10000 });
  }

  /** 进入制品管理页（二级菜单 key: artifact） */
  async gotoArtifactManagement() {
    await this.gotoMenu('artifact');
    await this.expectArtifactManagementVisible();
  }

  /** 进入 Helm Chart 制品页 */
  async gotoHelmChartArtifactManagement() {
    await this.gotoMenu('artifact', { activeTab: 'helm-chart' });
    await expect(this.page.getByText('制品管理', { exact: true }).last()).toBeVisible({ timeout: 10000 });
    await expect(this.page.getByText('Helm Chart', { exact: true }).first()).toBeVisible();
    await expect(this.page.locator('.helm-chart-table')).toBeVisible({ timeout: 10000 });
  }

  /** 打开 Helm Chart 版本详情 */
  async openHelmChartVersionDetail() {
    await this.page.locator('.helm-chart-table').getByRole('button', { name: '1.2.3' }).click();
    await this.expectHelmChartDetailVisible();
  }

  /** 晋级首个镜像 Tag */
  async promoteFirstImageTag() {
    await this.page.locator('.artifact-table').getByRole('button', { name: '晋级' }).first().click();
    const dialog = this.getDialog();
    await expect(dialog.getByText('确认晋级', { exact: true })).toBeVisible({ timeout: 10000 });
    await dialog.getByRole('button', { name: '确定' }).click();
    await this.expectImagePromoted();
  }

  /** 使用首行镜像 Tag 验证制品搜索 */
  async searchArtifactsByFirstRowTag() {
    const firstRow = this.page.locator('.artifact-table .vxe-table--fixed-left-wrapper .vxe-body--row').first();
    const tagCell = firstRow.locator('.vxe-body--column').nth(1);
    this.artifactSearchTag = (await tagCell.innerText()).trim();
    if (!this.artifactSearchTag) {
      throw new Error('制品列表首行未获取到镜像 Tag，无法执行查询验证');
    }

    const searchInput = this.page.locator('.bk-search-select .div-input');
    await searchInput.click();
    await this.page.locator('.bk-search-select-popover').getByText('Tag', { exact: true }).click();
    await searchInput.fill(this.artifactSearchTag);
    await searchInput.press('Enter');
    await expect(firstRow.getByText(this.artifactSearchTag, { exact: true })).toBeVisible({ timeout: 15000 });
  }

  /** 搜索 Helm Chart 版本 */
  async searchHelmChartByVersion() {
    this.helmChartSearchVersion = '1.2.3';
    const input = this.page.locator('input[type="search"]').first();
    await input.fill(this.helmChartSearchVersion);
    await input.press('Enter');
    await expect(this.page.getByText(this.helmChartSearchVersion, { exact: true }).first()).toBeVisible({
      timeout: 10000,
    });
  }

  /** 配置容器镜像生命周期 mock */
  async setupContainerImageLifecycleMock() {
    await this.routeImageLifecycle('success');
  }

  /** 配置容器镜像删除权限异常 mock */
  async setupContainerImagePermissionErrorMock() {
    await this.routeImageLifecycle('permission-error');
  }

  /** 配置 Helm Chart 生命周期 mock */
  async setupHelmChartLifecycleMock() {
    await this.routeHelmChartLifecycle();
  }

  /** 执行容器镜像一键同步 */
  async syncContainerImages() {
    await this.page.getByRole('button', { name: '一键同步' }).click();
    await this.expectImageSynced();
  }
}
