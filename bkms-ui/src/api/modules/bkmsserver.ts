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

/* eslint-disable @typescript-eslint/no-explicit-any */
/**
 * 手写兼容层：保留旧 `ApiServerService` 导出键名，实现优先委托 `~/api/modules/v1`。
 * 勿再对本文件运行旧版 gen-api；接口定义以 v1 模块与 swagger 为准。
 */
import { AppNetworkingService } from '~/api/modules/v1/app-networking';
import { AppComponentsService } from '~/api/modules/v1/app-components';
import { AppConfigFilesService } from '~/api/modules/v1/app-config-files';
import { AppService } from '~/api/modules/v1/app';
import { AppSpecService } from '~/api/modules/v1/app-spec';
import { ArrangementService } from '~/api/modules/v1/arrangement';
import { BuildAutodeployService } from '~/api/modules/v1/build-autodeploy';
import { BuildsService } from '~/api/modules/v1/builds';
import { BkintegrationsBkccService } from '~/api/modules/v1/bkintegrations-bkcc';
import { BkintegrationsBkciService } from '~/api/modules/v1/bkintegrations-bkci';
import { BkintegrationsBkmonitorService } from '~/api/modules/v1/bkintegrations-bkmonitor';
import { BkintegrationsBcsService } from '~/api/modules/v1/bkintegrations-bcs';
import { BkintegrationsBscpService } from '~/api/modules/v1/bkintegrations-bscp';
import { BkintegrationsKubeinsightService } from '~/api/modules/v1/bkintegrations-kubeinsight';
import { ClusterAddonService } from '~/api/modules/v1/cluster-addon';
import { ComponentDefsService } from '~/api/modules/v1/component-defs';
import { DeployService } from '~/api/modules/v1/deploy';
import { EnvService } from '~/api/modules/v1/env';
import { EnvvarsService } from '~/api/modules/v1/envvars';
import { HelmChartsService } from '~/api/modules/v1/helm-charts';
import { ImagesService } from '~/api/modules/v1/images';
import { InstanceService } from '~/api/modules/v1/instance';
import { OperationAuditService } from '~/api/modules/v1/operation-audit';
import { PolarisConfigService } from '~/api/modules/v1/polaris-config';
import { PortPoolService } from '~/api/modules/v1/port-pool';
import { TopologyService } from '~/api/modules/v1/topology';
import { WorkspaceComponentsService } from '~/api/modules/v1/workspace-components';
import { WorkspaceService } from '~/api/modules/v1/workspace';

