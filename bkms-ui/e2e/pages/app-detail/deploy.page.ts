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
import { type Locator, expect } from '@playwright/test';

import AppDetailBase from './app-detail-base.page';

type DeploymentTab = 'event' | 'history' | 'instance' | 'overview' | 'topo';

/**
 * 部署管理域 Page Object。
 *
 * 覆盖：立即部署、扩缩容（手动/自动）、移除部署、部署页签导航。
 * 只封装原子 UI 操作；复杂业务流程（部署/扩缩容/移除）由 action 层组合完成。
 * 实例列表相关操作与 mock 见 deploy-instance.page.ts。
 */
export default class DeployPage extends AppDetailBase {
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
    await this.safeWaitForNetworkIdle();
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

  private waitForScaleResponse(path: string, method: string) {
    return this.page.waitForResponse(
      response => response.request().method() === method && response.url().includes(path),
      { timeout: 30000 },
    );
  }

  /** 点击「立即部署」按钮（未部署空态时显示） */
  async clickQuicklyDeploy() {
    await this.clickButton('立即部署');
    await this.waitForSideslider();
  }

  /** 在更多菜单中点击「移除部署」，并等待确认弹窗出现 */
  async clickRemoveDeploy() {
    await this.page.getByRole('button', { name: '移除部署' }).click();
    await this.waitForDialog();
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
    await this.safeWaitForNetworkIdle();
    await this.openScaleSideslider();
    await this.selectAutoScaleMode();
    await expect(this.getAutoScaleMinInput()).toHaveValue(String(minReplicas), { timeout: 10000 });
    await expect(this.getAutoScaleMaxInput()).toHaveValue(String(maxReplicas), { timeout: 10000 });
    await expect(this.getAutoScaleCpuInput()).toHaveValue(String(cpuUtilization), { timeout: 10000 });
    await this.closeScaleSidesliderIfOpen();
  }

  /** 断言部署管理「实例列表」页签内容已渲染 */
  async expectDeploymentInstanceTabVisible() {
    await this.page.getByText('部署管理', { exact: true }).first().waitFor({ state: 'visible', timeout: 10000 });
    await this.page
      .getByText(/该环境尚未部署应用|部署状态|实例|镜像 Tag|暂无可用的环境/)
      .first()
      .waitFor({ state: 'visible', timeout: 15000 });
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
    const targetTab = this.page
      .locator('.tab-header-container .bk-tab-header-item')
      .filter({ hasText: tabTextMap[tab] });
    await expect(targetTab).toHaveClass(/active/, { timeout: 10000 });
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

  /** 断言页面处于「未部署」空态 */
  async expectUninstalled() {
    await this.page.getByText('该环境尚未部署应用').first().waitFor({ state: 'visible', timeout: 15000 });
    await this.page.getByRole('button', { name: '立即部署' }).waitFor({ state: 'visible', timeout: 10000 });
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

  /** 展开部署页右上角「更多」菜单 */
  async openMoreMenu() {
    await this.assertCurrentDeployReadyForOperation('移除部署');
    await this.page.locator('main .bkms-icon-more-fill').first().click();
    // 等待菜单项渲染完成（替代固定 sleep），也使「未部署时点开更多菜单」的失败信息更明确
    await this.page.getByRole('button', { name: '移除部署' }).waitFor({ state: 'visible', timeout: 5000 });
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
  }

  /** 选择自动调节模式 */
  async selectAutoScaleMode() {
    const slider = this.getScaleSideslider();
    await slider.getByRole('button', { name: '自动调节' }).click();
    await slider.getByText('最小实例数', { exact: true }).waitFor({ state: 'visible', timeout: 10000 });
    await slider.getByText('最大实例数', { exact: true }).waitFor({ state: 'visible', timeout: 10000 });
    await slider.getByText('触发条件', { exact: true }).waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 在部署 Sideslider 中选择第一个可用镜像 Tag */
  async selectFirstImageTag() {
    const sideslider = this.getSideslider();
    // sideslider 内的镜像 Tag 选择器（唯一一个 bk-select）
    const select = sideslider.locator('.bk-select').first();
    await select.click();
    await this.selectOption();
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

  /** 提交自动调节配置并等待页面状态稳定 */
  async submitAutoScale() {
    const responsePromise = this.waitForScaleResponse('/autoscaler', 'PUT');
    await this.getScaleSideslider().getByRole('button', { name: '确定' }).click();
    await this.confirmScaleModeSwitch('确认切换为自动');
    await this.assertApiResponseOk(await responsePromise, '配置自动调节');
    await this.waitForSidesliderSettled();
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
}
