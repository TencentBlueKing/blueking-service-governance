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
 * 应用配置域 Page Object。
 *
 * 覆盖：环境视角选择器、环境变量（默认视图 + 环境级变量侧栏）、
 * 资源规格卡片、开发模式开关。
 * 健康探针/生命周期/元数据/更新策略等 app-spec 卡片见 app-spec.page.ts。
 */
export default class AppConfigPage extends AppDetailBase {
  private getDevModeSection() {
    return this.page.locator('.bkms-content').filter({ hasText: '开发模式' }).first();
  }

  private getDevModeSwitcher() {
    return this.getDevModeSection().locator('.bk-switcher').first();
  }

  private getEnvPopover() {
    return this.page.locator('.c-env-select-v2-popover').first();
  }

  private getEnvSelect() {
    return this.page.locator('xpath=//div[normalize-space()="环境视角"]/ancestor::div[1]');
  }

  /** 等待环境选择器和配置卡片均更新到指定环境，避免在切换中的旧卡片上继续操作。 */
  private async waitForEnvConfigReady(envSelect: Locator, envDisplayName: string) {
    await expect(envSelect.getByText(envDisplayName, { exact: true })).toBeVisible({ timeout: 10000 });
    await this.getResourceSection()
      .getByText(`环境：${envDisplayName}`, { exact: true })
      .waitFor({ state: 'visible', timeout: 15000 });
    await this.safeWaitForNetworkIdle();
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
  }

  /** 点击开发模式开关并等待确认弹窗出现 */
  async clickDevModeSwitch() {
    await this.getDevModeSwitcher().click();
    await this.waitForDialog();
  }

