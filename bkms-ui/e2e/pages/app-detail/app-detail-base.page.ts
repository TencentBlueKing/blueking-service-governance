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
import { type Response, type Route, expect } from '@playwright/test';

import BasePage from '../base.page';

/**
 * 应用详情各业务域 Page Object 的公共基类。
 *
 * 承载跨域共享的能力：
 * - 应用详情二级菜单路由导航（`/:space/app/:name/:type/:menuName`）
 * - 接口响应断言与 route mock 填充工具
 * - Sideslider 关闭与收起稳定等待
 *
 * 各业务域（deploy / appConfig / appSpec / buildConfig / buildManagement / artifact）
 * 继承本类，最终由 `pages/app-detail.page.ts` 组合暴露给 steps / actions 使用。
 */
export default class AppDetailBase extends BasePage {
  protected async assertApiResponseOk(response: Response, action: string) {
    if (response.ok()) return;

    let body = '';
    try {
      body = await response.text();
    } catch {
      body = '<响应体读取失败>';
    }

    throw new Error(`${action}接口请求失败：${response.status()} ${response.url()}\n${body}`);
  }

  protected assertAppConfigured() {
    if (!this.config.app) {
      throw new Error(
        'testConfig.app 未配置：请配置 BKMS_TEST_DEFAULT_APP / BKMS_TEST_TRPC_APP / BKMS_TEST_HELM_APP，' +
          '或在当前空间创建类型匹配且名称以 e2e- 开头的测试应用，也可在 Scenario 打 @app:<name> 标签指定应用名称',
      );
    }
  }

  protected async closeVisibleSideslider(title?: string) {
    const titleLocator = title ? this.page.getByText(title, { exact: true }).last() : null;
    if (titleLocator && !(await titleLocator.isVisible().catch(() => false))) return;

    const closeButton = this.page.locator('.bk-sideslider-close:visible').last();
    if (await closeButton.isVisible().catch(() => false)) {
      await closeButton.click();
    } else {
      await this.page.keyboard.press('Escape');
    }
    if (titleLocator) {
      await expect(titleLocator)
        .toBeHidden({ timeout: 10000 })
        .catch(() => null);
    }
    await this.waitForSidesliderSettled();
  }

  protected async fulfillJson(route: Route, data: unknown, status = 200) {
    await route.fulfill({
      body: JSON.stringify(data),
      contentType: 'application/json',
      status,
    });
  }

  /**
   * 通用：进入应用详情某个二级菜单（遵循 `/:space/app/:name/:type/:menuName` 路由）
   *
   * 页面就绪不使用固定 sleep：domcontentloaded 后等待网络空闲（带超时兜底），
   * 内容加载由各域 goto* 方法的显式元素断言兜底。
   */
  async gotoMenu(menuName: string, query?: Record<string, string>) {
    this.assertAppConfigured();
    const search = query ? `?${new URLSearchParams(query).toString()}` : '';
    await this.page.goto(`/#/${this.config.space}/app/${this.config.app}/${this.config.appType}/${menuName}${search}`, {
      waitUntil: 'domcontentloaded',
    });
  }

  /** 等待 Sideslider 完全收起（隐藏即认为收起，网络空闲兜底数据刷新） */
  protected async waitForSidesliderSettled() {
    await this.page
      .waitForSelector('.bk-sideslider .bk-modal-body', { state: 'hidden', timeout: 5000 })
      .catch(() => null);
    await this.safeWaitForNetworkIdle();
  }
}