export const ApiServerService = {

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetUserStatistics: WorkspaceService.getUserStatistics,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListWorkspaceRoleMemberGroups: WorkspaceService.listWorkspaceRoleMemberGroups,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetWorkspace: WorkspaceService.getWorkspace,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListWorkspaces: WorkspaceService.listWorkspaces,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListWorkspacesOverview: WorkspaceService.listWorkspacesOverview,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateWorkspace: WorkspaceService.createWorkspace,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpdateWorkspaceInfo: WorkspaceService.updateWorkspaceInfo,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  AddWorkspaceUser: WorkspaceService.addWorkspaceUser,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  RemoveWorkspaceUser: WorkspaceService.removeWorkspaceUser,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  SetWorkspaceState: WorkspaceService.setWorkspaceState,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteWorkspace: WorkspaceService.deleteWorkspace,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateWorkspaceComponent: WorkspaceComponentsService.createWorkspaceComponent,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  PatchWorkspaceComponent: WorkspaceComponentsService.patchWorkspaceComponent,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteWorkspaceComponent: WorkspaceComponentsService.deleteWorkspaceComponent,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListWorkspaceComponents: WorkspaceComponentsService.listWorkspaceComponents,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateHelmChartBuild: HelmChartsService.createHelmChartBuild,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetHelmChartSemver: HelmChartsService.getHelmChartSemver,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListAppHelmCharts: HelmChartsService.listAppHelmCharts,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListHelmChartBuildRecords: HelmChartsService.listHelmChartBuildRecords,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetHelmChartFiles: HelmChartsService.getHelmChartFiles,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListChartVersions: HelmChartsService.listChartVersions,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetValuesFile: HelmChartsService.getValuesFile,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListPlaceholderVars: ArrangementService.listPlaceholderVars,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateAppConfigFile: AppConfigFilesService.createAppConfigFile,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpdateAppConfigFile: AppConfigFilesService.updateAppConfigFile,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListAppConfigFiles: AppConfigFilesService.listAppConfigFiles,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteAppConfigFile: AppConfigFilesService.deleteAppConfigFile,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetAppConfigFileDetails: AppConfigFilesService.getAppConfigFileDetails,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpdateAppConfigFileContent: AppConfigFilesService.updateAppConfigFileContent,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpdateAppConfigFileOverlayContent: AppConfigFilesService.updateAppConfigFileOverlayContent,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  PreviewOverlayMerge: AppConfigFilesService.previewOverlayMerge,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListAppConfigFileVersions: AppConfigFilesService.listAppConfigFileVersions,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CompareAppConfigFileVersions: AppConfigFilesService.compareAppConfigFileVersions,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  RollbackAppConfigFileVersion: AppConfigFilesService.rollbackAppConfigFileVersion,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteAppConfigFileVersion: AppConfigFilesService.deleteAppConfigFileVersion,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpdateBuildConfig: BuildsService.updateBuildConfig,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListBuildRecords: BuildsService.listBuildRecords,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateBuild: BuildsService.createBuild,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateTrpcBuildDeploy: BuildAutodeployService.createTrpcBuildDeploy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateTafBuildDeploy: BuildAutodeployService.createTafBuildDeploy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetRecommendedImageTag: BuildsService.getRecommendedImageTag,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListAppImages: ImagesService.listAppImages,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  RefreshAppImages: ImagesService.refreshAppImages,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  PromoteAppImage: ImagesService.promoteAppImage,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListImageTagDeployRecords: ImagesService.listImageTagDeployRecords,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListDeployableImageTags: ImagesService.listDeployableImageTags,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListHelmDeployRecords: DeployService.listHelmDeployRecords,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  PreviewHelmDeploy: DeployService.previewHelmDeploy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateHelmDeploy: DeployService.createHelmDeploy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  PreviewRollbackHelmDeploy: DeployService.previewRollbackHelmDeploy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  RollbackHelmDeploy: DeployService.rollbackHelmDeploy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteHelmDeploy: DeployService.deleteHelmDeploy,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListClusterAddons: ClusterAddonService.listClusterAddons,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpsertClusterAddon: ClusterAddonService.upsertClusterAddon,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteClusterAddon: ClusterAddonService.deleteClusterAddon,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListTrpcDeployRecords: DeployService.listTrpcDeployRecords,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateTrpcDeploy: DeployService.createTrpcDeploy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteTrpcDeploy: DeployService.deleteTrpcDeploy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListTrpcResourceSnapshots: DeployService.listTrpcResourceSnapshots,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetTrpcResourceSnapshot: DeployService.getTrpcResourceSnapshot,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetLatestTrpcDeployStatus: DeployService.getLatestTrpcDeployStatus,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListTafDeployRecords: DeployService.listTafDeployRecords,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateTafDeploy: DeployService.createTafDeploy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteTafDeploy: DeployService.deleteTafDeploy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListTafResourceSnapshots: DeployService.listTafResourceSnapshots,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetTafResourceSnapshot: DeployService.getTafResourceSnapshot,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetLatestTafDeployStatus: DeployService.getLatestTafDeployStatus,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListAppInstances: InstanceService.listAppInstances,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpdateAppInstances: InstanceService.updateAppInstances,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ScaleAppInstances: InstanceService.scaleAppInstances,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  BatchDeleteAppInstances: InstanceService.batchDeleteAppInstances,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpdateAppInstancePolaris: InstanceService.updateAppInstancePolaris,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateAppInstanceWebConsole: InstanceService.createAppInstanceWebConsole,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListAppInstanceLogs: InstanceService.listAppInstanceLogs,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListEvents: InstanceService.listEvents,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListTrpcAdminCmds: InstanceService.listTrpcAdminCmds,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ExecuteTrpcAdminCmd: InstanceService.executeTrpcAdminCmd,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ExecuteTafAdminCmd: InstanceService.executeTafAdminCmd,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateApp: AppService.createApp,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetAppIDAutoSuffix: AppService.getAppIDAutoSuffix,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetApp: AppService.getApp,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListAppEnvVars: EnvvarsService.listAppEnvVars,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListApps: AppService.listApps,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteApp: AppService.deleteApp,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpdateHelmSpec: AppService.updateHelmSpec,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpdateAppTrpcSpec: AppService.updateAppTrpcSpec,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpdateAppTafSpec: AppService.updateAppTafSpec,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetAppSpecOverview: AppSpecService.getAppSpecOverview,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetAppDefaultAppSpecResources: AppSpecService.getAppDefaultAppSpecResources,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  SetAppDefaultAppSpecResources: AppSpecService.setAppDefaultAppSpecResources,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetEnvAppSpecResources: AppSpecService.getEnvAppSpecResources,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetEnvEffectiveAppSpecResources: AppSpecService.getEnvEffectiveAppSpecResources,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  SetEnvAppSpecResources: AppSpecService.setEnvAppSpecResources,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteEnvAppSpecResources: AppSpecService.deleteEnvAppSpecResources,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetAppDefaultAppSpecUpdateStrategy: AppSpecService.getAppDefaultAppSpecUpdateStrategy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  SetAppDefaultAppSpecUpdateStrategy: AppSpecService.setAppDefaultAppSpecUpdateStrategy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetEnvAppSpecUpdateStrategy: AppSpecService.getEnvAppSpecUpdateStrategy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetEnvEffectiveAppSpecUpdateStrategy: AppSpecService.getEnvEffectiveAppSpecUpdateStrategy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  SetEnvAppSpecUpdateStrategy: AppSpecService.setEnvAppSpecUpdateStrategy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteEnvAppSpecUpdateStrategy: AppSpecService.deleteEnvAppSpecUpdateStrategy,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetEnvEffectiveAppSpecDevMode: AppSpecService.getEnvEffectiveAppSpecDevMode,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  SetEnvAppSpecDevMode: AppSpecService.setEnvAppSpecDevMode,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteEnvAppSpecDevMode: AppSpecService.deleteEnvAppSpecDevMode,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetAppDefaultAppSpecLifecycle: AppSpecService.getAppDefaultAppSpecLifecycle,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  SetAppDefaultAppSpecLifecycle: AppSpecService.setAppDefaultAppSpecLifecycle,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetEnvAppSpecLifecycle: AppSpecService.getEnvAppSpecLifecycle,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetEnvEffectiveAppSpecLifecycle: AppSpecService.getEnvEffectiveAppSpecLifecycle,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  SetEnvAppSpecLifecycle: AppSpecService.setEnvAppSpecLifecycle,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteEnvAppSpecLifecycle: AppSpecService.deleteEnvAppSpecLifecycle,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetAppDefaultAppSpecProbe: AppSpecService.getAppDefaultAppSpecProbe,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  SetAppDefaultAppSpecProbe: AppSpecService.setAppDefaultAppSpecProbe,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetEnvAppSpecProbe: AppSpecService.getEnvAppSpecProbe,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetEnvEffectiveAppSpecProbe: AppSpecService.getEnvEffectiveAppSpecProbe,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  SetEnvAppSpecProbe: AppSpecService.setEnvAppSpecProbe,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteEnvAppSpecProbe: AppSpecService.deleteEnvAppSpecProbe,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteEnvAppSpecProbeByType: AppSpecService.deleteEnvAppSpecProbeByType,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateAppComponent: AppComponentsService.createAppComponent,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  PatchAppComponent: AppComponentsService.patchAppComponent,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteAppComponent: AppComponentsService.deleteAppComponent,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateAppService: AppNetworkingService.createAppService,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListAppServices: AppNetworkingService.listAppServices,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteAppService: AppNetworkingService.deleteAppService,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpdateAppService: AppNetworkingService.updateAppService,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListTrafficLaneCandidateApps: AppNetworkingService.listTrafficLaneCandidateApps,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateEnv: EnvService.createEnv,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListEnvs: EnvService.listEnvs,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetEnv: EnvService.getEnv,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpdateEnvBasicInfo: EnvService.updateEnvBasicInfo,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpdateEnvCluster: EnvService.updateEnvCluster,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteEnv: EnvService.deleteEnv,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateScopedEnvVar: EnvvarsService.createScopedEnvVar,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpdateScopedEnvVar: EnvvarsService.updateScopedEnvVar,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteScopedEnvVar: EnvvarsService.deleteScopedEnvVar,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListPublicScopedEnvVars: EnvvarsService.listPublicScopedEnvVars,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListDetailedEnvScopedEnvVars: EnvvarsService.listDetailedEnvScopedEnvVars,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListDetailedAppEnvVars: EnvvarsService.listDetailedAppEnvVars,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListEnvAvailableEnvVars: EnvvarsService.listEnvAvailableEnvVars,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListEnvBgEnvVars: EnvvarsService.listEnvBgEnvVars,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListEnvTrafficLanes: EnvService.listEnvTrafficLanes,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListComponentDefs: ComponentDefsService.listComponentDefs,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateComponentDef: ComponentDefsService.createComponentDef,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  PatchComponentDef: ComponentDefsService.patchComponentDef,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteComponentDef: ComponentDefsService.deleteComponentDef,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListOperationRecords: OperationAuditService.listOperationRecords,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListOperationRecordFilterOptions: OperationAuditService.listOperationRecordFilterOptions,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListBSCPBizs: BkintegrationsBscpService.listBSCPBizs,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListBSCPServices: BkintegrationsBscpService.listBSCPServices,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListBSCPConfigs: BkintegrationsBscpService.listBSCPConfigs,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetBSCPConfig: BkintegrationsBscpService.getBSCPConfig,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListBKCCAuthorizedBusinesses: BkintegrationsBkccService.listBKCCAuthorizedBusinesses,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListBkCIOAuthGitProjects: BkintegrationsBkciService.listBkCIOAuthGitProjects,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetBkCIOAuthUrl: BkintegrationsBkciService.getBkCIOAuthUrl,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListBkCIPipelines: BkintegrationsBkciService.listBkCIPipelines,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetBkCIPipelineVariables: BkintegrationsBkciService.getBkCIPipelineVariables,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetBkCIPipeline: BkintegrationsBkciService.getBkCIPipeline,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListBCSAuthorizedProjects: BkintegrationsBcsService.listBCSAuthorizedProjects,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListClustersByProject: BkintegrationsBcsService.listClustersByProject,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListNamespacesByCluster: BkintegrationsBcsService.listNamespacesByCluster,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetLatestEnvReport: BkintegrationsKubeinsightService.getLatestEnvReport,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListAppPolarisConfigs: PolarisConfigService.listAppPolarisConfigs,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateAppPolarisConfig: PolarisConfigService.createAppPolarisConfig,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  PatchAppPolarisConfig: PolarisConfigService.patchAppPolarisConfig,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeleteAppPolarisConfig: PolarisConfigService.deleteAppPolarisConfig,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListAppPolarisConfigVars: PolarisConfigService.listAppPolarisConfigVars,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListPortPools: PortPoolService.listPortPools,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreatePortPool: PortPoolService.createPortPool,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  UpdatePortPool: PortPoolService.updatePortPool,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  DeletePortPool: PortPoolService.deletePortPool,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetApmServiceName: BkintegrationsBkmonitorService.getApmServiceName,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListApms: BkintegrationsBkmonitorService.listApms,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  CreateEnvApm: BkintegrationsBkmonitorService.createEnvApm,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  BindApmToEnv: BkintegrationsBkmonitorService.bindApmToEnv,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetEnvApm: BkintegrationsBkmonitorService.getEnvApm,

  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetResourceTopology: TopologyService.getResourceTopology,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetTopologyNodeDetail: TopologyService.getTopologyNodeDetail,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  ListTopologyNodeEvents: TopologyService.listTopologyNodeEvents,
  /**
   * @deprecated 请改用 `~/api/modules/v1` 下对应 Service（本属性已绑定 v1 实现）。
   */
  GetTopologyNodeManifest: TopologyService.getTopologyNodeManifest,
};
