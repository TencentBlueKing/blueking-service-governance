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
/** TC-19 北极星配置增删改：业务语义步骤。 */
import { Then, When } from '../fixtures/fixtures';

When('我添加从现有引入北极星配置', async ({ pages }) => {
  await pages.appDetailPage.polaris.createImportedPolarisConfig();
});

When('我编辑首条北极星配置', async ({ pages }) => {
  await pages.appDetailPage.polaris.editPrimaryPolarisConfig();
});

When('我删除首条北极星配置', async ({ pages }) => {
  await pages.appDetailPage.polaris.deletePrimaryPolarisConfig();
});

Then('北极星新增请求参数应正确', async ({ pages }) => {
  await pages.appDetailPage.polaris.expectCreateRequestParams();
});

Then('北极星编辑请求参数应正确', async ({ pages }) => {
  await pages.appDetailPage.polaris.expectPatchRequestParams();
});

Then('北极星删除请求应命中目标配置', async ({ pages }) => {
  await pages.appDetailPage.polaris.expectDeleteRequestTarget();
});
