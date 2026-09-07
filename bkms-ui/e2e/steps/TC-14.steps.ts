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
/** TC-14 构建管理记录列表：业务语义步骤。 */
import { Given, Then, When } from '../fixtures/fixtures';

Given('构建记录接口返回只读测试数据', async ({ pages }) => {
  await pages.appDetailPage.setupBuildRecordsReadonlyMock();
});

Given('构建记录接口先失败后恢复', async ({ pages }) => {
  await pages.appDetailPage.setupBuildRecordsErrorRecoveryMock();
});

Given('构建记录接口返回分页测试数据', async ({ pages }) => {
  await pages.appDetailPage.setupBuildRecordsPaginationMock();
});

Given('我在当前应用的构建管理页', async ({ pages }) => {
  await pages.appDetailPage.gotoBuildManagement();
});

Given('我在当前应用的构建管理页出现构建记录异常', async ({ pages }) => {
  await pages.appDetailPage.gotoMenu('build');
});

When('我按约定关键字搜索构建记录', async ({ pages }) => {
  await pages.appDetailPage.searchBuildRecordsByKnownKeyword();
});

When('我清空构建记录搜索', async ({ pages }) => {
  await pages.appDetailPage.clearBuildRecordSearch();
});

When('我刷新构建记录异常空态', async ({ pages }) => {
  await pages.appDetailPage.refreshBuildRecordsFromError();
});

When('我切换构建记录到第二页', async ({ pages }) => {
  await pages.appDetailPage.changeBuildRecordsToSecondPage();
});

When('我切换构建记录每页条数为 20', async ({ pages }) => {
  await pages.appDetailPage.changeBuildRecordsPageSizeToTwenty();
});

Then('构建管理页应展示构建记录列表和查询入口', async ({ pages }) => {
  await pages.appDetailPage.expectBuildManagementVisible();
});

Then('构建记录搜索结果应匹配关键字', async ({ pages }) => {
  await pages.appDetailPage.expectBuildRecordSearchResultMatches();
});

Then('构建记录列表应恢复完整数据', async ({ pages }) => {
  await pages.appDetailPage.expectBuildRecordsRestored();
});

Then('构建记录页应展示数据获取异常', async ({ pages }) => {
  await pages.appDetailPage.expectBuildRecordsErrorVisible();
});

Then('构建记录第二页应展示对应数据', async ({ pages }) => {
  await pages.appDetailPage.expectBuildRecordsSecondPageVisible();
});

Then('构建记录每页 20 条应展示完整数据', async ({ pages }) => {
  await pages.appDetailPage.expectBuildRecordsPageSizeTwentyVisible();
});
