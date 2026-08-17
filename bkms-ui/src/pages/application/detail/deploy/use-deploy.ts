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

import {
  type BuildRecordOutputObj,
  type CreateTafBuildDeployRequest,
  type CreateTrpcBuildDeployRequest,
} from '~/@types/v1/build-autodeploy';
import {
  type CreateTafDeployRequest,
  type CreateTrpcDeployRequest,
  type DeleteHelmDeployRequest,
  type DeleteTafDeployRequest,
  type DeleteTrpcDeployRequest,
  type EnvVarPreCheckOutput,
  type GetLatestAppModelDeployStatusOutput,
  type GetLatestTafDeployStatusRequest,
  type GetLatestTrpcDeployStatusRequest,
  type ListAppModelDeployRecordsOutput,
  type ListHelmDeployRecordsRequest,
  type ListTafDeployRecordsRequest,
  type ListTrpcDeployRecordsRequest,
  type PreCheckTafDeployEnvVarsRequest,
  type PreCheckTrpcDeployEnvVarsRequest,
} from '~/@types/v1/deploy';
import {
  type ExecuteTafAdminCmdOutput,
  type ExecuteTafAdminCmdRequest,
  type ExecuteTrpcAdminCmdOutput,
  type ExecuteTrpcAdminCmdRequest,
} from '~/@types/v1/instance';
import { v1Prefix } from '~/api/clients';
import { type Config } from '~/api/interceptors';
import { BuildAutodeployService, DeployService, InstanceService } from '~/api/modules/v1';
import { downloadByLink } from '~/common/util';

import type { ExtractData } from '~/@types/api';

/**
 * 支持部署的应用类型
 */
export type DeployableAppType = 'helm' | 'taf' | 'trpc';

/**
 * 最近一次重启日志
 */
export const RECENT_RESTART_LOG = 'recentRestartLog';

/**
 * 部署请求通用参数（直接部署与构建部署共用）
 */
export interface DeployParams {
  appID: string;
  /** 构建分支（构建部署时必填，直接部署时忽略） */
  branch?: string;
  envName: string;
  imageTag: string;
  replicas: number;
}

interface DeployAPIs {
  /**
   * 构建并部署（从源码构建）
   */
  buildAndCreateDeploy?: (
    params?: CreateTafBuildDeployRequest | CreateTrpcBuildDeployRequest,
    config?: Config,
  ) => Promise<BuildRecordOutputObj>;

  /**
   * 直接部署（已构建镜像）
   */
  createDeployDirectly?: (
    params?: CreateTafDeployRequest | CreateTrpcDeployRequest,
    config?: Config,
  ) => Promise<ExtractData<null>>;

  /**
   * 删除部署（下架当前环境最新版本）
   */
  deleteDeploy: (
    params?: DeleteHelmDeployRequest | DeleteTafDeployRequest | DeleteTrpcDeployRequest,
    config?: Config,
  ) => Promise<ExtractData<object>>;

  /**
   * 执行管理命令（helm 类型无此能力）
   */
  executeAdminCmd?: <
    Request = ExecuteTafAdminCmdRequest | ExecuteTrpcAdminCmdRequest,
    ResponseData = ExtractData<ExecuteTafAdminCmdOutput | ExecuteTrpcAdminCmdOutput>,
  >(
    params?: Request,
    config?: Config,
  ) => Promise<ResponseData>;

  /**
   * 获取部署记录列表
   */
  listDeployRecords: (
    params?: ListHelmDeployRecordsRequest | ListTafDeployRecordsRequest | ListTrpcDeployRecordsRequest,
    config?: Config,
  ) => Promise<ExtractData<ListAppModelDeployRecordsOutput>>;

  /**
   * 获取最新部署状态
   */
  listLatestDeployRecords?: (
    params?: GetLatestTafDeployStatusRequest | GetLatestTrpcDeployStatusRequest,
    config?: Config,
  ) => Promise<ExtractData<GetLatestAppModelDeployStatusOutput>>;

  /**
   * 部署前检查目标环境是否存在未定义的环境变量
   */
  preCheckDeployEnvVars?: (
    params?: PreCheckTafDeployEnvVarsRequest | PreCheckTrpcDeployEnvVarsRequest,
    config?: Config,
  ) => Promise<EnvVarPreCheckOutput>;
}

/**
 * 使用拼接 URL + a 标签方式下载实例日志（避免大文件占用内存）
 * @param params 下载参数
 * @param filename 下载文件名
 */
export async function downloadInstanceLog(params: {
  appID: string;
  envName: string;
  instanceID: string;
  previous?: boolean;
  trafficLaneName?: string;
}) {
  // 构建路径参数
  const path = `/apps/${encodeURIComponent(params.appID)}/envs/${encodeURIComponent(params.envName)}/instances/${encodeURIComponent(params.instanceID)}/logs/download`;

  // 构建查询参数
  const queryParams = new URLSearchParams();
  if (params.previous) {
    queryParams.append('previous', 'true');
  }
  if (params.trafficLaneName) {
    queryParams.append('trafficLaneName', params.trafficLaneName);
  }

  const queryString = queryParams.toString();
  const url = `${v1Prefix}${path}${queryString ? `?${queryString}` : ''}`;

  // 使用通用工具函数触发下载（浏览器会自动携带 Cookie 进行认证）
  downloadByLink(url, '');
}

/**
 * 根据应用类型获取对应的部署 API
 * @param appType - 应用类型 ('trpc' | 'taf' | 'helm')
 * @returns 部署相关的 API 方法集合
 *
 * @example
 * ```ts
 * const deployAPIs = useDeployAPIs('trpc');
 * ```
 */
export function useDeployAPIs(appType: DeployableAppType): DeployAPIs {
  const apiMapping: Record<DeployableAppType, DeployAPIs> = {
    trpc: {
      preCheckDeployEnvVars: DeployService.preCheckTrpcDeployEnvVars,
      listDeployRecords: DeployService.listTrpcDeployRecords,
      listLatestDeployRecords: DeployService.getLatestTrpcDeployStatus,
      createDeployDirectly: DeployService.createTrpcDeploy,
      buildAndCreateDeploy: BuildAutodeployService.createTrpcBuildDeploy,
      deleteDeploy: DeployService.deleteTrpcDeploy,
      executeAdminCmd: InstanceService.executeTrpcAdminCmd as DeployAPIs['executeAdminCmd'],
    },
    taf: {
      preCheckDeployEnvVars: DeployService.preCheckTafDeployEnvVars,
      listDeployRecords: DeployService.listTafDeployRecords,
      listLatestDeployRecords: DeployService.getLatestTafDeployStatus,
      createDeployDirectly: DeployService.createTafDeploy,
      buildAndCreateDeploy: BuildAutodeployService.createTafBuildDeploy,
      deleteDeploy: DeployService.deleteTafDeploy,
      executeAdminCmd: InstanceService.executeTafAdminCmd as DeployAPIs['executeAdminCmd'],
    },
    helm: {
      listDeployRecords: DeployService.listHelmDeployRecords,
      deleteDeploy: DeployService.deleteHelmDeploy as DeployAPIs['deleteDeploy'],
    },
  };

  return apiMapping[appType];
}
