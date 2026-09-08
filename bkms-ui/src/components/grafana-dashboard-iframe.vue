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
    class="h-full w-full"
    :loading="isLoading"
  >
    <iframe
      v-if="iframeUrl"
      allow="fullscreen"
      allowfullscreen
      frameborder="0"
      height="100%"
      :src="iframeUrl"
      width="100%"
      @error="isLoading = false"
      @load="isLoading = false"
    >
    </iframe>
  </Loading>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Loading } from 'bkui-vue';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import { useSpaceStore } from '~/stores/space';

  const props = defineProps<{
    url: string;
  }>();

  const spaceStore = useSpaceStore();
  const bizId = ref('');
  const isLoading = ref(false);

  const iframeUrl = computed(() => {
    if (!props.url || !bizId.value) return '';

    try {
      const baseUrl = `${import.meta.env.BK_MONITOR}`.replace(/\/$/, '');
      const url = new URL(props.url, `${baseUrl}/`);
      url.searchParams.set('bizId', bizId.value);
      url.searchParams.set('pure', '1');
      return url.href;
    } catch {
      return '';
    }
  });

  async function fetchBizId() {
    const workspaceData = await ApiServerService.GetWorkspace({
      workspaceID: spaceStore.currentSpace,
    }).catch(() => null);
    bizId.value = workspaceData?.bkSystems?.bkMonitorProjectID || '';
  }

  watch(
    iframeUrl,
    url => {
      isLoading.value = Boolean(url);
    },
    { immediate: true },
  );

  watch(
    () => spaceStore.currentSpace,
    space => {
      bizId.value = '';
      if (space) {
        fetchBizId();
      }
    },
    { immediate: true },
  );
</script>
