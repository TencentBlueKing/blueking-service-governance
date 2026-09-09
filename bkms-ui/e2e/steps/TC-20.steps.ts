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
/** TC-20 北极星关联环境侧栏：业务语义步骤。 */
import { Then, When } from '../fixtures/fixtures';

When('我打开首条北极星配置的关联环境侧栏', async ({ pages }) => {
  await pages.appDetailPage.polaris.openPrimaryAssociatedEnvSideslider();
});

Then('北极星关联环境侧栏应展示环境统计', async ({ pages }) => {
  await pages.appDetailPage.polaris.expectAssociatedEnvSidesliderVisible();
});