  /** 点击资源规格卡片上的「编辑」按钮 */
  async clickResourceEdit() {
    await this.getResourceSection().getByRole('button', { name: '编辑' }).click();
    // 等待编辑表单渲染完成（替代固定 sleep），后续填表操作可立即执行
    await this.getResourceSection().getByRole('spinbutton').first().waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 点击资源规格卡片的「恢复默认配置」按钮 */
  async clickResourceResetToDefault() {
    await this.getResourceSection().getByRole('button', { name: '恢复默认配置' }).click();
  }

  /** 点击资源规格卡片的「保存」按钮 */
  async clickResourceSave() {
    await this.getResourceSection().getByRole('button', { name: '保存' }).click();
  }

  /** 确保开发模式处于关闭状态，便于用例可重复运行 */
  async ensureDevModeDisabled() {
    await this.expectDevModeSectionVisible();
    if (!(await this.isDevModeEnabled())) return;

    await this.clickDevModeSwitch();
    await this.clickDevModeConfirm('disable');
    await this.expectDevModeDisabled();
  }

  /** 环境变量表格行（不含表头） */
  envVarRows() {
    return this.page.locator('.editable-variable-table tbody tr');
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

  /** 断言资源规格卡片包含指定文本（如 "0.5 核"） */
  async expectResourceContains(text: string) {
    await this.getResourceSection().getByText(text, { exact: false }).first().waitFor({ state: 'visible' });
  }

  /** 断言资源规格卡片处于查看态（编辑按钮再次可见） */
  async expectResourceInViewMode() {
    await this.getResourceSection().getByRole('button', { name: '编辑' }).waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 断言侧栏环境变量表格为空 */
  async expectSliderEnvVarRowsEmpty() {
    await expect(this.sliderEnvVarRows()).toHaveCount(0, { timeout: 10000 });
  }

  /** 断言侧栏环境变量表格已有数据 */
  async expectSliderEnvVarRowsVisible() {
    await expect.poll(() => this.sliderEnvVarRows().count(), { timeout: 10000 }).toBeGreaterThan(0);
  }

  /** 断言侧栏内环境变量搜索框可见 */
  async expectSliderEnvVarSearchVisible() {
    const slider = this.getSideslider();
    await slider.getByPlaceholder('搜索变量名、变量值、描述').waitFor({ state: 'visible', timeout: 10000 });
  }

  /** 在环境变量搜索框中输入关键字（默认环境变量视图下可见） */
  async fillEnvVarSearch(keyword: string) {
    const input = this.page.getByPlaceholder('搜索变量名、变量值、描述');
    await input.fill(keyword);
  }

  /** 在侧栏环境变量搜索框中输入关键字 */
  async fillSliderEnvVarSearch(keyword: string) {
    const slider = this.getSideslider();
    const input = slider.getByPlaceholder('搜索变量名、变量值、描述');
    await input.fill(keyword);
  }

  /** 获取资源规格卡片（BkmsContent 包裹块） */
  getResourceSection() {
    return this.page.locator('.bkms-content').filter({ hasText: '资源规格' }).first();
  }

  /** 进入应用配置页（应用配置 = 二级菜单 key: appConfig） */
  async gotoAppConfig() {
    await this.gotoMenu('appConfig');
  }

  /** 当前开发模式是否开启 */
  async isDevModeEnabled() {
    await this.getDevModeSwitcher().waitFor({ state: 'visible', timeout: 10000 });
    return this.getDevModeSwitcher().evaluate(element => element.classList.contains('is-checked'));
  }

  /** 点击「查看环境级变量」链接，打开侧栏并等待数据加载 */
  async openEnvLevelVarsSlider() {
    await this.page.getByText('查看环境级变量', { exact: false }).first().click();
    await this.waitForSideslider();
    // 等待侧栏内搜索框出现（意味着环境列表和变量数据已加载）
    const slider = this.getSideslider();
    await slider.getByPlaceholder('搜索变量名、变量值、描述').waitFor({ state: 'visible', timeout: 15000 });
  }

  /** 刷新应用配置页并重新选择第一个测试环境 */
  async reloadAppConfigAndSelectFirstTestEnv() {
    await this.page.reload({ waitUntil: 'domcontentloaded' });
    await this.safeWaitForNetworkIdle();
    await this.selectConfigFirstTestEnv();
    await this.expectDevModeSectionVisible();
  }

  /** 在部署配置的环境选择器中选择默认配置 */
  async selectConfigDefaultEnv() {
    const envSelect = this.getEnvSelect();
    if (
      await envSelect
        .getByText('默认配置', { exact: true })
        .isVisible()
        .catch(() => false)
    ) {
      await this.getResourceSection()
        .getByText('按环境', { exact: true })
        .waitFor({ state: 'visible', timeout: 15000 });
      return;
    }

    await envSelect.click();
    const envPopover = this.getEnvPopover();
    await envPopover.waitFor({ state: 'visible', timeout: 10000 });

    const defaultGroup = envPopover.locator(
      'xpath=.//*[normalize-space()="默认"]/ancestor::div[contains(@class, "flex-1")][1]',
    );
    await defaultGroup.locator('.env-list-scroll > div').filter({ hasText: '默认配置' }).first().click();
    await expect(envSelect.getByText('默认配置', { exact: true })).toBeVisible({ timeout: 10000 });
    await this.getResourceSection().getByText('按环境', { exact: true }).waitFor({ state: 'visible', timeout: 15000 });
    await this.safeWaitForNetworkIdle();
  }

  /** 在部署配置的环境选择器中选择第一个类型=「测试」的环境 */
  async selectConfigFirstTestEnv() {
    const envSelect = this.getEnvSelect();
    // 首次进入页面时，环境切换会被页面自身的 loading 锁保护。先等待配置区完成初始渲染，
    // 再触发选择，避免点击被忽略后仍停留在默认配置。
    await this.getResourceSection().waitFor({ state: 'visible', timeout: 15000 });
    // 页面刷新会保留已选环境；此时无需再次打开 Popover，避免把已就绪的选择器切回关闭状态。
    if (
      await envSelect
        .getByText('测试', { exact: true })
        .isVisible()
        .catch(() => false)
    )
      return;

    await envSelect.click();
    const envPopover = this.getEnvPopover();
    await envPopover.waitFor({ state: 'visible', timeout: 10000 });

    const testEnvColumn = envPopover.locator(
      'xpath=.//*[normalize-space()="测试"]/ancestor::div[contains(@class, "flex-1")][1]',
    );
    const option = testEnvColumn.locator('.env-list-scroll > div').first();
    const envDisplayName = await option.locator('.text-ov').innerText();
    await option.click();
    await this.waitForEnvConfigReady(envSelect, envDisplayName);
  }

  /** 环境级变量侧栏表格行（不含表头，兼容 bk-table / vxe-table） */
  sliderEnvVarRows() {
    const slider = this.getSideslider();
    return slider.locator('.bk-table-body tr, .vxe-table--body tr');
  }
}
