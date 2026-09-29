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
import { type Locator, type Route, expect } from '@playwright/test';

import AppDetailBase from './app-detail-base.page';

type CapturedRequest = {
  body: Record<string, unknown>;
  url: string;
};

type PolarisConfig = {
  depSvcInstID?: string;
  enableHealthCheck?: boolean;
  enableWeightFactor?: boolean;
  envDynamicWeights?: Record<string, boolean>;
  envStates?: Record<string, Record<string, unknown>>;
  envWeights?: Record<string, number>;
  instanceKey?: string;
  name?: string;
  operator?: string;
  polarisName?: string;
  polarisNamespace?: string;
  polarisToken?: string;
  registerMode?: string;
  scopeEnvNames?: string[];
  serviceLabels?: Record<string, string>;
  servicePort?: number;
};

/**
 * 北极星配置域 Page Object。
 *
 * 覆盖配置列表、添加、编辑、删除，以及关联环境侧栏的环境统计展示。
 */
export default class PolarisPage extends AppDetailBase {
  private createRequest?: CapturedRequest;

  private deleteRequestUrl = '';

  private envStatsRequestCount = 0;

  private listRequestCount = 0;

  private patchRequest?: CapturedRequest;

  private readonly primaryConfigName = 'e2e-polaris-primary';

  private readonly primaryPolarisName = 'e2e.service.primary';

  private state: {
    configs: PolarisConfig[];
  } = {
    configs: [],
  };

  private assertRequestBody(request: CapturedRequest | undefined, action: string) {
    if (!request) {
      throw new Error(`未捕获到${action}请求`);
    }
    return request.body;
  }

  private envList() {
    return [
      { displayName: '开发环境', id: 'env-dev', name: 'dev', status: 'Ready', type: 'development' },
      { displayName: '测试环境', id: 'env-test', name: 'test', status: 'Ready', type: 'test' },
      { displayName: '预发布环境', id: 'env-staging', name: 'staging', status: 'Ready', type: 'staging' },
      { displayName: '生产环境', id: 'env-prod', name: 'prod', status: 'Ready', type: 'production' },
    ];
  }

  private async fulfillPolarisRoute(route: Route) {
    const request = route.request();
    const url = new URL(request.url());
    const method = request.method();
    const pathname = url.pathname;

    if (method === 'GET' && pathname.endsWith(`/deps/polaris-configs/${this.primaryConfigName}/env-instance-stats`)) {
      this.envStatsRequestCount += 1;
      await this.fulfillJson(route, {
        data: {
          envStats: {
            prod: {
              healthyInstanceCount: 2,
              healthyInstanceWeight: 160,
              totalInstanceCount: 2,
              weightOverriddenInstanceCount: 1,
            },
            test: {
              healthyInstanceCount: 3,
              healthyInstanceWeight: 300,
              totalInstanceCount: 3,
              weightOverriddenInstanceCount: 0,
            },
          },
          totalHealthyInstanceCount: 5,
          totalHealthyInstanceWeight: 460,
        },
        status: 0,
      });
      return;
    }

    if (method === 'GET' && pathname.endsWith('/deps/polaris-configs')) {
      this.listRequestCount += 1;
      await this.fulfillJson(route, { data: this.state.configs, status: 0 });
      return;
    }

    if (method === 'POST' && pathname.endsWith('/deps/polaris-configs')) {
      const body = request.postDataJSON() as Record<string, unknown>;
      this.createRequest = { body, url: request.url() };
      this.state.configs.push({
        ...body,
        name: 'e2e-polaris-created',
        polarisToken: String(body.polarisToken || ''),
      });
      await this.fulfillJson(route, { data: { name: 'e2e-polaris-created' }, status: 0 });
      return;
    }

    if (method === 'PATCH' && pathname.endsWith(`/deps/polaris-configs/${this.primaryConfigName}`)) {
      const body = request.postDataJSON() as Record<string, unknown>;
      this.patchRequest = { body, url: request.url() };
      const updatedConfig = {
        ...this.state.configs[0],
        ...body,
        name: this.primaryConfigName,
        polarisName: this.primaryPolarisName,
      };
      this.state.configs[0] = updatedConfig;
      await this.fulfillJson(route, { data: updatedConfig, status: 0 });
      return;
    }

    if (method === 'DELETE' && pathname.endsWith(`/deps/polaris-configs/${this.primaryConfigName}`)) {
      this.deleteRequestUrl = request.url();
      this.state.configs = this.state.configs.filter(config => config.name !== this.primaryConfigName);
      await this.fulfillJson(route, { data: {}, status: 0 });
      return;
    }

    await route.fallback();
  }

