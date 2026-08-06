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

import { computed, ref } from 'vue';

import { useI18n } from 'vue-i18n';
import { ApiServerService } from '~/api/modules/bkmsserver';

import type { ClusterOutput } from '~/@types/v1/bkintegrations-bcs';

export type ClusterGroupType = 'independent' | 'managed' | 'shared' | 'virtual';

export type ClusterSelectorGroup = {
  list: ClusterSelectorItem[];
  title: string;
  type: ClusterGroupType;
};

export type ClusterSelectorItem = ClusterOutput & {
  id: string;
  name: string;
  type?: ClusterGroupType;
};

export type ClusterType = 'all' | ClusterGroupType;

type CompatibleClusterOutput = ClusterOutput & {
  clusterID?: string;
  clusterName?: string;
};

export default function useClusterSelector(
  // emits: any,
  projectId: string,
  clusterType: ClusterGroupType[] | ClusterType = ['independent', 'managed'],
) {
  const loading = ref(false);
  const clusterList = ref<ClusterSelectorItem[]>([]);
  // 获取集群列表
  getClusterList(projectId);
  const keyword = ref('');
  const { t } = useI18n();

  const collapseList = ref<Array<ClusterGroupType>>([]);
  const handleToggleCollapse = (type: ClusterGroupType) => {
    const index = collapseList.value.findIndex(item => item === type);
    if (index > -1) {
      collapseList.value.splice(index, 1);
    } else {
      collapseList.value.push(type);
    }
  };
  // 集群分类数据
  const clusterListByType = computed(() =>
    clusterList.value
      .filter(item => {
        const clusterID = item.id.toLocaleLowerCase();
        const clusterName = item.name.toLocaleLowerCase();
        const searchKey = keyword.value?.toLocaleLowerCase();
        return clusterID?.includes(searchKey) || clusterName?.includes(searchKey);
      })
      .reduce<ClusterSelectorGroup[]>(
        (list, item) => {
          if (item.type === 'virtual') {
            const data = list.find(item => item.type === 'virtual');
            // 虚拟集群属于共享集群中的一种
            data?.list.push(item);
          } else if (item?.isShared) {
            // 共享集群
            const data = list.find(item => item.type === 'shared');
            data?.list.push(item);
          } else {
            // 独立集群
            const data = list.find(item => item.type === 'independent');
            data?.list.push(item);
          }
          return list;
        },
        [
          {
            type: 'virtual',
            list: [],
            title: 'vCluster',
          },
          {
            type: 'managed',
            list: [],
            title: t('托管集群'),
          },
          {
            type: 'independent',
            list: [],
            title: t('自建集群'),
          },
          {
            type: 'shared',
            list: [],
            title: t('共享集群'),
          },
        ],
      )
      .filter(item => !!item.list.length),
  );
  // 当前场景的集群分类数据
  const clusterData = computed<ClusterSelectorGroup[]>(() => {
    if (Array.isArray(clusterType)) {
      return clusterListByType.value.filter(item => clusterType.includes(item.type));
    }
    return clusterType === 'all'
      ? clusterListByType.value
      : clusterListByType.value.filter(item => item.type === clusterType);
  });
  const isClusterDataEmpty = computed(() => clusterData.value.every(item => !item.list.length));

  async function getClusterList(projectId: string) {
    if (!projectId) return;
    loading.value = true;
    const res = await ApiServerService.ListClustersByProject({ projectID: projectId }, { validateCode: false }).catch(
      () => [],
    );
    clusterList.value = (res || []).map((item: CompatibleClusterOutput) => ({
      ...item,
      id: item.id || item.clusterID || '',
      name: item.name || item.clusterName || item.id || item.clusterID || '',
      type: item.type as ClusterGroupType | undefined,
    }));
    loading.value = false;
  }

  return {
    keyword,
    collapseList,
    isClusterDataEmpty,
    clusterData,
    loading,
    clusterList,
    handleToggleCollapse,
    getClusterList,
  };
}
