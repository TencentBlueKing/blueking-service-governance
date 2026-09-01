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
 * TC-04 修改部署资源规格：业务语义步骤。
 */
import { editResourceSpec, resetResourceToDefault } from '../actions/app-config.action';
import { Given, Then, When } from '../fixtures/fixtures';
import { transformFormData } from '../utils/form';

Given('我在当前应用的测试环境部署配置页', async ({ pages }) => {
  const { appDetailPage } = pages;
  await appDetailPage.gotoAppConfig();
  await appDetailPage.selectConfigFirstTestEnv();
});

When('我修改资源规格', async ({ pages, userData }, dataTable) => {
  const transformed = transformFormData(dataTable.rowsHash());
  await editResourceSpec({ appDetailPage: pages.appDetailPage, basePage: pages.basePage }, transformed);
  userData.resourceSpec = transformed;
});

When('我恢复默认资源规格配置', async ({ pages }) => {
  await resetResourceToDefault({ appDetailPage: pages.appDetailPage });
});

Then('资源规格应处于查看态', async ({ pages }) => {
  await pages.appDetailPage.expectResourceInViewMode();
});

Then('资源规格区域应包含 {string}', async ({ pages }, text: string) => {
  await pages.appDetailPage.expectResourceContains(text);
});