  private async getPolarisConfigEditor() {
    return this.preferTestId('polaris-config-editor', this.getSideslider());
  }

  private async getPolarisRowActionButton(actionName: string) {
    const rowIndex = await this.getPolarisTable()
      .locator('.vxe-body--row:visible')
      .evaluateAll(
        (rows, polarisName) => rows.findIndex(row => row.textContent?.includes(String(polarisName))),
        this.primaryPolarisName,
      );
    expect(rowIndex, `未找到北极星配置行：${this.primaryPolarisName}`).toBeGreaterThanOrEqual(0);
    return this.page.getByRole('button', { name: actionName }).nth(rowIndex);
  }

  private getPolarisTable() {
    return this.page.locator('.vxe-table').first();
  }

  private getSideFormItem(label: string) {
    return this.getSideslider().locator('.bk-form-item').filter({ hasText: label }).first();
  }

  private getVisibleTableRowByText(scope: Locator, rowText: string) {
    return scope.locator('.vxe-body--row:visible').filter({ hasText: rowText }).first();
  }

  private getVxeCell(row: Locator, cellIndex: number) {
    return row.locator('.vxe-body--column').nth(cellIndex);
  }

  private initialPolarisConfigs(): PolarisConfig[] {
    return [
      {
        enableHealthCheck: true,
        enableWeightFactor: true,
        envDynamicWeights: {
          test: true,
        },
        envStates: {
          prod: {
            appliedFields: {
              instanceKey: 'primary_ins',
              polarisToken: '****',
              servicePort: 8080,
            },
            status: 'deployed',
          },
          test: {
            appliedFields: {
              instanceKey: 'primary_ins',
              polarisToken: '****',
              servicePort: 8080,
            },
            status: 'deployed',
          },
        },
        envWeights: {
          prod: 80,
          staging: 60,
          test: 100,
        },
        instanceKey: 'primary_ins',
        name: this.primaryConfigName,
        polarisName: this.primaryPolarisName,
        polarisNamespace: 'Test',
        polarisToken: 'primary-token',
        registerMode: 'immediate',
        scopeEnvNames: ['test', 'staging'],
        serviceLabels: {
          region: 'ap-guangzhou',
          stage: 'primary',
        },
        servicePort: 8080,
      },
      {
        enableHealthCheck: false,
        enableWeightFactor: false,
        instanceKey: 'backup_ins',
        name: 'e2e-polaris-backup',
        polarisName: 'e2e.service.backup',
        polarisNamespace: 'Production',
        polarisToken: 'backup-token',
        registerMode: 'on_deploy',
        scopeEnvNames: [],
        serviceLabels: {},
        servicePort: 9090,
      },
    ];
  }

  private async preferTestId(id: string, fallback: Locator) {
    return (await this.testId(id).count()) > 0 ? this.testId(id) : fallback;
  }

  private async routeDeployStatuses() {
    await this.page.route('**/apps/*/deploy-statuses**', async route => {
      if (route.request().method() !== 'GET') {
        await route.fallback();
        return;
      }
      await this.fulfillJson(route, { data: [], status: 0 });
    });
  }

  private async routeEnvList() {
    await this.page.route('**/apps/*/envs**', async route => {
      const url = new URL(route.request().url());
      if (route.request().method() !== 'GET' || !url.pathname.endsWith('/envs')) {
        await route.fallback();
        return;
      }
      await this.fulfillJson(route, { data: this.envList(), status: 0 });
    });
  }

