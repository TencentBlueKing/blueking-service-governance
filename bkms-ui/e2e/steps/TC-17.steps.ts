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
/** TC-17 部署管理实例列表只读能力：业务语义步骤。 */
import { Given, Then, When } from '../fixtures/fixtures';

Given('部署实例列表接口返回只读测试数据', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.setupDeploymentInstanceMock();
});

When('我打开首行实例日志', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.openFirstInstanceLogSideslider();
});

When('我关闭实例日志侧栏', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.closeInstanceLogSideslider();
});

When('我打开首行实例灰度弹窗', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.openFirstInstanceGrayDialog();
});

When('我关闭实例操作弹窗', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.closeInstanceActionDialog();
});

When('我打开首行实例权重弹窗', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.openFirstInstanceWeightDialog();
});

When('我选中首个部署实例', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.selectFirstDeploymentInstance();
});

When('我打开部署实例管理命令侧栏', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.openSelectedInstanceAdminCommandSideslider();
});

When('我打开选中部署实例的删除确认弹窗', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.openSelectedDeploymentInstanceDeleteDialog();
});

When('我关闭实例删除确认弹窗', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.closeDeploymentInstanceDeleteDialog();
});

Then('实例列表应展示关键字段、行操作入口并建立 SSE 订阅', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.expectDeploymentInstanceListReadonlyVisible();
});

Then('实例日志侧栏应展示日志内容', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.expectInstanceLogSidesliderVisible();
});

Then('灰度实例弹窗应展示镜像选择', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.expectInstanceGrayDialogVisible();
});

Then('调整权重弹窗应展示权重表单', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.expectInstanceWeightDialogVisible();
});

Then('实例批量操作栏应展示操作入口', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.expectDeploymentInstanceBatchToolbarVisible();
});

Then('管理命令侧栏应展示已选实例和命令选项', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.expectInstanceAdminCommandSidesliderVisible();
});

Then('删除实例确认弹窗应展示选中的实例信息', async ({ pages }) => {
  await pages.appDetailPage.deployInstance.expectDeploymentInstanceDeleteDialogVisible();
});
