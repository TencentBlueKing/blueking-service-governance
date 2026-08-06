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
  import Layout from './skeleton-layout';

  const props = withDefaults(
    defineProps<{
      loading?: boolean;
    }>(),
    {
      loading: true,
    },
  );
</script>
<template>
  <template v-if="props.loading">
    <div class="flex flex-col w-full p-[20px]">
      <Layout.shape
        class="mb-[20px] h-[16px] w-[38%]"
        type="rect"
      />
      <Layout.shape
        class="mb-[16px] h-[16px]"
        type="rect"
      />
    </div>
  </template>
  <template v-else>
    <slot></slot>
  </template>
</template>