  private async routePolarisConfig() {
    await this.page.route('**/apps/*/deps/polaris-configs**', route => this.fulfillPolarisRoute(route));
  }

  private async setFieldValue(label: string, value: string) {
    const formItem = this.getSideFormItem(label);
    const input = formItem.locator('input').first();
    await input.fill(value);
  }

  /** 新增平台自动生成北极星配置，并等待创建请求完成 */
  async createAutoGeneratedPolarisConfig() {
    await this.setFieldValue('实例名称', 'auto_generated_ins');
    await this.setFieldValue('服务端口', '7171');
    const slider = this.getSideslider();
    await slider.getByTitle('测试环境').click();
    await this.getSideFormItem('北极星环境类型').getByText('Test', { exact: true }).click();
    await this.setFieldValue('北极星服务名', 'e2e.service.auto-generated');
    await this.getSideFormItem('生效方式').getByText('立即生效', { exact: true }).click();
    await this.getSideFormItem('权重因子').locator('.bk-switcher').click();

    const responsePromise = this.page.waitForResponse(
      response => response.request().method() === 'POST' && response.url().includes('/deps/polaris-configs'),
      { timeout: 30000 },
    );
    await slider.getByRole('button', { name: '确定' }).click();
    await this.assertApiResponseOk(await responsePromise, '创建平台自动生成北极星配置');
    await expect(this.page.getByText('e2e.service.auto-generated', { exact: true })).toBeVisible({ timeout: 10000 });
  }

  /** 新增从现有引入的北极星配置，并等待创建请求完成 */
  async createImportedPolarisConfig() {
    await this.openCreatePolarisImportSideslider();
    await this.submitImportedPolarisConfig();
  }

  /** 删除首条北极星配置，并等待删除请求完成 */
  async deletePrimaryPolarisConfig() {
    await (await this.getPolarisRowActionButton('删除')).click();
    const dialog = this.page.locator('.bk-modal-wrapper:visible').last();
    await expect(dialog.getByText('确认删除该北极星配置？', { exact: true })).toBeVisible({ timeout: 10000 });

    const responsePromise = this.page.waitForResponse(
      response => response.request().method() === 'DELETE' && response.url().includes('/deps/polaris-configs/'),
      { timeout: 30000 },
    );
    await dialog.getByRole('button', { name: '删除' }).click();
    await this.assertApiResponseOk(await responsePromise, '删除北极星配置');
    await expect(this.page.getByText(this.primaryPolarisName, { exact: true })).toHaveCount(0, { timeout: 10000 });
  }
  /** 编辑首条北极星配置，并等待更新请求完成 */
  async editPrimaryPolarisConfig() {
    await (await this.getPolarisRowActionButton('编辑')).click();
    const slider = this.getSideslider();
    await expect(slider.getByText('编辑北极星', { exact: true })).toBeVisible({ timeout: 10000 });

    await this.setFieldValue('实例名称', 'primary_ins_updated');
    await this.setFieldValue('服务端口', '8181');
    await this.setFieldValue('北极星Token', 'updated-token');
    await this.getSideFormItem('健康检查').locator('.bk-switcher').click();
    await this.getSideFormItem('服务标签').getByText('文本模式', { exact: true }).click();
    await this.getSideFormItem('服务标签').locator('textarea').fill('stage=updated\nlane=gray');

    const responsePromise = this.page.waitForResponse(
      response =>
        response.request().method() === 'PATCH' &&
        response.url().includes(`/deps/polaris-configs/${this.primaryConfigName}`),
      { timeout: 30000 },
    );
    await slider.getByRole('button', { name: '确定' }).click();
    await this.assertApiResponseOk(await responsePromise, '编辑北极星配置');
    await expect(this.page.getByText('8181', { exact: true }).first()).toBeVisible({ timeout: 10000 });
  }

