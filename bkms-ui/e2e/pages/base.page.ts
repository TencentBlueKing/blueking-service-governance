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
import { type APIRequestContext, type Locator, type Page, type PlaywrightTestArgs } from '@playwright/test';
import path from 'path';

import { type BkmsFixtures } from '../fixtures/fixtures';
import { NAVIGATION_ROUTE_MAP } from '../utils/config';
import { type Schema } from '../utils/form';

/** 构造 Page Object 所需的最小依赖（勿用整块 BkmsFixtures，否则与 basePage/homePage fixture 互相要求） */
export type BasePageDependencies = Pick<BkmsFixtures, 'testConfig'> & Pick<PlaywrightTestArgs, 'page' | 'request'>;

/**
 * Page Object 基类（描述 UI 基本结构）。
 *
 * 封装所有页面共享的能力：
 * - data-testid selector 快捷定位
 * - 页面就绪等待
 * - 截图
 * - 导航
 * - bkui-vue 组件通用操作（Select / Dialog / Sideslider / PopConfirm）
 */
export default class BasePage {
  protected config: BkmsFixtures['testConfig'];
  protected page: Page;
  protected request: APIRequestContext;

  constructor({ page, testConfig, request }: BasePageDependencies) {
    this.page = page;
    this.config = testConfig;
    this.request = request;
  }

  async clickButton(name: string) {
    await this.page.getByRole('button').filter({ hasText: name }).click();
  }

  async clickLink(name: string) {
    await this.page.getByRole('link', { name }).click();
    await this.waitForReady(2000);
  }

  async clickTab(tabName: string) {
    await this.page.locator('.bk-tab-header-item').filter({ hasText: tabName }).click();
    await this.waitForReady(2000);
  }

