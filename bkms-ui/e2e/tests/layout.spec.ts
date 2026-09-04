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
import { expect } from '@playwright/test';

import { test } from '../fixtures/fixtures';

/** 高度塌陷判定阈值：正常渲染数百像素 vs 塌陷 0/个位数像素，两侧留足余量 */
const MIN_LAYOUT_HEIGHT_PX = 200;

/**
 * 布局高度回归测试：断言「真实浏览器渲染高度」而非 DOM 结构。
 *
 * 背景：高度塌陷类回归（如 #135 detail.vue 包裹层丢失 h-full、#139 编辑器宿主塌陷）
 * 曾尝试用静态高度链扫描防护，但误报调试成本高、需随布局变更持续维护，已废弃（stash 留底）。
 * 本用例改为断言真实布局结果：高度链上任何一环断裂，最终渲染高度都会塌陷至 0/个位数像素，
 * 与正常渲染（数百像素）差距悬殊，阈值两侧留足余量，无需感知具体布局实现。
 */
test.describe('布局高度回归', () => {
  test('观测数据页面监控 iframe 高度不塌陷', async ({ pages }) => {
    const { appDetailPage } = pages;
    await appDetailPage.gotoMenu('observation');

    // iframe 由 bizId→iframeUrl 异步链驱动 v-if 渲染；src 指向监控平台（跨域），
    // 内容加载失败不影响元素本身高度，因此只等元素挂载、不等内容
    const iframe = appDetailPage.getMonitorIframe();
    await iframe.waitFor({ state: 'visible', timeout: 30000 });

    // 轮询真实渲染高度至稳定（替代固定 sleep，天然覆盖「挂载 + 布局完成」）
    await expect
      .poll(async () => (await iframe.boundingBox())?.height ?? 0, {
        timeout: 10000,
        message: '监控 iframe 高度塌陷（高度链断裂）',
      })
      .toBeGreaterThan(MIN_LAYOUT_HEIGHT_PX);
  });

  test('应用配置-框架配置文件 Monaco 编辑器高度不塌陷', async ({ pages }) => {
    const { appDetailPage } = pages;
    // activeTab 与 URL query 双向同步（useUrlQuerySync），直达框架配置文件 tab
    await appDetailPage.gotoMenu('appConfig', { activeTab: 'framework-config' });

    const editor = appDetailPage.getFrameworkMonacoEditor();
    await editor.waitFor({ state: 'visible', timeout: 30000 });

    await expect
      .poll(async () => (await editor.boundingBox())?.height ?? 0, {
        timeout: 10000,
        message: 'Monaco 编辑器高度塌陷（高度链断裂）',
      })
      .toBeGreaterThan(MIN_LAYOUT_HEIGHT_PX);
  });
});
