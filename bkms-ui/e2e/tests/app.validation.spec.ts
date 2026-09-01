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
import { expect } from '@playwright/test';

import { createTrpcApp } from '../actions/deploy.action';
import { fillFormByType } from '../actions/form.action';
import { TrpcFormCases } from '../data/trpc-data';
import { test } from '../fixtures/fixtures';

TrpcFormCases.forEach(c => {
  test(`TRPC 应用创建校验 - ${c.title}`, async ({ pages }) => {
    await createTrpcApp({ appPage: pages.appPage }, 'TRPC');

    await fillFormByType(pages.basePage, 'TRPC', c.data);
    await pages.basePage.clickButton('下一步');

    expect(pages.basePage.getFormItemContent('应用名称')).toHaveText(c.error);
  });
});