  async clickText(text: string) {
    await this.page.getByText(text, { exact: true }).click();
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  async fillForm<T extends Record<string, any>>(schema: Schema<T>, data: Partial<T>) {
    for (const key of Object.keys(schema)) {
      const field = schema[key];
      const value = data[key] ?? field?.default;

      // 👉 跳过未传值
      if (value === undefined) continue;

      // 👉 条件字段（关键 ⭐）
      if (field?.requiredIf && !field.requiredIf(data)) {
        continue;
      }

      console.log('start fill form item', field?.selector, field?.type, value);

      const selector = field?.selector || '';

      switch (field?.type) {
        case 'select':
          await this.searchOptionAndSelect(selector, value);
          break;

        case 'array':
          await this.fillFormItemArray(selector, value);
          break;

        default:
          await this.fillInput(selector, value);
      }
    }
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  async fillFormItemArray(label: string, values: any) {
    const valuesArray = Array.isArray(values) ? values : [values];
    const formContent = this.getFormItemContent(label);
    if (!(await formContent.count())) return;
    const addBtn = formContent.getByRole('button').first();
    if (!(await addBtn.count())) return;
    for (let i = 0; i < valuesArray.length; i++) {
      await addBtn.click();
      const input = await formContent.getByRole('textbox').nth(i);
      if (!(await input.count())) continue;
      await input.fill(valuesArray[i]);
    }
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  async fillInput(selector: Locator | string, value: any) {
    if (!selector) return;
    if (typeof selector === 'string') {
      const formItem = this.getFormItem(selector);
      if (!(await formItem.count())) return;
      await formItem.getByRole('textbox').fill(value);
    } else {
      await selector.fill(value);
    }
  }

  getCheckBox(index = 1) {
    return this.page.locator('.bk-checkbox-input').nth(index - 1);
  }

  getCheckboxes() {
    return this.page.locator('.bk-checkbox-input');
  }

  getDialog() {
    return this.page.locator('.bk-dialog .bk-modal-wrapper');
  }

  getFormItem(label: string) {
    return this.page.locator('.bk-form-item').filter({ hasText: label }).locator('.bk-form-content > *').first();
  }

  getFormItemContent(label: string) {
    return this.page.locator('.bk-form-item').filter({ hasText: label }).locator('.bk-form-content');
  }

  getModal() {
    return this.page.locator('.bk-modal-wrapper').first();
  }

  getPopConfirm() {
    return this.page.locator('.bk-pop-confirm');
  }

  getSelect() {
    return this.page.locator('.bk-select:not([data-testid="space-selector"])').first();
  }

  getSideslider() {
    return this.page.locator('.bk-sideslider-wrapper').or(this.page.locator('.bk-modal-wrapper')).first();
  }

  getVisibleSelectPopover() {
    return this.page.locator('.bk-popover.bk-pop2-content.visible.bk-select-popover').first();
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  mockData(url: string, data: any) {
    this.page.route(url, async route => {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(data),
      });
    });
  }

  async navTo(space: string, nav: string) {
    const route = NAVIGATION_ROUTE_MAP[nav as keyof typeof NAVIGATION_ROUTE_MAP] || nav;
    await this.page.goto(`/#/${space}/${route}`);
    await this.waitForReady(3000);
  }

  /**
   * 等待网络空闲，但带超时兜底。
   *
   * 说明：BKMS 这类管理台常有轮询 / 心跳接口，裸 `waitForLoadState('networkidle')`
   * 可能永远无法触发，直到挂满 navigationTimeout（30s）。这里给一个较短的显式超时，
   * 能达成就快速返回，达不成最多等待 timeout，避免隐性卡顿。
   */
  async safeWaitForNetworkIdle(timeout = 5000) {
    try {
      await this.page.waitForLoadState('networkidle', { timeout });
    } catch {
      // 存在持续网络请求时忽略，交由后续显式元素等待兜底
    }
  }

  async screenshot(name: string) {
    await this.page.screenshot({
      path: path.join(this.config.reportDir, `${name}.png`),
    });
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  async searchOptionAndSelect(selector: Locator | string, searchText: any) {
    if (!selector) return;
    if (typeof selector === 'string') {
      const formItem = this.getFormItem(selector);
      if (!(await formItem.count())) return;
      await formItem.click();
    } else {
      await selector.click();
    }
    const searchInput = this.getVisibleSelectPopover().locator('.bk-select-search-input');
    if (await searchInput.count()) {
      await searchInput.fill(searchText);
    }
    await this.selectOption(searchText);
  }

  /**
   * 在 bkui-vue Select 中选择指定选项。
   * 点击 select 后等待弹出层选项出现，再按文本匹配点击。
   */
  async selectOption(optionText?: string) {
    await this.page.waitForTimeout(500);
    const popContent = this.getVisibleSelectPopover();
    if (!(await popContent.count())) return;
    if (optionText) {
      const option = popContent
        .locator('.bk-select-option')
        .filter({ hasText: new RegExp(`^\\s*${optionText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`) });
      await option.first().click({ timeout: 10000 });
    } else {
      await popContent.locator('.bk-select-option').first().click();
    }

    await this.safeWaitForNetworkIdle();
    await this.page.waitForTimeout(300);
  }

  async selectOptionBy(selector: Locator | string, optionText: string) {
    if (!selector) return;
    if (typeof selector === 'string') {
      const formItem = this.getFormItem(selector);
      if (!formItem) return;
      await formItem.click();
    } else {
      await selector.click();
    }
    await this.selectOption(optionText);
  }

  /**
   * 通过 data-testid 定位元素（与 v-test 指令配合）。
   * 命名规范: {模块}-{组件}-{行为}
   */
  testId(id: string) {
    return this.page.locator(`[data-testid="${id}"]`);
  }

  /**
   * 验证 AccessToken 认证是否成功。
   */
  async verifyAccessToken() {
    const response = await this.request.get('/simple_account/info', {
      headers: {
        Authorization: `Bearer ${this.config.token}`,
      },
    });
    return response.status() === 200;
  }

  async waitForDialog() {
    await this.page.waitForSelector('.bk-dialog .bk-modal-wrapper', { state: 'visible', timeout: 15000 });
  }

  async waitForDialogClosed() {
    await this.page.waitForSelector('.bk-dialog .bk-modal-wrapper', { state: 'hidden', timeout: 15000 });
  }

  /**
   * 等待 bk-modal-wrapper 可见
   */
  async waitForModal() {
    await this.page.waitForSelector('.bk-modal-wrapper', { state: 'visible', timeout: 15000 });
    await this.page.waitForTimeout(500);
  }

  async waitForPopConfirm() {
    await this.page.waitForSelector('.bk-pop-confirm', { state: 'visible', timeout: 10000 });
    await this.page.waitForTimeout(500);
  }

  async waitForReady(extraMs = 2000) {
    await this.safeWaitForNetworkIdle();
    if (extraMs > 0) await this.page.waitForTimeout(extraMs);
  }
  async waitForSideslider() {
    await this.page.locator('.bk-sideslider .bk-modal-body:visible').first().waitFor({
      state: 'visible',
      timeout: 15000,
    });
    await this.page.waitForTimeout(1000);
  }

  async waitSelectReady() {
    await this.getVisibleSelectPopover()
      .locator('.bk-select-option')
      .first()
      .waitFor({ state: 'visible', timeout: 10000 });
  }
}
