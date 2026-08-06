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

import { ref } from 'vue';

import { type HelmDeployRecordOutputObj } from '~/@types/v1/deploy';
import { type TrafficLaneOutput } from '~/@types/v1/env';
import { type ChartVersionOutputObj } from '~/@types/v1/helm-charts';
import { DeployService, EnvService, HelmChartsService } from '~/api/modules/v1';
import { useDeployStatusMap } from '~/composables/use-deploy-status';
import { useAppDetail } from '~/stores/app-detail';
import { useSpaceStore } from '~/stores/space';

interface IPrams {
  envName: string;
  keyword?: string;
  page: number;
  pageSize: number;
  trafficLaneName?: string;
}

export const deployHistoryList = ref<HelmDeployRecordOutputObj[]>([]);
export const count = ref<number>(0);
// 最新部署状态
export const latestDeployStatus = ref<string>('');

export const chartList = ref<ChartVersionOutputObj[]>([]);

export const useHelmDeploy = () => {
  const spaceStore = useSpaceStore();
  const { currentSpace } = spaceStore;
  const { helmStatusTextMap: statusTextMap, helmStatusColorMap: statusColorMap } = useDeployStatusMap();
  // 获取部署历史
  async function handleListDeployHistories(params: IPrams) {
    if (!params?.envName || !currentSpace) return;
    const res = await DeployService.listHelmDeployRecords({
      ...params,
      appID: appDetailStore.appID,
      keyword: params.keyword || '',
    }).catch(() => ({ count: '0', results: [] }));
    deployHistoryList.value = res?.results || [];
    count.value = Number(res.count);
    if (params.page === 1 && res.results && res.results.length > 0) {
      updateLatestDeployStatusFromList(res.results);
    }
  }

  // 从列表中更新最新部署状态
  function updateLatestDeployStatusFromList(list: HelmDeployRecordOutputObj[]) {
    if (!list.length) {
      latestDeployStatus.value = '';
      return;
    }
    const latestDeploy = list.reduce((prev, current) =>
      new Date(prev.updatedAt!) > new Date(current.updatedAt!) ? prev : current,
    );
    latestDeployStatus.value = latestDeploy.status!;
  }

  // 泳道列表
  const laneList = ref<TrafficLaneOutput[]>([]);
  // 获取当前环境下的泳道列表
  async function handleGetLanesList(env?: string) {
    if (!env) return;
    const res = await EnvService.listEnvTrafficLanes({
      workspaceID: currentSpace,
      envName: env,
    }).catch(() => []);
    laneList.value = res;
  }

  const appDetailStore = useAppDetail();
  // Chart 版本
  async function handleGetChartList(appID?: string) {
    chartList.value = await HelmChartsService.listChartVersions({
      appID: appID || appDetailStore.appID,
    }).catch(() => []);
  }

  return {
    chartList,
    laneList,
    deployHistoryList,
    count,
    latestDeployStatus,
    statusColorMap,
    statusTextMap,
    handleListDeployHistories,
    handleGetChartList,
    handleGetLanesList,
  };
};
