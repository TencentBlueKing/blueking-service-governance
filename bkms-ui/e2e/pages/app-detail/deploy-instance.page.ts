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
import { expect } from '@playwright/test';

import AppDetailBase from './app-detail-base.page';

/**
 * 部署管理实例列表域 Page Object。
 *
 * 覆盖：实例列表展示、行操作（日志/灰度/调整权重）、批量操作栏、管理命令侧栏、
 * 联邦与特性环境的只读展示，以及实例列表相关接口 mock。
 * 部署/扩缩容/移除等流程见 deploy.page.ts。
 */
export default class DeployInstancePage extends AppDetailBase {
  /** 特性环境 mock 数据（「共 N 个」等断言由该数组派生，避免多处硬编码） */
  private readonly featureEnvs = [
    {
      cluster: { clusterID: 'e2e-cluster', namespace: 'e2e-feature-waiting-ns' },
      createdAt: '2026-01-01T10:00:00Z',
      creator: 'e2e-user',
      deployStatuses: [],
      displayName: 'e2e-feature-waiting',
      id: 'feature-env-waiting',
      name: 'e2e-feature-waiting',
      sourceEnv: { displayName: '测试环境', id: 'env-e2e-test', isDeleted: false, name: 'test' },
      status: 'Ready',
      type: 'test',
    },
    {
      cluster: { clusterID: 'e2e-cluster', namespace: 'e2e-feature-deployed-ns' },
      createdAt: '2026-01-01T10:01:00Z',
      creator: 'e2e-user',
      deployStatuses: [{ deployStatus: 'deployed', imageTag: 'v1.0.0' }],
      displayName: 'e2e-feature-deployed',
      id: 'feature-env-deployed',
      name: 'e2e-feature-deployed',
      sourceEnv: { displayName: '测试环境', id: 'env-e2e-test', isDeleted: false, name: 'test' },
      status: 'Ready',
      type: 'test',
    },
    {
      cluster: { clusterID: 'e2e-cluster', namespace: 'e2e-feature-deploying-ns' },
      createdAt: '2026-01-01T10:02:00Z',
      creator: 'e2e-user',
      deployStatuses: [{ deployStatus: 'deploying', imageTag: 'v1.0.1' }],
      displayName: 'e2e-feature-deploying',
      id: 'feature-env-deploying',
      name: 'e2e-feature-deploying',
      sourceEnv: { displayName: '测试环境', id: 'env-e2e-test', isDeleted: false, name: 'test' },
      status: 'Ready',
      type: 'test',
    },
  ];

  private instanceListRequestCount = 0;

  private instanceWatchRequestCount = 0;

  private deploymentInstance(index: number, overrides: Record<string, unknown> = {}) {
    return {
      age: `${index + 1}m`,
      deployID: 'deploy-e2e-001',
      id: `e2e-instance-${String(index).padStart(2, '0')}`,
      image: `registry.example.com/bkms/e2e-service:v1.0.${index % 3}`,
      ip: `10.0.0.${index}`,
      isHealthy: true,
      nodeIP: `192.168.0.${index}`,
      polarisInfos: [
        {
          enableHealthCheck: true,
          ip: `10.0.0.${index}`,
          isHealthy: true,
          isIsolated: false,
          port: 8080,
          serviceName: 'e2e-service',
          serviceNamespace: 'e2e',
          staticWeight: '100',
          weight: '100',
        },
      ],
      resources: {
        cpuLimits: '1',
        cpuRequests: '500m',
        memoryLimits: '1Gi',
        memoryRequests: '512Mi',
      },
      restartCount: String(index % 5),
      status: 'Running',
      ...overrides,
    };
  }

  /**
   * 三行差异化数据：行 1 正常（主实例）、行 2 健康状态异常（UnHealthy）、
   * 行 3 未注册北极星（polarisInfos 为空，展示 --）。
   */
  private deploymentInstances() {
    return Array.from({ length: 3 }, (_, index) =>
      this.deploymentInstance(index + 1, {
        ...(index === 0 ? { id: 'e2e-instance-primary', restartCount: '9' } : {}),
        ...(index === 1 ? { isHealthy: false } : {}),
        ...(index === 2 ? { polarisInfos: [] } : {}),
      }),
    );
  }

