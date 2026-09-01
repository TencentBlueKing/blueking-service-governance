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
 * TC-02 扩缩容：业务语义步骤。
 *
 * 「我在当前应用的部署管理页」「当前应用未部署则先部署 N 个实例」
 * 与「实例列表应至少出现 N 个 Pod…」已在 TC-01.steps.ts 中注册，
 * 此文件仅注册扩缩容特有步骤。
 */
import { configureAutoScale, scaleAppManually } from '../actions/deploy.action';
import { Then, When } from '../fixtures/fixtures';

When('我使用手动调节扩缩容到 {int} 副本', async ({ pages }, replicas: number) => {
  await scaleAppManually({ appDetailPage: pages.appDetailPage }, replicas);
});

When(
  '我配置自动调节最小 {int} 最大 {int} CPU 使用率 {int}%',
  async ({ pages }, minReplicas: number, maxReplicas: number, cpuUtilization: number) => {
    await configureAutoScale({ appDetailPage: pages.appDetailPage }, { cpuUtilization, maxReplicas, minReplicas });
  },
);

Then(
  '自动调节配置应展示最小 {int} 最大 {int} CPU 使用率 {int}%',
  async ({ pages }, minReplicas: number, maxReplicas: number, cpuUtilization: number) => {
    await pages.appDetailPage.expectAutoScaleConfig({ cpuUtilization, maxReplicas, minReplicas });
  },
);
