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
 * TC-09 开发模式：业务语义步骤。
 */
import {
  cancelEnableDevMode,
  disableDevMode,
  enableDevMode,
  ensureDevModeDisabled,
  reloadAndSelectTestEnv,
} from '../actions/app-config.action';
import { Then, When } from '../fixtures/fixtures';

When('我尝试开启开发模式后取消', async ({ pages }) => {
  await cancelEnableDevMode({ appDetailPage: pages.appDetailPage });
});

When('我确认关闭开发模式', async ({ pages }) => {
  await disableDevMode({ appDetailPage: pages.appDetailPage });
});

When('我确认开启开发模式', async ({ pages }) => {
  await enableDevMode({ appDetailPage: pages.appDetailPage });
});

When('我确保开发模式已关闭', async ({ pages }) => {
  await ensureDevModeDisabled({ appDetailPage: pages.appDetailPage });
});

When('我刷新页面并切换到测试环境', async ({ pages }) => {
  await reloadAndSelectTestEnv({ appDetailPage: pages.appDetailPage });
});

Then('开发模式区域应展示关闭状态', async ({ pages }) => {
  await pages.appDetailPage.expectDevModeSectionVisible();
  await pages.appDetailPage.expectDevModeDisabled();
});

Then('开发模式应保持关闭', async ({ pages }) => {
  await pages.appDetailPage.expectDevModeDisabled();
});

Then('开发模式应展示开启后的操作步骤', async ({ pages }) => {
  await pages.appDetailPage.expectDevModeEnabledStepsVisible();
});
