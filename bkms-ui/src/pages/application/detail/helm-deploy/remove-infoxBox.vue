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
  <div class="text-left text-[14px]">
    <div class="mb-[16px]">
      <div>
        <span class="text-[#4D4F56]">{{ t('环境') }}：</span>
        <span class="text-[#313238]">{{ displayName || curEnv }}</span>
      </div>
      <p
        v-if="laneName"
        class="mt-[8px]"
      >
        <span class="text-[#4D4F56]">{{ t('泳道') }}：</span>
        <span class="text-[#313238]">{{ laneName }}</span>
      </p>
    </div>
    <div class="px-[16px] py-[12px] bg-[#f6f7fa] rounded-[2px] text-[14px]">
      <p>{{ t('移除后将无法恢复，请谨慎操作') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
  interface Props {
    curEnv: string;
    displayName?: string;
    laneName?: string;
    t: (key: string) => string;
  }

  defineProps<Props>();
</script>
