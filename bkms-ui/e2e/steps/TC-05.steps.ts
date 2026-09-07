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
import { Given, Then, When } from '../fixtures/fixtures';

Given('我在当前应用的环境变量页', async ({ pages }) => {
  const { appDetailPage, basePage } = pages;
  await appDetailPage.appConfig.gotoAppConfig();
  await basePage.clickTab('环境变量');
});

When('我点击查看环境级变量', async ({ pages }) => {
  await pages.appDetailPage.appConfig.openEnvLevelVarsSlider();
});

When('我在侧栏搜索环境变量 {string}', async ({ pages }, keyword: string) => {
  await pages.appDetailPage.appConfig.fillSliderEnvVarSearch(keyword);
});

When('我清空侧栏环境变量搜索', async ({ pages }) => {
  await pages.appDetailPage.appConfig.clearSliderEnvVarSearch();
});

Then('环境变量搜索框应可见', async ({ pages }) => {
  await pages.appDetailPage.appConfig.expectEnvVarSearchVisible();
});

Then('环境级变量侧栏应可见', async ({ pages }) => {
  await pages.appDetailPage.appConfig.expectEnvLevelVarsSliderVisible();
});

Then('侧栏环境变量搜索框应可见', async ({ pages }) => {
  await pages.appDetailPage.appConfig.expectSliderEnvVarSearchVisible();
});

Then('侧栏环境变量表格应有数据', async ({ pages }) => {
  await pages.appDetailPage.appConfig.expectSliderEnvVarRowsVisible();
});

Then('侧栏环境变量表格应无数据', async ({ pages }) => {
  await pages.appDetailPage.appConfig.expectSliderEnvVarRowsEmpty();
});
