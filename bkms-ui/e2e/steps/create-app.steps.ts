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
 * 应用创建相关语义步骤。
 * 可被任何需要"先创建一个应用"的 .feature 复用，不绑定具体 TC。
 */
import { createTrpcApp, fillAppForm, waitForAppCreated } from '../actions/deploy.action';
import { Then, When } from '../fixtures/fixtures';
import { transformFormData } from '../utils/form';

When('开始创建 {string} 应用', async ({ pages }, appType) => {
  await createTrpcApp({ appPage: pages.appPage }, appType);
});

// 新增表单类型：
// 1. 在 e2e/data/<xxx>-data.ts 定义 Schema
// 2. 在 e2e/actions/form.action.ts 扩展 FormType 联合类型 + schemaMap
// 3. 此处复用 fillAppForm（基本信息 → 下一步 → 参数配置）
When('填写 TRPC 表单', async ({ pages, userData }, dataTable) => {
  const transformedData = transformFormData(dataTable.rowsHash());
  await fillAppForm({ basePage: pages.basePage }, 'TRPC', transformedData);
  userData.formData = transformedData;
});

When('提交 {string} 表单', async ({ pages }) => {
  await pages.basePage.clickButton('创建');
});

Then('{string} 应用创建成功', async ({ page }) => {
  await waitForAppCreated({ page });
});
