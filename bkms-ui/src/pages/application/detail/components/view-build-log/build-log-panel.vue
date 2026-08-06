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
  <div class="flex h-full flex-col p-24px">
    <BuildTips
      :build-info="props.buildInfo"
      class="shrink-0"
      @detail="handleGotoPipelineDetail"
    />
    <InstanceLog
      class="build-log-panel min-h-0 flex-1"
      :loading="loading"
      :logs="logs"
      :title="$t('构建日志')"
      @download="downloadLogs"
      @refresh="fetchLogs"
    />
  </div>
</template>

<script lang="ts" setup>
  import { watch } from 'vue';

  import { gotoPipelineDetail } from '~/common/util';
  import InstanceLog from '~/pages/application/detail/deploy/instance-list/components/instance-log.vue';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';

  import BuildTips from './build-tips.vue';
  import { useBuildLog } from './use-build-log';

  import type { BuildLogPanelProps } from './type';

  const props = defineProps<BuildLogPanelProps>();
  const appDetailStore = useAppDetail();
  const spaceStore = useSpaceStore();

  /** 日志请求、下载及连接回收统一由 Hook 管理。 */
  const { downloadLogs, fetchLogs, loading, logs, resetLogs } = useBuildLog(() => ({
    appID: appDetailStore.appID,
    buildID: props.buildInfo.buildID,
  }));

  /** 跳转到当前构建对应的蓝盾流水线。 */
  function handleGotoPipelineDetail() {
    const bkciProjectId = spaceStore.workspaceDetail?.bkSystems?.bkCIProjectID || '';
    gotoPipelineDetail(bkciProjectId, props.buildInfo.pipelineID, props.buildInfo.buildID);
  }

  watch(
    [() => props.active, () => props.buildInfo.buildID],
    ([active, buildID]) => {
      if (active && buildID) {
        fetchLogs();
      } else {
        resetLogs();
      }
    },
    { immediate: true },
  );
</script>

<style scoped>
  .build-log-panel :deep(.bk-task-log-scroll) {
    font-family: Consolas, Monaco, 'Courier New', monospace;
  }

  .build-log-panel :deep(.log-item) {
    white-space: pre-wrap;
  }
</style>
