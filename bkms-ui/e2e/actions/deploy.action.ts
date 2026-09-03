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
import { type Page, expect } from '@playwright/test';

import { type FormType, fillFormByType } from './form.action';

import type AppDetailPage from '../pages/app-detail.page';
import type AppPage from '../pages/app.page';
import type BasePage from '../pages/base.page';

// ─── 应用创建相关 ──────────────────────────────────────────────────

/** 自动调节：打开 Sideslider → 选择自动调节 → 设置最小/最大副本和 CPU 阈值 → 确认 */
export async function configureAutoScale(
  { appDetailPage }: { appDetailPage: AppDetailPage },
  {
    cpuUtilization,
    maxReplicas,
    minReplicas,
  }: {
    cpuUtilization: number;
    maxReplicas: number;
    minReplicas: number;
  },
) {
  await appDetailPage.openScaleSideslider();
  await appDetailPage.selectAutoScaleMode();
  await appDetailPage.setAutoScaleReplicas({ maxReplicas, minReplicas });
  await appDetailPage.setAutoScaleCpuUtilization(cpuUtilization);
  await appDetailPage.submitAutoScale();
}

/** 创建 TRPC 应用：进入应用列表 → 点「创建应用」→ 选应用类型 → 下一步 */
export async function createTrpcApp({ appPage }: { appPage: AppPage }, appType: string) {
  await appPage.goto();
  await appPage.clickButton('创建应用');
  await appPage.chooseAppType(appType);
  await appPage.clickButton('下一步');
}

/**
 * 执行立即部署：打开 Sideslider → 设置实例数 → 选第一个可用镜像 → 提交。
 * 关闭 Sideslider 后不做部署完成断言，断言交由 SSE 驱动的实例列表等待完成。
 */
export async function deployApp({ appDetailPage }: { appDetailPage: AppDetailPage }, replicas: number) {
  await appDetailPage.clickQuicklyDeploy();
  await appDetailPage.setDeployReplicas(replicas);
  await appDetailPage.selectFirstImageTag();
  await appDetailPage.submitSideslider();
}

// ─── 部署管理（部署/扩缩容/移除） ───────────────────────────────────

/** 等待 SSE 同步后的实例数稳定到期望值（达不到则抛错） */
export async function expectInstanceCount(
  { appDetailPage }: { appDetailPage: AppDetailPage },
  expected: number,
  opts?: { timeoutMs?: number },
) {
  const ok = await appDetailPage.waitForInstanceCount(expected, {
    timeout: opts?.timeoutMs ?? 180000,
  });
  if (!ok) {
    throw new Error(`期望实例数达到 ${expected}，等待 SSE 更新超时未满足`);
  }
}

/** 等待 SSE 同步后的实例数精确匹配，并要求每个 Pod 都处于 Running/Healthy */
export async function expectInstanceReadyCount(
  { appDetailPage }: { appDetailPage: AppDetailPage },
  expected: number,
  opts?: { timeoutMs?: number },
) {
  const ok = await appDetailPage.waitForInstanceReadyCount(expected, {
    timeout: opts?.timeoutMs ?? 180000,
  });
  if (!ok) {
    throw new Error(`期望实例数精确达到 ${expected} 且全部 Running/Healthy，等待 SSE 更新超时未满足`);
  }
}

/** 验证移除部署确认弹窗要求输入环境名称 */
export async function expectRemoveDeployConfirmationGuard({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.openMoreMenu();
  await appDetailPage.clickRemoveDeploy();
  await appDetailPage.expectRemoveDeployConfirmationGuard();
}

/**
 * 填写应用创建的两段式表单：基本信息 → 下一步 → 参数配置。
 * 最终的「创建」按钮点击由 step 独立触发，便于分步断言。
 */
export async function fillAppForm<K extends FormType>(
  { basePage }: { basePage: BasePage },
  formType: K,
  data: Record<string, unknown>,
) {
  await fillFormByType(basePage, formType, data as Parameters<typeof fillFormByType<K>>[2]);
  await basePage.clickButton('下一步');
  await fillFormByType(basePage, formType, data as Parameters<typeof fillFormByType<K>>[2]);
}

/** 移除部署：更多菜单 → 移除部署 → 输入环境名 → 删除 */
export async function removeDeploy({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.openMoreMenu();
  await appDetailPage.clickRemoveDeploy();
  await appDetailPage.confirmRemoveDeploy();
}

/** 手动扩缩容：打开 Sideslider → 选择手动调节 → 设置副本数 → 确认 */
export async function scaleAppManually({ appDetailPage }: { appDetailPage: AppDetailPage }, replicas: number) {
  await appDetailPage.openScaleSideslider();
  await appDetailPage.selectManualScaleMode();
  await appDetailPage.setScaleReplicas(replicas);
  await appDetailPage.submitManualScale();
}

/** 等待应用创建成功 */
export async function waitForAppCreated({ page }: { page: Page }) {
  await expect(page.getByText('应用创建成功').first()).toBeVisible({ timeout: 10000 });
}