  /** 断言关联环境侧栏展示已部署、未部署环境及北极星实例统计 */
  async expectAssociatedEnvSidesliderVisible() {
    await expect.poll(() => this.envStatsRequestCount, { timeout: 10000 }).toBeGreaterThanOrEqual(1);
    const slider = this.getSideslider();
    await expect(slider.getByText('关联环境', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(slider.getByText(this.primaryPolarisName, { exact: true })).toBeVisible();
    await expect(slider.getByText('已部署环境', { exact: false })).toBeVisible();
    await expect(slider.getByText('未部署环境', { exact: false })).toBeVisible();
    await expect(slider.getByText('测试环境', { exact: true })).toBeVisible();
    await expect(slider.getByText('生产环境', { exact: true })).toBeVisible();
    await expect(slider.getByText('预发布环境', { exact: true })).toBeVisible();

    const deployedSection = slider.locator('section').filter({ hasText: '已部署环境' }).first();
    const undeployedSection = slider.locator('section').filter({ hasText: '未部署环境' }).first();
    const testRow = this.getVisibleTableRowByText(deployedSection, '测试环境');
    await expect(this.getVxeCell(testRow, 1)).toContainText(/^100/);
    await expect(this.getVxeCell(testRow, 2)).toHaveText('3');
    await expect(this.getVxeCell(testRow, 3)).toContainText('300');
    await expect(this.getVxeCell(testRow, 3)).toContainText('65.22%');

    const stagingRow = this.getVisibleTableRowByText(undeployedSection, '预发布环境');
    await expect(this.getVxeCell(stagingRow, 1)).toHaveText('60');
  }

  /** 断言平台自动生成新增请求不传从现有引入专属 Token */
  async expectAutoGeneratedCreateRequestParams() {
    const body = this.assertRequestBody(this.createRequest, '创建平台自动生成北极星配置');
    expect(body).toEqual(
      expect.objectContaining({
        createNewService: true,
        enableWeightFactor: true,
        instanceKey: 'auto_generated_ins',
        polarisName: 'e2e.service.auto-generated',
        polarisNamespace: 'Test',
        registerMode: 'immediate',
        servicePort: 7171,
      }),
    );
    expect(body).not.toHaveProperty('polarisToken');
    expect(body.operator).toEqual(expect.stringMatching(/\S/));
  }

  /** 断言新增侧边栏展示平台自动生成模式字段 */
  async expectCreatePolarisAutoMode() {
    const editor = await this.getPolarisConfigEditor();
    const typeOptions = await this.preferTestId('polaris-config-type-options', this.getSideFormItem('类型'));
    const ownerField = await this.preferTestId('polaris-config-auto-owner-field', this.getSideFormItem('北极星负责人'));
    const weightFactorField = await this.preferTestId(
      'polaris-config-auto-weight-factor-field',
      this.getSideFormItem('权重因子'),
    );

    await expect(typeOptions.getByRole('radio', { name: '平台自动生成' })).toBeChecked();
    await expect(editor.getByText('北极星Token', { exact: true })).toHaveCount(0);
    await expect(ownerField).toBeVisible();
    await expect(weightFactorField).toBeVisible();
  }
  /** 断言新增侧边栏展示从现有引入模式字段 */
  async expectCreatePolarisImportMode() {
    const editor = await this.getPolarisConfigEditor();
    const typeOptions = await this.preferTestId('polaris-config-type-options', this.getSideFormItem('类型'));
    const tokenField = await this.preferTestId(
      'polaris-config-import-token-field',
      this.getSideFormItem('北极星Token'),
    );

    await expect(typeOptions.getByRole('radio', { name: '从现有引入' })).toBeChecked();
    await expect(tokenField).toBeVisible();
    await expect(editor.getByText('北极星负责人', { exact: true })).toHaveCount(0);
    await expect(editor.getByText('权重因子', { exact: true })).toHaveCount(0);
  }
  /** 断言新增北极星配置请求参数 */
  async expectCreateRequestParams() {
    const body = this.assertRequestBody(this.createRequest, '创建北极星配置');
    expect(body).toEqual(
      expect.objectContaining({
        createNewService: false,
        enableHealthCheck: true,
        instanceKey: 'created_ins',
        polarisName: 'e2e.service.created',
        polarisNamespace: 'Production',
        polarisToken: 'created-token',
        registerMode: 'on_deploy',
        serviceLabels: {
          lane: 'stable',
          stage: 'created',
        },
        servicePort: 7070,
      }),
    );
    expect(body).not.toHaveProperty('enableWeightFactor');
    expect(body).not.toHaveProperty('operator');
    expect([...(body.scopeEnvNames as string[])].sort()).toEqual(['prod', 'test']);
  }

  /** 断言删除北极星配置请求命中目标配置 */
  async expectDeleteRequestTarget() {
    expect(this.deleteRequestUrl).toContain(`/deps/polaris-configs/${this.primaryConfigName}`);
  }
  /** 断言编辑北极星配置请求参数 */
  async expectPatchRequestParams() {
    const body = this.assertRequestBody(this.patchRequest, '编辑北极星配置');
    expect(this.patchRequest?.url).toContain(`/deps/polaris-configs/${this.primaryConfigName}`);
    expect(body).toEqual(
      expect.objectContaining({
        enableHealthCheck: false,
        instanceKey: 'primary_ins_updated',
        polarisToken: 'updated-token',
        scopeEnvNames: ['test', 'staging'],
        serviceLabels: {
          lane: 'gray',
          stage: 'updated',
        },
        servicePort: 8181,
      }),
    );
    expect(body).not.toHaveProperty('enableWeightFactor');
    expect(body).not.toHaveProperty('operator');
  }

  /** 断言北极星配置页列表已加载 */
  async expectPolarisConfigListVisible() {
    await expect.poll(() => this.listRequestCount, { timeout: 10000 }).toBeGreaterThanOrEqual(1);
    const table = this.getPolarisTable();
    await expect(this.page.getByText('添加北极星', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(table).toBeVisible();
    for (const column of ['服务名', '环境类型', '实例名称', '服务端口', '可用环境', '健康检查']) {
      await expect(table.getByText(column, { exact: true }).first()).toBeVisible();
    }
    await expect(this.page.getByText('操作', { exact: true }).last()).toBeVisible();
    await expect(table.getByText(this.primaryPolarisName, { exact: true }).first()).toBeVisible({ timeout: 10000 });
    await expect(this.page.getByRole('button', { name: '关联环境' }).first()).toBeVisible();
    await expect(this.page.getByRole('button', { name: '编辑' }).first()).toBeVisible();
    await expect(this.page.getByRole('button', { name: '删除' }).first()).toBeVisible();
  }

  /** 断言编辑侧边栏保留从现有引入模式，并限制不可编辑字段 */
  async expectPrimaryImportedPolarisEditMode() {
    const typeOptions = await this.preferTestId('polaris-config-type-options', this.getSideFormItem('类型'));
    const namespaceOptions = await this.preferTestId(
      'polaris-config-namespace-options',
      this.getSideFormItem('北极星环境类型'),
    );
    const serviceNameField = await this.preferTestId(
      'polaris-config-service-name-field',
      this.getSideFormItem('北极星服务名'),
    );
    const tokenField = await this.preferTestId(
      'polaris-config-import-token-field',
      this.getSideFormItem('北极星Token'),
    );
    const importRadio = typeOptions.getByRole('radio', { name: '从现有引入' });
    const namespaceRadio = namespaceOptions.getByRole('radio', { name: 'Test' });
    const serviceNameInput = serviceNameField.locator('input');
    const tokenInput = tokenField.locator('input');

    await expect(importRadio).toBeChecked();
    await expect(importRadio).toBeDisabled();
    await expect(namespaceRadio).toBeChecked();
    await expect(namespaceRadio).toBeDisabled();
    await expect(serviceNameInput).toHaveValue(this.primaryPolarisName);
    await expect(serviceNameInput).toHaveAttribute('readonly', '');
    await expect(tokenInput).toHaveValue('primary-token');
    await expect(tokenInput).toBeEditable();
  }

  /** 进入北极星配置页（二级菜单 key: polaris） */
  async gotoPolarisConfig() {
    await this.gotoMenu('polaris');
    await this.expectPolarisConfigListVisible();
  }

  /** 打开新增侧边栏并选择平台自动生成模式 */
  async openCreatePolarisAutoSideslider() {
    const addButton = await this.preferTestId(
      'polaris-config-add-button',
      this.page.getByRole('button', { name: '添加北极星' }),
    );
    await addButton.click();
    const editor = await this.getPolarisConfigEditor();
    await expect(editor).toBeVisible({ timeout: 10000 });
    const typeOptions = await this.preferTestId('polaris-config-type-options', this.getSideFormItem('类型'));
    await typeOptions.getByText('平台自动生成', { exact: true }).click();
  }

  /** 打开新增侧边栏并选择从现有引入模式 */
  async openCreatePolarisImportSideslider() {
    const addButton = await this.preferTestId(
      'polaris-config-add-button',
      this.page.getByRole('button', { name: '添加北极星' }),
    );
    await addButton.click();
    const editor = await this.getPolarisConfigEditor();
    await expect(editor).toBeVisible({ timeout: 10000 });
    const typeOptions = await this.preferTestId('polaris-config-type-options', this.getSideFormItem('类型'));
    await typeOptions.getByText('从现有引入', { exact: true }).click();
  }

  /** 打开首条北极星配置的关联环境侧栏 */
  async openPrimaryAssociatedEnvSideslider() {
    await (await this.getPolarisRowActionButton('关联环境')).click();
  }
  /** 打开首条从现有引入配置的编辑侧边栏 */
  async openPrimaryImportedPolarisEditor() {
    const editButton = await this.preferTestId(
      `polaris-config-edit-${this.primaryConfigName}`,
      await this.getPolarisRowActionButton('编辑'),
    );
    await editButton.click();
    await expect(await this.getPolarisConfigEditor()).toBeVisible({ timeout: 10000 });
  }

  /** 配置北极星页面接口 mock */
  async setupPolarisConfigMock() {
    this.createRequest = undefined;
    this.deleteRequestUrl = '';
    this.envStatsRequestCount = 0;
    this.listRequestCount = 0;
    this.patchRequest = undefined;
    this.state = {
      configs: this.initialPolarisConfigs(),
    };
    await this.routeEnvList();
    await this.routeDeployStatuses();
    await this.routePolarisConfig();
  }

  /** 提交已打开的从现有引入北极星配置 */
  async submitImportedPolarisConfig() {
    const slider = this.getSideslider();
    await expect(slider.getByText('添加北极星', { exact: true })).toBeVisible({ timeout: 10000 });

    await this.setFieldValue('实例名称', 'created_ins');
    await this.setFieldValue('服务端口', '7070');
    await slider.getByTitle('测试环境').click();
    await slider.getByTitle('生产环境').click();
    await this.getSideFormItem('类型').getByText('从现有引入', { exact: true }).click();
    await this.getSideFormItem('北极星环境类型').getByText('Production', { exact: true }).click();
    await this.setFieldValue('北极星服务名', 'e2e.service.created');
    await this.setFieldValue('北极星Token', 'created-token');
    await this.getSideFormItem('生效方式').getByText('下次部署生效', { exact: true }).click();
    await this.getSideFormItem('健康检查').locator('.bk-switcher').click();
    await this.getSideFormItem('服务标签').getByText('文本模式', { exact: true }).click();
    await this.getSideFormItem('服务标签').locator('textarea').fill('stage=created\nlane=stable');

    const responsePromise = this.page.waitForResponse(
      response => response.request().method() === 'POST' && response.url().includes('/deps/polaris-configs'),
      { timeout: 30000 },
    );
    await slider.getByRole('button', { name: '确定' }).click();
    await this.assertApiResponseOk(await responsePromise, '创建北极星配置');
    await expect(this.page.getByText('e2e.service.created', { exact: true })).toBeVisible({ timeout: 10000 });
  }
}
