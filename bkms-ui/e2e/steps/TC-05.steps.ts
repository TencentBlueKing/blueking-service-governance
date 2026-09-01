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
 * TC-05 环境变量查看与搜索：业务语义步骤。
 */
import { expect } from '@playwright/test';

import { Given, Then, When } from '../fixtures/fixtures';

Given('我在当前应用的环境变量页', async ({ pages }) => {
  const { appDetailPage, basePage } = pages;
  await appDetailPage.gotoAppConfig();
  await basePage.clickTab('环境变量');
});

When('我点击查看环境级变量', async ({ pages }) => {
  await pages.appDetailPage.openEnvLevelVarsSlider();
});

When('我在侧栏搜索环境变量 {string}', async ({ pages }, keyword: string) => {
  await pages.appDetailPage.fillSliderEnvVarSearch(keyword);
  await pages.basePage.waitForReady(1000);
});

When('我清空侧栏环境变量搜索', async ({ pages }) => {
  await pages.appDetailPage.clearSliderEnvVarSearch();
  await pages.basePage.waitForReady(1000);
});

Then('环境变量搜索框应可见', async ({ pages }) => {
  await pages.appDetailPage.expectEnvVarSearchVisible();
});

Then('环境级变量侧栏应可见', async ({ pages }) => {
  await pages.appDetailPage.expectEnvLevelVarsSliderVisible();
});

Then('侧栏环境变量搜索框应可见', async ({ pages }) => {
  await pages.appDetailPage.expectSliderEnvVarSearchVisible();
});

Then('侧栏环境变量表格应有数据', async ({ pages }) => {
  const count = await pages.appDetailPage.sliderEnvVarRows().count();
  expect(count).toBeGreaterThan(0);
});

Then('侧栏环境变量表格应无数据', async ({ pages }) => {
  const count = await pages.appDetailPage.sliderEnvVarRows().count();
  expect(count).toBe(0);
});
