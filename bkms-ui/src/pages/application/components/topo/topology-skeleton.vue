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
  <div class="flex flex-col h-full custom-resource-topology">
    <!-- 顶部工具栏骨架 -->
    <div class="flex items-center justify-between h-[48px] py-[8px] px-[16px] shrink-0 shadow-[0_2px_4px_0_#0000001a]">
      <div class="flex items-center gap-[8px]">
        <Layout.shape
          :height="32"
          :width="80"
        />
        <Layout.shape
          :height="32"
          :width="80"
        />
      </div>
      <div class="flex items-center gap-[8px]">
        <Layout.shape
          :height="32"
          :width="560"
        />
        <Layout.shape
          :height="32"
          :width="32"
        />
      </div>
    </div>
    <!-- 下方主体区域骨架 -->
    <div class="flex flex-1 mt-[2px] min-h-0">
      <!-- 左侧统计面板骨架 -->
      <div class="w-[260px] py-[12px] px-[16px] flex flex-col shadow-[0_2px_4px_0_#0000000f]">
        <!-- 状态标题行 -->
        <div class="flex items-center justify-between h-[40px] px-[10px]">
          <Layout.shape
            :height="16"
            :width="32"
          />
          <Layout.shape
            :height="16"
            :width="16"
          />
        </div>
        <!-- 状态选择器 -->
        <Layout.shape :height="32" />
        <!-- 分类标题 -->
        <div class="flex items-center h-[40px] px-[10px] mt-[4px]">
          <Layout.shape
            :height="16"
            :width="48"
          />
        </div>
        <!-- 资源行 -->
        <div
          v-for="i in 3"
          :key="i"
          class="flex items-center justify-between h-[32px] px-[10px]"
        >
          <Layout.shape
            :height="16"
            :width="80"
          />
          <Layout.shape
            :height="20"
            :width="24"
          />
        </div>
        <!-- 分类标题 -->
        <div class="flex items-center h-[40px] px-[10px] mt-[4px]">
          <Layout.shape
            :height="16"
            :width="32"
          />
        </div>
        <!-- 资源行 -->
        <div class="flex items-center justify-between h-[32px] px-[10px]">
          <Layout.shape
            :height="16"
            :width="72"
          />
          <Layout.shape
            :height="20"
            :width="24"
          />
        </div>
        <!-- 分类标题 -->
        <div class="flex items-center h-[40px] px-[10px] mt-[4px]">
          <Layout.shape
            :height="16"
            :width="32"
          />
        </div>
        <!-- 资源行 -->
        <div class="flex items-center justify-between h-[32px] px-[10px]">
          <Layout.shape
            :height="16"
            :width="88"
          />
          <Layout.shape
            :height="20"
            :width="24"
          />
        </div>
      </div>
      <!-- 右侧拓扑图区域骨架 -->
      <div class="flex-1 flex items-center justify-center">
        <div class="flex flex-col items-center gap-[16px]">
          <Layout.shape
            :height="120"
            type="circle"
            :width="120"
          />
          <Layout.shape
            :height="16"
            :width="200"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import Layout from '~/components/skeleton/skeleton-layout';
</script>
