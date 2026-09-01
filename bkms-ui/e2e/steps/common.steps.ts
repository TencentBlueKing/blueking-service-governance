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
/**
 * ====================================================================
 * BKMS E2E BDD 通用步骤定义
 * ====================================================================
 *
 * 所有步骤通过 Page Object（pages fixture）操作，不直接写 CSS selector。
 *
 * 覆盖：导航、交互、等待、截图、断言。
 * AI 生成场景时应优先复用这些步骤，仅在需要时生成自定义步骤。
 * ====================================================================
 */
import { expect } from '@playwright/test';

import { Given, Then, When } from '../fixtures/fixtures';

Given('进入 {string} 空间的 {string} 导航', async ({ pages, testConfig }, space, nav) => {
  // 与 @space:default tag 语义对齐：字面量 "default" 回退到 testConfig.space（由环境变量注入）
  const resolvedSpace = space === 'default' ? testConfig.space : space;
  await pages.basePage.navTo(resolvedSpace, nav);
});

Given('AccessToken 认证已配置', async ({ pages }) => {
  const basePage = pages.basePage;
  const result = await basePage.verifyAccessToken();
  expect(result).toBe(true);
});

// ─── 点击步骤 ────────────────────────────────────────────────────

When('点击 {string}', async ({ page }, name) => {
  await page.getByText(name).click();
});

When('点击 {string} 按钮', async ({ pages }, name) => {
  const basePage = pages.basePage;
  await basePage.clickButton(name);
});

When('点击 {string} 链接', async ({ pages }, linkText) => {
  const basePage = pages.basePage;
  await basePage.clickLink(linkText);
});

When('点击 {string} 文本', async ({ pages }, text) => {
  const basePage = pages.basePage;
  await basePage.clickText(text);
});

When('点击 Tab 页签 {string}', async ({ pages }, tabName) => {
  const basePage = pages.basePage;
  await basePage.clickTab(tabName);
});

When('点击弹窗中的 {string} 按钮', async ({ pages }, buttonName) => {
  const basePage = pages.basePage;
  const dialog = basePage.getDialog();
  await dialog.getByRole('button', { name: buttonName }).click();
});

// ─── 填写步骤 ────────────────────────────────────────────────────

When('在输入框填写 {string}', async ({ page }, value) => {
  const input = page.getByRole('textbox').first();
  await input.click({ clickCount: 3 });
  await input.fill(value);
});

When('在 {string} 输入框填写 {string}', async ({ pages }, label, value) => {
  const basePage = pages.basePage;
  await basePage.getFormItem(label).fill(value);
});

// ─── 下拉选择步骤 ──────────────────────────────────────────────

When('在下拉框中选择 {string}', async ({ pages }, envName) => {
  const basePage = pages.basePage;
  await basePage.getSelect().click();
  await basePage.selectOption(envName);
});

When('在 {string} 下拉框中选择 {string}', async ({ pages }, label, optionText) => {
  const basePage = pages.basePage;
  await basePage.getFormItem(label).click();
  await basePage.selectOption(optionText);
});

When('勾选 Checkbox', async ({ pages }) => {
  const basePage = pages.basePage;
  await basePage.getCheckBox(1).click();
});

When('勾选第 {int} 个 Checkbox', async ({ pages }, index) => {
  const basePage = pages.basePage;
  await basePage.getCheckBox(index).click();
});

When('勾选所有 Checkbox', async ({ pages }) => {
  const basePage = pages.basePage;
  const checkboxes = basePage.getCheckboxes();
  for (let i = 1; i <= (await checkboxes.count()); i++) {
    await basePage.getCheckBox(i).click();
  }
});

When('等待页面加载', async ({ pages }) => {
  const basePage = pages.basePage;
  await basePage.waitForReady(2000);
});

When('等待 {int} 毫秒', async ({ pages }, ms) => {
  const basePage = pages.basePage;
  await basePage.waitForReady(ms);
});

When('等待弹窗出现', async ({ pages }) => {
  const basePage = pages.basePage;
  await basePage.waitForDialog();
});

When('等待弹窗关闭', async ({ pages }) => {
  const basePage = pages.basePage;
  await basePage.waitForDialogClosed();
});

When('等待下拉选项加载', async ({ pages }) => {
  const basePage = pages.basePage;
  await basePage.waitSelectReady();
});

When('截图 {string}', async ({ pages }, name) => {
  const basePage = pages.basePage;
  await basePage.screenshot(name);
});

Then('页面 URL 应包含 {string}', async ({ page }, text) => {
  expect(page.url()).toContain(text);
});

Then('页面标题应包含 {string}', async ({ page }, text) => {
  await expect(page).toHaveTitle(new RegExp(text));
});

Then('应该看到 {string}', async ({ page }, text) => {
  await expect(page.getByText(text).first()).toBeVisible({ timeout: 10000 });
});

Then('不应该看到 {string}', async ({ page }, text) => {
  await expect(page.getByText(text)).not.toBeVisible({ timeout: 10000 });
});

Then('{string} 按钮应可见', async ({ page }, name) => {
  await expect(page.getByRole('button', { name })).toBeVisible({ timeout: 10000 });
});

Then('{string} 按钮应不可见', async ({ page }, name) => {
  await expect(page.getByRole('button', { name })).not.toBeVisible({ timeout: 10000 });
});

Then('弹窗应包含 {string}', async ({ pages }, text) => {
  const basePage = pages.basePage;
  const dialog = basePage.getDialog();
  await expect(dialog.getByText(text)).toBeVisible({ timeout: 10000 });
});

Then('页面应包含 {string} 和 {string}', async ({ page }, text1, text2) => {
  await expect(page.getByText(text1).first()).toBeVisible({ timeout: 10000 });
  await expect(page.getByText(text2).first()).toBeVisible({ timeout: 10000 });
});

// ─── 页面刷新步骤 ──────────────────────────────────────────────

When('刷新页面', async ({ page }) => {
  await page.reload({ waitUntil: 'domcontentloaded' });
  // 网络空闲带超时兜底：存在轮询接口时不阻塞满 navigationTimeout
  await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => {});
  await page.waitForTimeout(2000);
});
