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
/** TC-18 部署管理实例列表特殊环境：业务语义步骤。 */
import { Given, Then, When } from '../fixtures/fixtures';

Given('联邦环境部署实例列表接口返回测试数据', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.setupDeploymentInstanceMock({ federation: true });
});

When('我打开应用关联特性环境侧栏', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.openFeatureEnvSideslider();
});

Then('联邦环境实例列表应通过轮询展示数据并禁用灰度', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.expectFederationDeploymentInstanceListVisible();
});

Then('特性环境侧栏应展示部署入口和销毁保护', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.expectFeatureEnvSidesliderVisible();
  await pages.appDetailPage.deployInstance.expectFeatureEnvDestroyGuardVisible();
});
