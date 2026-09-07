/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */
import AppConfigPage from './app-detail/app-config.page';
import AppDetailBase from './app-detail/app-detail-base.page';
import AppSpecPage from './app-detail/app-spec.page';
import ArtifactPage from './app-detail/artifact.page';
import BuildConfigPage from './app-detail/build-config.page';
import BuildManagementPage from './app-detail/build-management.page';
import DeployPage from './app-detail/deploy.page';
import { type BasePageDependencies } from './base.page';

/**
 * 应用详情页面 Page Object（组合门面）。
 *
 * 应用详情路由格式：`/:space/app/:name/:type/:menuName`
 * 目前 E2E 覆盖的应用类型默认为 TRPC，后续如需 Helm / TAF 可通过 `type` 参数覆盖。
 *
 * 实现按业务域拆分在 `pages/app-detail/` 目录下，各自继承 AppDetailBase：
 * - `deploy`          部署管理：部署、扩缩容、移除部署、实例列表
 * - `appConfig`       应用配置：环境视角、环境变量、资源规格、开发模式
 * - `appSpec`         运行配置卡片：健康探针、生命周期、元数据、更新策略
 * - `buildConfig`     构建配置（基本信息页）
 * - `buildManagement` 构建管理：构建记录、构建日志
 * - `artifact`        制品管理：容器镜像、Helm Chart
 *
 * steps / actions 通过 `pages.appDetailPage.<domain>.<method>()` 访问；
 * 跨域通用能力（gotoMenu 等）由 AppDetailBase 直接提供。
 */
export default class AppDetailPage extends AppDetailBase {
  /** 应用配置域：环境视角选择、环境变量、资源规格、开发模式 */
  readonly appConfig: AppConfigPage;

  /** 应用运行配置域：健康探针、生命周期、元数据、更新策略 */
  readonly appSpec: AppSpecPage;

  /** 制品管理域：容器镜像、Helm Chart */
  readonly artifact: ArtifactPage;

  /** 构建配置域（基本信息页的构建配置卡片与编辑侧栏） */
  readonly buildConfig: BuildConfigPage;

  /** 构建管理域：构建记录、构建日志 */
  readonly buildManagement: BuildManagementPage;

  /** 部署管理域：部署、扩缩容、移除部署、实例列表 */
  readonly deploy: DeployPage;

  constructor(deps: BasePageDependencies) {
    super(deps);
    this.appConfig = new AppConfigPage(deps);
    this.appSpec = new AppSpecPage(deps);
    this.artifact = new ArtifactPage(deps);
    this.buildConfig = new BuildConfigPage(deps);
    this.buildManagement = new BuildManagementPage(deps);
    this.deploy = new DeployPage(deps);
  }

  /** 框架配置文件 tab 的 Monaco 编辑器（monaco 实例根，高度随宿主高度链塌陷） */
  getFrameworkMonacoEditor() {
    return this.page.locator('.monaco-editor').first();
  }

  /**
   * 观测数据页的监控 iframe（MonitorIframe 渲染）。
   * 按 src 的 `#/apm/` 路由特征过滤（buildIframeUrl 产物），避免未来页面出现其他 iframe
   * 时 first() 静默命中错误元素；不用 data-testid 是因为 e2e 纪律禁止修改前端源码。
   */
  getMonitorIframe() {
    return this.page.locator('iframe[src*="#/apm/"]').first();
  }
}
