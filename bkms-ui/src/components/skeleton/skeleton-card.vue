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

  interface CardChildSizeType {
    height: number;
    width: number;
  }

  const props = withDefaults(
    defineProps<{
      childrenSize?: CardChildSizeType[];
      count?: number;
      height?: number;
      loading?: boolean;
      width?: number;
    }>(),
    {
      loading: true,
      count: 5,
      width: 280,
      height: 150,
      childrenSize: () => [],
    },
  );

  const { count, width, height, childrenSize, loading } = toRefs(props);

  if (!childrenSize.value?.length) {
    for (let i = 0; i < count.value; i++) {
      childrenSize.value.push({ width: width.value, height: height.value });
    }
  }
</script>
<template>
  <template v-if="loading">
    <div class="flex flex-wrap w-full p-[20px]">
      <template
        v-for="index in count"
        :key="index"
      >
        <Layout.shape
          class="mr-[20px] mb-[20px]"
          :height="childrenSize[index - 1]?.height"
          type="rect"
          :width="childrenSize[index - 1]?.width"
        >
          <template #content>
            <Layout.paragraph />
          </template>
        </Layout.shape>
      </template>
    </div>
  </template>
  <template v-else>
    <slot></slot>
  </template>
</template>
