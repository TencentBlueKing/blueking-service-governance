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
/** TC-16 构建产物生命周期：业务语义步骤。 */
import { Given, Then, When } from '../fixtures/fixtures';

Given('容器镜像接口返回生命周期测试数据', async ({ pages }) => {
  await pages.appDetailPage.artifact.setupContainerImageLifecycleMock();
});

Given('容器镜像接口返回删除权限异常测试数据', async ({ pages }) => {
  await pages.appDetailPage.artifact.setupContainerImagePermissionErrorMock();
});

Given('Helm Chart 接口返回生命周期测试数据', async ({ pages }) => {
  await pages.appDetailPage.artifact.setupHelmChartLifecycleMock();
});

Given('我在 Helm 应用的 Helm Chart 制品页', async ({ pages }) => {
  await pages.appDetailPage.artifact.gotoHelmChartArtifactManagement();
});

When('我执行容器镜像一键同步', async ({ pages }) => {
  await pages.appDetailPage.artifact.syncContainerImages();
});

When('我晋级首个容器镜像 Tag', async ({ pages }) => {
  await pages.appDetailPage.artifact.promoteFirstImageTag();
});

When('我删除首个容器镜像 Tag', async ({ pages }) => {
  await pages.appDetailPage.artifact.deleteFirstImageTag();
});

When('我按约定版本搜索 Helm Chart', async ({ pages }) => {
  await pages.appDetailPage.artifact.searchHelmChartByVersion();
});

When('我打开 Helm Chart 版本详情', async ({ pages }) => {
  await pages.appDetailPage.artifact.openHelmChartVersionDetail();
});

When('我新建 Helm Chart 版本构建', async ({ pages }) => {
  await pages.appDetailPage.artifact.createHelmChartVersionBuild();
});

Then('容器镜像同步结果应展示', async ({ pages }) => {
  await pages.appDetailPage.artifact.expectImageSynced();
});

Then('容器镜像应展示已晋级状态', async ({ pages }) => {
  await pages.appDetailPage.artifact.expectImagePromoted();
});

Then('容器镜像 Tag 应从列表移除', async ({ pages }) => {
  await pages.appDetailPage.artifact.expectImageDeleted();
});

Then('容器镜像删除权限错误应展示构建配置引导', async ({ pages }) => {
  await pages.appDetailPage.artifact.expectImageDeletePermissionErrorVisible();
});

Then('Helm Chart 搜索结果应匹配版本', async ({ pages }) => {
  await pages.appDetailPage.artifact.expectHelmChartSearchResultMatches();
});

Then('Helm Chart 版本详情应展示文件内容', async ({ pages }) => {
  await pages.appDetailPage.artifact.expectHelmChartDetailVisible();
});

Then('Helm Chart 构建记录侧栏应展示目标构建', async ({ pages }) => {
  await pages.appDetailPage.artifact.expectHelmChartBuildRecordVisible();
});
