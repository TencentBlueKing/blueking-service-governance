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

import { type BuildRecordOutputObj, type CreateBuildDeployRequest } from '~/@types/v1/build-autodeploy';
import {
  type CreateBKMSAppDeployRequest,
  type DeleteBKMSAppDeployRequest,
  type DeleteHelmDeployRequest,
  type DeployPreCheckOutput,
  type GetLatestAppModelDeployStatusOutput,
  type GetLatestBKMSAppDeployStatusRequest,
  type ListAppModelDeployRecordsOutput,
  type ListBKMSAppDeployRecordsRequest,
  type ListHelmDeployRecordsRequest,
  type PreCheckBKMSAppDeployRequest,
} from '~/@types/v1/deploy';
import { v1Prefix } from '~/api/clients';
import { type Config } from '~/api/interceptors';
import { BuildAutodeployService, DeployService } from '~/api/modules/v1';
import { downloadByLink } from '~/common/util';
import { type AppType, isHelmLikeAppType } from '~/composables/app-type';

import type { ExtractData } from '~/@types/api';

/**
 * 支持部署的应用类型
 */
export type DeployableAppType = AppType;

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
  /** 发布策略 ID：指定时按策略分批更新实例，仅镜像直连部署支持；不指定时一次更新全部实例 */
  releaseStrategyID?: string;
  replicas: number;
  /** 代码修订：分支名、Tag 名或 CommitID */
  revision?: string;
  /** 代码来源类型 */
  revisionType?: 'branch' | 'commit' | 'tag';
}

interface DeployAPIs {
  /**
   * 构建并部署（从源码构建）
   */
  buildAndCreateDeploy?: (params?: CreateBuildDeployRequest, config?: Config) => Promise<BuildRecordOutputObj>;

  /**
   * 直接部署（已构建镜像）
   */
  createDeployDirectly?: (params?: CreateBKMSAppDeployRequest, config?: Config) => Promise<ExtractData<null>>;

  /**
   * 删除部署（下架当前环境最新版本）
   */
  deleteDeploy: (
    params?: DeleteBKMSAppDeployRequest | DeleteHelmDeployRequest,
    config?: Config,
  ) => Promise<ExtractData<object>>;

  /**
   * 获取部署记录列表
   */
  listDeployRecords: (
    params?: ListBKMSAppDeployRecordsRequest | ListHelmDeployRecordsRequest,
    config?: Config,
  ) => Promise<ExtractData<ListAppModelDeployRecordsOutput>>;

  /**
   * 获取最新部署状态
   */
  listLatestDeployRecords?: (
    params?: GetLatestBKMSAppDeployStatusRequest,
    config?: Config,
  ) => Promise<ExtractData<GetLatestAppModelDeployStatusOutput>>;

  /**
   * 部署前检查目标环境的环境变量和必选集群组件
   */
  preCheckDeploy?: (params?: PreCheckBKMSAppDeployRequest, config?: Config) => Promise<DeployPreCheckOutput>;
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
 * 实例管理命令由管理命令侧栏按当前 framework 选择接口，并单独校验 adminCommand 能力。
 * @param appType - 应用详情中的真实类型
 * @returns 部署相关的 API 方法集合
 *
 * @example
 * ```ts
 * const deployAPIs = useDeployAPIs('trpc');
 * ```
 */
export function useDeployAPIs(appType: DeployableAppType): DeployAPIs {
  if (isHelmLikeAppType(appType)) {
    return {
      listDeployRecords: DeployService.listHelmDeployRecords,
      deleteDeploy: DeployService.deleteHelmDeploy as DeployAPIs['deleteDeploy'],
    };
  }

  // 统一部署接口兼容存量 trpc/taf 与新 bkmsApp（含 blank），无需按框架分流。
  return {
    preCheckDeploy: DeployService.preCheckBKMSAppDeploy,
    listDeployRecords: DeployService.listBKMSAppDeployRecords,
    listLatestDeployRecords: DeployService.getLatestBKMSAppDeployStatus,
    createDeployDirectly: DeployService.createBKMSAppDeploy,
    buildAndCreateDeploy: BuildAutodeployService.createBuildDeploy,
    deleteDeploy: DeployService.deleteBKMSAppDeploy,
  };
}
