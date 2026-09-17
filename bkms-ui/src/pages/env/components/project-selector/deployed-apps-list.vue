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
  <div class="border border-[#e5e7eb] text-[#4d4f56]">
    <div
      class="grid grid-cols-2 h-[40px] items-center border-b border-b-[#e5e7eb] bg-[#f8f8f9] text-[#313238] text-[13px]"
    >
      <div class="min-w-0 px-[16px]">
        {{ $t('应用名称') }}
      </div>
      <div class="min-w-0 px-[16px]">
        {{ $t('部署状态') }}
      </div>
    </div>
    <div class="max-h-[200px] overflow-y-auto">
      <div
        v-for="app in deployedApps"
        :key="`${app.appName}-${app.appType}-${app.deployStatus}`"
        class="grid grid-cols-2 h-[40px] items-center border-b border-b-[#e5e7eb] last:border-b-0"
      >
        <div
          class="min-w-0 overflow-hidden text-ellipsis whitespace-nowrap px-[16px]"
          :title="app.appName"
        >
          {{ app.appName }}
        </div>
        <div class="min-w-0 overflow-hidden text-ellipsis whitespace-nowrap px-[16px]">
          <div class="flex w-fit items-center">
            <StatusDotIcon
              :icon="getDeployStatusInfo(app.appType, app.deployStatus).icon"
              :size="12"
            />
            <span class="text-[#4D4F56]">{{ getDeployStatusInfo(app.appType, app.deployStatus).text || '--' }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import StatusDotIcon from '~/components/status-dot-icon.vue';
  import { useDeployStatusMap } from '~/composables/use-deploy-status';

  import type { EnvAppDeployStatusOutput } from '~/@types/v1/env';

  defineProps<{
    deployedApps: EnvAppDeployStatusOutput[];
  }>();

  const { getDeployStatusInfo } = useDeployStatusMap();
</script>
