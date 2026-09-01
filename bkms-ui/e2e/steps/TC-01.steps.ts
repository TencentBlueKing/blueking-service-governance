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
 * TC-01 部署应用：业务语义步骤。
 * UI 原子操作下沉到 AppDetailPage；业务流程封装在 actions/deploy.action.ts。
 */
import { deployApp, expectInstanceCount, expectInstanceReadyCount } from '../actions/deploy.action';
import { Given, test, Then, When } from '../fixtures/fixtures';

Given('我在当前应用的部署管理页', async ({ pages }) => {
  await pages.appDetailPage.gotoDeployment();
});

Given('当前应用已部署则跳过本用例', async ({ pages }) => {
  const deployed = await pages.appDetailPage.isDeployed();
  test.skip(deployed, '当前默认应用已部署，跳过未部署空态校验');
});

Given('当前应用未部署则先部署 {int} 个实例', async ({ pages }, replicas: number) => {
  if (await pages.appDetailPage.isDeployed()) return;

  await deployApp({ appDetailPage: pages.appDetailPage }, replicas);
  await expectInstanceReadyCount({ appDetailPage: pages.appDetailPage }, replicas, { timeoutMs: 180000 });
});

When('我立即部署 {int} 个实例', async ({ pages }, replicas: number) => {
  await deployApp({ appDetailPage: pages.appDetailPage }, replicas);
});

Then('应用应处于未部署状态', async ({ pages }) => {
  await pages.appDetailPage.expectUninstalled();
});

Then('实例列表应至少出现 {int} 个 Pod，最多等待 {int} 秒', async ({ pages }, expected: number, timeoutSec: number) => {
  await expectInstanceCount({ appDetailPage: pages.appDetailPage }, expected, {
    timeoutMs: timeoutSec * 1000,
  });
});

Then(
  '实例列表应出现 {int} 个 Running 且 Healthy 的 Pod，最多等待 {int} 秒',
  async ({ pages }, expected: number, timeoutSec: number) => {
    await expectInstanceReadyCount({ appDetailPage: pages.appDetailPage }, expected, {
      timeoutMs: timeoutSec * 1000,
    });
  },
);
