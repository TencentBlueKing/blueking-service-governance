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
/** TC-13 制品管理：只读查看步骤。 */
import { Given, Then, When } from '../fixtures/fixtures';

Given('我在当前应用的制品管理页', async ({ pages }) => {
  await pages.appDetailPage.gotoArtifactManagement();
});

Then('制品管理页应展示镜像列表和查询入口', async ({ pages }) => {
  await pages.appDetailPage.expectArtifactManagementVisible();
});

When('我展开制品列表首行', async ({ pages }) => {
  await pages.appDetailPage.expandFirstArtifactRow();
});

Then('制品列表首行详情应展示', async ({ pages }) => {
  await pages.appDetailPage.expectFirstArtifactRowExpanded();
});

When('我收起制品列表首行', async ({ pages }) => {
  await pages.appDetailPage.collapseFirstArtifactRow();
});

Then('制品列表首行详情应隐藏', async ({ pages }) => {
  await pages.appDetailPage.expectFirstArtifactRowCollapsed();
});

When('我按制品列表首行 Tag 查询', async ({ pages }) => {
  await pages.appDetailPage.searchArtifactsByFirstRowTag();
});

Then('制品列表应仅展示匹配的 Tag', async ({ pages }) => {
  await pages.appDetailPage.expectArtifactSearchResultMatches();
});

Then('制品列表首行详情应展示完整制品字段和部署记录状态', async ({ pages }) => {
  await pages.appDetailPage.expectFirstArtifactRowDetailVisible();
});