  private getFeatureEnvRow(displayName: string) {
    return this.page.locator('tr, .bk-table-row').filter({ hasText: displayName }).first();
  }

  private getInstanceBatchToolbar() {
    // 「管理命令」按钮仅存在于批量操作栏（行内操作列无此按钮），其直接父容器即工具栏根节点，
    // 据此与实例表格内的同名「监控/灰度」行按钮区分。
    return this.page.getByRole('button', { name: '管理命令', exact: true }).locator('xpath=ancestor::div[1]');
  }

  private getInstanceTable() {
    return this.page.locator('.instance-table').first();
  }

  private getManagementCommandConfigCard() {
    // ToggleCard 的「配置命令」标题与内容共用第三层父容器，借此限定到管理命令侧栏中的命令选择区。
    return this.page.getByText('配置命令', { exact: true }).locator('xpath=ancestor::div[3]');
  }

  private getManagementCommandSelect() {
    return this.getManagementCommandConfigCard().locator('.bk-select:visible').last();
  }

  private getVisibleInstanceRows() {
    return this.getInstanceTable().locator('.vxe-table--main-wrapper .vxe-body--row:visible');
  }

  private async routeAdminCommands() {
    await this.page.route('**/apps/*/envs/*/instances/admin-cmds**', async route => {
      if (route.request().method() !== 'GET') {
        await route.fallback();
        return;
      }
      await this.fulfillJson(route, {
        data: {
          results: ['/cmd/e2e-health', '/cmd/e2e-version'],
        },
        status: 0,
      });
    });
  }

  private async routeAppEnvs({ federation = false } = {}) {
    await this.page.route('**/apps/*/envs**', async route => {
      const request = route.request();
      const url = new URL(request.url());
      if (request.method() !== 'GET' || !/\/apps\/[^/]+\/envs$/.test(url.pathname)) {
        await route.fallback();
        return;
      }

      await this.fulfillJson(route, {
        data: [
          {
            appIDs: ['e2e-app'],
            cluster: {
              clusterID: federation ? 'e2e-federation-cluster' : 'e2e-cluster',
              clusterType: federation ? 'federation' : 'kubernetes',
              isFederation: federation,
              namespace: 'e2e-namespace',
            },
            displayName: federation ? '联邦测试环境' : '测试环境',
            id: federation ? 'env-e2e-federation' : 'env-e2e-test',
            kind: 'standard',
            name: federation ? 'e2e-federation' : 'test',
            status: 'Ready',
            type: 'test',
          },
        ],
        status: 0,
      });
    });
  }

  private async routeDeployLatestStatus() {
    await this.page.route('**/apps/*/envs/*/{trpc-deploys,taf-deploys}/latest-status**', async route => {
      if (route.request().method() !== 'GET') {
        await route.fallback();
        return;
      }
      await this.fulfillJson(route, {
        data: {
          deployID: 'deploy-e2e-001',
          hasDeployRecord: true,
          imageTag: 'v1.0.0',
          operator: 'e2e-user',
          stage: 'deploy',
          status: 'deployed',
        },
        status: 0,
      });
    });
  }

  private async routeEffectiveResources() {
    await this.page.route('**/apps/*/envs/*/app-spec/resources/effective**', async route => {
      if (route.request().method() !== 'GET') {
        await route.fallback();
        return;
      }
      await this.fulfillJson(route, {
        data: {
          replicas: 12,
          resources: {
            cpuLimits: '1',
            cpuRequests: '500m',
            memoryLimits: '1Gi',
            memoryRequests: '512Mi',
          },
        },
        status: 0,
      });
    });
  }

  private async routeFeatureEnvs() {
    await this.page.route('**/apps/*/feat-envs**', async route => {
      const request = route.request();
      const url = new URL(request.url());
      if (request.method() !== 'GET' || !url.pathname.endsWith('/feat-envs')) {
        await route.fallback();
        return;
      }

      await this.fulfillJson(route, {
        data: this.featureEnvs,
        status: 0,
      });
    });
  }

