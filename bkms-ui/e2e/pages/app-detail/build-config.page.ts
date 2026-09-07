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

type BuilderConfigSourceName = '代码仓库' | '流水线' | '源码仓库' | '镜像仓库';

type BuilderConfigSourceType = 'codeRepository' | 'imageRegistry' | 'pipeline';

/**
 * 构建配置域 Page Object（基本信息页）。
 *
 * 覆盖：构建配置卡片查看态、编辑构建配置 Sideslider
 * （来源切换：代码仓库 / 镜像仓库 / 流水线）、必填校验与保存。
 */
export default class BuildConfigPage extends AppDetailBase {
  private builderConfigInvalidFieldLabel = '';

  private builderConfigInvalidSubmitted = false;

  private builderConfigOriginalDefaultBranch = '';

  private builderConfigOriginalImageRegistryName = '';

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

  private getBuilderConfigSourceButton(name: BuilderConfigSourceName) {
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

  private async isBuilderConfigSourceDisabled(name: BuilderConfigSourceName) {
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

  private waitForBuilderConfigSaveResponse() {
    return this.page.waitForResponse(
      response => response.request().method() === 'PUT' && response.url().includes('/build-configs'),
      { timeout: 30000 },
    );
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

  /** 获取构建配置卡片（BkmsContent 包裹块） */
  getBuildConfigSection(): Locator {
    return this.page.locator('.bkms-content').filter({ hasText: '构建配置' }).first();
  }

  /** 进入基本信息页（基本信息 = 二级菜单 key: info） */
  async gotoBaseInfo() {
    await this.gotoMenu('info');
    await this.expectBuilderConfigSectionVisible();
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
    await this.expectBuilderConfigSectionVisible();
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
    return true;
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
}
