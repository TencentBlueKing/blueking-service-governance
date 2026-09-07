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
/** TC-15 构建日志与状态：业务语义步骤。 */
import { Given, Then, When } from '../fixtures/fixtures';

Given('构建日志接口返回流式日志', async ({ pages }) => {
  await pages.appDetailPage.setupBuildLogStreamMock();
});

Given('构建记录接口返回失败状态测试数据', async ({ pages }) => {
  await pages.appDetailPage.setupBuildRecordsWithFailedStatusesMock();
});

When('我打开首行构建日志', async ({ pages }) => {
  await pages.appDetailPage.openFirstBuildLog();
});

Then('构建日志侧栏应展示成功状态和日志内容', async ({ pages }) => {
  await pages.appDetailPage.expectBuildLogPanelVisible('构建成功');
});

Then('构建日志侧栏应展示失败状态和日志内容', async ({ pages }) => {
  await pages.appDetailPage.expectBuildLogPanelVisible('构建失败');
});

Then('构建失败记录应可见', async ({ pages }) => {
  await pages.appDetailPage.expectBuildFailedRecordsVisible();
});
