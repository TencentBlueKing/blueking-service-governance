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
 * TC-03 移除部署：业务语义步骤。
 *
 * 「我在当前应用的部署管理页」「当前应用未部署则先部署 N 个实例」
 * 与「应用应处于未部署状态」已在 TC-01.steps.ts 中注册，此处仅注册 TC-03 特有步骤。
 */
import { expectRemoveDeployConfirmationGuard, removeDeploy } from '../actions/deploy.action';
import { Then, When } from '../fixtures/fixtures';

Then('移除部署确认应要求输入环境名称', async ({ pages }) => {
  await expectRemoveDeployConfirmationGuard({ appDetailPage: pages.appDetailPage });
});

When('我执行移除部署', async ({ pages }) => {
  await removeDeploy({ appDetailPage: pages.appDetailPage });
});
