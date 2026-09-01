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
 * TC-10 生命周期配置：业务语义步骤。
 */
import {
  editLifecycleAndCancel,
  LIFECYCLE_CANCEL_COMMAND,
  LIFECYCLE_SAVED_COMMAND,
  LIFECYCLE_SAVED_GRACE_PERIOD,
  saveValidLifecycle,
  selectLifecycleCustomCommandOption,
  submitInvalidLifecycle,
} from '../actions/app-config.action';
import { Given, Then, When } from '../fixtures/fixtures';

Given('我在当前应用的默认部署配置页', async ({ pages }) => {
  const { appDetailPage } = pages;
  await appDetailPage.gotoAppConfig();
  await appDetailPage.selectConfigDefaultEnv();
});

When('我编辑生命周期后取消', async ({ pages }) => {
  await editLifecycleAndCancel({ appDetailPage: pages.appDetailPage });
});

When('我选择生命周期自定义命令选项', async ({ pages }) => {
  await selectLifecycleCustomCommandOption({ appDetailPage: pages.appDetailPage });
});

When('我保存有效生命周期配置', async ({ pages }) => {
  await saveValidLifecycle({ appDetailPage: pages.appDetailPage });
});

When('我提交无效生命周期配置', async ({ pages }) => {
  await submitInvalidLifecycle({ appDetailPage: pages.appDetailPage });
});

Then('生命周期不应包含取消测试命令', async ({ pages }) => {
  await pages.appDetailPage.expectLifecycleTextHidden(LIFECYCLE_CANCEL_COMMAND);
});

Then('生命周期区域应展示当前配置', async ({ pages }) => {
  await pages.appDetailPage.expectLifecycleSectionVisible();
  await pages.appDetailPage.expectLifecycleInViewMode();
});

Then('生命周期应展示自定义命令输入项', async ({ pages }) => {
  await pages.appDetailPage.expectLifecycleCustomCommandEditorVisible();
});

Then('生命周期应处于查看态', async ({ pages }) => {
  await pages.appDetailPage.expectLifecycleInViewMode();
});

Then('生命周期应展示必填校验提示', async ({ pages }) => {
  await pages.appDetailPage.expectLifecycleValidationVisible('必填项');
});

Then('生命周期应展示已保存配置', async ({ pages }) => {
  await pages.appDetailPage.expectLifecycleContains(LIFECYCLE_SAVED_COMMAND);
  await pages.appDetailPage.expectLifecycleContains(`${LIFECYCLE_SAVED_GRACE_PERIOD} 秒`);
});