  private async routeImages() {
    await this.page.route('**/apps/*/images**', async route => {
      const request = route.request();
      const url = new URL(request.url());
      if (request.method() !== 'GET' || !url.pathname.endsWith('/images')) {
        await route.fallback();
        return;
      }
      await this.fulfillJson(route, {
        data: {
          count: '2',
          productionEnvNames: [],
          results: [
            {
              builtAt: '2026-01-01T10:00:00Z',
              deployedEnvs: [],
              digest: 'sha256:e2e001',
              repository: 'registry.example.com/bkms/e2e-service',
              size: '1024',
              tag: 'v1.0.0',
            },
            {
              builtAt: '2026-01-01T10:01:00Z',
              deployedEnvs: [],
              digest: 'sha256:e2e002',
              repository: 'registry.example.com/bkms/e2e-service',
              size: '1024',
              tag: 'v1.0.1',
            },
          ],
          snapshotStatus: { refreshStatus: 'idle' },
        },
        status: 0,
      });
    });
  }

  private async routeInstanceList({ federation = false } = {}) {
    const instances = this.deploymentInstances();
    await this.page.route('**/apps/*/envs/*/instances**', async route => {
      const request = route.request();
      const url = new URL(request.url());
      if (request.method() !== 'GET' || !url.pathname.endsWith('/instances')) {
        await route.fallback();
        return;
      }
      this.instanceListRequestCount += 1;

      if (federation) {
        // 联邦环境前端只做全量轮询 List、不建 Watch（use-instance-list-watch.ts poll 分支），响应不带 resourceVersion；
        // 普通环境分支缺失 resourceVersion 前端会直接抛错，因此两种响应结构必须区分。
        await this.fulfillJson(route, {
          data: {
            count: String(instances.length),
            results: instances,
          },
          status: 0,
        });
        return;
      }

      await this.fulfillJson(route, {
        data: {
          count: String(instances.length),
          resourceVersion: 'rv-e2e-001',
          results: instances,
        },
        status: 0,
      });
    });
  }

  private async routeInstanceLogs() {
    await this.page.route('**/apps/*/envs/*/instances/*/logs**', async route => {
      if (route.request().method() !== 'GET') {
        await route.fallback();
        return;
      }
      await this.fulfillJson(route, {
        data: [{ content: 'e2e instance log line', timestamp: '2026-01-01T10:00:10Z' }],
        status: 0,
      });
    });
  }

  private async routeInstanceWatch() {
    await this.page.route('**/apps/*/envs/*/instances/watch**', async route => {
      if (route.request().method() !== 'GET') {
        await route.fallback();
        return;
      }
      this.instanceWatchRequestCount += 1;
      await route.fulfill({
        body: ['event: message', 'data: {"type":"ENDED","reason":"watch timeout"}', '', ''].join('\n'),
        contentType: 'text/event-stream',
        status: 200,
      });
    });
  }

  private async visibleInstanceRowTexts() {
    const texts = await this.instanceRows().evaluateAll(rows =>
      rows
        .filter(row => {
          const rect = row.getBoundingClientRect();
          const style = window.getComputedStyle(row);
          return rect.width > 0 && rect.height > 0 && style.display !== 'none' && style.visibility !== 'hidden';
        })
        .map(row => row.textContent?.replace(/\s+/g, ' ').trim() || ''),
    );

    const instanceStatePattern = /\b(Running|Pending|Failed|Succeeded|Unknown|Healthy|Unhealthy|Error)\b/;
    return texts.filter(
      text => text && !text.includes('暂无数据') && !text.includes('No Data') && instanceStatePattern.test(text),
    );
  }

  /** 关闭批量删除实例的 InfoBox 确认弹窗 */
  async closeDeploymentInstanceDeleteDialog() {
    const title = this.page.getByText('确认删除该实例', { exact: false });
    await this.page.getByRole('button', { name: '取消', exact: true }).last().click();
    await expect(title).toBeHidden({ timeout: 10000 });
  }

