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

import { v1Fetch } from '~/api/clients';
import { BkintegrationsBkmonitorService, BkmonitorDashboardService } from '~/api/modules/v1';

import type {
  DashboardDirectoryOutput,
  DashboardOutput as MonitorDashboardOutput,
} from '~/@types/v1/bkintegrations-bkmonitor';
import type { DashboardOutput as AppDashboardOutput, EmptyOutput } from '~/@types/v1/bkmonitor-dashboard';

/** 应用已绑定的仪表盘 */
export interface AppDashboardBinding {
  title: string;
  uid: string;
  url: string;
}

/** 监控平台中可选的仪表盘 */
export interface MonitorDashboardOption {
  folderTitle: string;
  title: string;
  uid: string;
  url: string;
}

const dashboards = ref<AppDashboardBinding[]>([]);
const loading = ref(false);
const monitorDashboards = ref<MonitorDashboardOption[]>([]);
const monitorDashboardsLoading = ref(false);
// 自增序号：仅允许最后一次请求写入状态，避免快速切换应用/空间时旧响应覆盖新数据
let fetchDashboardsSeq = 0;
let fetchMonitorDashboardsSeq = 0;

/** 组合式函数入口：提供应用仪表盘的列表状态与增删改查方法 */
export function useAppDashboards() {
  /** 拉取应用已绑定的仪表盘列表，带序号防竞态 */
  async function fetchDashboards(appID: string) {
    const seq = ++fetchDashboardsSeq;
    if (!appID) {
      dashboards.value = [];
      return dashboards.value;
    }

    loading.value = true;
    try {
      const data = await BkmonitorDashboardService.listAppDashboards({ appID });
      const nextDashboards = (Array.isArray(data) ? data : []).map(normalizeAppDashboard).filter(item => item.uid);
      if (seq === fetchDashboardsSeq) {
        dashboards.value = nextDashboards;
      }
      return nextDashboards;
    } finally {
      if (seq === fetchDashboardsSeq) {
        loading.value = false;
      }
    }
  }

  /** 拉取监控平台空间内可选的仪表盘，带序号防竞态 */
  async function fetchMonitorDashboards(workspaceID: string) {
    const seq = ++fetchMonitorDashboardsSeq;
    if (!workspaceID) {
      monitorDashboards.value = [];
      return monitorDashboards.value;
    }

    monitorDashboardsLoading.value = true;
    try {
      const data = await BkintegrationsBkmonitorService.listDashboardDirectoryTree({ workspaceID });
      const nextMonitorDashboards = flattenMonitorDashboards(Array.isArray(data) ? data : []);
      if (seq === fetchMonitorDashboardsSeq) {
        monitorDashboards.value = nextMonitorDashboards;
      }
      return nextMonitorDashboards;
    } finally {
      if (seq === fetchMonitorDashboardsSeq) {
        monitorDashboardsLoading.value = false;
      }
    }
  }

  /** 新建绑定，成功后回拉最新列表 */
  async function createDashboard(appID: string, payload: { title: string; uid: string }) {
    await BkmonitorDashboardService.createAppDashboard({
      appID,
      title: payload.title,
      uid: payload.uid,
    });
    return fetchDashboards(appID);
  }

  /** 更新绑定：uid 未变更时只改标题，否则需走 body 传新 uid */
  async function updateDashboard(appID: string, uid: string, payload: { title: string; uid: string }) {
    if (payload.uid === uid) {
      await BkmonitorDashboardService.updateAppDashboard({
        appID,
        title: payload.title,
        uid,
      });
    } else {
      await updateAppDashboardWithBodyUid(appID, uid, payload);
    }
    return fetchDashboards(appID);
  }

  /** 删除绑定，成功后回拉最新列表 */
  async function deleteDashboard(appID: string, uid: string) {
    await BkmonitorDashboardService.deleteAppDashboard({ appID, uid });
    return fetchDashboards(appID);
  }

  return {
    createDashboard,
    dashboards,
    deleteDashboard,
    fetchDashboards,
    fetchMonitorDashboards,
    loading,
    monitorDashboards,
    monitorDashboardsLoading,
    updateDashboard,
  };
}

/** 将空间的目录树拍平为仪表盘选项，按 uid 去重 */
function flattenMonitorDashboards(tree: DashboardDirectoryOutput[]): MonitorDashboardOption[] {
  const map = new Map<string, MonitorDashboardOption>();

  tree.forEach(folder => {
    const folderTitle = folder.title || '';
    folder.dashboards?.forEach(item => {
      const option = normalizeMonitorDashboard(item, folderTitle);
      if (option.uid && !map.has(option.uid)) {
        map.set(option.uid, option);
      }
    });
  });

  return [...map.values()];
}

/** 归一化应用绑定项，标题缺省时回退为 uid */
function normalizeAppDashboard(item: AppDashboardOutput): AppDashboardBinding {
  const uid = item.uid || '';
  return {
    title: item.title || uid,
    uid,
    url: item.url || '',
  };
}

/** 归一化监控平台仪表盘，标题缺省时回退为 uid */
function normalizeMonitorDashboard(item: MonitorDashboardOutput, folderTitle: string): MonitorDashboardOption {
  const uid = item.uid || '';
  return {
    folderTitle,
    title: item.title || uid,
    uid,
    url: item.url || '',
  };
}

/** 直连底层请求：SDK 生成的更新接口不支持在 body 中传新 uid，故手动组装 PUT */
function updateAppDashboardWithBodyUid(appID: string, uid: string, payload: { title: string; uid: string }) {
  return v1Fetch.put<{ title: string; uid: string }, EmptyOutput>(
    `/apps/${encodeURIComponent(appID)}/bkmonitor/dashboards/${encodeURIComponent(uid)}`,
  )({
    title: payload.title,
    uid: payload.uid,
  });
}
