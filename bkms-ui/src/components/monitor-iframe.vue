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
  <Loading
    class="w-full h-full"
    :loading="isLoading"
  >
    <iframe
      allow="fullscreen"
      allowfullscreen
      frameborder="0"
      height="100%"
      :src="url"
      width="100%"
      @error="isLoading = false"
      @load="isLoading = false"
    >
    </iframe>
  </Loading>
</template>

<script lang="ts" setup>
  import { computed, ref } from 'vue';

  import { Loading } from 'bkui-vue';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import { objectToUrlParams } from '~/common/util';
  import { useSpaceStore } from '~/stores/space';

  interface BaseQueryParams {
    apm_submenu?: number;
    needMenu?: boolean | number;
  }

  interface IProps {
    baseQueryParams: BaseQueryParams;
    observabilityQuery: ObservabilityType; // 观测参数
    type: 'application' | 'service';
  }
  interface ObservabilityType {
    dashboardId: string;
    'filter-app_name': string;
    'filter-service_name'?: string;
    from: string;
    interval: string;
    isGroupByLimit: boolean;
    method: string;
    preciseFilter: boolean;
    queryString: string;
    refreshInterval: number;
    sceneType: string;
    timezone: string;
    to: string;
  }

  const props = defineProps<IProps>();
  const spaceStore = useSpaceStore();

  const isLoading = ref(true);
  const bizId = ref<string>('');

  const baseQueryParams = computed(() => ({
    ...props.baseQueryParams,
    bizId: bizId.value,
  }));

  const observabilityQuery = computed(() => ({
    ...props.observabilityQuery,
    sceneId: props.type === 'service' ? 'apm_service' : 'apm_application',
  }));

  const url = computed(() => {
    const baseUrl = `${import.meta.env.BK_MONITOR}`;
    const prefixParams = objectToUrlParams(baseQueryParams.value);
    const suffixParams = objectToUrlParams(observabilityQuery.value);
    return `${baseUrl}/?${prefixParams}#/apm/${props.type}?${suffixParams}`;
  });

  // 获取 bizId (bkMonitorProjectID)
  async function fetchBizId() {
    const workspaceData = await ApiServerService.GetWorkspace({
      workspaceID: spaceStore.currentSpace,
    }).catch(() => null);
    if (workspaceData) {
      bizId.value = workspaceData.bkSystems?.bkMonitorProjectID || '';
    }
  }

  fetchBizId();
</script>