  /** 关闭当前实例操作弹窗 */
  async closeInstanceActionDialog() {
    await this.getDialog().getByRole('button', { name: '取消' }).click();
    await this.waitForDialogClosed();
  }

  /** 关闭当前实例日志侧栏 */
  async closeInstanceLogSideslider() {
    await this.closeVisibleSideslider('实例日志');
  }

  /** 断言实例列表批量操作栏已启用（限定在工具栏容器内，避免与行内同名按钮混淆） */
  async expectDeploymentInstanceBatchToolbarVisible() {
    const toolbar = this.getInstanceBatchToolbar();
    await expect(toolbar).toBeVisible({ timeout: 10000 });
    for (const action of ['监控', '灰度', '删除', '管理命令']) {
      await expect(toolbar.getByRole('button', { name: action, exact: true })).toBeVisible();
    }
    // 勾选后表格顶部出现跨页选择信息条（「已选择 “N” 条，」）
    await expect(
      this.getInstanceTable()
        .getByText(/已选择|已跨页全选/)
        .first(),
    ).toBeVisible({ timeout: 10000 });
  }

  /** 断言批量删除实例的确认弹窗展示当前选中实例信息 */
  async expectDeploymentInstanceDeleteDialogVisible() {
    await expect(this.page.getByText('确认删除该实例', { exact: false })).toBeVisible({ timeout: 10000 });
    await expect(this.page.getByText('实例: e2e-instance-primary (10.0.0.1)', { exact: true })).toBeVisible();
    await expect(this.page.getByText('此操作将删除该实例，并调整实例数', { exact: true })).toBeVisible();
    await expect(this.page.getByRole('button', { name: '取消', exact: true }).last()).toBeVisible();
  }

  /** 断言实例列表展示关键列、首行数据、行操作入口与差异化状态（UnHealthy / 北极星空态） */
  async expectDeploymentInstanceListReadonlyVisible() {
    const table = this.getInstanceTable();
    await expect(table).toBeVisible({ timeout: 10000 });
    for (const column of [
      '实例',
      '镜像 Tag',
      'Pod IP',
      'Node IP',
      '实例状态',
      '健康状态',
      '北极星状态',
      'Restart',
      'Age',
      '操作',
    ]) {
      await expect(table.getByText(column, { exact: true }).first()).toBeVisible();
    }
    await expect(this.getVisibleInstanceRows()).toHaveCount(3, { timeout: 10000 });
    await expect(table.getByText('e2e-instance-primary', { exact: true }).first()).toBeVisible();
    await expect(table.getByText('Running', { exact: true }).first()).toBeVisible();
    await expect(table.getByText('Healthy', { exact: true }).first()).toBeVisible();
    // 差异化数据：行 2 健康状态异常展示 UnHealthy；行 3 未注册北极星展示 --
    await expect(table.getByText('UnHealthy', { exact: true }).first()).toBeVisible();
    await expect(table.getByText('--', { exact: true }).first()).toBeVisible();
    for (const action of ['灰度', '日志', '监控', '登录', '调整权重']) {
      await expect(table.getByRole('button', { name: action }).first()).toBeVisible();
    }
    await expect.poll(() => this.instanceListRequestCount, { timeout: 10000 }).toBeGreaterThanOrEqual(1);
    await expect.poll(() => this.instanceWatchRequestCount, { timeout: 10000 }).toBeGreaterThanOrEqual(1);
  }

