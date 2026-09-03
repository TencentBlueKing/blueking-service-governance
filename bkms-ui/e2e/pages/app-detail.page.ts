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
import { type Locator, type Response, expect } from '@playwright/test';

import BasePage from './base.page';

type BuilderConfigSourceType = 'codeRepository' | 'imageRegistry' | 'pipeline';

type DeploymentTab = 'event' | 'history' | 'instance' | 'overview' | 'topo';

type HealthProbeHttpConfig = {
  failureThreshold?: string;
  initialDelaySeconds?: string;
  periodSeconds?: string;
  port: string;
  successThreshold?: string;
  timeoutSeconds?: string;
  url: string;
};

type LifecycleShellCommandConfig = {
  command: string;
  gracePeriod: string;
};

type MetadataSectionKey = 'annotations' | 'labels';

type UpdateStrategyConfig = {
  maxSurge: string;
  maxUnavailable: string;
};

/**
 * 应用详情页面 Page Object
 *
 * 应用详情路由格式：`/:space/app/:name/:type/:menuName`
 * 目前 E2E 覆盖的应用类型默认为 TRPC，后续如需 Helm / TAF 可通过 `type` 参数覆盖。
 *
 * 只封装原子 UI 操作；复杂业务流程（部署/扩缩容/移除部署）由 action 层组合完成。
 */
export default class AppDetailPage extends BasePage {
  private artifactSearchTag = '';

  private builderConfigInvalidFieldLabel = '';

  private builderConfigInvalidSubmitted = false;

  private builderConfigOriginalDefaultBranch = '';

  private builderConfigOriginalImageRegistryName = '';

  private async assertApiResponseOk(response: Response, action: string) {
    if (response.ok()) return;

    let body = '';
    try {
      body = await response.text();
    } catch {
      body = '<响应体读取失败>';
    }

    throw new Error(`${action}接口请求失败：${response.status()} ${response.url()}\n${body}`);
  }

  private assertAppConfigured() {
    if (!this.config.app) {
      throw new Error(
        'testConfig.app 未配置：请配置 BKMS_TEST_DEFAULT_APP / BKMS_TEST_TRPC_APP / BKMS_TEST_HELM_APP，' +
          '或在当前空间创建类型匹配且名称以 e2e- 开头的测试应用，也可在 Scenario 打 @app:<name> 标签指定应用名称',
      );
    }
  }

  private async assertCurrentDeployReadyForOperation(action: string) {
    if (!(await this.isDeployed(1000))) {
      throw new Error(`当前环境尚未部署应用，无法执行${action}；请先确保 TC-01 部署应用成功后再运行该用例`);
    }
  }

  private async closeScaleSidesliderIfOpen() {
    if (
      !(await this.getScaleSidesliderTitle()
        .isVisible()
        .catch(() => false))
    )
      return;

    const slider = this.page
      .locator('.bk-sideslider-wrapper:visible, .bk-modal-wrapper:visible')
      .filter({ hasText: '扩缩容配置' })
      .first();
    await slider.getByRole('button', { name: '取消' }).click();
    await this.page
      .waitForSelector('.bk-sideslider .bk-modal-body', { state: 'hidden', timeout: 10000 })
      .catch(() => null);
    await this.waitForReady(500);
  }

  private async confirmScaleModeSwitch(confirmText: string) {
    const confirmButton = this.page.getByRole('button', { name: confirmText }).last();
    try {
      await confirmButton.waitFor({ state: 'visible', timeout: 1500 });
      await confirmButton.click();
      return true;
    } catch {
      return false;
    }
  }

  private async fillHealthProbeInput(section: Locator, label: string, value: string) {
    const input = this.getHealthProbeInput(section, label);
    await input.click({ clickCount: 3 });
    await input.fill(value);
  }

  private async fillLifecycleGracePeriod(value: string) {
    const input = this.getLifecycleSection().getByRole('spinbutton').last();
    await input.click({ clickCount: 3 });
    await input.fill(value);
  }

  private async fillLifecycleShellCommand(value: string) {
    const textarea = this.getLifecycleSection().locator('textarea').first();
    await textarea.click({ clickCount: 3 });
    await textarea.fill(value);
  }

  private async fillUpdateStrategyInput(label: string, value: string) {
    const input = this.getUpdateStrategyInput(label);
    await input.click({ clickCount: 3 });
    await input.fill(value);
  }

  private getAutoScaleCpuInput() {
    const slider = this.getScaleSideslider();
    return slider
      .locator('.bk-form-item:visible')
      .filter({ hasText: '触发条件' })
      .first()
      .getByRole('spinbutton')
      .first();
  }

  private getAutoScaleMaxInput() {
    const slider = this.getScaleSideslider();
    return slider.locator('.bk-form-item:visible').filter({ hasText: '最大实例数' }).getByRole('spinbutton').first();
  }

  private getAutoScaleMinInput() {
    const slider = this.getScaleSideslider();
    return slider.locator('.bk-form-item:visible').filter({ hasText: '最小实例数' }).getByRole('spinbutton').first();
  }

  private getBuilderConfigDefaultBranchInput() {
    return this.getBuilderConfigFormItem('默认分支').getByRole('textbox').first();
  }

  private getBuilderConfigFormItem(label: string) {
    return this.getBuilderConfigSideslider()
      .locator(
        `xpath=.//*[contains(@class, "bk-form-label") and normalize-space()="${label}"]/ancestor::*[contains(@class, "bk-form-item")][1]`,
      )
      .first();
  }

  private getBuilderConfigImageRegistryInput() {
    return this.getBuilderConfigFormItem('镜像仓库').getByRole('textbox').first();
  }

  private getBuilderConfigSideslider() {
    return this.page.locator('.bk-sideslider-wrapper, .bk-modal-wrapper').filter({ hasText: '编辑构建配置' }).first();
  }

  private getBuilderConfigSourceButton(name: '代码仓库' | '流水线' | '源码仓库' | '镜像仓库') {
    const slider = this.getBuilderConfigSideslider();
    return slider
      .locator('.bk-radio-button')
      .filter({ hasText: name })
      .or(slider.getByRole('button', { name }))
      .first();
  }

  private getBuilderConfigSourceFormItem() {
    return this.getBuilderConfigFormItem('来源').or(this.getBuilderConfigFormItem('镜像来源')).first();
  }

  private async getBuilderConfigSourceType(): Promise<BuilderConfigSourceType> {
    if (
      await this.getBuilderConfigFormItem('代码库')
        .isVisible()
        .catch(() => false)
    ) {
      return 'codeRepository';
    }
    if (
      await this.getBuilderConfigFormItem('镜像仓库')
        .isVisible()
        .catch(() => false)
    ) {
      return 'imageRegistry';
    }
    if (
      await this.getBuilderConfigSideslider()
        .getByText(/查看操作指引|请先选择流水线|流水线无参数配置|流水线参数配置/)
        .first()
        .isVisible()
        .catch(() => false)
    ) {
      return 'pipeline';
    }

    throw new Error('无法识别当前构建配置来源表单');
  }

  private getBuilderConfigSubmitButton() {
    const slider = this.getBuilderConfigSideslider();
    return slider
      .getByRole('button', { name: '保存' })
      .or(slider.getByRole('button', { name: '确定' }))
      .first();
  }

