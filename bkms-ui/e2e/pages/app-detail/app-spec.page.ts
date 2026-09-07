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
 * 应用运行配置（app-spec）域 Page Object。
 *
 * 覆盖应用配置页内的四类配置卡片：
 * - 健康探针（存活/就绪/启动）
 * - 生命周期（preStop 命令）
 * - 元数据配置（Labels / Annotations）
 * - 更新策略
 */
export default class AppSpecPage extends AppDetailBase {
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

  private getHealthProbeCard(label: string): Locator {
    return this.getHealthProbeSection().locator('.probe-view-section, .probe-card').filter({ hasText: label }).first();
  }

  private getHealthProbeEditSection(label: string): Locator {
    return this.getHealthProbeSection().locator('.probe-card').filter({ hasText: label }).first();
  }

  private getHealthProbeInput(section: Locator, label: string) {
    const formItem = section.locator('.bk-form-item').filter({ hasText: label }).first();
    return formItem.getByRole('spinbutton').or(formItem.getByRole('textbox')).first();
  }

  private getHealthProbeSection(): Locator {
    return this.page.locator('.bkms-content').filter({ hasText: '健康探针' }).first();
  }

  private getHealthProbeViewSection(label: string): Locator {
    return this.getHealthProbeSection().locator('.probe-view-section').filter({ hasText: label }).first();
  }

  private getLifecyclePreStopSwitcher() {
    return this.getLifecycleSection().locator('.bk-switcher').first();
  }

  private getLifecycleSection(): Locator {
    return this.page.locator('.bkms-content').filter({ hasText: '生命周期' }).first();
  }

  private getMetadataCard(label: string): Locator {
    return this.getMetadataSection().locator('.metadata-card').filter({ hasText: label }).first();
  }

  private getMetadataSection(): Locator {
    return this.page.locator('.bkms-content').filter({ hasText: '元数据配置' }).first();
  }

  private getUpdateStrategyInput(label: string) {
    return this.getUpdateStrategySection()
      .locator('.bk-form-item')
      .filter({ hasText: label })
      .getByRole('textbox')
      .first();
  }

  private getUpdateStrategySection(): Locator {
    return this.page.locator('.bkms-content').filter({ hasText: '更新策略' }).first();
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

  /** 点击健康探针编辑态的「保存」按钮（校验断言由调用方轮询等待） */
  async clickHealthProbeSave(label: string) {
    await this.getHealthProbeEditSection(label).getByRole('button', { name: '保存' }).click();
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

  /** 点击生命周期编辑态的「保存」按钮（校验断言由调用方轮询等待） */
  async clickLifecycleSave() {
    await this.getLifecycleSection().getByRole('button', { name: '保存' }).click();
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

  /** 点击元数据配置卡片的「保存」按钮（校验断言由调用方轮询等待） */
  async clickMetadataSave(label: string) {
    await this.getMetadataCard(label).getByRole('button', { name: '保存' }).click();
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
  }

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

  /** 点击更新策略编辑态的「保存」按钮（校验断言由调用方轮询等待） */
  async clickUpdateStrategySave() {
    await this.getUpdateStrategySection().getByRole('button', { name: '保存' }).click();
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
  }

  /** 确保生命周期处于编辑态 */
  async ensureLifecycleEditMode() {
    const saveButton = this.getLifecycleSection().getByRole('button', { name: '保存' });
    if (await saveButton.isVisible().catch(() => false)) {
      return;
    }

    await this.clickLifecycleEdit();
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

  /** 填写更新策略配置 */
  async fillUpdateStrategyConfig(config: UpdateStrategyConfig) {
    await this.fillUpdateStrategyInput('最大超出数量', config.maxSurge);
    await this.fillUpdateStrategyInput('最大不可用数量', config.maxUnavailable);
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

  /** 在元数据配置卡片中切换到文本模式 */
  async selectMetadataTextMode(label: string) {
    const card = this.getMetadataCard(label);
    await card.getByText('文本模式').click();
    await card.getByRole('textbox').first().waitFor({ state: 'visible', timeout: 10000 });
  }
}
