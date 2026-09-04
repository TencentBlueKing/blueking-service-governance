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
 * 团队入口（e2e/tests + pnpm test:spec）。
 * 对偶 aitest 副本：tests/e2e/build-management-execute-build.spec.ts（个人本地 aitest run）。
 * 改断言时请两边同步，避免漂移。
 *
 * Fixture 约定（公开测试应用名，可进仓库）：
 * - e2e-build-pipeline：固定流水线构建（分支为 Input）
 * - e2e-trpc：固定源码仓库构建（分支为 Select）
 * 可用 BKMS_TEST_BUILD_PIPELINE_APP / BKMS_TEST_BUILD_REPO_APP 覆盖；空间来自 BKMS_TEST_DEFAULT_SPACE。
 */
import { type Page, type Response, expect } from '@playwright/test';

import { test } from '../fixtures/fixtures';

/** 推荐镜像 Tag API 路径片段（build-management → useRecommendTag） */
const RECOMMENDED_TAG_API = 'recommended-image-tag';

/** 流水线构建专用应用（无 repoAlias → Input） */
const DEFAULT_PIPELINE_APP = 'e2e-build-pipeline';
/** 源码仓库构建专用应用（有 repoAlias → Select） */
const DEFAULT_REPO_APP = 'e2e-trpc';
/** 构建管理路由中的应用类型段 */
const DEFAULT_APP_TYPE = 'trpc';

type BuildMode = 'pipeline' | 'repo';

/**
 * 打开弹层后校验构建模式，避免专用应用被改成另一种来源后静默错位。
 */
async function assertBranchMode(page: Page, mode: BuildMode) {
  const branchItem = branchFormItem(page);
  await expect(branchItem).toBeVisible();
  const select = branchItem.locator('.repo-ref-select');
  const app = resolveBuildApp(mode);

  if (mode === 'pipeline') {
    await expect(
      select,
      `应用 ${app} 应为流水线构建（分支 Input）。若已被改成源码仓库构建，请改回或设置 BKMS_TEST_BUILD_PIPELINE_APP`,
    ).toHaveCount(0);
    return;
  }

  await expect(
    select,
    `应用 ${app} 应为源码仓库构建（分支 Select）。若已被改成流水线构建，请改回或设置 BKMS_TEST_BUILD_REPO_APP`,
  ).toBeVisible();
}

/** 「代码分支」表单项容器 */
function branchFormItem(page: Page) {
  return page.locator('.bk-form-item').filter({ has: page.getByText('代码分支', { exact: true }) });
}

/**
 * 拼构建管理页 hash 路由：/#/{space}/app/{app}/{appType}/build
 */
function buildManageUrl(space: string, mode: BuildMode): string {
  if (!space?.trim()) {
    throw new Error('缺少空间：请在 e2e/.env.test 配置 BKMS_TEST_DEFAULT_SPACE');
  }
  const app = resolveBuildApp(mode);
  const appType = process.env.BKMS_TEST_BUILD_APP_TYPE?.trim() || DEFAULT_APP_TYPE;
  return `/#/${space.trim()}/app/${app}/${appType}/build`;
}

/** 打开「执行构建」Popover 并等待「执行配置」表单出现 */
async function openExecuteBuildPopover(page: Page, tagApiPending?: Promise<Response>) {
  // 排除镜像仓库模式下的 disabled「执行构建」按钮
  const executeBtn = page.getByRole('button', { name: '执行构建' }).filter({ hasNot: page.locator('[disabled]') });
  await expect(executeBtn).toBeVisible();
  await executeBtn.click();

  const configTitle = page.getByText('执行配置', { exact: true });
  // Popover 为 manual 触发，偶发首次 click 未展开时再点一次
  try {
    await expect(configTitle).toBeVisible({ timeout: 3000 });
  } catch {
    await executeBtn.click();
    await expect(configTitle).toBeVisible();
  }

  if (tagApiPending) {
    await tagApiPending;
  }
}

/** 解析可覆盖的约定应用名 */
function resolveBuildApp(mode: BuildMode): string {
  if (mode === 'pipeline') {
    return process.env.BKMS_TEST_BUILD_PIPELINE_APP?.trim() || DEFAULT_PIPELINE_APP;
  }
  return process.env.BKMS_TEST_BUILD_REPO_APP?.trim() || DEFAULT_REPO_APP;
}

/** 「版本号（镜像 Tag）」输入框 */
function tagInput(page: Page) {
  return page.getByPlaceholder(/请输入，例如v1\.0\.0-alpha\.1/);
}

/** 等待推荐 Tag 接口真实响应（替代 aitest apiRecorder） */
function waitForRecommendTag(page: Page) {
  return page.waitForResponse(response => response.url().includes(RECOMMENDED_TAG_API));
}

/**
 * 构建管理 - 执行构建弹层回归
 * 覆盖 build-management.vue / repo-ref-select.vue 集成交互。
 */
