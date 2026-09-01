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
 * TC-06 健康探针配置：业务语义步骤。
 */
import {
  editHealthProbeAndCancel,
  getHealthProbeConfig,
  HEALTH_PROBE_CLAMPED_PORT,
  saveValidHealthProbe,
  submitInvalidHealthProbe,
} from '../actions/app-config.action';
import { Then, When } from '../fixtures/fixtures';

Then('健康探针区域应展示三个探针卡片', async ({ pages }) => {
  await pages.appDetailPage.expectHealthProbeCardsVisible();
});

When('我编辑 {string} 后取消', async ({ pages }, label: string) => {
  await editHealthProbeAndCancel({ appDetailPage: pages.appDetailPage, label });
});

When('我保存有效 {string} 配置', async ({ pages }, label: string) => {
  await saveValidHealthProbe({ appDetailPage: pages.appDetailPage, label });
});

When('我提交无效 {string} 配置', async ({ pages }, label: string) => {
  await submitInvalidHealthProbe({ appDetailPage: pages.appDetailPage, label });
});

Then('{string} 不应包含取消测试路径', async ({ pages }, label: string) => {
  const { cancelPath } = getHealthProbeConfig(label);
  await pages.appDetailPage.expectHealthProbeTextHidden(label, cancelPath);
});

Then('{string} 应处于查看态', async ({ pages }, label: string) => {
  await pages.appDetailPage.expectHealthProbeInViewMode(label);
});

Then('{string} 应展示校验提示并限制异常端口', async ({ pages }, label: string) => {
  await pages.appDetailPage.expectHealthProbeValidationVisible('检查路径不能为空');
  await pages.appDetailPage.expectHealthProbeInputValue(label, '检查端口', HEALTH_PROBE_CLAMPED_PORT);
});

Then('{string} 应展示已保存配置', async ({ pages }, label: string) => {
  const { savedPath, savedPort } = getHealthProbeConfig(label);
  await pages.appDetailPage.expectHealthProbeContains(label, savedPath);
  await pages.appDetailPage.expectHealthProbeContains(label, savedPort);
});
