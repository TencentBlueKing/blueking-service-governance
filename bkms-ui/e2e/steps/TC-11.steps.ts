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
 * TC-11 tRPC 构建配置侧边栏：业务语义步骤。
 */
import { Given, Then, When } from '../fixtures/fixtures';

Given('我在 tRPC 应用的基本信息页', async ({ pages }) => {
  await pages.appDetailPage.gotoBaseInfo();
});

When('我保存有效构建配置', async ({ pages }) => {
  await pages.appDetailPage.saveValidBuilderConfig();
});

When('我打开构建配置编辑侧栏', async ({ pages }) => {
  await pages.appDetailPage.openBuilderConfigSideslider();
});

When('我取消构建配置编辑', async ({ pages }) => {
  await pages.appDetailPage.cancelBuilderConfigEdit();
});

When('我提交无效构建配置', async ({ pages, userData }) => {
  userData.builderConfigInvalidSubmitted = await pages.appDetailPage.submitInvalidBuilderConfig();
});

When('我在构建配置侧栏切换来源', async ({ pages }) => {
  await pages.appDetailPage.switchBuilderConfigSource();
});

Then('构建配置保存成功并关闭侧栏', async ({ pages }) => {
  await pages.appDetailPage.expectBuilderConfigSaveCompleted();
});

Then('构建配置侧栏应展示当前来源表单', async ({ pages }) => {
  await pages.appDetailPage.expectBuilderConfigCurrentSourceFormVisible();
});

Then('构建配置侧栏应展示必填校验提示', async ({ pages, userData }) => {
  if (userData.builderConfigInvalidSubmitted === false) {
    await pages.appDetailPage.expectBuilderConfigCurrentSourceFormVisible();
    return;
  }
  await pages.appDetailPage.expectBuilderConfigRequiredValidationVisible();
});

Then('构建配置区域应可见', async ({ pages }) => {
  await pages.appDetailPage.expectBuilderConfigSectionVisible();
});

Then('编辑构建配置侧栏应关闭', async ({ pages }) => {
  await pages.appDetailPage.expectBuilderConfigSidesliderClosed();
});

Then('编辑构建配置侧栏应可见', async ({ pages }) => {
  await pages.appDetailPage.expectBuilderConfigSidesliderVisible();
});
