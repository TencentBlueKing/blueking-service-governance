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
  <div class="px-[24px] py-[20px] h-full overflow-auto">
    <Skeleton
      :full-height="false"
      :loading="isLoading || appDetailStore.loading"
    >
      <template #loading>
        <Layout.shape
          :height="28"
          width="100%"
        />
        <div class="my-[16px] pl-[16px]">
          <Layout.shape
            class="mt-[12px]"
            :height="32"
            :width="240"
          />
          <Layout.shape
            class="mt-[12px] mx-[16px]"
            :height="32"
          />
          <Layout.shape
            class="mt-[12px]"
            :height="32"
            :width="110"
          />
          <Layout.table class="mt-[12px] pb-20px" />
        </div>
      </template>
      <!-- 环境变量 -->
      <AppEnvVariableManagement />
    </Skeleton>
  </div>
</template>

<script setup lang="ts">
  import { onBeforeMount, ref } from 'vue';

  import Layout from '~/components/skeleton/skeleton-layout';
  import AppEnvVariableManagement from '~/pages/application/detail/base-info/trpc/app-env-variable-management.vue';
  import { useAppDetail } from '~/stores/app-detail';

  const appDetailStore = useAppDetail();

  /**
   * 获取应用数据
   */
  const isLoading = ref(false);
  async function getData() {
    isLoading.value = true;
    try {
      await appDetailStore.fetchAppDetail();
    } finally {
      isLoading.value = false;
    }
  }

  onBeforeMount(() => {
    getData();
  });
</script>