  /** 断言已部署特性环境销毁入口展示移除部署引导 */
  async expectFeatureEnvDestroyGuardVisible() {
    const deployedRow = this.getFeatureEnvRow('e2e-feature-deployed');
    const destroyButton = deployedRow.getByRole('button', { name: '销毁' });
    await expect(destroyButton).toBeDisabled({ timeout: 10000 });
    await destroyButton.hover({ force: true });
    await expect(this.page.getByText('无法销毁特性环境', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(this.page.getByText('该特性环境有部署实例，请先移除部署后再销毁环境')).toBeVisible();
    await expect(this.page.getByText('立即移除部署')).toBeVisible();
  }

  /** 断言应用关联特性环境侧栏展示列表与操作保护 */
  async expectFeatureEnvSidesliderVisible() {
    await expect(this.page.getByText('应用关联的特性环境', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(this.page.getByText(`共 ${this.featureEnvs.length} 个`, { exact: true })).toBeVisible();
    for (const column of ['环境展示名称', '部署状态', '来源环境', '命名空间', '创建时间', '操作']) {
      await expect(this.page.getByText(column, { exact: true }).last()).toBeVisible();
    }
    const waitingRow = this.getFeatureEnvRow('e2e-feature-waiting');
    const deployedRow = this.getFeatureEnvRow('e2e-feature-deployed');
    await expect(waitingRow).toBeVisible();
    await expect(waitingRow.getByText('待部署', { exact: true })).toBeVisible();
    await expect(waitingRow.getByRole('button', { name: '部署', exact: true })).toBeVisible();
    await expect(deployedRow).toBeVisible();
    await expect(deployedRow.getByText('已部署', { exact: true })).toBeVisible();
  }

  /** 断言联邦环境实例列表走只读展示，行内与批量操作栏的灰度入口均禁用 */
  async expectFederationDeploymentInstanceListVisible() {
    const table = this.getInstanceTable();
    await expect(table).toBeVisible({ timeout: 10000 });
    await expect(this.getVisibleInstanceRows()).toHaveCount(3, { timeout: 10000 });
    await expect.poll(() => this.instanceListRequestCount, { timeout: 10000 }).toBeGreaterThanOrEqual(1);
    expect(this.instanceWatchRequestCount).toBe(0);
    await expect(this.getInstanceTable().getByRole('button', { name: '灰度' }).first()).toBeDisabled();
    await expect(this.getInstanceBatchToolbar().getByRole('button', { name: '灰度', exact: true })).toBeDisabled();
  }

  /** 断言管理命令侧栏展示已选实例和 mock 命令 */
  async expectInstanceAdminCommandSidesliderVisible() {
    await expect(this.page.getByText('管理命令', { exact: true }).last()).toBeVisible({ timeout: 10000 });
    await expect(this.page.getByText(/已选\s*1\s*个实例/).last()).toBeVisible({ timeout: 10000 });
    await expect(this.page.getByText('/cmd/e2e-health', { exact: true })).toBeVisible({ timeout: 10000 });
  }

  /** 断言灰度弹窗可见 */
  async expectInstanceGrayDialogVisible() {
    await expect(this.getDialog().getByText('灰度实例', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(this.getDialog().getByText('镜像 Tag', { exact: true })).toBeVisible();
  }

  /** 断言实例日志侧栏展示 mock 日志 */
  async expectInstanceLogSidesliderVisible() {
    await expect(this.page.getByText('实例日志', { exact: true }).last()).toBeVisible({ timeout: 10000 });
    await expect(this.page.getByText('e2e instance log line')).toBeVisible({ timeout: 10000 });
  }

  /** 断言调整权重弹窗可见 */
  async expectInstanceWeightDialogVisible() {
    await expect(this.getDialog().getByText('调整权重', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(this.getDialog().getByText(/权重/).first()).toBeVisible();
  }

  /** 获取实例列表主表的行，排除固定选择列和操作列复制出的 VXE 行 */
  instanceRows() {
    return this.page.locator('.instance-table .vxe-table--main-wrapper .vxe-table--body tr');
  }

  /** 打开应用关联特性环境侧栏 */
  async openFeatureEnvSideslider() {
    await this.page.locator('.feature-env-entry').click();
    // 最小等待：侧栏标题出现即视为打开完成；完整断言交由 expectFeatureEnvSidesliderVisible
    await expect(this.page.getByText('应用关联的特性环境', { exact: true })).toBeVisible({ timeout: 10000 });
  }

  /** 打开实例列表首行灰度弹窗 */
  async openFirstInstanceGrayDialog() {
    await this.getInstanceTable().getByRole('button', { name: '灰度' }).first().click();
    await expect(this.getDialog().getByText('灰度实例', { exact: true })).toBeVisible({ timeout: 10000 });
  }

  /** 打开实例列表首行日志侧栏 */
  async openFirstInstanceLogSideslider() {
    await this.getInstanceTable().getByRole('button', { name: '日志' }).first().click();
    await expect(this.page.getByText('实例日志', { exact: true }).last()).toBeVisible({ timeout: 10000 });
  }

  /** 打开实例列表首行调整权重弹窗 */
  async openFirstInstanceWeightDialog() {
    await this.getInstanceTable().getByRole('button', { name: '调整权重' }).first().click();
    await expect(this.getDialog().getByText('调整权重', { exact: true })).toBeVisible({ timeout: 10000 });
  }

  /** 打开选中实例的删除确认弹窗 */
  async openSelectedDeploymentInstanceDeleteDialog() {
    await this.getInstanceBatchToolbar().getByRole('button', { name: '删除', exact: true }).click();
    await this.expectDeploymentInstanceDeleteDialogVisible();
  }

  /** 打开选中实例的管理命令侧栏，并展开命令选择下拉（使命令选项渲染供断言） */
  async openSelectedInstanceAdminCommandSideslider() {
    await this.getInstanceBatchToolbar().getByRole('button', { name: '管理命令', exact: true }).click();
    await expect(this.page.getByText(/已选\s*1\s*个实例/).last()).toBeVisible({ timeout: 10000 });
    await this.getManagementCommandSelect().click();
  }

  /** 选中实例列表首行 */
  async selectFirstDeploymentInstance() {
    const firstRow = this.page.locator('.instance-table .vxe-table--fixed-left-wrapper .vxe-body--row').first();
    await firstRow.locator('.bk-checkbox').first().click();
    // 最小等待：勾选后跨页选择信息条出现即视为选中生效；完整断言交由 expectDeploymentInstanceBatchToolbarVisible
    await this.getInstanceTable()
      .getByText(/已选择|已跨页全选/)
      .first()
      .waitFor({ state: 'visible', timeout: 10000 });
  }

  /**
   * 配置部署实例列表相关接口 mock。
   *
   * 普通环境：实例列表响应带 resourceVersion，并以 SSE ENDED 事件终止 watch（模拟一次正常续流周期）。
   * 联邦环境（federation=true）：环境为 federation 集群，实例响应不带 resourceVersion 且不注册 watch
   * 路由 —— 前端联邦分支只做全量轮询 List、不建 Watch（use-instance-list-watch.ts poll 分支）。
   */
  async setupDeploymentInstanceMock({ federation = false }: { federation?: boolean } = {}) {
    this.instanceListRequestCount = 0;
    this.instanceWatchRequestCount = 0;
    await this.routeAppEnvs({ federation });
    await this.routeDeployLatestStatus();
    await this.routeEffectiveResources();
    await this.routeFeatureEnvs();
    await this.routeInstanceList({ federation });
    // 联邦环境也注册 Watch 路由用于记录误请求；断言会确保该分支不实际建立 Watch。
    await this.routeInstanceWatch();
    await this.routeInstanceLogs();
    await this.routeAdminCommands();
    await this.routeImages();
  }

  /** 等待 SSE 推送后的实例数满足期望（达成返回 true，超时返回 false） */
  async waitForInstanceCount(expected: number, { timeout = 180000 } = {}) {
    try {
      await expect.poll(() => this.instanceRows().count(), { timeout }).toBeGreaterThanOrEqual(expected);
      return true;
    } catch {
      return false;
    }
  }

  /** 等待 SSE 推送后实例数精确匹配且每行状态均为 Running/Healthy */
  async waitForInstanceReadyCount(expected: number, { timeout = 180000 } = {}) {
    try {
      await expect
        .poll(
          async () => {
            const rowTexts = await this.visibleInstanceRowTexts();
            return (
              rowTexts.length === expected &&
              rowTexts.every(text => text.includes('Running') && text.includes('Healthy'))
            );
          },
          { timeout },
        )
        .toBe(true);
      return true;
    } catch {
      return false;
    }
  }
}
