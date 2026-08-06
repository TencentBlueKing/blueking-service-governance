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

<script setup lang="ts">
  import { toRefs } from 'vue';

  import Layout from './skeleton-layout';

  const props = withDefaults(
    defineProps<{
      loading?: boolean;
      rows?: number;
    }>(),
    {
      loading: true,
      rows: 6,
    },
  );

  const { rows, loading } = toRefs(props);
</script>
<template>
  <div v-if="loading">
    <div class="grid w-full bg-[#ffffff]">
      <div class="grid grid-cols-1 gap-[26px]">
        <Layout.shape
          v-for="(_, index) in rows"
          :key="index"
          class="rounded-sm"
          :height="index === 0 ? '44px' : '20px'"
          type="rect"
          width="100%"
        />
      </div>
    </div>
  </div>
  <div v-else>
    <slot></slot>
  </div>
</template>