test.describe('构建管理-执行构建', () => {
  test.describe.configure({ mode: 'serial' });

  test('流水线模式：代码分支为可编辑 Input 而非 Select', async ({ page, testConfig }) => {
    await page.goto(buildManageUrl(testConfig.space, 'pipeline'));
    await openExecuteBuildPopover(page);
    await assertBranchMode(page, 'pipeline');

    const branchInput = branchFormItem(page).locator('input');
    await expect(branchInput).toBeVisible();
    await expect(branchInput).toBeEnabled();

    await page.getByRole('button', { name: '取消' }).click();
    await expect(page.getByText('执行配置', { exact: true })).toBeHidden();
  });

  test('打开执行构建弹层后版本号自动填充', async ({ page, testConfig }) => {
    await page.goto(buildManageUrl(testConfig.space, 'pipeline'));
    await expect(
      page.getByRole('button', { name: '执行构建' }).filter({ hasNot: page.locator('[disabled]') }),
    ).toBeVisible();

    const tagApiPending = waitForRecommendTag(page);
    await openExecuteBuildPopover(page, tagApiPending);
    await assertBranchMode(page, 'pipeline');

    const tagField = tagInput(page);
    await expect(tagField).toBeVisible();
    await expect(tagField).not.toHaveValue('');

    await page.getByRole('button', { name: '取消' }).click();
    await expect(page.getByText('执行配置', { exact: true })).toBeHidden();
  });

  test('代码仓库模式：代码分支为 Select 且版本号自动填充', async ({ page, testConfig }) => {
    await page.goto(buildManageUrl(testConfig.space, 'repo'));
    await expect(
      page.getByRole('button', { name: '执行构建' }).filter({ hasNot: page.locator('[disabled]') }),
    ).toBeVisible();

    const tagApiPending = waitForRecommendTag(page);
    await openExecuteBuildPopover(page, tagApiPending);
    await assertBranchMode(page, 'repo');

    const tagField = tagInput(page);
    await expect(tagField).toBeVisible();
    await expect(tagField).not.toHaveValue('');
  });

  test('流水线模式：分支 Input 立即 trim；仅实质变化才请求推荐 Tag', async ({ page, testConfig }) => {
    await page.goto(buildManageUrl(testConfig.space, 'pipeline'));
    await expect(
      page.getByRole('button', { name: '执行构建' }).filter({ hasNot: page.locator('[disabled]') }),
    ).toBeVisible();

    const openTagPending = waitForRecommendTag(page);
    await openExecuteBuildPopover(page, openTagPending);
    await assertBranchMode(page, 'pipeline');

    const branchInput = branchFormItem(page).locator('input');
    await expect(branchInput).toBeVisible();

    const currentBranch = (await branchInput.inputValue()).trim();
    const baselineBranch = currentBranch || 'master';
    if (!currentBranch) {
      const seedTagPending = waitForRecommendTag(page);
      await branchInput.fill(baselineBranch);
      await expect(branchInput).toHaveValue(baselineBranch);
      await seedTagPending;
    }

    await test.step('仅首尾空格：防抖窗口内不新增推荐 Tag 请求', async () => {
      let recommendTagHits = 0;
      const onResponse = (response: Response) => {
        if (response.url().includes(RECOMMENDED_TAG_API)) {
          recommendTagHits += 1;
        }
      };
      page.on('response', onResponse);

      try {
        await branchInput.fill(`  ${baselineBranch}  `);

        const startedAt = Date.now();
        await expect
          .poll(() => {
            if (recommendTagHits > 0) return 'extra-hit';
            return Date.now() - startedAt >= 700 ? 'stable' : 'waiting';
          })
          .toBe('stable');
      } finally {
        page.off('response', onResponse);
      }
    });

    await test.step('分支实质变化：绑定值为 trim 结果，并触发推荐 Tag 请求', async () => {
      const nextBranch = `${baselineBranch}-e2e-trim`;
      const changeTagPending = waitForRecommendTag(page);
      await branchInput.fill(`  ${nextBranch}  `);
      await expect(branchInput).toHaveValue(nextBranch);
      await changeTagPending;
    });

    await test.step('清空分支：防抖窗口内不请求推荐 Tag', async () => {
      let recommendTagHits = 0;
      const onResponse = (response: Response) => {
        if (response.url().includes(RECOMMENDED_TAG_API)) {
          recommendTagHits += 1;
        }
      };
      page.on('response', onResponse);

      try {
        await branchInput.fill('');
        await expect(branchInput).toHaveValue('');

        const startedAt = Date.now();
        await expect
          .poll(() => {
            if (recommendTagHits > 0) return 'extra-hit';
            return Date.now() - startedAt >= 700 ? 'stable' : 'waiting';
          })
          .toBe('stable');
      } finally {
        page.off('response', onResponse);
      }
    });

    await page.getByRole('button', { name: '取消' }).click();
    await expect(page.getByText('执行配置', { exact: true })).toBeHidden();
  });
});
