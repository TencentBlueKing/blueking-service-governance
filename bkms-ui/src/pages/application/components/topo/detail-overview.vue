<!--
 - TencentBlueKing is pleased to support the open source community by making
 - 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 - Copyright (C) Tencent. All rights reserved.
 - Licensed under the MIT License (the "License"); you may not use this file except
 - in compliance with the License. You may obtain a copy of the License at
 -
 -  http://opensource.org/licenses/MIT
 -
 - Unless required by applicable law or agreed to in writing, software distributed under
 - the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 - either express or implied. See the License for the specific language governing permissions and
 - limitations under the License.
 -
 - We undertake not to change the open source license (MIT license) applicable
 - to the current version of the project delivered to anyone in the future.
-->

<template>
  <div class="p-[24px] h-full overflow-y-auto">
    <Skeleton
      :full-height="false"
      :loading="loading"
    >
      <template #loading>
        <div class="flex flex-col gap-[16px]">
          <Layout.shape
            :height="32"
            :width="120"
          />
          <div class="grid grid-cols-2 gap-4">
            <Layout.shape
              :height="24"
              width="80%"
            />
            <Layout.shape
              :height="24"
              width="80%"
            />
            <Layout.shape
              :height="24"
              width="60%"
            />
            <Layout.shape
              :height="24"
              width="60%"
            />
          </div>
          <Layout.shape
            class="mt-[8px]"
            :height="32"
            :width="120"
          />
          <div class="grid grid-cols-2 gap-4">
            <Layout.shape
              :height="24"
              width="70%"
            />
            <Layout.shape
              :height="24"
              width="70%"
            />
          </div>
        </div>
      </template>
      <!-- 基础信息 -->
      <ToggleCard
        class="mb-[16px]"
        :name="$t('基础信息')"
        type="normal"
      >
        <div class="grid grid-cols-2 gap-4 text-[12px]">
          <FieldItem
            :field-value="$t('基本信息')"
            :value="nodeData?.kind || '--'"
          />
          <FieldItem :field-value="$t('状态')">
            <template #value>
              <span class="flex items-center gap-[6px]">
                <TopologyStatusIcon
                  :size="12"
                  :type="nodeData?.nodeStatus || 'unknown'"
                />
                <span
                  v-bk-tooltips="{
                    content: nodeData?.reason || '',
                    disabled: !nodeData?.reason,
                  }"
                >
                  {{ nodeData?.status || '--' }}
                </span>
              </span>
            </template>
          </FieldItem>
          <FieldItem
            v-for="[key, value] in extrasEntries"
            :key="key"
            :field-value="extrasLabelMap[key] || key"
            :value="value || '--'"
          />
        </div>
      </ToggleCard>

      <!-- 资源使用 -->
      <!-- <ToggleCard
        v-if="resourceUsageEntries.length > 0"
        class="mb-[16px]"
        :name="$t('资源使用')"
        type="normal"
      >
        <div class="grid grid-cols-2 gap-4 text-[12px]">
          <FieldItem
            v-for="[key, value] in resourceUsageEntries"
            :key="key"
            :field-value="resourceUsageLabelMap[key] || key"
            :value="value || '--'"
          />
        </div>
      </ToggleCard> -->

      <!-- Conditions -->
      <ToggleCard
        class="mb-[16px]"
        name="Conditions"
        type="normal"
      >
        <Table
          :data="conditions"
          :max-height="600"
        >
          <template #empty>
            <TableException type="empty" />
          </template>
          <TableColumn
            :label="$t('类型')"
            show-overflow="tooltip"
            :width="150"
          >
            <template #default="{ row }">
              {{ row.type || '--' }}
            </template>
          </TableColumn>
          <TableColumn
            :label="$t('状态')"
            show-overflow="tooltip"
            :width="100"
          >
            <template #default="{ row }">
              {{ row.status || '--' }}
            </template>
          </TableColumn>
          <TableColumn
            :label="$t('原因')"
            show-overflow="tooltip"
            :width="180"
          >
            <template #default="{ row }">
              {{ row.reason || '--' }}
            </template>
          </TableColumn>
          <TableColumn
            :label="$t('信息')"
            :min-width="200"
            show-overflow="tooltip"
          >
            <template #default="{ row }">
              {{ row.message || '--' }}
            </template>
          </TableColumn>
          <TableColumn
            :label="$t('最后变更时间')"
            :width="180"
          >
            <template #default="{ row }">
              {{ row.lastTransitionTime ? formatTimeByTimezone(row.lastTransitionTime) : '--' }}
            </template>
          </TableColumn>
        </Table>
      </ToggleCard>

      <!-- 空状态 -->
      <Exception
        v-if="!loading && !nodeData"
        scene="empty"
        type="empty"
      >
        {{ $t('暂无数据') }}
      </Exception>
    </Skeleton>
  </div>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Exception } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { TopologyNodeDetail } from '~/@types/topology';
  import { formatTimeByTimezone } from '~/common/util';
  import FieldItem from '~/components/field-item.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import Skeleton from '~/components/skeleton/skeleton.vue';
  import TableException from '~/components/table-exception.vue';
  import ToggleCard from '~/components/toggle-card.vue';

  import TopologyStatusIcon from './topology-status-icon.vue';

  import type { TopoNodeData } from './types';

  const props = defineProps<{
    appId: string;
    // detail: null | TopologyNodeDetail;
    envName: string;
    loading: boolean;
    nodeData: null | (TopologyNodeDetail & TopoNodeData);
  }>();

  const { t } = useI18n();

  /** 资源使用相关的 key 集合 */
  const RESOURCE_USAGE_KEYS = new Set(['cpu', 'cpuUsage', 'memory', 'memoryUsage']);

  /** extras 字段的中文映射 */
  const extrasLabelMap: Record<string, string> = {
    image: t('镜像'),
    ip: 'Pod IP',
    readyReplicas: t('就绪副本数'),
    replicas: t('期望副本数'),
    ports: t('端口'),
    selector: t('选择器'),
    clusterIP: 'Cluster IP',
    type: t('服务类型'),
    host: t('域名'),
    qosClass: 'QoS',
    nodeName: t('节点'),
    restartCount: t('重启次数'),
  };

  /** 资源使用字段映射 */
  // const resourceUsageLabelMap: Record<string, string> = {
  //   cpu: 'CPU',
  //   cpuUsage: 'CPU',
  //   memory: t('内存'),
  //   memoryUsage: t('内存'),
  // };

  /** 过滤出非资源使用类的 extras */
  const extrasEntries = computed(() => {
    const extras = props.nodeData?.extras;
    if (!extras) return [];
    return Object.entries(extras).filter(
      ([key, value]) => !RESOURCE_USAGE_KEYS.has(key) && value !== undefined && value !== '',
    );
  });

  /** 资源使用类的 extras */
  // const resourceUsageEntries = computed(() => {
  //   const extras = props.detail?.extras;
  //   if (!extras) return [];
  //   return Object.entries(extras).filter(
  //     ([key, value]) => RESOURCE_USAGE_KEYS.has(key) && value !== undefined && value !== '',
  //   );
  // });

  const conditions = computed(() => props.nodeData?.conditions ?? []);
</script>
