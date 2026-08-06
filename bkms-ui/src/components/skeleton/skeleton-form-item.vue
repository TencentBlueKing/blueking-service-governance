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
      itemHeight?: number;
      itemWidth?: number;
      labelHeight?: number;
      labelWidth?: number;
      loading?: boolean;
    }>(),
    {
      loading: true,
      labelWidth: 100,
      labelHeight: 30,
      itemWidth: 200,
      itemHeight: 30,
    },
  );

  const { labelWidth, labelHeight, itemWidth, itemHeight, loading } = toRefs(props);
</script>
<template>
  <template v-if="loading">
    <div class="flex">
      <Layout.shape
        class="mr-[20px]"
        :height="labelHeight"
        type="rect"
        :width="labelWidth"
      />
      <Layout.shape
        class="mr-[20px]"
        :height="itemHeight"
        type="rect"
        :width="itemWidth"
      />
    </div>
  </template>
  <template v-else>
    <slot></slot>
  </template>
</template>