  private getDevModeSwitcher() {
    return this.getDevModeSection().locator('.bk-switcher').first();
  }

  private getHealthProbeInput(section: Locator, label: string) {
    const formItem = section.locator('.bk-form-item').filter({ hasText: label }).first();
    return formItem.getByRole('spinbutton').or(formItem.getByRole('textbox')).first();
  }

  private getLifecyclePreStopSwitcher() {
    return this.getLifecycleSection().locator('.bk-switcher').first();
  }

  private async getRemoveDeployConfirmName(dialog: Locator) {
    const text = (await dialog.textContent()) || '';
    return text.match(/环境名称[：:]\s*([^\s，。,.]+)/)?.[1]?.trim() || this.config.env;
  }

  private getScaleSideslider(): Locator {
    // bkui Sideslider 的 header/body/footer 不稳定落在同一 wrapper 下；打开后用唯一标题约束页面级查询。
    return this.page.locator('body');
  }

  private getScaleSidesliderTitle() {
    return this.page.getByText('扩缩容配置', { exact: true }).first();
  }

  private getUpdateStrategyInput(label: string) {
    return this.getUpdateStrategySection()
      .locator('.bk-form-item')
      .filter({ hasText: label })
      .getByRole('textbox')
      .first();
  }

  private async isBuilderConfigSourceDisabled(name: '代码仓库' | '流水线' | '源码仓库' | '镜像仓库') {
    const slider = this.getBuilderConfigSideslider();
    const radio = slider.getByRole('radio', { name }).first();
    if (await radio.isVisible().catch(() => false)) {
      return radio.isDisabled();
    }

    const button = slider.getByRole('button', { name }).first();
    if (await button.isVisible().catch(() => false)) {
      return button.isDisabled();
    }

    return false;
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

  private waitForBuilderConfigSaveResponse() {
    return this.page.waitForResponse(
      response => response.request().method() === 'PUT' && response.url().includes('/build-configs'),
      { timeout: 30000 },
    );
  }

  private waitForScaleResponse(path: string, method: string) {
    return this.page.waitForResponse(
      response => response.request().method() === method && response.url().includes(path),
      { timeout: 30000 },
    );
  }

  private async waitForSidesliderSettled() {
    await this.page
      .waitForSelector('.bk-sideslider .bk-modal-body', { state: 'hidden', timeout: 5000 })
      .catch(() => null);
    await this.waitForReady(1000);
  }

  /** 取消构建配置编辑并处理离开确认 */
  async cancelBuilderConfigEdit() {
    await this.getBuilderConfigSideslider().getByRole('button', { name: '取消' }).click();
    const leaveButton = this.page.getByRole('button', { name: '离开' }).last();
    if (
      await leaveButton.waitFor({ state: 'visible', timeout: 3000 }).then(
        () => true,
        () => false,
      )
    ) {
      await leaveButton.click();
    }
    await this.expectBuilderConfigSidesliderClosed();
  }

  /** 清空环境变量搜索框 */
  async clearEnvVarSearch() {
    const input = this.page.getByPlaceholder('搜索变量名、变量值、描述');
    await input.fill('');
  }

  /** 清空侧栏环境变量搜索框 */
  async clearSliderEnvVarSearch() {
    const slider = this.getSideslider();
    const input = slider.getByPlaceholder('搜索变量名、变量值、描述');
    await input.fill('');
  }

  /** 点击开发模式确认弹窗的「取消」按钮 */
  async clickDevModeCancel() {
    await this.getDialog().getByRole('button', { name: '取消' }).click();
    await this.waitForDialogClosed();
  }

  /** 点击开发模式确认弹窗的确认按钮，并等待接口成功 */
  async clickDevModeConfirm(action: 'disable' | 'enable') {
    const responsePromise = this.page.waitForResponse(
      response =>
        response.request().method() === (action === 'enable' ? 'PUT' : 'DELETE') &&
        response.url().includes('/app-spec/dev-mode'),
      { timeout: 30000 },
    );
    const buttonName = action === 'enable' ? '确认开启' : '确认关闭';

    await this.getDialog().getByRole('button', { name: buttonName }).click();
    await this.assertApiResponseOk(await responsePromise, `${buttonName}开发模式`);
    await this.waitForDialogClosed();
    await this.waitForReady(1000);
  }

  /** 点击开发模式开关并等待确认弹窗出现 */
  async clickDevModeSwitch() {
    await this.getDevModeSwitcher().click();
    await this.waitForDialog();
  }

  /** 点击健康探针编辑态的「取消」按钮 */
  async clickHealthProbeCancel(label: string) {
    await this.getHealthProbeEditSection(label).getByRole('button', { name: '取消' }).click();
    await this.expectHealthProbeInViewMode(label);
  }

  /** 点击健康探针卡片的编辑入口 */
  async clickHealthProbeEdit(label: string) {
    const card = this.getHealthProbeViewSection(label);
    const configureButton = card.getByRole('button', { name: '立即配置' });

    if (await configureButton.isVisible().catch(() => false)) {
      await configureButton.click();
    } else {
      await card.locator('.card-header button').first().click();
    }

    await this.getHealthProbeEditSection(label).getByText('探测方法').waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 点击健康探针编辑态的「保存」按钮 */
  async clickHealthProbeSave(label: string) {
    await this.getHealthProbeEditSection(label).getByRole('button', { name: '保存' }).click();
    await this.page.waitForTimeout(500);
  }

  /** 点击健康探针编辑态的「保存」按钮，并等待保存接口成功 */
  async clickHealthProbeSaveAndWait(label: string) {
    const responsePromise = this.page.waitForResponse(
      response => response.request().method() === 'PUT' && response.url().includes('/app-spec/probe'),
      { timeout: 30000 },
    );

    await this.getHealthProbeEditSection(label).getByRole('button', { name: '保存' }).click();
    await this.assertApiResponseOk(await responsePromise, '保存健康探针');
    await this.expectHealthProbeInViewMode(label);
    await this.waitForReady(1000);
  }

  /** 点击生命周期编辑态的「取消」按钮 */
  async clickLifecycleCancel() {
    await this.getLifecycleSection().getByRole('button', { name: '取消' }).click();
    await this.expectLifecycleInViewMode();
  }

  /** 点击生命周期卡片的编辑入口 */
  async clickLifecycleEdit() {
    await this.getLifecycleSection().getByRole('button', { name: '编辑' }).click();
    await this.getLifecycleSection().getByRole('button', { name: '保存' }).waitFor({
      state: 'visible',
      timeout: 10000,
    });
  }

  /** 点击生命周期编辑态的「保存」按钮 */
  async clickLifecycleSave() {
    await this.getLifecycleSection().getByRole('button', { name: '保存' }).click();
    await this.page.waitForTimeout(500);
  }

  /** 点击生命周期编辑态的「保存」按钮，并等待保存接口成功 */
  async clickLifecycleSaveAndWait() {
    const responsePromise = this.page.waitForResponse(
      response =>
        response.request().method() === 'PUT' &&
        (response.url().includes('/app-spec/default-lifecycle') || response.url().includes('/app-spec/lifecycle')),
      { timeout: 30000 },
    );

    await this.getLifecycleSection().getByRole('button', { name: '保存' }).click();
    await this.assertApiResponseOk(await responsePromise, '保存生命周期');
    await this.expectLifecycleInViewMode();
    await this.waitForReady(1000);
  }

  /** 点击元数据配置卡片的「取消」按钮 */
  async clickMetadataCancel(label: string) {
    await this.getMetadataCard(label).getByRole('button', { name: '取消' }).click();
    await this.expectMetadataInViewMode(label);
  }

  /** 点击元数据配置卡片的编辑入口 */
  async clickMetadataEdit(label: string) {
    const card = this.getMetadataCard(label);
    const configureButton = card.getByRole('button', { name: '立即配置' });

    if (await configureButton.isVisible().catch(() => false)) {
      await configureButton.click();
    } else {
      await card.locator('button').first().click();
    }

    await card.getByText('表格模式').waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 点击元数据配置卡片的「保存」按钮 */
  async clickMetadataSave(label: string) {
    await this.getMetadataCard(label).getByRole('button', { name: '保存' }).click();
    await this.page.waitForTimeout(500);
  }

  /** 点击元数据配置卡片的「保存」按钮，并等待保存接口成功 */
  async clickMetadataSaveAndWait(label: string, sectionKey: MetadataSectionKey) {
    const responsePromise = this.page.waitForResponse(
      response => response.request().method() === 'PUT' && response.url().includes(`/app-spec/${sectionKey}`),
      { timeout: 30000 },
    );

    await this.getMetadataCard(label).getByRole('button', { name: '保存' }).click();
    await this.assertApiResponseOk(await responsePromise, `保存${label}`);
    await this.expectMetadataInViewMode(label);
    await this.waitForReady(1000);
  }

  /** 点击「立即部署」按钮（未部署空态时显示） */
  async clickQuicklyDeploy() {
    await this.clickButton('立即部署');
    await this.waitForSideslider();
  }

  // ─── 部署管理页：立即部署 Sideslider ────────────────────────────────

  /** 在更多菜单中点击「移除部署」，并等待确认弹窗出现 */
  async clickRemoveDeploy() {
    await this.page.getByRole('button', { name: '移除部署' }).click();
    await this.waitForDialog();
  }

  /** 点击资源规格卡片上的「编辑」按钮 */
  async clickResourceEdit() {
    await this.getResourceSection().getByRole('button', { name: '编辑' }).click();
    await this.page.waitForTimeout(500);
  }

  /** 点击资源规格卡片的「恢复默认配置」按钮 */
  async clickResourceResetToDefault() {
    await this.getResourceSection().getByRole('button', { name: '恢复默认配置' }).click();
  }

  /** 点击资源规格卡片的「保存」按钮 */
  async clickResourceSave() {
    await this.getResourceSection().getByRole('button', { name: '保存' }).click();
  }

  // ─── 部署管理页：扩缩容 PopConfirm ──────────────────────────────────

  /** 点击更新策略编辑态的「取消」按钮 */
  async clickUpdateStrategyCancel() {
    await this.getUpdateStrategySection().getByRole('button', { name: '取消' }).click();
    await this.expectUpdateStrategyInViewMode();
  }

  /** 点击更新策略卡片的编辑入口 */
  async clickUpdateStrategyEdit() {
    await this.getUpdateStrategySection().getByRole('button', { name: '编辑' }).click();
    await this.getUpdateStrategyInput('最大超出数量').waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 点击更新策略编辑态的「保存」按钮 */
  async clickUpdateStrategySave() {
    await this.getUpdateStrategySection().getByRole('button', { name: '保存' }).click();
    await this.page.waitForTimeout(500);
  }

  /** 点击更新策略编辑态的「保存」按钮，并等待保存接口成功 */
  async clickUpdateStrategySaveAndWait() {
    const responsePromise = this.page.waitForResponse(
      response => response.request().method() === 'PUT' && response.url().includes('/app-spec/update-strategy'),
      { timeout: 30000 },
    );

    await this.getUpdateStrategySection().getByRole('button', { name: '保存' }).click();
    await this.assertApiResponseOk(await responsePromise, '保存更新策略');
    await this.expectUpdateStrategyInViewMode();
    await this.waitForReady(1000);
  }

  /** 收起制品列表首行详情 */
  async collapseFirstArtifactRow() {
    const firstRow = this.page.locator('.artifact-table .vxe-table--body .vxe-body--row').first();
    await firstRow.click();
  }

  /** 在「移除部署」确认弹窗中输入当前环境名并点击「删除」 */
  async confirmRemoveDeploy() {
    const dialog = this.getDialog();
    const confirmName = await this.getRemoveDeployConfirmName(dialog);
    if (!confirmName) {
      throw new Error('无法从移除部署确认弹窗中解析待确认的环境名称');
    }

    const input = dialog.getByRole('textbox').first();
    await input.fill(confirmName);
    await expect(input).toHaveValue(confirmName, { timeout: 10000 });

    const deleteButton = dialog.getByRole('button', { name: '删除' });
    await expect(deleteButton).toBeEnabled({ timeout: 10000 });
    const responsePromise = this.page.waitForResponse(
      response =>
        response.request().method() === 'DELETE' &&
        (response.url().includes('/trpc-deploys') ||
          response.url().includes('/taf-deploys') ||
          response.url().includes('/helm-deploys')),
      { timeout: 30000 },
    );
    await deleteButton.click();
    await this.assertApiResponseOk(await responsePromise, '移除部署');
    await this.waitForDialogClosed();
  }

  /** 确保开发模式处于关闭状态，便于用例可重复运行 */
  async ensureDevModeDisabled() {
    await this.expectDevModeSectionVisible();
    await this.waitForReady(1000);
    if (!(await this.isDevModeEnabled())) return;

    await this.clickDevModeSwitch();
    await this.clickDevModeConfirm('disable');
    await this.expectDevModeDisabled();
  }

  // ─── 部署管理页：移除部署 ──────────────────────────────────────────

  /** 确保生命周期处于编辑态 */
  async ensureLifecycleEditMode() {
    const saveButton = this.getLifecycleSection().getByRole('button', { name: '保存' });
    if (await saveButton.isVisible().catch(() => false)) {
      return;
    }

    await this.clickLifecycleEdit();
  }

  /** 环境变量表格行（不含表头） */
  envVarRows(): Locator {
    return this.page.locator('.editable-variable-table tbody tr');
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

  /** 断言自动调节配置已从服务端回显 */
  async expectAutoScaleConfig({
    cpuUtilization,
    maxReplicas,
    minReplicas,
  }: {
    cpuUtilization: number;
    maxReplicas: number;
    minReplicas: number;
  }) {
    await this.page.reload({ waitUntil: 'domcontentloaded' });
    await this.waitForReady(2000);
    await this.openScaleSideslider();
    await this.selectAutoScaleMode();
    await expect(this.getAutoScaleMinInput()).toHaveValue(String(minReplicas), { timeout: 10000 });
    await expect(this.getAutoScaleMaxInput()).toHaveValue(String(maxReplicas), { timeout: 10000 });
    await expect(this.getAutoScaleCpuInput()).toHaveValue(String(cpuUtilization), { timeout: 10000 });
    await this.closeScaleSidesliderIfOpen();
  }

  /** 断言部署管理 Header Tab 已切换到目标页签 */
  async expectDeploymentTabActive(tab: DeploymentTab) {
    const tabTextMap: Record<DeploymentTab, string> = {
      event: '事件',
      history: '部署历史',
      instance: '实例列表',
      overview: '部署总览',
      topo: '资源拓扑',
    };
    const targetTab = this.page.locator('.tab-header-container .bk-tab-header-item').filter({ hasText: tabTextMap[tab] });
    await expect(targetTab).toHaveClass(/active/, { timeout: 10000 });
  }

  /** 断言部署管理「实例列表」页签内容已渲染 */
  async expectDeploymentInstanceTabVisible() {
    await this.page.getByText('部署管理', { exact: true }).first().waitFor({ state: 'visible', timeout: 10000 });
    await this.page
      .getByText(/该环境尚未部署应用|部署状态|实例|镜像 Tag|暂无可用的环境/)
      .first()
      .waitFor({ state: 'visible', timeout: 15000 });
  }

  /** 断言构建配置侧栏展示当前来源对应表单 */
  async expectBuilderConfigCurrentSourceFormVisible() {
    const sourceType = await this.getBuilderConfigSourceType();
    if (sourceType === 'codeRepository') {
      for (const label of ['代码库', '默认分支', '构建目录', 'Dockerfile 路径', '构建参数', '推荐版本号']) {
        await this.getBuilderConfigFormItem(label).waitFor({ state: 'visible', timeout: 10000 });
      }
      return;
    }
    if (sourceType === 'imageRegistry') {
      for (const label of ['镜像仓库', '镜像凭证']) {
        await this.getBuilderConfigFormItem(label).waitFor({ state: 'visible', timeout: 10000 });
      }
      return;
    }

    await this.getBuilderConfigFormItem('流水线').waitFor({ state: 'visible', timeout: 10000 });
    await this.getBuilderConfigSideslider()
      .getByText('需要保证流水线会将构建的镜像推送到当前空间的镜像仓库下', { exact: false })
      .waitFor({ state: 'visible', timeout: 10000 });
    await this.getBuilderConfigFormItem('推荐版本号').waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 断言构建配置必填校验可见；无可清空字段的来源只断言当前表单仍可用 */
  async expectBuilderConfigRequiredValidationVisible() {
    if (!this.builderConfigInvalidSubmitted) {
      await this.expectBuilderConfigCurrentSourceFormVisible();
      return;
    }

    const formItem = this.getBuilderConfigFormItem(this.builderConfigInvalidFieldLabel || '默认分支');
    const validation = formItem
      .getByText(/必填|不能为空|required/i)
      .or(formItem.locator('.bk-form-error, .bk-form-error-tips, .bk-form-error-text'))
      .first();
    await validation.waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 断言构建配置保存完成后回到基本信息页 */
  async expectBuilderConfigSaveCompleted() {
    await this.expectBuilderConfigSidesliderClosed();
    await this.expectBuilderConfigSectionVisible();
  }

  /** 断言构建配置区域可见 */
  async expectBuilderConfigSectionVisible() {
    const section = this.getBuildConfigSection();
    await section.waitFor({ state: 'visible', timeout: 15000 });
    await section
      .getByText('来源：', { exact: true })
      .or(section.getByText('应用镜像来源', { exact: false }))
      .first()
      .waitFor({ state: 'visible', timeout: 10000 });
    await section.getByRole('button', { name: '编辑' }).waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 断言编辑构建配置侧栏关闭 */
  async expectBuilderConfigSidesliderClosed() {
    await this.page.waitForSelector('.bk-sideslider .bk-modal-body', { state: 'hidden', timeout: 30000 });
  }

  /** 断言编辑构建配置侧栏可见 */
  async expectBuilderConfigSidesliderVisible() {
    const slider = this.getBuilderConfigSideslider();
    await slider.waitFor({ state: 'visible', timeout: 10000 });
    await this.getBuilderConfigSourceFormItem().waitFor({ state: 'visible', timeout: 10000 });
    if (
      await this.getBuilderConfigSourceButton('代码仓库')
        .isVisible()
        .catch(() => false)
    ) {
      await this.getBuilderConfigSourceButton('代码仓库').waitFor({ state: 'visible', timeout: 10000 });
      await this.getBuilderConfigSourceButton('镜像仓库').waitFor({ state: 'visible', timeout: 10000 });
    } else {
      await this.getBuilderConfigSourceButton('源码仓库').waitFor({ state: 'visible', timeout: 10000 });
      await this.getBuilderConfigSourceButton('流水线').waitFor({ state: 'visible', timeout: 10000 });
    }
    await this.getBuilderConfigSubmitButton().waitFor({ state: 'visible', timeout: 10000 });
    await slider.getByRole('button', { name: '取消' }).waitFor({ state: 'visible', timeout: 10000 });
    await this.expectBuilderConfigCurrentSourceFormVisible();
  }

  /** 断言开发模式处于关闭状态 */
  async expectDevModeDisabled() {
    await expect(this.getDevModeSwitcher()).toHaveClass(/is-unchecked/, { timeout: 10000 });
    await this.getDevModeSection().getByText('开启后，仍需执行以下流程', { exact: false }).waitFor({
      state: 'hidden',
      timeout: 10000,
    });
  }

  /** 断言开发模式开启后的操作步骤可见 */
  async expectDevModeEnabledStepsVisible() {
    await expect(this.getDevModeSwitcher()).toHaveClass(/is-checked/, { timeout: 10000 });
    await this.getDevModeSection().getByText('开启后，仍需执行以下流程', { exact: false }).waitFor({
      state: 'visible',
      timeout: 10000,
    });
    await this.getDevModeSection().getByText('执行部署', { exact: true }).waitFor({
      state: 'visible',
      timeout: 10000,
    });
    await this.getDevModeSection().getByRole('button', { name: '去部署' }).waitFor({
      state: 'visible',
      timeout: 10000,
    });
    await this.getDevModeSection().getByText('登录 bkms-cli', { exact: true }).waitFor({
      state: 'visible',
      timeout: 10000,
    });
    await this.getDevModeSection().getByRole('button', { name: '查看 Token' }).waitFor({
      state: 'visible',
      timeout: 10000,
    });
    await this.getDevModeSection().getByText('使用 bkms-cli 发布二进制', { exact: true }).waitFor({
      state: 'visible',
      timeout: 10000,
    });
  }

  /** 断言开发模式区域可见 */
  async expectDevModeSectionVisible() {
    await this.getDevModeSection().waitFor({ state: 'visible', timeout: 15000 });
    await this.getDevModeSwitcher().waitFor({ state: 'visible', timeout: 10000 });
    await this.getDevModeSection()
      .getByText('支持通过 bkms-cli 上传二进制的方式热更新服务', { exact: false })
      .waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 断言环境级变量侧栏可见 */
  async expectEnvLevelVarsSliderVisible() {
    await this.getSideslider().waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 断言环境变量搜索框可见 */
  async expectEnvVarSearchVisible() {
    await this.page.getByPlaceholder('搜索变量名、变量值、描述').waitFor({ state: 'visible', timeout: 10000 });
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

  /** 断言健康探针区域三张卡片可见 */
  async expectHealthProbeCardsVisible() {
    await this.getHealthProbeSection().waitFor({ state: 'visible', timeout: 15000 });
    for (const label of ['存活探针', '就绪探针', '启动探针']) {
      await this.getHealthProbeCard(label).waitFor({ state: 'visible', timeout: 10000 });
    }
  }

  /** 断言健康探针卡片包含指定文本 */
  async expectHealthProbeContains(label: string, text: string) {
    await this.getHealthProbeCard(label).getByText(text, { exact: false }).first().waitFor({
      state: 'visible',
      timeout: 10000,
    });
  }

  /** 断言健康探针编辑态表单字段值 */
  async expectHealthProbeInputValue(label: string, inputLabel: string, value: string) {
    await expect(this.getHealthProbeInput(this.getHealthProbeEditSection(label), inputLabel)).toHaveValue(value, {
      timeout: 10000,
    });
  }

  /** 断言健康探针卡片处于查看态 */
  async expectHealthProbeInViewMode(label: string) {
    await this.getHealthProbeViewSection(label).waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 断言健康探针卡片不包含指定文本 */
  async expectHealthProbeTextHidden(label: string, text: string) {
    await this.getHealthProbeCard(label).getByText(text, { exact: false }).first().waitFor({
      state: 'hidden',
      timeout: 10000,
    });
  }

  /** 断言健康探针表单校验提示可见 */
  async expectHealthProbeValidationVisible(text: string) {
    await this.getHealthProbeSection().getByText(text, { exact: false }).first().waitFor({
      state: 'visible',
      timeout: 10000,
    });
  }

  /** 断言生命周期区域包含指定文本 */
  async expectLifecycleContains(text: string) {
    await this.getLifecycleSection().getByText(text, { exact: false }).first().waitFor({
      state: 'visible',
      timeout: 10000,
    });
  }

  /** 断言生命周期自定义命令 shell 编辑器可见 */
  async expectLifecycleCustomCommandEditorVisible() {
    const section = this.getLifecycleSection();
    await section.getByText('自定义命令', { exact: true }).waitFor({
      state: 'visible',
      timeout: 10000,
    });
    await section.locator('.pre-stop-exec-mode').getByText('shell', { exact: true }).waitFor({
      state: 'visible',
      timeout: 10000,
    });
    await section.locator('textarea').first().waitFor({
      state: 'visible',
      timeout: 10000,
    });
  }

  /** 断言生命周期卡片处于查看态 */
  async expectLifecycleInViewMode() {
    await this.getLifecycleSection().getByRole('button', { name: '编辑' }).waitFor({
      state: 'visible',
      timeout: 10000,
    });
  }

  /** 断言生命周期区域可见 */
  async expectLifecycleSectionVisible() {
    await this.getLifecycleSection().waitFor({ state: 'visible', timeout: 15000 });
    await this.expectLifecycleContains('退出前命令 (preStop)');
    await this.expectLifecycleContains('优雅退出时间');
  }

  /** 断言生命周期区域不包含指定文本 */
  async expectLifecycleTextHidden(text: string) {
    await this.getLifecycleSection().getByText(text, { exact: false }).first().waitFor({
      state: 'hidden',
      timeout: 10000,
    });
  }

  /** 断言生命周期表单校验提示可见 */
  async expectLifecycleValidationVisible(text: string) {
    await this.getLifecycleSection().getByText(text, { exact: false }).first().waitFor({
      state: 'visible',
      timeout: 10000,
    });
  }

  /** 断言元数据配置区域两张卡片可见 */
  async expectMetadataCardsVisible() {
    await this.getMetadataSection().waitFor({ state: 'visible', timeout: 15000 });
    for (const label of ['标签（Labels）', '注解（Annotations）']) {
      await this.getMetadataCard(label).waitFor({ state: 'visible', timeout: 10000 });
    }
  }

  // ─── 部署管理页：实例列表断言 ─────────────────────────────────

  /** 断言元数据配置卡片包含指定文本 */
  async expectMetadataContains(label: string, text: string) {
    await this.getMetadataCard(label).getByText(text, { exact: false }).first().waitFor({
      state: 'visible',
      timeout: 10000,
    });
  }

  /** 断言元数据配置卡片处于查看态 */
  async expectMetadataInViewMode(label: string) {
    await this.getMetadataCard(label).getByText('表格模式').waitFor({ state: 'hidden', timeout: 10000 });
  }

  /** 断言元数据配置卡片不包含指定文本 */
  async expectMetadataTextHidden(label: string, text: string) {
    await this.getMetadataCard(label).getByText(text, { exact: false }).first().waitFor({
      state: 'hidden',
      timeout: 10000,
    });
  }

  /** 断言元数据配置表单校验提示可见 */
  async expectMetadataValidationVisible(text: string) {
    await this.getMetadataSection().getByText(text, { exact: false }).first().waitFor({
      state: 'visible',
      timeout: 10000,
    });
  }

  /** 断言移除部署弹窗必须输入环境名称才允许删除 */
  async expectRemoveDeployConfirmationGuard() {
    const dialog = this.getDialog();
    const confirmName = await this.getRemoveDeployConfirmName(dialog);
    if (!confirmName) {
      throw new Error('无法从移除部署确认弹窗中解析待确认的环境名称');
    }

    const input = dialog.getByRole('textbox').first();
    const deleteButton = dialog.getByRole('button', { name: '删除' });
    await input.fill(`${confirmName}-wrong`);
    await expect(deleteButton).toBeDisabled({ timeout: 10000 });
    await input.fill(confirmName);
    await expect(deleteButton).toBeEnabled({ timeout: 10000 });
    await dialog.getByRole('button', { name: '取消' }).click();
    await this.waitForDialogClosed();
  }

  /** 断言资源规格卡片包含指定文本（如 "0.5 核"） */
  async expectResourceContains(text: string) {
    await this.getResourceSection().getByText(text, { exact: false }).first().waitFor({ state: 'visible' });
  }

  /** 断言资源规格卡片处于查看态（编辑按钮再次可见） */
  async expectResourceInViewMode() {
    await this.getResourceSection().getByRole('button', { name: '编辑' }).waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 断言侧栏内环境变量搜索框可见 */
  async expectSliderEnvVarSearchVisible() {
    const slider = this.getSideslider();
    await slider.getByPlaceholder('搜索变量名、变量值、描述').waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 断言页面处于「未部署」空态 */
  async expectUninstalled() {
    await this.page.getByText('该环境尚未部署应用').first().waitFor({ state: 'visible', timeout: 15000 });
    await this.page.getByRole('button', { name: '立即部署' }).waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 断言更新策略卡片包含指定文本 */
  async expectUpdateStrategyContains(text: string) {
    await this.getUpdateStrategySection().getByText(text, { exact: false }).first().waitFor({
      state: 'visible',
      timeout: 10000,
    });
  }

  /** 断言更新策略卡片处于查看态 */
  async expectUpdateStrategyInViewMode() {
    await this.getUpdateStrategySection().getByRole('button', { name: '编辑' }).waitFor({
      state: 'visible',
      timeout: 10000,
    });
  }

  /** 断言更新策略卡片不包含指定文本 */
  async expectUpdateStrategyTextHidden(text: string) {
    await this.getUpdateStrategySection().getByText(text, { exact: false }).first().waitFor({
      state: 'hidden',
      timeout: 10000,
    });
  }

  /** 断言更新策略表单校验提示可见 */
  async expectUpdateStrategyValidationVisible(text: string) {
    await this.getUpdateStrategySection().getByText(text, { exact: false }).first().waitFor({
      state: 'visible',
      timeout: 10000,
    });
  }

  /** 在环境变量搜索框中输入关键字（默认环境变量视图下可见） */
  async fillEnvVarSearch(keyword: string) {
    const input = this.page.getByPlaceholder('搜索变量名、变量值、描述');
    await input.fill(keyword);
  }

  /** 填写 HTTP 健康探针配置 */
  async fillHealthProbeHttpConfig(label: string, config: HealthProbeHttpConfig) {
    const section = this.getHealthProbeEditSection(label);

    await this.selectHealthProbeMethod(label, 'HTTP');
    await this.fillHealthProbeInput(section, '检查路径', config.url);
    await this.fillHealthProbeInput(section, '检查端口', config.port);
    if (config.initialDelaySeconds !== undefined) {
      await this.fillHealthProbeInput(section, '延迟探测时间', config.initialDelaySeconds);
    }
    if (config.timeoutSeconds !== undefined) {
      await this.fillHealthProbeInput(section, '探测超时时间', config.timeoutSeconds);
    }
    if (config.periodSeconds !== undefined) {
      await this.fillHealthProbeInput(section, '探测频率', config.periodSeconds);
    }
    if (config.successThreshold !== undefined) {
      await this.fillHealthProbeInput(section, '连续探测成功次数', config.successThreshold);
    }
    if (config.failureThreshold !== undefined) {
      await this.fillHealthProbeInput(section, '连续探测失败次数', config.failureThreshold);
    }
  }

  /** 填写生命周期 shell 命令配置 */
  async fillLifecycleShellCommandConfig(config: LifecycleShellCommandConfig) {
    await this.selectLifecycleShellCommandMode();
    await this.fillLifecycleShellCommand(config.command);
    await this.fillLifecycleGracePeriod(config.gracePeriod);
  }

  /** 填写元数据配置文本模式内容 */
  async fillMetadataText(label: string, value: string) {
    const textarea = this.getMetadataCard(label).getByRole('textbox').first();
    await textarea.click({ clickCount: 3 });
    await textarea.fill(value);
  }

  /** 在侧栏环境变量搜索框中输入关键字 */
  async fillSliderEnvVarSearch(keyword: string) {
    const slider = this.getSideslider();
    const input = slider.getByPlaceholder('搜索变量名、变量值、描述');
    await input.fill(keyword);
  }

  /** 填写更新策略配置 */
  async fillUpdateStrategyConfig(config: UpdateStrategyConfig) {
    await this.fillUpdateStrategyInput('最大超出数量', config.maxSurge);
    await this.fillUpdateStrategyInput('最大不可用数量', config.maxUnavailable);
  }

  /** 获取构建配置卡片（BkmsContent 包裹块） */
  getBuildConfigSection(): Locator {
    return this.page.locator('.bkms-content').filter({ hasText: '构建配置' }).first();
  }

  /** 获取开发模式区域 */
  getDevModeSection(): Locator {
    return this.page.locator('.bkms-content').filter({ hasText: '开发模式' }).first();
  }

  /** 获取健康探针任意状态卡片 */
  getHealthProbeCard(label: string): Locator {
    return this.getHealthProbeSection().locator('.probe-view-section, .probe-card').filter({ hasText: label }).first();
  }

  /** 获取健康探针编辑态卡片 */
  getHealthProbeEditSection(label: string): Locator {
    return this.getHealthProbeSection().locator('.probe-card').filter({ hasText: label }).first();
  }

  /** 获取健康探针区域 */
  getHealthProbeSection(): Locator {
    return this.page.locator('.bkms-content').filter({ hasText: '健康探针' }).first();
  }

  /** 获取健康探针查看态卡片 */
  getHealthProbeViewSection(label: string): Locator {
    return this.getHealthProbeSection().locator('.probe-view-section').filter({ hasText: label }).first();
  }

  /** 获取生命周期区域 */
  getLifecycleSection(): Locator {
    return this.page.locator('.bkms-content').filter({ hasText: '生命周期' }).first();
  }

  // ─── 应用配置页：部署配置 EnvSelect（环境视角） ─────────────────────

  /** 获取元数据配置卡片 */
  getMetadataCard(label: string): Locator {
    return this.getMetadataSection().locator('.metadata-card').filter({ hasText: label }).first();
  }

  /** 获取元数据配置区域 */
  getMetadataSection(): Locator {
    return this.page.locator('.bkms-content').filter({ hasText: '元数据配置' }).first();
  }

  /** 获取资源规格卡片（BkmsContent 包裹块） */
  getResourceSection(): Locator {
    return this.page.locator('.bkms-content').filter({ hasText: '资源规格' }).first();
  }

  /** 获取更新策略区域 */
  getUpdateStrategySection(): Locator {
    return this.page.locator('.bkms-content').filter({ hasText: '更新策略' }).first();
  }

  /** 进入应用配置页（应用配置 = 二级菜单 key: appConfig） */
  async gotoAppConfig() {
    await this.gotoMenu('appConfig');
  }

  /** 进入制品管理页（二级菜单 key: artifact） */
  async gotoArtifactManagement() {
    await this.gotoMenu('artifact');
    await this.expectArtifactManagementVisible();
  }

  /** 进入基本信息页（基本信息 = 二级菜单 key: info） */
  async gotoBaseInfo() {
    await this.gotoMenu('info');
    await this.expectBuilderConfigSectionVisible();
  }

  /** 进入部署管理页（部署管理 = 二级菜单 key: deployment） */
  async gotoDeployment(tab: DeploymentTab = 'instance') {
    await this.gotoMenu('deployment', { activeTab: tab });
    await this.expectDeploymentTabActive(tab);
    if (tab === 'instance') {
      await this.expectDeploymentInstanceTabVisible();
    }
  }

  /**
   * 通用：进入应用详情某个二级菜单（遵循 `/:space/app/:name/:type/:menuName` 路由）
   */
  async gotoMenu(menuName: string, query?: Record<string, string>) {
    this.assertAppConfigured();
    const search = query ? `?${new URLSearchParams(query).toString()}` : '';
    await this.page.goto(`/#/${this.config.space}/app/${this.config.app}/${this.config.appType}/${menuName}${search}`);
    await this.waitForReady(3000);
  }

  /** 获取实例列表主表的行，排除固定选择列和操作列复制出的 VXE 行 */
  instanceRows(): Locator {
    return this.page.locator('.instance-table .vxe-table--main-wrapper .vxe-table--body tr');
  }

  /**
   * 当前应用是否已部署。
   * 通过「该环境尚未部署应用」空态文案判断：可见即未部署，超时未见即视为已部署。
   */
  async isDeployed(timeoutMs = 5000): Promise<boolean> {
    try {
      await this.page.getByText('该环境尚未部署应用').first().waitFor({ state: 'visible', timeout: timeoutMs });
      return false;
    } catch {
      return true;
    }
  }

  /** 当前开发模式是否开启 */
  async isDevModeEnabled() {
    await this.getDevModeSwitcher().waitFor({ state: 'visible', timeout: 10000 });
    return this.getDevModeSwitcher().evaluate(element => element.classList.contains('is-checked'));
  }

  /** 打开编辑构建配置侧栏 */
  async openBuilderConfigSideslider() {
    this.builderConfigInvalidSubmitted = false;
    this.builderConfigInvalidFieldLabel = '';
    this.builderConfigOriginalDefaultBranch = '';
    this.builderConfigOriginalImageRegistryName = '';
    await this.getBuildConfigSection().getByRole('button', { name: '编辑' }).click();
    await this.waitForSideslider();
    await this.expectBuilderConfigSidesliderVisible();
  }

  /** 点击「查看环境级变量」链接，打开侧栏并等待数据加载 */
  async openEnvLevelVarsSlider() {
    await this.page.getByText('查看环境级变量', { exact: false }).first().click();
    await this.waitForSideslider();
    // 等待侧栏内搜索框出现（意味着环境列表和变量数据已加载）
    const slider = this.getSideslider();
    await slider.getByPlaceholder('搜索变量名、变量值、描述').waitFor({ state: 'visible', timeout: 15000 });
    // 等待 loading 消失、表格数据就绪
    await this.waitForReady(2000);
  }

  /** 展开部署页右上角「更多」菜单 */
  async openMoreMenu() {
    await this.assertCurrentDeployReadyForOperation('移除部署');
    await this.page.locator('main .bkms-icon-more-fill').first().click();
    await this.page.waitForTimeout(300);
  }

  /** 打开扩缩容配置 Sideslider */
  async openScaleSideslider() {
    await this.assertCurrentDeployReadyForOperation('扩缩容');
    await this.clickButton('扩缩容');
    await this.waitForSideslider();
    const slider = this.getScaleSideslider();
    await slider.getByText('扩缩容配置').waitFor({ state: 'visible', timeout: 10000 });
    await slider.getByRole('button', { name: '手动调节' }).waitFor({ state: 'visible', timeout: 10000 });
    await slider.getByRole('button', { name: '自动调节' }).waitFor({ state: 'visible', timeout: 10000 });
    await this.page.waitForTimeout(500);
  }

  // ─── 应用配置页：环境变量 Tab ─────────────────────────────────────

  /** 刷新应用配置页并重新选择第一个测试环境 */
  async reloadAppConfigAndSelectFirstTestEnv() {
    await this.page.reload({ waitUntil: 'domcontentloaded' });
    await this.waitForReady(2000);
    await this.selectConfigFirstTestEnv();
    await this.expectDevModeSectionVisible();
  }

  /** 保存当前有效构建配置，并等待保存接口成功 */
  async saveValidBuilderConfig() {
    if (this.builderConfigInvalidSubmitted) {
      const sourceType = await this.getBuilderConfigSourceType();
      if (sourceType === 'codeRepository') {
        const input = this.getBuilderConfigDefaultBranchInput();
        await input.click({ clickCount: 3 });
        await input.fill(this.builderConfigOriginalDefaultBranch);
        await expect(input).toHaveValue(this.builderConfigOriginalDefaultBranch, { timeout: 10000 });
      } else if (sourceType === 'imageRegistry') {
        const input = this.getBuilderConfigImageRegistryInput();
        await input.click({ clickCount: 3 });
        await input.fill(this.builderConfigOriginalImageRegistryName);
        await expect(input).toHaveValue(this.builderConfigOriginalImageRegistryName, { timeout: 10000 });
      }
    }

    const responsePromise = this.waitForBuilderConfigSaveResponse();
    await this.getBuilderConfigSubmitButton().click();
    await this.assertApiResponseOk(await responsePromise, '保存构建配置');
    await this.expectBuilderConfigSidesliderClosed();
    await this.waitForReady(1000);
    await this.expectBuilderConfigSectionVisible();
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

  /** 选择自动调节模式 */
  async selectAutoScaleMode() {
    const slider = this.getScaleSideslider();
    await slider.getByRole('button', { name: '自动调节' }).click();
    await slider.getByText('最小实例数', { exact: true }).waitFor({ state: 'visible', timeout: 10000 });
    await slider.getByText('最大实例数', { exact: true }).waitFor({ state: 'visible', timeout: 10000 });
    await slider.getByText('触发条件', { exact: true }).waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 在部署配置的环境选择器中选择默认配置 */
  async selectConfigDefaultEnv() {
    const envSelect = this.page.locator('xpath=//div[normalize-space()="环境视角"]/ancestor::div[1]');
    await envSelect.click();
    const envPopover = this.page.locator('.c-env-select-v2-popover.visible').first();
    await envPopover.waitFor({ state: 'visible', timeout: 10000 });

    const defaultGroup = envPopover.locator(
      'xpath=.//*[normalize-space()="默认"]/ancestor::div[contains(@class, "flex-1")][1]',
    );
    await defaultGroup.locator('.env-list-scroll > div').filter({ hasText: '默认配置' }).first().click();
    await this.getLifecycleSection().getByText('生命周期').waitFor({ state: 'visible', timeout: 10000 });
    await this.waitForReady(1500);
  }

  /** 在部署配置的环境选择器中选择第一个类型=「测试」的环境 */
  async selectConfigFirstTestEnv() {
    const envSelect = this.page.locator('xpath=//div[normalize-space()="环境视角"]/ancestor::div[1]');
    await envSelect.click();
    const envPopover = this.page.locator('.c-env-select-v2-popover.visible').first();
    await envPopover.waitFor({ state: 'visible', timeout: 10000 });

    const testEnvColumn = envPopover.locator(
      'xpath=.//*[normalize-space()="测试"]/ancestor::div[contains(@class, "flex-1")][1]',
    );
    const option = testEnvColumn.locator('.env-list-scroll > div').first();
    await option.click();
    await this.getResourceSection().getByText('资源规格').waitFor({ state: 'visible', timeout: 10000 });
    await this.waitForReady(1500);
  }

  /** 在部署 Sideslider 中选择第一个可用镜像 Tag */
  async selectFirstImageTag() {
    const sideslider = this.getSideslider();
    // sideslider 内的镜像 Tag 选择器（唯一一个 bk-select）
    const select = sideslider.locator('.bk-select').first();
    await select.click();
    await this.selectOption();
  }

  /** 选择健康探针探测方法 */
  async selectHealthProbeMethod(label: string, method: string) {
    const section = this.getHealthProbeEditSection(label);
    const select = section.locator('.bk-form-item').filter({ hasText: '探测方法' }).locator('.bk-select').first();
    await select.click();
    await this.selectOption(method);
  }

  /** 选择生命周期 shell 命令模式 */
  async selectLifecycleShellCommandMode() {
    const section = this.getLifecycleSection();
    const switcher = this.getLifecyclePreStopSwitcher();
    await switcher.waitFor({ state: 'visible', timeout: 10000 });
    if (await switcher.evaluate(element => element.classList.contains('is-unchecked'))) {
      await switcher.click();
    }

    await section.getByText('自定义命令', { exact: true }).click();
    await section.locator('.pre-stop-exec-mode').getByText('shell', { exact: true }).click();
    await section.locator('textarea').first().waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 选择手动调节模式 */
  async selectManualScaleMode() {
    const slider = this.getScaleSideslider();
    await slider.getByRole('button', { name: '手动调节' }).click();
    await slider
      .locator('.bk-form-item')
      .filter({ hasText: '实例数' })
      .getByRole('spinbutton')
      .first()
      .waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 在元数据配置卡片中切换到文本模式 */
  async selectMetadataTextMode(label: string) {
    const card = this.getMetadataCard(label);
    await card.getByText('文本模式').click();
    await card.getByRole('textbox').first().waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 设置自动调节 CPU 使用率阈值 */
  async setAutoScaleCpuUtilization(cpuUtilization: number) {
    const triggerSection = this.getScaleSideslider()
      .locator('.bk-form-item:visible')
      .filter({ hasText: '触发条件' })
      .first();
    const metricInput = triggerSection.getByRole('textbox').first();
    await metricInput.waitFor({ state: 'visible', timeout: 10000 });
    if ((await metricInput.inputValue()) !== 'CPU 使用率') {
      await metricInput.click();
      await this.selectOption('CPU 使用率');
    }
    const input = this.getAutoScaleCpuInput();
    await input.click({ clickCount: 3 });
    await input.fill(String(cpuUtilization));
  }

  /** 设置自动调节最小/最大副本数 */
  async setAutoScaleReplicas({ maxReplicas, minReplicas }: { maxReplicas: number; minReplicas: number }) {
    const minInput = this.getAutoScaleMinInput();
    const maxInput = this.getAutoScaleMaxInput();

    await minInput.click({ clickCount: 3 });
    await minInput.fill(String(minReplicas));
    await maxInput.click({ clickCount: 3 });
    await maxInput.fill(String(maxReplicas));
  }

  /** 设置「立即部署」Sideslider 的实例数 */
  async setDeployReplicas(count: number) {
    const sideslider = this.getSideslider();
    const input = sideslider.getByRole('spinbutton').first();
    if ((await input.inputValue()) === String(count)) return;
    await input.click({ clickCount: 3 });
    await input.fill(String(count));
  }

  /** 设置扩缩容副本数 */
  async setScaleReplicas(count: number) {
    const slider = this.getScaleSideslider();
    const input = slider.locator('.bk-form-item').filter({ hasText: '实例数' }).getByRole('spinbutton').first();
    await input.click({ clickCount: 3 });
    await input.fill(String(count));
  }

  /** 环境级变量侧栏表格行（不含表头，兼容 bk-table / vxe-table） */
  sliderEnvVarRows(): Locator {
    const slider = this.getSideslider();
    return slider.locator('.bk-table-body tr, .vxe-table--body tr');
  }

  /** 提交自动调节配置并等待页面状态稳定 */
  async submitAutoScale() {
    const responsePromise = this.waitForScaleResponse('/autoscaler', 'PUT');
    await this.getScaleSideslider().getByRole('button', { name: '确定' }).click();
    await this.confirmScaleModeSwitch('确认切换为自动');
    await this.assertApiResponseOk(await responsePromise, '配置自动调节');
    await this.waitForSidesliderSettled();
  }

  /** 提交无效构建配置。按当前来源清空一个必填字段；无可清空字段的来源保持当前有效表单 */
  async submitInvalidBuilderConfig() {
    this.builderConfigInvalidSubmitted = false;
    this.builderConfigInvalidFieldLabel = '';
    const sourceType = await this.getBuilderConfigSourceType();

    if (sourceType === 'codeRepository') {
      const input = this.getBuilderConfigDefaultBranchInput();
      this.builderConfigOriginalDefaultBranch = await input.inputValue();
      this.builderConfigInvalidFieldLabel = '默认分支';
      await input.click({ clickCount: 3 });
      await input.fill('');
    } else if (sourceType === 'imageRegistry') {
      const input = this.getBuilderConfigImageRegistryInput();
      this.builderConfigOriginalImageRegistryName = await input.inputValue();
      this.builderConfigInvalidFieldLabel = '镜像仓库';
      await input.click({ clickCount: 3 });
      await input.fill('');
    } else {
      await this.expectBuilderConfigCurrentSourceFormVisible();
      return false;
    }

    await this.getBuilderConfigSubmitButton().click();
    this.builderConfigInvalidSubmitted = true;
    await this.page.waitForTimeout(500);
    return true;
  }

  /** 提交手动扩缩容配置并等待页面状态稳定 */
  async submitManualScale() {
    const responsePromise = this.waitForScaleResponse('/instances/operations/scale', 'PUT');
    const toggleResponsePromise = this.page
      .waitForResponse(
        response => response.request().method() === 'PATCH' && response.url().includes('/autoscaler/toggle'),
        { timeout: 30000 },
      )
      .catch(() => null);

    await this.getScaleSideslider().getByRole('button', { name: '确定' }).click();
    const switchedFromAuto = await this.confirmScaleModeSwitch('确认切换为手动');

    await this.assertApiResponseOk(await responsePromise, '手动扩缩容');
    if (switchedFromAuto) {
      const toggleResponse = await toggleResponsePromise;
      if (!toggleResponse) throw new Error('关闭自动调节接口请求超时：PATCH /autoscaler/toggle');
      await this.assertApiResponseOk(toggleResponse, '关闭自动调节');
    }
    await this.waitForSidesliderSettled();
  }

  /** 点击 Sideslider 底部提交按钮并等待页面状态稳定 */
  async submitSideslider() {
    const sideslider = this.getSideslider();
    const submitButton = sideslider
      .getByRole('button', { name: '确定' })
      .or(sideslider.getByRole('button', { name: '部署' }))
      .first();
    const responsePromise = this.page.waitForResponse(
      response =>
        response.request().method() === 'POST' &&
        (response.url().includes('/trpc-deploys') || response.url().includes('/taf-deploys')),
      { timeout: 30000 },
    );
    await submitButton.click();
    await this.assertApiResponseOk(await responsePromise, '部署应用');
    await this.waitForSidesliderSettled();
  }

  /** 切换构建配置来源，并等待目标表单渲染 */
  async switchBuilderConfigSource() {
    const sourceType = await this.getBuilderConfigSourceType();
    if (sourceType === 'codeRepository') {
      const imageRegistryButton = this.getBuilderConfigSourceButton('镜像仓库');
      if (await imageRegistryButton.isVisible().catch(() => false)) {
        await imageRegistryButton.click();
        await this.getBuilderConfigFormItem('镜像仓库').waitFor({ state: 'visible', timeout: 10000 });
        return;
      }

      await this.getBuilderConfigSourceButton('流水线').click();
      await this.getBuilderConfigFormItem('流水线').waitFor({ state: 'visible', timeout: 10000 });
      return;
    }

    const repoButton =
      sourceType === 'imageRegistry'
        ? this.getBuilderConfigSourceButton('代码仓库')
        : this.getBuilderConfigSourceButton('源码仓库');
    if (await this.isBuilderConfigSourceDisabled(sourceType === 'imageRegistry' ? '代码仓库' : '源码仓库')) {
      await this.expectBuilderConfigCurrentSourceFormVisible();
      return;
    }

    await repoButton.click();
    await this.getBuilderConfigFormItem('代码库').waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 等待 SSE 推送后的实例数满足期望（达成返回 true，超时返回 false） */
  async waitForInstanceCount(expected: number, { timeout = 180000 } = {}) {
    try {
      await expect
        .poll(() => this.instanceRows().count(), { timeout })
        .toBeGreaterThanOrEqual(expected);
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
